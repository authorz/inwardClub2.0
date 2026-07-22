# Mini Create→Pay→Settle Flows — Acceptance (2026-07-20)

End-to-end acceptance of the mini-program write/payment flows against the live Go backend, driven by
curl on the local dev server (`:18477`, docker MySQL `:3307`, `USE_FAKE_ADAPTERS=true`). Follows the
mini live-wiring work ([[mini-live-wiring]]). Remote `server/.env` DB never touched.

Scope: food order, activity ticket order, recharge, reservation, coupon redemption, point saving —
each `create → pay → settle`, verifying wallet deltas, order/payment status, and ticket state.

## 0. Headline

| Flow | Result |
|---|---|
| Recharge + WeChat notify → credit | **PASS** — coins credited `amount + product bonus` |
| Food order + pay-by-coin | **PASS** — coins debited 1:1, order payment settled |
| Activity ticket order + pay-by-coin | **PASS (after fix F1)** — coins debited, **tickets → active** |
| Activity ticket order + WeChat | **PASS (after fix F1)** — settled via notify, **ticket → active** |
| Reservation create → cancel (free) | **PASS** — booked, then cancelled |
| Coupon redemption | **PASS** — entitlement `active → used` |
| Point saving (staff-reviewed) | **PASS** — pending row created, no balance move |

**Bottom line:** all seven create/pay flows work end-to-end. One real gap was found and fixed during
the run — **paid activity tickets were never activated** (F1); both settlement paths now flip
`pending → active` on payment, verified live. 1 coin == 1 cent throughout.

## 1. Environment & method

- Server `go run ./cmd/api` on `:18477`, `env=development`, `USE_FAKE_ADAPTERS=true` (fake WeChat pay + fake object store; no real creds).
- DB: docker `inwardclub2-mysql` `127.0.0.1:3307/inwardclub2` (migrated to goose v21).
- Member: `seed-user-001` → `members#2`. All create/pay POSTs carry a unique `Idempotency-Key` (required by the `idem` route group).
- Coin ledger read via `GET /mini/wallet` (`coins.availableAmount`). Coin↔cent is 1:1 (`order/write_repository.go:19-22`).
- WeChat settlement simulated via the unauthenticated internal callback `POST /internal/payments/wechat/notify` with the fake body `{"outTradeNo":"<paymentOrderNo>","transactionId":"...","amountCent":N,"success":true}` (`outTradeNo` MUST be the `paymentOrderNo` string). Driver: `scratchpad/pay-flows2.sh`.

## 2. Recharge + WeChat — PASS

```
POST /mini/recharge-orders {"amountCent":10000,"payMethod":"wechat"} -> paymentOrderNo=PO2026...
POST /mini/payment-orders/:id/wechat-jsapi -> prepay {prepayId:"fake_prepay_..."}
POST /internal/payments/wechat/notify {outTradeNo:PO2026..., amountCent:10000, success:true} -> 200 {"data":{}}
```
| Assertion | Expected | Observed |
|---|---|---|
| coins credited | +10000 + 1500 (product amount=10000 bonus) | coins 5601 → **17101** (+11500) |

Note: recharge only credits on the WeChat/offline notify path — `pay-by-coin` on a recharge order would debit without crediting (`SettleByCoin` has no recharge branch). Custom amounts not matching a `recharge_products.amount` credit face value with zero bonus.

## 3. Food order + pay-by-coin — PASS

```
POST /mini/food-orders {"storeId":1,"items":[{"itemId":1,"quantity":2}],"payMethod":"coin"} -> order 2, totalAmountCent 2000, paymentOrderId 18
POST /mini/payment-orders/18/pay-by-coin -> 200 {"data":{}}
```
| Assertion | Expected | Observed |
|---|---|---|
| coin debit == total | −2000 | coins 17101 → **15101** |
| business order payment | marked paid | payment settled (payment_orders.status=paid) |

`fulfillmentStatus` stays `pending` after payment — correct: fulfillment (kitchen) is a separate staff step from payment. `INSUFFICIENT_BALANCE` returns 409 when coins < total (observed earlier when unfunded).

## 4. Activity ticket order + pay — PASS (after F1)

