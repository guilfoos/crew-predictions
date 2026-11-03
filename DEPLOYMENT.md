# Deployment Guide

This document covers deployment configuration, troubleshooting, and operational procedures for the Crew Predictions application.

---

## Quick Start Checklist

Before deploying to production for the first time:

- [ ] Firebase project created with Hosting, Auth, Firestore
- [ ] Workload Identity Federation configured (see `infra/wif-setup.md`)
- [ ] Service account granted all required IAM roles
- [ ] Firebase Auth test accounts created (see [Smoke Test Accounts](#smoke-test-accounts))
- [ ] GCS bucket created for frontend artifacts
- [ ] GitHub secrets configured (see [Environment Secrets](#environment-secrets))
- [ ] `FIREBASE_AUTH_DOMAIN` set to `.firebaseapp.com` domain (see [Critical Configuration](#critical-configuration))

---

## Critical Configuration

### Firebase Auth Domain

**The `FIREBASE_AUTH_DOMAIN` environment variable must be the `.firebaseapp.com` domain, NOT the `.web.app` hosting domain.**

| Environment | Correct Value | Wrong Value |
|-------------|---------------|-------------|
| Staging | `subtle-canto-513501-i3.firebaseapp.com` | `subtle-canto-513501-i3.web.app` |
| Production | `crew-predictions-510902.firebaseapp.com` | `crew-predictions-510902.web.app` |

**Why:** Firebase Auth validates session cookies and OAuth redirects against this domain. Using the `.web.app` domain will cause authentication loops where users are stuck on `/login` even after successful OAuth.

**Symptoms of wrong configuration:**
- User clicks "Sign in with Google" → OAuth completes → redirects back to app → immediately redirects back to `/login`
- `getGoogleRedirectResult()` returns `null` in browser console
- No errors in Cloud Run logs (the request never reaches the backend)

---

### Cloud Run Authentication

The Cloud Run service **must** be deployed with `--allow-unauthenticated` flag.

```bash
gcloud run deploy crew-predictions \
  --allow-unauthenticated \
  ...
```

**Why:** Firebase Hosting rewrites (`/api/**`, `/auth/**`, `/admin/**`) proxy requests to Cloud Run without IAM credentials. Without this flag, all API requests return **403 Forbidden**.

**Symptoms of missing flag:**
- Frontend loads correctly
- API endpoints return 403 when accessed directly or via Firebase rewrites
- `curl https://<cloud-run-url>/api/matches` returns HTML 403 page

---

## Firebase Configuration Files

The project uses three Firebase configuration files for different environments:

| File | Purpose | Public Directory | Used By |
|------|---------|------------------|---------|
| `firebase.json` | Local development with emulators | `public/` | `npm run dev` |
| `firebase.staging.json` | Staging CI/CD deployment | `dist/` | `deploy-staging` workflow |
| `firebase.prod.json` | Production CI/CD deployment | `dist/` | `deploy-prod` workflow |

**Key differences:**
- **Local (`firebase.json`)**: Uses `public/` directory for static assets during emulator testing
- **Staging/Prod (`*.json`)**: Uses `dist/` directory (Vite build output) with Cloud Run rewrites

**Common mistake:** Deploying production with `firebase.json` instead of `firebase.prod.json` results in deploying empty `public/` directory instead of the built frontend.

---

## Smoke Test Accounts

Before running smoke tests, create test accounts in Firebase Auth for each environment.

### Staging Environment

1. Go to [Firebase Console](https://console.firebase.google.com/) → select `subtle-canto-513501-i3` → Authentication → Users
2. Add user: `smoke1@test.com` with password from `SMOKE_TEST_PASSWORD` GitHub secret
3. Add user: `smoke2@test.com` with same password

### Production Environment

1. Go to [Firebase Console](https://console.firebase.google.com/) → select `crew-predictions-510902` → Authentication → Users
2. Add user: `smoke1@test.com` with password from `SMOKE_TEST_PASSWORD` GitHub secret
3. Add user: `smoke2@test.com` with same password

**Note:** Accounts must exist in the correct Firebase project. Staging accounts won't work for production tests and vice versa.

**Why test accounts are needed:** The smoke test suite includes email/password authentication scenarios that require pre-existing accounts. Google sign-in smoke tests only verify redirect initiation, not full authentication.

---

## Environment Secrets

The following GitHub repository secrets must be configured:

| Secret | Staging Value | Production Value | Notes |
|--------|---------------|------------------|-------|
| `WIF_PROVIDER` | `projects/525186631552/locations/global/workloadIdentityPools/github/providers/github-provider` | Same | WIF pool + provider path |
| `WIF_SERVICE_ACCOUNT` | `github-actions@crew-predictions-510902.iam.gserviceaccount.com` | Same | Service account email |
| `STAGING_FIREBASE_API_KEY` | (from Firebase web app) | N/A | Staging Firebase web app API key |
| `PROD_FIREBASE_API_KEY` | N/A | (from Firebase web app) | Production Firebase web app API key |
| `SMOKE_TEST_PASSWORD` | (your choice) | (same password) | Password for smoke test accounts |

**How to create Firebase web apps:**
1. Firebase Console → Project Settings → Your apps → Add app → Web
2. Register with app name (e.g., "Staging Web App", "Production Web App")
3. Copy the `apiKey`, `projectId`, `appId`, and `measurementId` from the config object

---

## Frontend Artifact Storage

Frontend builds are stored in GCS bucket `gs://crew-predictions-frontend/`:

| File | Purpose | Created By |
|------|---------|------------|
| `sha-<commit>.zip` | Build for specific commit hash | `deploy-staging` workflow |
| `latest.zip` | Most recent successful staging build | `deploy-staging` workflow |
| `prod.zip` | Last successful production deployment | `deploy-prod` workflow |
| `prod-previous.zip` | Previous production (for rollback) | `deploy-prod` workflow |

**Required permissions:** The github-actions service account needs:
- `storage.objects.create` - Upload new builds
- `storage.objects.delete` - Clean up old rollback targets
- `storage.objects.get` - Download builds for deployment

---

## First Production Deployment

The initial production deploy will show warnings about missing rollback targets:

```
sha-.zip not found in GCS — falling back to latest.zip
DEVELOP_SHA empty (squash/rebase merge?) — falling back to :latest
```

**This is expected.** There's no previous version to rollback to yet. The workflow handles this gracefully with `|| true` to continue the deploy.

After the first successful deploy:
- `prod.zip` will contain the current build
- `prod-previous.zip` will be empty/non-existent
- Subsequent deploys will properly maintain rollback targets

---

## CLOUD_TASKS_TARGET_URL

This environment variable points to the Cloud Run service URL for background task polling:

| Environment | Value |
|-------------|-------|
| Staging | `https://crew-predictions-staging-<hash>.a.run.app/admin/poll-scores` |
| Production | `https://crew-predictions-<hash>.a.run.app/admin/poll-scores` |

**Important:** This must match the Cloud Run service for the **current environment**, not the other one. Wrong configuration causes background tasks to poll the wrong environment.

**How to find your Cloud Run URL:**
```bash
gcloud run services describe crew-predictions --project=crew-predictions-510902 --format="value(status.url)"
```

---

## Troubleshooting

### Auth Loop at /login

**Symptoms:** User completes Google OAuth but is redirected back to `/login` instead of `/matches`

**Diagnosis:**
1. Check `FIREBASE_AUTH_DOMAIN` environment variable
2. Should be `.firebaseapp.com` domain, NOT `.web.app`
3. Clear browser cache and test again

**Fix:** Update `FIREBASE_AUTH_DOMAIN` and redeploy

---

### API Returns 403 Forbidden

**Symptoms:** Frontend loads, but API endpoints return 403

**Diagnosis:**
```bash
curl -s https://<cloud-run-url>/api/matches
# Returns HTML 403 page
```

**Fix:** Redeploy Cloud Run with `--allow-unauthenticated` flag

---

### Frontend Shows "Firebase Hosting Setup Complete"

**Symptoms:** Visiting the deployed URL shows the default Firebase placeholder page

**Diagnosis:**
1. Check if `latest.zip` exists in GCS: `gs://crew-predictions-frontend/latest.zip`
2. Check `firebase.prod.json` is being used in deploy step
3. Verify `public` directory in config matches build output (`dist/`)

**Fix:** Ensure `deploy-prod` uses `--config firebase.prod.json` and that file has `"public": "dist"`

---

### Smoke Tests Fail After Manual Auth Works

**Symptoms:** You can sign in manually in the browser, but smoke tests fail with timeout

**Diagnosis:**
1. Verify test accounts exist in Firebase Auth for the correct project
2. Check password matches `SMOKE_TEST_PASSWORD` GitHub secret exactly
3. Try manual sign-in with the test account credentials

**Fix:** Create/recreate the test accounts with correct password

---

## Rollback Procedure

If production deployment fails:

1. **Automatic rollback:** The `deploy-prod` workflow has a `Rollback on failure` step that runs automatically on failure
2. **Manual rollback:**
   ```bash
   gcloud run deploy crew-predictions \
     --image=us-east5-docker.pkg.dev/crew-predictions-510902/crew-predictions/crew-predictions:prod \
     --region=us-east5 \
     --update-secrets=ADMIN_KEY=admin-key:latest,SESSION_SECRET=session-secret:latest \
     --project=crew-predictions-510902
   ```

3. **Firebase Hosting rollback:**
   ```bash
   gcloud storage cp gs://crew-predictions-frontend/prod-previous.zip rollback.zip
   unzip rollback.zip -d dist/
   npx firebase deploy --only hosting --project crew-predictions-510902
   ```

---

## CI/CD Workflow

```
push to develop
    ↓
[build-and-test] → Go + Vue unit tests
    ↓
[deploy-staging] → Cloud Run + Firebase Hosting + GCS upload
    ↓
[smoke-staging] → Verify staging deployment
    ↓
create PR develop → main
    ↓
[build-and-test + smoke-staging run on PR]
    ↓
approve + merge to main
    ↓
[deploy-prod] → Cloud Run + Firebase Hosting
    ↓
[smoke-prod] → Verify production deployment
```

**Branch protection:** `main` should require PR + passing status checks before merge

**Status checks to require:**
- `build-and-test` (Go + Vue unit tests)
- `smoke-staging` (staging smoke tests)

**Do NOT require:** `smoke-prod` (runs after merge, can't be a gate)