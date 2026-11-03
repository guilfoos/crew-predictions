# Custom Domain Migration Guide

This document outlines the steps to migrate from Firebase default domains to custom domains for production use.

**Status:** Not started — requires domain purchase and DNS configuration

---

## Current URLs (Pre-Migration)

| Service | Current URL |
|---------|-------------|
| Frontend (Firebase Hosting) | `https://crew-predictions-510902.firebaseapp.com` |
| Backend (Cloud Run) | `https://crew-predictions-525186631552.us-east5.run.app` |

**Problem:** These URLs are not user-friendly for sharing with fans.

---

## Target URLs (Post-Migration)

| Service | Target Domain | Routing |
|---------|---------------|---------|
| Frontend (Firebase Hosting) | `https://crew96predictions.fans` | Direct |
| API (Cloud Run via Firebase) | `https://api.crew96predictions.fans` | Firebase → Cloud Run |

**Note:** Cloud Run custom domain mapping is not available in all regions. Instead, we use Firebase Hosting rewrites to route `/api/**` paths to Cloud Run.

---

## Prerequisites

### 1. Domain Registration

**Choose a domain registrar:**
- GoDaddy
- Namecheap
- Google Domains (being migrated to Squarespace)
- Cloudflare
- Route 53 (AWS)

**Suggested domain names:**
- `crewpredictions.com`
- `crew96predictions.com`
- `crew-predictions.com`
- `crewpredictions.io`

### 2. DNS Management Access

Ensure you have access to manage DNS records (CNAME, TXT) for your domain.

### 3. Firebase Project Access

You need **Owner** or **Editor** role on the Firebase project `crew-predictions-510902`.

### 4. GCP Project Access

You need **Owner** or **Editor** role on the GCP project `crew-predictions-510902`.

---

## Migration Steps

### Phase 1: Purchase Domain

1. Choose and purchase your domain from registrar
2. Note your DNS management URL
3. Keep domain registration active (renew annually)

### Phase 2: Firebase Hosting Custom Domain

