# Backend Acceptance Smoke Test — 2026-07-18 (membership-tiers / recharge-products / payment-orders / member wallet)

Scope: read/write smoke test of `server` against local dev MySQL/Redis
(`deploy/docker-compose.dev.yaml`, already running, left up). Migrations at
version 17, dev data seeded. API started with `HTTP_ADDR=:8099`, stopped at
end of test. No `.go` files modified; no commits made.

Note: an earlier same-day report (`backend-smoke-2026-07-18.md`) flagged
`GET /api/v2/admin/payment-orders/:id` as unwired. As of this run the route
**exists** (`router.go:122`, `paymentAdminHandler.GetPaymentOrder`) and returns
the app's JSON 404 envelope for an unknown id — that gap appears resolved.

## Endpoints exercised

| Endpoint | Method | Result |
|---|---|---|
| `/api/v2/admin/membership-tiers` | GET | PASS (200, 3 seeded tiers) |
| `/api/v2/admin/membership-tiers` | POST | PASS (200, create) |
| `/api/v2/admin/membership-tiers/:id` | GET/PATCH | PASS (200) |
| `/api/v2/admin/membership-tiers/:id/disable` | POST | PASS (200 disable; 404 on unknown id) |
| `/api/v2/admin/membership-tiers` | POST (missing name) | PASS (400 INVALID_ARGUMENT) |
| `/api/v2/admin/recharge-products` | GET | PASS (200, 3 seeded products) |
| `/api/v2/admin/recharge-products` | POST | PASS (200, create) |
| `/api/v2/admin/recharge-products/:id` | GET/PATCH | PASS (200) |
| `/api/v2/admin/recharge-products/:id/disable` | POST | PASS (200 disable; 404 on unknown id) |
| `/api/v2/admin/recharge-products` | POST (amount=0) | PASS (400 INVALID_ARGUMENT) |
| `/api/v2/admin/payment-orders` | GET | PASS (200, empty — no seeded orders) |
| `/api/v2/admin/payment-orders/:id` | GET | PASS (404 JSON envelope on unknown id; route confirmed wired) |
| `/api/v2/store/payment-orders` | GET | PASS (200, empty) |
| `/api/v2/admin/members/:id` | GET | PASS (200, seeded member#2 + wallet) |
| `/api/v2/store/members/:id` | GET | PASS (404 — member#2 has no `business_orders` at store#1; correct store-scope isolation per checklist §9) |
| `/api/v2/admin/members/:id/wallet-adjustments` | POST | PASS (201 credit applied; balance reflected immediately on re-GET) |
| — same, missing `Idempotency-Key` | POST | PASS (400 IDEMPOTENCY_KEY_REQUIRED) |
| — same, invalid `direction` | POST | PASS (400, "direction must be credit or debit") |
| `/api/v2/store/members/:id/wallet-adjustments` | POST (out-of-scope member) | PASS (404, store isolation holds) |
| cross-audience: store token → admin route | GET | PASS (401, audience rejected) |

## Pass/fail summary

All 19 exercised checks passed. No 500s, no schema mismatches, no auth bypass.

## Regressions found

None. Store/admin scope isolation, idempotency enforcement, and validation on
membership-tiers and recharge-products all behave as specified. The
previously-flagged missing payment-order detail route is now present and
working.

## Test artifacts left in DB (seed-safe, dev-only)

- `membership_tiers` id=4 `smoke-tier-01`, disabled.
- `recharge_products` id=4 `smoke-pack-01`, disabled.
- member#2 wallet `coins` balance +100 (one wallet_ledger append), reason
  "smoke test credit".

## Environment teardown

- API process (port 8099) killed after testing.
- MySQL/Redis docker-compose containers left running (pre-existing dev state).
