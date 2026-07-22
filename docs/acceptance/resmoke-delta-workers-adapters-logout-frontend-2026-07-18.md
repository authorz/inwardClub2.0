# Re-smoke Delta — Workers · Adapters · Logout invalidation · Frontend contract fixes (2026-07-18)

**This is a delta, not a full checklist.** It lists *only* what must be re-verified
after the current wave lands. The baseline stays valid unless a row below changes it:

- Base backend checklist: `docs/BACKEND_ACCEPTANCE_CHECKLIST.md` (§4 order, §5 blockers).
- Last full backend sweep: `docs/acceptance/backend-smoke-2026-07-18-full-sweep.md`
  (194 checks, 0×5xx, all §5 PASS, **F1 logout gap open**).
- Remaining-gap map this wave closes: `docs/acceptance/server-not-implemented-audit-2026-07-18.md`
  (§1.A workers, §1.B adapters).

Anchor state at time of writing (**verify these still describe the *pre-wave* code, then
confirm the wave changed them**):

- Workers: `cmd/worker/main.go` registers all 10 tasks as `logHandler` no-ops.
  `outbox.Write` is called in **one** place only — `payment/settlement_repository.go:49`
  (topic `payment:post-process`). **No outbox→asynq relay and no periodic scheduler exist.**
- Adapters: only `buildObjectStore` (`app.go:215`) branches on `USE_FAKE_ADAPTERS`.
  WeChat-login / WeChat-pay / offline-acquirer are hardcoded to fakes at `app.go:91-93`;
  printer has console CRUD only, no client. `config.go:226` requires creds for Qiniu only.
- Logout: `authn/middleware.go` `RequireAuth` (L28-60) is stateless — no `token_version`
  compare. `token_version` column already exists and is bumped on logout (refresh already
  honours it). Fix is middleware/denylist only; **no migration expected**.

---

## 0. Pre-flight (do first, every run)

1. **Rebuild ALL binaries from current source** — `api`, `worker`, and any new relay/scheduler
   cmd. The 2026-07-18 partial reports failed only because a *stale binary* lagged the source
   (see `BACKEND_ACCEPTANCE_CHECKLIST.md` §7). Kill old `api`/`worker` processes on shared ports
   first; run the fresh build on a private port.
   ```
   go build -o <tmp>/api    ./cmd/api
   go build -o <tmp>/worker ./cmd/worker
   # + any new dispatcher/scheduler cmd introduced by the wave
   ```
2. **Static checks must be green before any interface test:**
   ```
   go vet ./...        go build ./...
   go test ./...       go test -race ./...
   ```
   New worker/adapter code should ship with unit tests — confirm `cmd/worker` and the adapter
   packages now have `*_test.go` (there were **none** pre-wave). Only touch the test harness if a
   build break there blocks the smoke; no product-code changes in a re-smoke.
3. **Run the async stack**, not just the API: start `cmd/worker`, and confirm whatever now
   dispatches outbox rows to asynq (a relay in `cmd/api`/`cmd/worker` or a new cmd) and whatever
   schedules the time-driven sweeps is actually running. If nothing enqueues, every §A task
   below will silently no-op regardless of handler quality.
4. **DB safety unchanged:** local dev MySQL `127.0.0.1:3307/inwardclub2` + Redis `:6379`,
   migrations `up`, seed accounts. Never export `server/.env`'s remote `MYSQL_DSN`.

---

## A. Workers — re-verify the full chain, per task

The unit under test is the **whole pipeline**, not just the handler:
`business tx → outbox_events row (or scheduled tick) → relay → asynq enqueue → handler → side effect`.
For each task: fire the trigger, then assert the observable side effect in the DB/log.
A green log line is **not** acceptance — the pre-wave handler already logged and returned nil.

| Task | Trigger to fire | Observable side effect to assert | Frontend it unblocks |
|---|---|---|---|
| `payment:post-process` | Complete a mini WeChat-pay order (or `/internal/payments/wechat/notify`) | order → paid, settlement/coin credit + ledger append, no double-credit on replay | mini pay, admin/store settlement |
| `offline-collection:expire` | Create an offline-collection order, let TTL pass | order auto-closed after timeout | store-console |
| `print:receipt` | **Producer wired:** settle a store-bound order via `SettleWeChat` / `SettleOffline` / `SettleByCoin`, with an `active` `printer_devices` row for that store | `print:receipt` outbox event (payload = `printer.Job`) written in the settlement tx → dispatched → print job issued to printer client (fake records call; real prints); recharge (no store) and stores with no active device print nothing; exactly one receipt per paid order (idem `payment:{id}:print-receipt`) | store-console |
| `reservation:expire` | Create a reservation, pass its expiry window | reservation state machine advances to expired/released | mini + store-console |
| `activity-order:expire` | Create unpaid activity order, pass TTL | order expired **and stock released** | mini |
| `ticket-coupon:expire` | Have a ticket/coupon reach its expiry | ticket/coupon marked expired | mini |
| `benefit:vip-monthly` | Trigger the monthly VIP benefit tick | benefit granted once per member per period (idempotent) | mini (depends on rule/tier map) |
| `rule:post-process` | Emit a rule-eval event | rule evaluated, resulting benefit applied | three consoles (indirect) |
| `asset:pending-cleanup` | Leave a `pending` asset past its grace | pending asset GC'd | three consoles (indirect) |
| `report:rollup` | Run the rollup tick | `reporting_daily` refreshed → `/admin/reports/*` and mini `/rankings` change | admin + mini |

