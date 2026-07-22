# Backend Acceptance — Full Route Sweep (2026-07-18)

Fresh, whole-surface acceptance sweep of the `server/` Go API (`cmd/api`) covering
**every route registered in `internal/bootstrap/router.go`** — all three audiences
(mini / admin / store) plus `/internal`. Supersedes the two earlier same-day
partial reports (`backend-smoke-2026-07-18.md`,
`backend-smoke-2026-07-18-membership-recharge.md`), whose open item was
"running binaries were stale relative to source". This run rebuilt from current
source, so that blocker no longer applies.

## Headline

- **194 request-level checks, 0× 5xx, 0× NOT_IMPLEMENTED.** Every registered
  route answered with either a success or the correct, intentional error
  envelope. The `stub()` / `NOT_IMPLEMENTED` skeleton handlers are no longer
  wired anywhere in `router.go`; the whole documented surface is implemented.
- Status distribution: `200`×149, `201`×5, `400`×17 (all expected validation),
  `404`×13 (all expected not-found / store-scope isolation), `401`×8 (all
  expected audience isolation), `409`×1 (idempotency replay), `403`×1 (asset
  purpose permission).
- All checklist §5 hard blockers **PASS**.
- **One finding (functional gap vs checklist §4.4):** logout does not invalidate
  already-issued *access* tokens — see "Findings" below. Not a §5 blocker.

## Environment & method

- **Binary:** rebuilt fresh from current source — `go build -o <jobtmp>/api-bin
  ./cmd/api` — and run on `HTTP_ADDR=:18120` (ports 8080/8099/18110 were occupied
  by pre-existing, older `api` processes belonging to other jobs; those were left
  untouched). Started before the sweep, killed after. This is the key difference
  from the earlier reports.
- **Database:** local dev docker MySQL `inward:inward@tcp(127.0.0.1:3307)/inwardclub2`
  + Redis `127.0.0.1:6379` (`deploy/docker-compose.dev.yaml`, already up ~3 days,
  left up). Migrations current at **v17** (`go run ./cmd/migrate status/up`);
  seed accounts present (login succeeded, no re-seed needed).
- **Why not `server/.env`:** `config.Load()` does **not** auto-read `.env` (no
  godotenv; only env vars + optional `CONFIG_FILE`). `server/.env`'s `MYSQL_DSN`
  points at a **remote** host (`119.29.35.118:3306`) that may be production. To
  honour "avoid destructive DB operations," the sweep used the local dev DB — the
  same target the checklist §1 and all prior acceptance runs use. The remote DSN
  was never exported and never touched. Non-secret env otherwise matched `.env`
  defaults (`USE_FAKE_ADAPTERS=true`, so WeChat/Qiniu/printer/acquirer are fakes).
- **Seed accounts:** `superadmin`/`password` (aud=admin, id 1), `storeadmin`/`password`
  (aud=store, id 2, store 1), mini member via fake `wechat/login` (aud=mini).

## Static checks (all pass)

```
go vet ./...        → clean
go build ./...      → clean
go test ./...       → ok (all module + platform packages; no failures)
go test -race ./... → ok (all packages, race detector clean)
```

## Interface coverage — PASS by area

Every row below returned `200`/`201` for the happy path and the correct error
envelope for the negative/validation path. Full transcripts are in the job's
`all_results.txt`; representative evidence excerpts follow the table.

