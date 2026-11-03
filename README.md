# Crew Predictions

Public humiliation as a service for #Crew96 fandom.

Pick your scores for upcoming Columbus Crew matches, get ranked against other fans using podcast scoring formats, and be wrong in public. It's tradition.

Predictions lock at published kickoff time. No excuses.

---

## Scoring Formats

**Aces Radio**
| Outcome | Points |
|---|---|
| Exact score | +15 |
| Correct result (wrong score) | +10 |
| Flipped same scoreline (predict Crew 3–2, actual opponent 3–2 Crew) | −15 |
| Anything else | 0 |

**Upper 90 Club** — three independent points per match:
| Condition | Points |
|---|---|
| Correct match result (win/draw/loss) | +1 |
| Correct Columbus Crew goal count | +1 |
| Correct opponent goal count | +1 |

**Grouchy™** — one point for landing in the right outcome bucket:
| Columbus margin | Category |
|---|---|
| +2 or more | Win by 2+ |
| +1 | Win by 1 |
| 0 | Draw |
| −1 | Lose by 1 |
| −2 or worse | Lose by 2+ |

Predict the correct bucket: +1. Wrong bucket: 0. Exact score doesn't matter.

---

## Running Locally

**Prerequisites:** Go, Node.js, Firebase CLI

```bash
# Install dependencies
npm install

# Terminal 1 — emulators + Go server
./dev.sh

# Terminal 2 — Vue dev server (hot reload)
npm run dev
```

The macOS development environment is different enough that a few changes are required.
```bash
# Mac specific setup beyond npm install
brew install go
brew install firebase-cli
brew install java

./dev.mac.sh
```

Open `http://localhost:5173`.

**Run tests:**
```bash
go test ./...          # Go unit tests
npm run test:unit      # Vue component tests (Vitest)
npm test               # Full e2e suite (Playwright BDD, runs against local emulator)
npm run test:smoke     # Staging smoke suite (requires test accounts in Firebase Auth)
```

**Smoke test accounts:** Before running `npm run test:smoke`, create test accounts in Firebase Auth:
- `smoke1@test.com` and `smoke2@test.com` with password from `SMOKE_TEST_PASSWORD` secret
- See [DEPLOYMENT.md - Smoke Test Accounts](DEPLOYMENT.md#smoke-test-accounts) for details

**Staging smoke suite** requires `STAGING_API_KEY` and `SMOKE_TEST_PASSWORD` env vars. In CI these are GitHub secrets. Locally:
```bash
STAGING_API_KEY=... SMOKE_TEST_PASSWORD=... npm run test:smoke
```

**Google sign-in — manual verification required on staging:**
The smoke suite verifies that clicking "Sign in with Google" initiates the redirect (page navigates to Google's OAuth flow). It cannot complete the full Google sign-in automatically because Google's OAuth consent screen blocks automated browsers. After any deploy that touches the auth flow, manually verify on your staging URL:
1. Click "Sign in with Google" on desktop Chrome
2. Complete the OAuth flow
3. Confirm you land on `/matches` and your email appears in the header
4. Repeat on mobile (iOS Safari and Android Chrome)

**Critical:** If you see an auth loop (redirects back to `/login` after OAuth completes), check that `FIREBASE_AUTH_DOMAIN` is set to the `.firebaseapp.com` domain, NOT the `.web.app` domain. See [DEPLOYMENT.md - Firebase Auth Domain](DEPLOYMENT.md#firebase-auth-domain).

---

## Linting

Linters are configured for **local use only** — there's no CI gate. Run them on demand.

**Go** ([`.golangci.yml`](.golangci.yml)):

```bash
golangci-lint run ./...      # find issues
golangci-lint fmt ./...      # auto-apply gofmt + goimports
```

The linter must be at least v2.x and built with a Go runtime ≥ the project's `go.mod` toolchain (currently 1.26). Install/upgrade with:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

**Vue / TypeScript** ([`eslint.config.mjs`](eslint.config.mjs)):

```bash
npm run lint        # find issues
npm run lint:fix    # auto-fix what's autofixable (most Vue formatting rules)
```

Both linters expect a clean tree before commit. The configs are tuned to skip noisy style nags (Go: `hugeParam`/`rangeValCopy`/`paramTypeCombine`/`QF1001`; TypeScript: `no-explicit-any` is allowed in `e2e/` and test files).

---

## Status

See [BACKLOG.md](BACKLOG.md) for what's done and what's next.

See [ARCHITECTURE.md](ARCHITECTURE.md) for how the pieces fit together.

See [DEPLOYMENT.md](DEPLOYMENT.md) for deployment configuration, troubleshooting, and operational procedures.

---

## What This Is Not

- Official. The podcasts don't know this exists.
- Finished. It's a side project built for fun and to learn Go + Vue + GCP.