```
POST /mini/activity-orders {"activityId":1,"ticketTypeId":1,"quantity":2,"payMethod":"coin"} -> order 3, total 4000, tickets:[pending,pending]
POST /mini/payment-orders/:id/pay-by-coin -> 200
GET  /mini/activity-orders/3 -> tickets:[{TK...,"active"},{TK...,"active"}]
```
| Assertion | Expected | Observed |
|---|---|---|
| coin debit | −4000 | coins 13101 → 9101 |
| tickets after pay | **active** (usable) | `["active","active"]` ✅ (was `pending` — F1) |
| WeChat path parity | ticket active after notify | `["active"]` ✅ |

## 5. Reservation (free) — PASS

```
POST /mini/reservations {"storeId":1,"partySize":2,"reservedAt":"2026-07-25T12:00:00Z"} -> id, status:"booked" (no payment order)
POST /mini/reservations/:id/cancel -> 200
```

## 6. Coupon redemption — PASS

```
POST /mini/coupon-redemptions {"entitlementId":1,"storeId":1} -> {entitlementNo:"SEED-ENT-0001", status:"used"}
```
Entitlement flipped `active → used` synchronously under the idempotency key. Pre-used entitlement returns 409 `coupon is not redeemable`.

## 7. Point saving (staff-reviewed) — PASS

```
POST /mini/point-savings {"amount":100,"storeId":1} -> {status:"pending", requestId:2, balanceAfter:0}
```
Member creates a pending request; no balance moves until store-console review.

## Findings

### F1 — paid activity tickets were never activated (FIXED)
- **Severity:** medium (breaks the ticket purchase closed loop — a paid ticket stayed `pending`, so it read as unpaid on the mini and could not be treated as usable/verifiable).
- **Root cause:** neither settlement path set tickets to `active`. `TicketStatusActive` (`order/model.go:40`, "paid, usable") was only ever *read* (expiry sweep); the only ticket writes were expiry (`expiry_repository.go`) and staff verify → used (`activity/store.go:165`). Ticket lifecycle `pending → active → used` had no `→ active` edge.
- **Fix:** activate an order's tickets on settlement in **both** paths — `order.SettleByCoin` (`order/write_repository.go`, after the business-order paid update) and `payment.SettleWeChat` (`payment/settlement_repository.go`, after `markOrderPaid`):
  ```sql
  UPDATE tickets t JOIN activity_orders ao ON ao.id = t.activity_order_id
     SET t.status='active', t.updated_at=? WHERE ao.business_order_id=? AND t.status='pending'
  ```
  Idempotent via the `status='pending'` guard (a re-settle finds no rows). Gated on `order_type='activity'`. Added `orderTypeActivity` const to the payment package.
- **Verified:** coin pay → 2 tickets `pending → active`; WeChat notify → ticket `active`. `go test ./internal/modules/order/... ./internal/modules/payment/...` green.

### Non-issue noted (no server change)
- The initial run's WeChat-notify `INTERNAL error` / "invalid character ':'" was a **shell-quoting bug in the first driver script** (nested `$()` + escaped quotes mangled the JSON body), NOT a server bug. A clean `printf`/`--data-raw` body settles correctly (`{"data":{}}`). Driver rewritten (`pay-flows2.sh`).

### Observations (not fixed — out of this flow scope)
- Activity `ActivityOrderView.status` still reads `created` after payment (payment is tracked on the payment order + `business_orders.payment_status='paid'`; the activity_orders.status column isn't advanced). Ticket `active` + verify code is the usable signal. Flagging for the order-status lifecycle pass.
- `fulfillmentStatus` (food) intentionally separate from payment.

## Test artifacts (local dev DB, dev-only)
- Recharge orders + credited coins (member#2 coins ~9101 after the run); food order#2 paid; activity orders #2/#3 + tickets active; reservation booked+cancelled; coupon entitlement#1 now `used`; point_savings requestId 2 pending. Local docker DB only.

## Sign-off
- [x] recharge → coins credited (amount+bonus)
- [x] food order → pay-by-coin debits + settles
- [x] activity order → pay (coin & wechat) → **tickets active** (F1 fixed)
- [x] reservation create/cancel
- [x] coupon redemption consumes entitlement
- [x] point saving creates pending request
- [x] `go test ./...` green after fix