| Area | Audience | Reads | Representative writes | Result |
|---|---|---|---|---|
| Health / ready | internal | `/internal/health`,`/ready` | — | PASS (200) |
| Auth login / me / refresh | mini/admin/store | `/me` all three | login, refresh (new pair) | PASS |
| Assets upload-credentials | mini/admin/store | — | 200 fake qiniu creds; purpose ACL 403 | PASS |
| Stores + settings + banners | mini/admin/store | list/detail/settings/banners | banner create→patch→delete | PASS |
| Catalog categories/items/variants | mini/admin/store | list/detail/variants | category/item create→update→delete | PASS |
| Activities + sessions + ticket-types | mini/admin/store | list/detail/sessions/ticket-types | (console reads; writes idem-guarded) | PASS |
| Coupons (templates/redemptions) | mini/admin/store | templates, applicable-items, redemptions | template create→update→delete | PASS |
| Membership tiers / recharge products | mini/admin | public + admin list/detail | create→patch→disable; validation 400 | PASS |
| Rankings | mini | `/rankings` (empty) | — | PASS (200) |
| Wallet + ledger | mini/admin/store | wallet, ledger (scope-filtered) | admin wallet-adjustment (+idem) | PASS |
| Sign-in / point savings / withdrawals | mini | — | sign-in +100pts; savings/withdrawal pending | PASS |
| Orders (food/recharge/activity) | mini | list/detail | create → validation 400 (wired) | PASS |
| Reservations / waitlist | mini | tables/seats/list/detail | create → validation 400 (wired) | PASS |
| Payment transactions / orders | admin/store | list + detail (404 on unknown id) | — | PASS |
| Refunds / refund-orders | admin/store | list (read alias) | — | PASS |
| Members + wallet-adjustments | admin/store | list/detail (store scope-isolated) | admin credit (+idem, replay 409) | PASS |
| Staff / cashier accounts | admin/store | list | create→patch→disable→pwd-reset | PASS |
| Store-admin / admin accounts | admin | list | — | PASS |
| Printer devices | admin/store | list | store create→patch→delete | PASS |
| Rule definitions | admin | list | create→publish→disable | PASS |
| Reports (9 endpoints) | admin | overview/revenue/…/stores | — | PASS (200, real aggregates) |
| Diagnostics error-events | admin | list (captured own 404s) | — | PASS |
| Audit logs | admin | list | — | PASS |
| Reconcile framework | CLI | `cmd/reconcile` → JSON report | — | PASS (source skipped, target counts) |

Evidence excerpts:

- `GET /api/v2/admin/reports/overview` →
  `{"storeCount":1,"memberCount":3,"orderCount":0,"grossSalesCent":0,"couponsIssued":2,"couponsRedeemed":1}`
- `POST /api/v2/mini/sign-ins` (idem key) → `201 {"date":"2026-07-18","pointsEarned":100,"streakDays":1}`;
  a second same-day sign-in with a *different* idem key returned `201` again but
  **did not double-award** — member#3 points balance stayed `100` with exactly
  one `sign_in` ledger row (daily-idempotent, verified via `/wallet/ledger`).
- `POST /api/v2/admin/members/2/wallet-adjustments` (idem) → `201 balanceAfter:101`;
  **replay with same key → `409 CONFLICT "adjustment already recorded"`** and the
  ledger appended exactly one row (100→101). Missing key → `400 IDEMPOTENCY_KEY_REQUIRED`;
  invalid direction → `400 "direction must be credit or debit"`.

## Checklist §5 hard blockers — all PASS

1. **Audience isolation.** store-token→admin, admin-token→store, mini-token→admin,
   admin-token→mini-authed, no-token, and garbage-token all → `401 UNAUTHENTICATED`.
   The three middleware instances are audience-bound (`registerMini/Admin/Store`
   each build their own `authn.NewMiddleware`).
2. **Store scope not settable by request.** `POST /api/v2/store/banners` with a
   spoofed `"storeId":999` created a banner pinned to **store 1** (from token),
   not 999. `GET /api/v2/store/members/2` and store wallet-adjustment on the
   out-of-scope member#2 → `404` (member not in store 1's scope). Scope comes only
   from `storescope.Inject()`, never the body/query.
3. **Wallet ledger append-only.** `wallet_ledger_entries` is written by `INSERT`
   only — no `UPDATE`/`DELETE` in `internal/modules/wallet/*` (`points_repository.go:151`,
   `wallet.go:193`). Runtime behaviour matches: each adjustment appends a row with
   a running `balanceAfter`; idempotent replay is rejected rather than re-applied.
4. **No Alipay fields in mini DTOs.** No `alipay`/`支付宝` token in any `*/dto.go`;
   the only reference lives in `internal/modules/payment/gateways.go` (the offline
   aggregate channel), as designed.
5. **Single unified WeChat callback.** Exactly one route:
   `POST /internal/payments/wechat/notify` (`router.go:35`).

Also: **0 HTTP 500s and 0 ERROR-level log lines** across all 194 calls.

## Findings

### F1 — Logout does not invalidate outstanding access tokens (gap vs checklist §4.4)

**Severity:** medium (functional gap; **not** a §5 release blocker).
**Status:** FAIL against checklist §4.4 ("`POST */auth/logout` 后旧 accessToken 应失效").