1. **Navigate to Firebase Console**
   - Go to [Firebase Console](https://console.firebase.google.com/)
   - Select project `crew-predictions-510902`
   - Click **Hosting** in the left sidebar

2. **Add Custom Domain**
   - Click **Add custom domain**
   - Enter your domain (e.g., `crewpredictions.com`)
   - Firebase will show required DNS records

3. **Verify Domain Ownership**
   - Add TXT record to your DNS:
     ```
     Type: TXT
     Host: @ (or leave blank)
     Value: google-site-verification=<code-from-firebase>
     TTL: 1 minute (Cloudflare: use "1" or "Automatic")
     ```

4. **Configure DNS for Firebase Hosting**
   - Add CNAME record:
     ```
     Type: CNAME
     Host: www (or @ for root domain)
     Value: <your-project>.firebaseapp.com.
     TTL: 1 minute (Cloudflare: use "1" or "Automatic")
     ```

5. **Wait for SSL Provisioning**
   - Firebase automatically provisions Let's Encrypt SSL
   - This typically takes 10-30 minutes
   - Can take up to 24 hours in rare cases

6. **Verify Deployment**
   - Visit `https://crewpredictions.com`
   - Confirm site loads with valid SSL certificate

### Phase 3: Firebase Hosting API Rewrite

Instead of Cloud Run domain mapping (not available in all regions), use Firebase Hosting rewrites:

1. **Add rewrite rule to `firebase.json`**:
   ```json
   {
     "hosting": {
       "rewrites": [
         {
           "source": "/api/**",
           "run": {
             "serviceId": "crew-predictions",
             "region": "us-east5"
           }
         }
       ]
     }
   }
   ```

2. **Deploy Firebase Hosting**:
   ```bash
   firebase deploy --only hosting
   ```

3. **Verify API routing**:
   - Test: `curl https://api.crew96predictions.fans/api/matches`
   - Should route to Cloud Run service automatically

**Note:** The `api` subdomain uses the same CNAME as the main site — Firebase routes based on the path.

### Phase 4: Update Firebase Configuration

1. **Update `FIREBASE_AUTH_DOMAIN`**
   - Change from: `crew-predictions-510902.firebaseapp.com`
   - Change to: `crewpredictions.com` (your domain)

2. **Update Frontend Firebase Config**
   - In `src/firebase.ts`, update the `authDomain` field:
     ```typescript
     const firebaseConfig = {
       // ... other fields
       authDomain: 'crewpredictions.com', // Update this
       // ... other fields
     }
     ```

3. **Update CI/CD Environment Variables**
   - In GitHub Actions workflow (`ci.yml`):
     - `FIREBASE_AUTH_DOMAIN`: Update to new domain
     - `CLOUD_TASKS_TARGET_URL`: Update to `https://api.crewpredictions.com/admin/poll-scores`

### Phase 5: Update GitHub Secrets

Update or add these secrets in GitHub:

| Secret Name | New Value |
|-------------|-----------|
| `PROD_FIREBASE_API_KEY` | (unchanged) |
| `FIREBASE_AUTH_DOMAIN` | `crewpredictions.com` |
| `CLOUD_TASKS_TARGET_URL` | `https://api.crewpredictions.com/admin/poll-scores` |

### Phase 6: Testing Checklist

After all changes are deployed:

- [ ] Frontend loads at custom domain with valid SSL
- [ ] Google sign-in completes successfully
- [ ] Email/password sign-in works
- [ ] API endpoints respond (`/api/matches`, `/api/leaderboard`)
- [ ] Firebase Hosting rewrites to Cloud Run for `/api/**` paths
- [ ] Mobile browsers work correctly
- [ ] No mixed-content warnings in browser console
- [ ] Session cookies persist after page refresh
- [ ] Background task polling works (check Cloud Run logs)

### Phase 7: Rollback Plan

If something breaks:

1. **Revert DNS records**
   - Remove CNAME records pointing to Firebase/Cloud Run
   - Revert to default `.firebaseapp.com` URLs

2. **Revert environment variables**
   - Set `FIREBASE_AUTH_DOMAIN` back to `.firebaseapp.com`
   - Redeploy via CI/CD

3. **Verify rollback**
   - Test at original `.firebaseapp.com` URL
   - Confirm all functionality restored

---

## DNS Records Summary

### For `crew96predictions.fans` (root domain)

```
Type: TXT
Host: @
Value: google-site-verification=<firebase-code>
TTL: 1 minute (Cloudflare: "1" or "Automatic")

Type: CNAME (or ALIAS/ANAME if supported)
Host: www
Value: crew-predictions-510902.firebaseapp.com.
TTL: 1 minute (Cloudflare: "1" or "Automatic")

Type: CNAME
Host: api
Value: crew-predictions-510902.firebaseapp.com.
TTL: 1 minute (Cloudflare: "1" or "Automatic")
```

**Note:** The `api` CNAME points to Firebase Hosting (not Cloud Run directly). Firebase routes `/api/**` paths to Cloud Run via rewrite rules.

**Note:** Cloud Run domain mapping may use a different verification method. Follow the exact instructions in Cloud Run Console.

---

## Important Notes

### DNS Propagation

- DNS changes can take **24-48 hours** to fully propagate globally
- SSL certificate provisioning can take **up to 24 hours**
- Plan accordingly — don't deploy during a match if possible

### Firebase Auth Domain

**Critical:** The `FIREBASE_AUTH_DOMAIN` must match your custom domain exactly. If it doesn't:
- OAuth redirects will fail
- Session cookies won't validate
- Users will be stuck in auth loops

### Subdomain vs Root Domain

- **Root domain** (`crewpredictions.com`): Requires ALIAS/ANAME record (not all registrars support this)
- **Subdomain** (`www.crewpredictions.com`): Standard CNAME record, works everywhere
- **Recommendation:** Start with `www.crewpredictions.com`, then add root domain redirect

### Cloud Run Domain Mapping

Cloud Run custom domains work differently than Firebase:
- Each domain/subdomain needs separate mapping
- SSL is provisioned automatically
- You can map multiple domains to the same service

---

## Cost Estimate

| Item | Cost |
|------|------|
| Domain registration (annual) | $10-20/year |
| Firebase Hosting custom domain | Free |
| Cloud Run custom domain | Free |
| SSL certificates | Free (Let's Encrypt) |

**Total:** ~$15-25/year for domain registration

---

## References

- [Firebase Hosting Custom Domains](https://firebase.google.com/docs/hosting/custom-domain)
- [Cloud Run Custom Domains](https://cloud.google.com/run/docs/mapping-custom-domains)
- [DNS Records Explained](https://www.cloudflare.com/learning/dns/dns-records/)

---

## Timeline Estimate

| Phase | Duration | Notes |
|-------|----------|-------|
| Domain purchase | 5 minutes | Immediate |
| DNS configuration | 15 minutes | Add records |
| DNS propagation | 24-48 hours | Wait time |
| SSL provisioning | 10-30 minutes | Automatic |
| Testing | 30 minutes | Thorough testing |
| **Total active time** | **~1 hour** | Plus propagation wait |

---

## Success Criteria

After migration:
- [ ] Users can access site at custom domain
- [ ] SSL certificate is valid (no browser warnings)
- [ ] All authentication flows work
- [ ] API endpoints respond correctly
- [ ] No functionality regressions
- [ ] Mobile and desktop both work

---

**Document Owner:** Brian (original author)
**Last Updated:** 2026-11-02 (fork migration)
**Status:** Ready to start — awaiting domain purchase