Cross-cutting worker checks:
- **Enqueue actually happens.** Confirm each fired trigger writes/enqueues its topic (pre-wave only
  `payment:post-process` had an `outbox.Write`; the other 9 need enqueue or a periodic schedule).
- **Idempotency / no double side-effect** on relay retry or duplicate delivery (money + benefits especially).
- **Failure path:** a handler error must **not** ack; assert retry/backoff and that no partial side
  effect leaks. `outbox_events.status` transitions `pending → dispatched` (and `failed` on error).
- **Time-driven vs event-driven:** expiry sweeps + `report:rollup` are clock-driven — verify the
  scheduler ticks them; they won't fire from an HTTP call alone.

---

## B. Adapters — re-verify BOTH flag states

Real impls only matter if `USE_FAKE_ADAPTERS=false` actually selects them. Pre-wave, only the object
store branched; the other three were hardcoded fakes. So test the **flag matrix**:

**B1 — `USE_FAKE_ADAPTERS=true` (regression / offline).** The full-sweep baseline must still pass end
to end with zero network dependency. Fakes for WeChat login/pay, offline acquirer, printer, Qiniu.
This is the default and the CI/dev contract — a real-adapter wave must not break offline runs.

**B2 — `USE_FAKE_ADAPTERS=false` (real path).**
- **Config fail-fast:** with the flag off and creds **missing**, `config.Load()` must reject startup
  for *each* newly-real adapter (pre-wave it only required Qiniu creds — `config.go:226`). Verify
  wechat/pay/offline/printer creds are now validated too. This is testable without real creds.
- **Wiring branch:** confirm `app.go` now selects real vs fake per adapter (like `buildObjectStore`),
  i.e. L91-93 fakes are no longer hardcoded.
- **Live smoke (only if sandbox/real creds available):** WeChat login `code2session` + phone bind;
  WeChat-pay JSAPI prepay + `/internal/payments/wechat/notify`; offline aggregate collection
  (支付宝 channel, store-console); printer (芯烨云) issues a receipt. If creds are unavailable, record
  as **deferred** and rely on B1 + config fail-fast + wiring branch — do not fake a pass.

| Adapter | Fake still OK (B1) | Real selected + creds validated (B2) | Live smoke (creds-gated) |
|---|---|---|---|
| WeChat login (code2session + phone) | ☐ | ☐ | ☐ mini login / phone-binding |
| WeChat-pay JSAPI gateway | ☐ | ☐ | ☐ mini pay + notify callback |
| Offline aggregate acquirer | ☐ | ☐ | ☐ store offline-collection |
| Printer (芯烨云) | ☐ | ☐ | ☐ store receipt (pairs with `print:receipt`) |
| Qiniu object store (already real) | ☐ | ☐ (regression only) | ☐ asset upload three ends |

---

## C. Logout invalidation (F1) — the fix, per audience

F1 baseline: `POST */auth/logout` bumps `token_version` (refresh + future mints killed) but the
stateless access-token middleware never checks it, so an old **access** token still works up to the
2h TTL. The wave resolves this one of two ways — verify whichever shipped:

- **Enforced:** `RequireAuth` (or a denylist/cache) now compares `claims.TokenVersion` to the
  account's current value for access tokens. → Re-run BACKEND_ACCEPTANCE_CHECKLIST §4.4.
- **Relaxed:** §4.4 wording changed to "logout kills refresh + future issuance; existing access
  tokens expire within the 2h TTL." → Then C's pass criterion is only the refresh-kill below.

Verify (repeat for **all three** audiences — mini / admin / store; F1 reproduced identically on each):
1. Login → capture access + refresh. `GET */auth/me` (or `/mini/me`) → 200.
2. `POST */auth/logout` → 200.
3. **Same access token** → `/me`:
   - Enforced path: **401** (was 200 — the F1 fix). ← primary check
   - Relaxed path: 200 is accepted **only if** §4.4 was formally relaxed and recorded.