Reproduction (all three audiences, identical result):

```
POST /api/v2/admin/auth/logout      → 200 {"data":{}}
GET  /api/v2/admin/auth/me   (SAME access token, after logout) → 200  ← expected 401
# same for store and mini
```

**Root cause (code, not harness).** Logout *does* bump `token_version`
(`auth/service.go:144-150` → `repository.go:88/130`:
`UPDATE … SET token_version = token_version + 1`), and the **refresh** path
re-checks it (`auth/service.go:107,119`: `TokenVersion != claims.TokenVersion`),
so refresh tokens and future mints are correctly killed. But the **access-token
path** (`internal/platform/authn/middleware.go` `RequireAuth`) is fully
**stateless** — it verifies the JWT signature/audience/subject/scope and does
**no DB lookup and never compares `claims.TokenVersion`** against the account's
current value. So an access token issued before logout keeps working until it
expires naturally (access TTL = 2h).

This is the classic stateless-JWT trade-off, but it contradicts both the
acceptance criterion and the intent stated in the logout code comment
("invalidates **all** of a member's tokens"). Two reasonable resolutions (product
decision, **no code changed in this run**):

- Accept the stateless model and **relax checklist §4.4** to "logout invalidates
  refresh tokens / future issuance; existing access tokens expire within the 2h
  access TTL"; or
- Enforce it: have `RequireAuth` (or a lightweight cache/denylist) check
  `token_version` for access tokens too, at the cost of a per-request lookup.

**Resolution (2026-07-18):** the second option was taken. `RequireAuth` now
checks `token_version` for access tokens via a per-audience `TokenVersionChecker`
(members for mini, admin_accounts for admin/store); a pre-logout access token now
returns `401 session expired` on all three consoles. See the "修复（§4.4
access-token 失效）" entry in `docs/BACKEND_ACCEPTANCE_CHECKLIST.md` for the
per-request-lookup cost and the deliberate no-cache decision.

### Note — idempotency semantics are claim-and-reject, not response-replay

A repeated write with an already-used `Idempotency-Key` returns **`409 CONFLICT`**
(`idempotency.Claim` persists the key under a unique constraint;
`internal/platform/idempotency/idempotency.go`), rather than replaying the
original `2xx` body. This safely guarantees "no double side-effect" (verified: no
double ledger append), but callers retrying after a network timeout must treat a
`409` on a known key as "already applied," not as a fresh failure. Behaviour is
consistent across mini/admin/store money writes. Recorded as an observation, not
a defect.

## Test artifacts left in the local dev DB (dev-only, clearly labelled `s0718b`)

CRUD flows that have a DELETE endpoint (banners, catalog categories/items,
coupon-templates, printer-devices — admin and store) were run create→update→delete
and **left no residue**. Disable-only resources leave a labelled, disabled row:

- `membership_tiers` id=5 `s0718b-tier` (disabled); `recharge_products` id=5
  `s0718b-pack` (disabled); `rule_definitions` id=3 `s0718b_RULE` (disabled).
- admin `staff_accounts` id=3 `s0718b-staff2` (disabled); store cashier
  (admin_accounts) id=9 `s0718bcashier` (disabled); store `staff_accounts` id=4
  `s0718b-sstaff2` (disabled).
- member#2 `coins` +1 (balance 101), one `wallet_ledger_entries` append (reason `s0718b`).
- member#3: sign-in +100 `points` (one ledger row); one pending `point_savings`
  row (amount 10) and one pending `point_withdrawals` row (amount 5) awaiting
  store review; nickname set to `s0718b-nick`.
- 3 fake asset-credential rows (metadata only; `USE_FAKE_ADAPTERS`).
- `token_version` bumped for superadmin/storeadmin/two mini members (logout tests).
- These pre-date-safe artifacts sit alongside the earlier runs' `smoke-tier-01`/
  `smoke-pack-01` (id 4) and member#2's original `+100` credit.

## Environment teardown

- The fresh `api-bin` process (port 18120) was killed after the sweep.
- The other pre-existing `api` processes (ports 8080/8099/18110) and the dev
  MySQL/Redis docker containers were left as-is (they belong to other jobs / the
  normal dev workflow).
- **No `.go` source files were modified. No product code changed.** Only this
  report and the checklist §7 record were written; the reconcile report was
  written to `server/tmp/reconciliation-0718.json`.
