#!/usr/bin/env bash
# Recreate or sync the github-actions service account's IAM bindings on
# both the prod and staging GCP projects, plus the Workload Identity
# Federation impersonation grant on the SA itself.
#
# Idempotent: gcloud add-iam-policy-binding is a no-op when the
# (member, role, condition) triple is already present, so re-running
# this script is safe.
#
# Use cases:
#   - Recovery after accidental deletion of bindings
#   - Onboarding a fresh GCP project + WIF setup
#   - Verifying current IAM matches the documented intent
#
# Companion doc: infra/wif-setup.md

set -euo pipefail

SA='github-actions@crew-predictions.iam.gserviceaccount.com'
REPO='guilfoos/crew-predictions'
PROD_PROJECT='crew-predictions'
STAGING_PROJECT='crew-predictions-staging'
PROD_PROJECT_NUMBER='525186631552'

# Roles required on the prod project. The frontend artifact bucket
# (gs://crew-predictions-frontend) and the Docker registry
# (us-east5-docker.pkg.dev/crew-predictions/...) both live in prod, so
# both staging and prod deploys access them via these prod bindings.
PROD_ROLES=(
  roles/run.admin                            # Deploy/update Cloud Run service
  roles/iam.serviceAccountUser               # Act as runtime SA during deploy
  roles/artifactregistry.writer              # Push Docker images
  roles/storage.admin                        # Read/write frontend zip bucket
  roles/firebasehosting.admin                # Deploy Hosting
  roles/firebaserules.admin                  # Deploy firestore.rules
  roles/datastore.indexAdmin                 # Deploy firestore.indexes.json
  roles/serviceusage.serviceUsageConsumer    # Firebase CLI preflight
)

# Roles required on the staging project. No artifactregistry/storage
# bindings here — those resources live in prod and are reachable via
# the prod-project bindings above.
STAGING_ROLES=(
  roles/run.admin
  roles/iam.serviceAccountUser
  roles/firebasehosting.admin
  roles/firebaserules.admin
  roles/datastore.indexAdmin
  roles/serviceusage.serviceUsageConsumer
)

grant_project_role() {
  local project=$1 role=$2
  printf '  %-40s %s\n' "${project}" "${role}"
  gcloud projects add-iam-policy-binding "${project}" \
    --member="serviceAccount:${SA}" \
    --role="${role}" \
    --condition=None \
    --quiet \
    > /dev/null
}

echo "Granting bindings on ${PROD_PROJECT}..."
for r in "${PROD_ROLES[@]}"; do
  grant_project_role "${PROD_PROJECT}" "${r}"
done

echo
echo "Granting bindings on ${STAGING_PROJECT}..."
for r in "${STAGING_ROLES[@]}"; do
  grant_project_role "${STAGING_PROJECT}" "${r}"
done

echo
echo "Granting workloadIdentityUser on the SA itself (allow GitHub OIDC tokens"
echo "for ${REPO} to impersonate this SA)..."
gcloud iam service-accounts add-iam-policy-binding "${SA}" \
  --project="${PROD_PROJECT}" \
  --role='roles/iam.workloadIdentityUser' \
  --member="principalSet://iam.googleapis.com/projects/${PROD_PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/attribute.repository/${REPO}" \
  --quiet \
  > /dev/null
echo "  ✓"

echo
echo "Done. The github-actions SA now has the documented bindings."
echo "Verify with: gcloud projects get-iam-policy ${PROD_PROJECT} --format=json"
