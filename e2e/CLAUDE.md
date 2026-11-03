# e2e/CLAUDE.md

BDD guidance for the Playwright e2e suite in this directory.

---

## BDD Dual-Loop TDD

Every feature increment starts from a failing Playwright scenario and is driven inward through unit-level red-green-refactor cycles.

### Outer loop

1. **Red** — Write one Gherkin scenario describing the next observable user behavior. Immediately write stub step definitions for any new steps (missing steps cause `bddgen` to abort, which blocks all other tests from running — this hides regressions). Run `npm test`. Confirm: the new scenario fails, all existing scenarios pass. Do not proceed until the failure matches intent.
2. **Inner loop** — Repeat until you believe the scenario can pass:
   - **Red** — Write the smallest failing unit test for the next missing piece. One test at a time. Run it. Confirm it fails.
   - **Green** — Write the minimum production code to pass that one test. Only what the test demands. Run it. **Commit.**
   - **Refactor** — Clean up covered code only. All tests stay green. **Commit.**
3. **Coverage gate** — Run `go test ./... -cover`. Any uncovered branch means production code was written without a test — go back to the inner loop.
4. **Green (scenario)** — Run `npm test`. Still failing? Identify the next missing piece and return to the inner loop. Passes? **Commit.**
5. **Refactor (scenario)** — Refactor across modules if needed. All tests stay green. **Commit.**
6. Repeat from step 1.

### Absolute rules

- **No production code without a red test.** Every `if`, every `return`, every error branch — driven by a failing test first. No exceptions for "wiring", "infrastructure", or "obvious" code.
- **This applies to Vue code too.** Before creating any `.vue` or `.ts` in `src/`, write a failing Vitest test first.
- **One branch, one test, one commit.** Never write a whole function and test it afterward.
- **Always run the full suite.** `go test ./...` and `npm run test:unit` at every red/green/refactor step.
- **Never skip red.** If you cannot articulate exactly why the test fails, stop.
- **Exception — external HTTP calls:** Note the gap as tech debt; do not silently accept low coverage.

---

## Conventions

**Feature files:** `e2e/features/` · **Step definitions:** `e2e/steps/`
**Smoke features:** `e2e/smoke/features/` · **Smoke steps:** `e2e/smoke/steps/`

Always run `npx bddgen` after editing a `.feature` file.

**@reset tag:** A `Before` hook calls `DELETE /admin/reset` before each `@reset` scenario. Any feature that mutates match/prediction/result stores must carry `@reset`. Auth-only features omit it and run in parallel via the `auth` Playwright project.

**Why the `app` project runs `workers: 1`:** every `@reset` scenario calls `DELETE /admin/reset` against the shared Go server. The Go server holds match/prediction/result/user state in process memory in TEST_MODE — there's no isolation between concurrent requests. Running `@reset` scenarios in parallel would cause races (Worker A seeds `m-A`, Worker B resets and wipes `m-A`, Worker A asserts on `m-A` → fails). Don't try to "fix" `workers: 1` by removing it; the right fix is per-worker server isolation, captured in BACKLOG with a 90s runtime trigger condition.

**Smoke suite:** `npm run test:smoke` runs in CI after `deploy-staging` against the live staging URL. Scope: "did the deploy come up + can users sign in + do core API endpoints respond." Scope explicitly does NOT include third-party integrations like ESPN — if ESPN is down, smoke must still pass. Don't add smoke assertions for `[data-testid="match-events"]`, `match-detail-attendance`, `home-logo`, `match-referee`, etc. (those are exercised by the local e2e suite, which now uses fixture-backed summaries via `espn.FixtureFetcher` in TEST_MODE).

**Smoke test accounts:** Before running smoke tests, create test accounts in Firebase Auth for the target environment:
- `smoke1@test.com` and `smoke2@test.com` with password from `SMOKE_TEST_PASSWORD` GitHub secret
- See [DEPLOYMENT.md - Smoke Test Accounts](../DEPLOYMENT.md#smoke-test-accounts) for setup instructions

It is not a substitute for the local e2e suite — run `npm test` locally first.

**TEST_MODE swaps live ESPN for local fixtures.** The Go server in TEST_MODE (`PORT=8092`, used by `npm test`) routes match-summary fetches to `internal/espn/testdata/summary_<matchID>.json` via `espn.FixtureFetcher` instead of calling `site.api.espn.com`. Tests that exercise event-timeline / attendance / logos / referee should seed real ESPN match IDs that have fixtures in `testdata/` (currently 761573, 761499, 761451, 761461, 761552, 401869714). Tests that don't care about summary data can use any seed ID — missing fixtures return an empty `MatchSummary` so lazy-fetch becomes a no-op.

---

## Environment

`npm test` starts two servers in parallel: the Go API server on :8092 (TEST_MODE) and `vite preview` on :8083. Playwright's `baseURL` is :8083, which proxies `/api`, `/auth`, and `/admin` to :8092 — mirroring the production architecture (Firebase Hosting → Cloud Run). Emulators must already be running (`./dev.sh` handles that). `dev.sh` and `npm test` can run simultaneously — no port conflict.

Do **not** set `GOOGLE_CLOUD_PROJECT` in the playwright `webServer` env — it triggers Firestore and breaks test isolation.
