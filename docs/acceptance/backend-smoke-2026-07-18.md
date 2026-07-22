# Backend Acceptance Smoke Test — 2026-07-18

Scope: read-only smoke test of `server` (Go backend, `cmd/api`) against a local dev
MySQL/Redis (docker-compose.dev.yaml, already running, left up). Migrations
applied to version 17 (`go run ./cmd/migrate up`) and dev data seeded
(`go run ./cmd/migrate seed`). API started with `HTTP_ADDR=:8081` (default
`:8080` was occupied by an unrelated docker port mapping on this host) and
stopped again at the end of the test. No `.go` files were modified; no commits
made.

Auth: `POST /api/v2/admin/auth/login` with seeded `superadmin`/`password` →
super_admin JWT. `POST /api/v2/store/auth/login` with seeded
`storeadmin`/`password` → store_admin JWT (obtained but not needed once the
admin-side store detail/settings endpoints covered the same routes).

## Endpoints exercised

| # | Endpoint | Method | Result |
|---|---|---|---|
| a | `/api/v2/admin/payment-orders` | GET | PASS (200, empty list — no seeded payment orders) |
| b | payment-orders detail by id | GET | **BLOCKED — route does not exist** |
| c | `/api/v2/admin/rule-definitions` | POST | PASS (200, created id=2) |
| d | `/api/v2/admin/rule-definitions/2/publish` | POST | PASS (200, status→published) |
| e | `/api/v2/admin/rule-definitions/2/disable` | POST | PASS (200, status→disabled) |
| f | `/api/v2/admin/error-events` | GET | PASS (200, empty list — in-memory diagnostics feed, nothing captured yet) |
| g | `/api/v2/admin/payment-channel-settings` | GET | PASS (200, wechat+offline channels, both enabled) — read-only, no PUT performed |
| h | `/api/v2/admin/stores/1` | GET | PASS (200, seeded store#1) |
| i | `/api/v2/admin/stores/1/settings` | GET | PASS (200, `{"settings":{}}`) |

## Evidence excerpts

- a: `{"data":[],"meta":{"page":1,"pageSize":20,"total":0}}` (200)
- b: `GET /api/v2/admin/payment-orders/1` → gin's bare `404 page not found` (not
  the app's JSON 404 envelope), confirming no route is registered — see below.
- c: `{"data":{"id":2,"ruleKey":"SMOKE_TEST_RULE","scopeType":"global","version":1,"configJson":{"note":"smoke test"},"enabled":false,"status":"draft",...}}` (200)
- d: `{"data":{...,"enabled":true,"status":"published",...}}` (200)
- e: `{"data":{...,"enabled":false,"status":"disabled",...}}` (200)
- f: `{"data":[],"meta":{"page":1,"pageSize":20,"total":0}}` (200)
- g: `{"data":[{"channel":"offline","displayName":"线下聚合收款","enabled":true},{"channel":"wechat","displayName":"微信支付","enabled":true}]}` (200)
- h: `{"data":{"id":1,"name":"示例门店","address":"示例地址 1 号","businessHours":"10:00-22:00","status":"active"}}` (200)
- i: `{"data":{"settings":{}}}` (200)

## Regressions / gaps found

1. **Payment-orders detail endpoint is not wired.** `internal/bootstrap/router.go`
   only registers `GET /api/v2/admin/payment-orders` (list) and
   `GET /api/v2/store/payment-orders` (list) via
   `paymentAdminHandler.ListPaymentOrders` / `paymentStoreHandler.ListPaymentOrders`.
   Neither `internal/modules/payment/admin_handler.go` nor `store_handler.go`
   defines a get-by-id handler, and no `/payment-orders/:id` route exists
   anywhere in the router. This is likely intentional/pending (a later
   milestone) rather than a broken handler, but it means "GET payment-order
   detail" cannot be smoke-tested until the route is added — flagging per the
   acceptance checklist rather than fabricating a result.
2. No other regressions observed — all other exercised endpoints returned 200
   with sane payloads and no 500s/auth failures/schema mismatches.

## Test artifact

- Created `rule_definitions` row id=2, `ruleKey=SMOKE_TEST_RULE`, taken through
  draft → published → disabled. Final state is `disabled`/`enabled=false`.
- There is no DELETE endpoint for rule-definitions (confirmed via grep of
  `router.go` and `admin/handler.go`), so "disabled" is the natural terminal
  state for a smoke-test artifact and was left as-is. It is clearly named
  (`SMOKE_TEST_RULE`) so it's easy to identify/purge later via direct DB access
  or once a delete/cleanup endpoint exists.

## Environment teardown

- The API process (`go run ./cmd/api`, port 8081) was killed after testing.
- The dev MySQL/Redis docker-compose containers (`inwardclub2-mysql`,
  `inwardclub2-redis`) were already running before this session (uptime "3
  days") and were left running, matching the project's normal dev workflow
  (`deploy/docker-compose.dev.yaml` is meant to stay up while iterating on the
  Go binary).
- No `.go` files were modified. No git commits were made. This report is the
  only file written, under `docs/acceptance/`.
