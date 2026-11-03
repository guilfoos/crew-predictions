# Firebase Authentication

Catalog of Firebase Auth configuration that lives in the Firebase Console for both `crew-predictions` (prod) and `crew-predictions-staging`. Firebase doesn't have a clean export-to-file tool for these; this doc is the source of truth — keep it in sync when changing console settings.

## Sign-in providers

Both projects must have these providers enabled, with identical settings:

| Provider           | Enabled? | Notes                                                                                                                   |
| ------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------- |
| Email / Password   | Yes      | Used by the e2e suite (smoke tests sign in with seeded accounts) and any user who chooses email signup. Email link sign-in NOT enabled — keep it that way unless we add UI for it. |
| Google             | Yes      | Web OAuth client configured per project. Frontend uses `signInWithRedirect` (NOT popup) for compatibility with Firebase Hosting auth domain. |
| Anonymous          | No       | Guest predictions use localStorage instead — see `src/guestPredictions.ts`. Do not enable.                              |
| Phone / SMS        | No       | Costs money, no use case.                                                                                               |
| Other (Apple, Facebook, Twitter, GitHub, Microsoft, Yahoo) | No | None of these are supported.                                                                            |

## Authorized domains

The Firebase Auth provider redirect flow only accepts callbacks from these domains. Any new domain (custom domain, preview channel, etc.) must be added explicitly.

### Production (`crew-predictions`)
- `crew96predictions.fans` (custom domain)
- `api.crew96predictions.fans` (custom domain for API)
- `crew-predictions.web.app` (Firebase default, still authorized)
- `localhost` (for local dev only — leaving this enabled in prod is intentional)

### Staging (`crew-predictions-staging`)
- `crew-predictions-staging.web.app`
- `crew-predictions-staging.firebaseapp.com`
- `localhost`

If a Google sign-in flow ever returns `auth/unauthorized-domain`, the calling domain isn't in the list above. Add it via:
**Firebase Console → Authentication → Settings → Authorized domains**.

## OAuth client (Google provider)

The Google sign-in provider uses a Web OAuth 2.0 client per project. Client IDs are public (visible in `/auth/config.js`); client secrets are stored by Firebase and accessed only by the Auth backend.

If the OAuth client is ever regenerated:
1. **Firebase Console → Authentication → Sign-in method → Google → Web SDK configuration** — copy the new client ID and secret.
2. Update both projects' GCP OAuth consent screens to include any newly added authorized domains.
3. Verify staging end-to-end (sign in via Google) before touching prod.

The **`authDomain`** in the Firebase web SDK config (set via `FIREBASE_AUTH_DOMAIN` env var on Cloud Run) must match your deployed domain exactly:

- **Production:** `crew96predictions.fans` (custom domain)
- **Staging:** `subtle-canto-513501-i3.firebaseapp.com`

**Historical note:** We previously learned that `*.firebaseapp.com` works correctly while `*.web.app` caused Google sign-in failures in some browsers. We've now migrated to a custom domain which avoids both pitfalls.

## Email templates

Firebase Console → Authentication → Templates. Default English templates are in use; we have not customized:

- Email address verification
- Password reset
- Email address change

If we ever rebrand or want a more #Crew96 voice in these emails, customize via the console and capture the templates here.

## SMS region policy

Not configured (we don't use phone auth). If we ever enable phone sign-in, restrict to specific country codes via:

```bash
# Allow only US, CA, MX initially
gcloud identity-toolkit projects update --project=crew-predictions \
  --sms-region-policy='ALLOW_BY_DEFAULT' \
  --sms-allowed-regions=US,CA,MX
```

## Recovery: how to recreate auth from scratch

If the Auth config is wiped (accidental delete, project rebuild):

1. Re-enable Email/Password and Google providers (settings above).
2. Re-add every domain in **Authorized domains**.
3. For Google provider: link to the existing OAuth client in the GCP project (or recreate it via **GCP Console → APIs & Services → Credentials**), then paste the client ID + secret into the Firebase Console.
4. Verify each user record migrated correctly via Firestore Auth import (if user data was preserved).
5. Test sign-in flows on staging before considering prod recovery complete.