4. **Same refresh token** → `POST */auth/refresh` → **401/invalid** (must already hold; guard against regression).
5. Fresh login after logout → new pair works (didn't lock the account out).
6. **Regression watch (enforced path):** the added per-request `token_version` lookup must not
   break audience isolation (§5 #1) and must not add a 5xx or material latency under the sweep.
   Confirm 0×5xx and no new ERROR log lines, as in the full sweep.

---

## D. Frontend contract fixes — per console

Re-verify against the **real** Go API, not the mock, where the fix targeted real drift. Baseline for
each: `pnpm lint && pnpm typecheck && pnpm build` all green (build-approval note:
`[[pnpm-build-approval-sandbox]]`).

**D1 — mini-program** (`services/api.js` is the single server→page normalization seam):
- Flip `config/env.js` `useMock:false` against a live server and walk the smoke pages.
- Re-verify the seam fixes hold on real payloads: login token read `d.token || d` (app.js ×2,
  pages/index); `normalizeStore` (open/hours/lat-lng), `normalizeItem` (stock←stockQuantity),
  `normalizeWallet`/`normalizeLedgerEntry`.
- Confirm the **known backend gaps** still degrade gracefully, not crash: `/me` minimal (no
  tier/growth/memberNo/avatar/isStaff); tiers/rankings/invitations return lists not aggregates;
  recharge-products `amount`/`bonusAmount` (no priceCent); activity view missing ticketTypes;
  order/reservation views missing display fields; coupon redeem body shape. If the wave added any of
  these server fields, switch that page from mock-tolerant to real and re-verify.

**D2 — admin-console** (already aligned to server DTOs 2026-07-18):
- Re-verify the field renames against live `/api/v2/admin/*`: MembershipTier `threshold`;
  RechargeProduct `amount`+`bonusAmount`+`assetType`+`sortOrder`; Store `phone`/`status` + settings
  blob PUT wrapping; PaymentOrder `paymentOrderNo`/`payMethod`; Refund `refundOrderNo`/`status`
  (read-only); RuleDefinition `ruleKey`/`scopeType` + `configJson` on create; Accounts variant-driven.
- If the wave also fixed the flagged adjacent `PaymentTransactionListView` mismatch
  (transactionNo/paymentNo/payChannel vs server paymentOrderNo/payMethod/status), re-verify that page.
- Still-blocked deps to confirm are *unchanged or resolved*: refund approval state machine,
  super_admin CRUD, tiers/recharge list pagination/filter.

**D3 — store-console** (phase-1 skeleton; `src/api/http.ts` hard-blocks `/admin/*`, allows `/store/*`):
- Re-verify only `/store/*` calls flow; auth store validates `aud=store` / subject whitelist /
  non-empty `store_id`; high-risk writes auto-attach `Idempotency-Key`.
- If the wave wired the newly-mounted store reservation routes (`GET /store/reservations`,
  `/{id}`, `POST /{id}/arrive` — landed Wave 0) into the UI, smoke list/detail/arrive against live API.

---

## Recommended re-smoke order

Ordered by dependency so a failure stops the chain at its true cause:

1. **Pre-flight §0** — rebuild all binaries, static checks (`vet`/`build`/`test`/`race`), start
   api + worker + relay/scheduler. *Gate: all green, async stack confirmed running.*
2. **Baseline regression** — re-run the full-sweep happy paths under `USE_FAKE_ADAPTERS=true`
   (§B1) to prove the wave didn't regress the 194-check green baseline. *Gate: 0×5xx, §5 blockers PASS.*
3. **C — Logout invalidation** — cheapest, highest-signal, no external deps; three audiences.
   *Gate: enforced→401 on reused access token (or §4.4 formally relaxed) + refresh still killed.*
4. **A — Workers** — the biggest surface. Do the pure-DB/time-driven tasks first (`reservation:expire`,
   `activity-order:expire`, `ticket-coupon:expire`, `offline-collection:expire`, `asset:pending-cleanup`,
   `report:rollup`) since they need no real adapter; then the adapter-coupled ones
   (`payment:post-process`, `print:receipt`) after §B.
5. **B — Adapters** — B1 fake regression (folds into step 2), then B2 config fail-fast + wiring branch
   (no creds needed), then creds-gated live smoke last. Unblocks the adapter-coupled workers in step 4.
6. **D — Frontend contracts** — after the backend is green, per console D1/D2/D3 against the live API;
   re-run each console's `lint/typecheck/build`.
7. **Regression tail** — `cmd/reconcile` report still generates; sweep for new ERROR log lines /
   5xx introduced by the async stack or the middleware lookup.

**§5 hard blockers are unchanged and still absolute.** Add two wave-specific stop conditions:
worker money/benefit tasks that double-apply on retry, and (enforced path) a logout-invalidation
check that leaks across audiences or 5xxes.

## Sign-off

- [ ] §0 pre-flight green; async stack running
- [ ] §B1 fake-adapter baseline regression green (no drop vs full-sweep)
- [ ] §C logout invalidation verified (three audiences) — enforced or §4.4 relaxed & recorded
- [ ] §A each worker task's side effect observed + idempotent + failure path safe
- [ ] §B2 real-adapter wiring branch + config fail-fast; live smoke done or deferred-with-reason
- [ ] §D mini / admin / store contract fixes verified on live API; lint/typecheck/build green
- [ ] Record results in a dated `docs/acceptance/backend-smoke-<date>-*.md` and update
      `BACKEND_ACCEPTANCE_CHECKLIST.md` §7 (and §4.4 if relaxed)
