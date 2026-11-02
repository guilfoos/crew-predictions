# internal/CLAUDE.md

Go backend guidance for packages in this directory.

---

## Firestore Integration Tests

Every new method on `FirestoreUserStore` (or any Firestore-backed store) requires an integration test that round-trips through the emulator **before** writing the implementation. The e2e suite runs against `MemoryUserStore` — it will never catch a Firestore read/write bug.

- Integration tests live in `*_test.go` files with `//go:build integration`
- Run with: `FIRESTORE_EMULATOR_HOST=localhost:8091 go test -tags integration ./internal/repository/...`
- A test must exercise the full write → read cycle (e.g. `UpdateScores` then `GetByID`/`GetAll`) to catch missing fields in `toUser()`

This was learned the hard way: `UpdateScores` wrote scoring fields to Firestore correctly, but `toUser()` never read them back. Every unit and e2e test passed because they used `MemoryUserStore`. The bug only surfaced on the live staging environment.

---

## Firestore Security Rules

This app has **no client-side Firestore access**. All reads/writes go through the Go service via the Admin SDK, which bypasses security rules. The frontend never imports the Firestore web SDK — its only contract with the backend is the `/api/*` and `/auth/*` HTTP endpoints. The project ID is publicly visible (served at `/auth/config.js` for Firebase Auth), so any rule that permits client access is an external surface anyone with the project ID can hit.

**Posture: deny-all.** The deployed rules must be:

```
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /{document=**} {
      allow read, write: if false;
    }
  }
}
```

Audited 2026-05-07 via `/firestore-security-rules-auditor` — score 5/5. `firestore.rules` and `firestore.indexes.json` are checked into the repo and deployed to both projects from CI on every push. A 21-probe Vitest suite (`@firebase/rules-unit-testing`) validates deny-all behavior against the emulator on every CI run.

**When you change Firestore rules:**
1. Edit `firestore.rules` in the repo.
2. Run `/firestore-security-rules-auditor` against the diff. Address findings before deploy.
3. CI deploys to staging on push to develop and to prod on merge to main. Verify the deploy took via Firebase MCP `firebase_get_security_rules`.
4. Prod and staging rules must be identical unless there is a documented reason in the PR description.

**If a frontend Firestore SDK call is ever introduced** (e.g., real-time listeners), the deny-all rule will silently block it. Either re-architect the rules to support that specific access pattern (with a fresh audit), or refactor back to an HTTP endpoint that the Go server proxies. Do not relax deny-all without an audit.

---

## Firestore Indexes

`firestore.indexes.json` is checked in as `{"indexes": [], "fieldOverrides": []}`. Currently every server query is either a doc-ID lookup, a single-field equality (`predictions.where('MatchID', '==', x)`, auto-indexed for free), or a `GetAll()` followed by in-memory filter/sort. No compound indexes are needed today.

**When to add a real index — watch for these signals:**

1. **p95 latency on `/api/leaderboard` exceeds 500ms consistently.** Today this endpoint reads only `users.GetAll()`; predictions are pre-aggregated onto user docs by the recalculator. Latency past this threshold means user-doc count has outgrown the in-memory pattern.
2. **Firestore reads/day approaching ~50k.** Free tier covers up to 50k reads/day; past that, `GetAll()`-style scans start costing real money. Switching to bounded queries (`where season == X`, `limit(N)`) cuts read volume sharply.
3. **Cloud Run memory or CPU spikes during recalculator runs.** The recalculator scans `predictions.GetAll() + users.GetAll()`. Memory pressure here means it's time to scope to one season at a time via a `season` field on predictions.
4. **Predictions collection > 100k docs OR active users > 10k.** Even without latency issues, the in-memory recalc strategy stops scaling around these volumes.

**Minimum-effort upgrades when a signal fires:**

- *Top-N leaderboard:* add a single-field index on `users.acesRadioPoints` (free, auto-created on first query) and switch `leaderboard.go` to `users.orderBy('acesRadioPoints', 'desc').limit(100)`. Caps per-request reads regardless of total user count.
- *Per-season recalc:* add a `season` field to each prediction doc, then index it. Drop recalc cost roughly in proportion to the number of archived seasons.
- *Per-season top-N leaderboard:* this is the only path that actually requires a **compound** index — `where season == X order by acesRadioPoints desc`. Defer until both #1 and #4 fire.

When any of these triggers, capture the new query in a Go integration test before adding the index, run `firebase deploy --only firestore:indexes`, then verify via `firestore_list_indexes` that the deploy landed. Add a `BACKLOG.md` Done entry recording which signal triggered the change.
