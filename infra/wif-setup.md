# Workload Identity Federation (GitHub Actions ↔ GCP)

GitHub Actions authenticates to GCP without long-lived JSON service account keys via Workload Identity Federation (WIF). When a workflow runs, GitHub mints a short-lived OIDC token; GCP's WIF provider verifies the token's claims (specifically `repository == 'guilfoos/crew-predictions'`) and exchanges it for a Google access token bound to a service account.

This doc captures the configuration so it can be re-created or audited without poking through three cloud consoles.

## Service account

`github-actions@crew-predictions.iam.gserviceaccount.com`

Lives in the **prod** project, but holds IAM bindings on **both** projects (cross-project access). The CI workflow references it via the `WIF_SERVICE_ACCOUNT` GitHub repo secret.

## Workload Identity Pool & Provider

Both live in the **prod** project at the global location.

| Resource | Value |
| --- | --- |
| Pool ID | `github` |
| Pool full name | `projects/525186631552/locations/global/workloadIdentityPools/github` |
| Provider ID | `github-provider` |
| Provider full name | `projects/525186631552/locations/global/workloadIdentityPools/github/providers/github-provider` |
| Issuer URI | `https://token.actions.githubusercontent.com` |
| Attribute condition | `attribute.repository == 'guilfoos/crew-predictions'` |
| Attribute mapping | `attribute.repository = assertion.repository`, `google.subject = assertion.sub` |

The attribute condition is the security boundary: only workflow runs originating from this specific GitHub repo can impersonate the service account. Forking the repo or running from a different repo will fail token exchange.

## Required IAM bindings

The `github-actions@` service account holds asymmetric bindings on prod vs. staging because some GCP resources (the frontend GCS bucket and the Docker registry) live only in the prod project and are accessed cross-project. All bindings are unconditional (`--condition=None`).

The authoritative grant list lives in [`infra/scripts/grant-github-actions-roles.sh`](scripts/grant-github-actions-roles.sh) — re-run that script to recover or sync bindings idempotently. The tables below are the human-readable mirror.

### Prod project (`crew-predictions`)

| Role | Purpose |
| --- | --- |
| `roles/run.admin` | Deploy/update the prod Cloud Run service |
| `roles/iam.serviceAccountUser` | Act as the Cloud Run runtime service account during deploy |
| `roles/artifactregistry.writer` | Push Docker images to `us-east5-docker.pkg.dev/crew-predictions/crew-predictions/` |
| `roles/storage.admin` | Read/write frontend zip artifacts in `gs://crew-predictions-frontend/` (used by both prod *and* staging deploys) |
| `roles/firebasehosting.admin` | Deploy prod frontend to Firebase Hosting |
| `roles/firebaserules.admin` | Deploy `firestore.rules` to prod |
| `roles/datastore.indexAdmin` | Deploy `firestore.indexes.json` to prod |
| `roles/serviceusage.serviceUsageConsumer` | Required preflight for Firebase CLI deploys |

### Staging project (`crew-predictions-staging`)

| Role | Purpose |
| --- | --- |
| `roles/run.admin` | Deploy/update the staging Cloud Run service |
| `roles/iam.serviceAccountUser` | Act as the staging Cloud Run runtime SA during deploy |
| `roles/firebasehosting.admin` | Deploy staging frontend to Firebase Hosting |
| `roles/firebaserules.admin` | Deploy `firestore.rules` to staging |
| `roles/datastore.indexAdmin` | Deploy `firestore.indexes.json` to staging |
| `roles/serviceusage.serviceUsageConsumer` | Required preflight for Firebase CLI deploys |

Note: staging does **not** need its own `storage.admin` or `artifactregistry.writer` because the frontend bucket and Docker registry both live in the prod project — the prod-side bindings cover both deploys.

### What the deployer does NOT need (and why)

| Role | Why we don't grant it to the deployer |
| --- | --- |
| `roles/secretmanager.secretAccessor` | The deployer wires `--update-secrets=ADMIN_KEY=admin-key:latest` — Cloud Run resolves the secret at request time using the *runtime* SA. The deployer never reads the value. |
| `roles/datastore.user` | Runtime data access role. Belongs on the Cloud Run runtime SA, not the deployer. |
| `roles/firebase.admin` | Too broad — superseded by the narrower `firebasehosting.admin` + `firebaserules.admin`. |
| `roles/cloudfunctions.developer` | We don't deploy Cloud Functions through this SA. The billing-killswitch in `infra/billing-killswitch/` was deployed manually; if it's ever wired into CI, add the role then. |

## Granting / syncing roles

```bash
infra/scripts/grant-github-actions-roles.sh
```

The script is idempotent (gcloud `add-iam-policy-binding` no-ops if the binding already exists). Run it after recovery or when adding a new required role — update the script *and* the tables above in the same PR so they stay in sync.

For one-off manual grants, the first time you grant a role on a project that has any conditional binding, gcloud forces you to specify a condition. Always pass `--condition=None`:

```bash
gcloud projects add-iam-policy-binding "${PROJECT}" \
  --member="serviceAccount:github-actions@crew-predictions.iam.gserviceaccount.com" \
  --role='roles/<some-role>' \
  --condition=None
```

## Recovery: re-create WIF from scratch

If the pool or provider is deleted, GitHub Actions will fail every deploy with a 401 from the token exchange. Recovery:

```bash
PROJECT_ID='crew-predictions'
PROJECT_NUMBER='525186631552'
REPO='guilfoos/crew-predictions'

# Pool
gcloud iam workload-identity-pools create github \
  --project="${PROJECT_ID}" --location=global \
  --display-name='GitHub Actions'

# Provider
gcloud iam workload-identity-pools providers create-oidc github-provider \
  --project="${PROJECT_ID}" --location=global \
  --workload-identity-pool=github \
  --display-name='GitHub Provider' \
  --attribute-mapping='google.subject=assertion.sub,attribute.repository=assertion.repository' \
  --attribute-condition="attribute.repository=='${REPO}'" \
  --issuer-uri='https://token.actions.githubusercontent.com'

# Allow the SA to be impersonated by tokens matching the repo
gcloud iam service-accounts add-iam-policy-binding \
  "github-actions@${PROJECT_ID}.iam.gserviceaccount.com" \
  --project="${PROJECT_ID}" \
  --role='roles/iam.workloadIdentityUser' \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/attribute.repository/${REPO}"
```

Then update the GitHub repo secrets `WIF_PROVIDER` and `WIF_SERVICE_ACCOUNT` to match the new resources (the pool / provider full names from above and the SA email).

## GitHub repo secrets used by CI

| Secret | Value shape | Notes |
| --- | --- | --- |
| `WIF_PROVIDER` | `projects/525186631552/locations/global/workloadIdentityPools/github/providers/github-provider` | Pool + provider full path |
| `WIF_SERVICE_ACCOUNT` | `github-actions@crew-predictions.iam.gserviceaccount.com` | Service account email |
| `STAGING_FIREBASE_API_KEY` | `AIza...` | Public Firebase web SDK API key for staging — secret because we don't want it indexed in commit history, but it's served to clients at runtime |
| `PROD_FIREBASE_API_KEY` | `AIza...` | Same, for prod |
| `SMOKE_TEST_PASSWORD` | string | Password for the seeded staging smoke-test accounts |
