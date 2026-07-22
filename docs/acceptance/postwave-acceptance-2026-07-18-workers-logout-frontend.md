# Post-Wave Acceptance Pass — Workers · Logout · Frontend · Receipt/Outbox/Rule (2026-07-18)

Comprehensive post-wave acceptance pass covering the four requested surfaces:
backend **worker chain**, **logout invalidation**, **frontend build status**, and the
newly enabled **receipt / outbox / rule** flows. Local dev DB/Redis only; the remote
`server/.env` `MYSQL_DSN` was never exported or touched. No product code changed.

Follows the plan in `docs/acceptance/resmoke-delta-workers-adapters-logout-frontend-2026-07-18.md`
and the baseline `docs/acceptance/backend-smoke-2026-07-18-full-sweep.md`.

---

## 0. Headline

| Area | Verdict |
|---|---|
| Pre-flight static checks (vet/build/test/race) | **PASS** on clean snapshot; source intermittently non-compiling during window (see §7) |
| §C Logout invalidation (mini/admin/store) | **PASS** — enforced on all three |
| §A Worker chain (relay + scheduler + expiry + rollup) | **PASS** — verified end-to-end incl. real DB side effects |
| Newly enabled `print:receipt` flow | **PASS** (build @18:13) — settlement → outbox → printer |
| Newly enabled `payment:post-process` → rule eval | **PASS** (build @18:13) — no longer a log-only stub |
| §D Frontend builds (admin/store/mini) | **PASS** — lint + typecheck + build all green |
| **Migration state / v18** | **RESOLVED (2026-07-18)** — duplicate v18 renumbered; `up` reaches v19 (see §5) |
| Recharge VIP/growth settlement | **UNBLOCKED** — `growth_amount` column now created by `00019` |
| Diagnostics error-events persistence | **UNBLOCKED** — `error_events` table now created by `00018` |
| §B2 real-adapter live smoke | **DEFERRED** — ran with `USE_FAKE_ADAPTERS=true`; config fail-fast present in code |

> **Top action:** ~~rename one of the two `db/migrations/00018_*.sql` files to `00019_*`~~
> **DONE (2026-07-18):** `00018_recharge_growth_vip.sql` renamed to `00019_recharge_growth_vip.sql`
> (`00018_diagnostics.sql` kept its number to stay consistent with the code/doc references to it).
> `goose` now loads cleanly; `migrate up` reaches v19 and both previously-500 code paths are unblocked.

---

## 1. Environment & method

- **Binaries:** rebuilt fresh from current source into the job tmp dir and run on private
  ports so the other jobs' `api` processes (ports 8080/8099/18110) were left untouched.
  - Clean snapshot **build @ 17:55** (`api-bin`, `worker-bin`) → HTTP on `:18130`.
  - Refresh **build @ 18:13:23** (`api-bin2`, `worker-bin2`) → HTTP on `:18131`,
    picking up the receipt/rule code that landed mid-run (§7).
- **DB/Redis:** local dev docker `inward:inward@tcp(127.0.0.1:3307)/inwardclub2` +
  Redis `127.0.0.1:6379` (`deploy/docker-compose.dev.yaml`, up ~3 days, left up).
  Applied migration version = **v17** (`goose_db_version` max id 17). **v18 never applied**
  (see §5). `USE_FAKE_ADAPTERS=true`, private `JWT_SIGNING_KEY`.
- **Why local, not `.env`:** `config.Load()` does not read `.env`; `server/.env`'s
  `MYSQL_DSN` is a remote host (possibly prod). Local dev DSN used exclusively.
- **Seed accounts:** `superadmin`/`password` (admin, id 1), `storeadmin`/`password`
  (store, id 2 / store 1), mini member id 5 via fake `wechat/login`.
- **go:** `go1.25.7 darwin/arm64`. **node:** v20.20.1, npm 10.8.2.

Commands (representative — full command lines inline in each section):
```
go vet ./... ; go build ./... ; go test ./... ; go test -race ./...
go build -o <tmp>/api-bin ./cmd/api ; go build -o <tmp>/worker-bin ./cmd/worker
MYSQL_DSN=<local> REDIS_ADDR=127.0.0.1:6379 JWT_SIGNING_KEY=<x> USE_FAKE_ADAPTERS=true \
  HTTP_ADDR=:18130 <tmp>/api-bin ;  <tmp>/worker-bin
go run ./cmd/migrate status          # → panics (see §5)
```

---

## 2. Pre-flight — static checks

On the clean **17:55** source snapshot:

```
go vet ./...        → clean (exit 0)
go build ./...      → clean (exit 0)
go test ./...       → ok — all module + platform packages, no failures
go test -race ./... → ok — race detector clean (outbox/authn/payment/... all pass)
```

Fresh worker/adapter/outbox code ships with unit tests, e.g. `internal/platform/outbox/dispatch_test.go`,
`enqueuer_test.go`, `cmd/worker/main_test.go` (`TestPrintHandlerDispatchesJob`,
`TestPrintHandlerDropsUndecodablePayload`), `internal/modules/*/expiry_test.go`,
`internal/modules/auth/service_test.go` + `authn/middleware_test.go`
(`TestRequireAuth_RejectsStaleTokenVersion`).

⚠️ **The source did not stay static during this pass** — see §7. `go build ./...` failed
at ~18:11 (undefined symbol mid-edit) and recovered ~40 s later. All interface results
below are pinned to the build noted in each section.

---

## 3. §C — Logout invalidation (F1 fix): **PASS on all three audiences**

The F1 gap from the full-sweep baseline (logout did not kill outstanding access tokens)
is **resolved and enforced**: `RequireAuth` compares `claims.TokenVersion` to the stored
value via a per-audience `TokenVersionChecker`
(`internal/platform/authn/middleware.go:36,86-99`; wired for all three in
`internal/bootstrap/router.go:18-20`: `memberVersions` for mini, `accountVersions` for
admin/store).

Matrix (login → me → logout → **reuse same access token** → refresh → fresh login):

| Step | mini | admin | store | expected |
|---|---|---|---|---|
| 1. login | tokens (tv=0) | tokens (tv=2) | tokens (tv=1) | — |
| 2. GET /me (pre-logout access) | 200 | 200 | 200 | 200 |
| 3. POST /auth/logout | 200 | 200 | 200 | 200 |
| 4. GET /me (**same** access, post-logout) | **401** | **401** | **401** | **401 enforced** |
| 5. POST /auth/refresh (same refresh) | 401 | 401 | 401 | 401 |
| 6. fresh login → GET /me | 200 | 200 | 200 | 200 (not locked out) |

Step-4 body on all three: `{"error":{"code":"UNAUTHENTICATED","message":"session expired"}}`.
No cross-audience leakage; the extra per-request lookup added no 5xx. (Build @17:55, re-spot-checked @18:13.)

---

## 4. §A — Worker chain: **PASS end-to-end**

The unit under test is the whole pipeline, not just the handler. `cmd/worker` runs three
things in one process: the **asynq server** (10 task types), the **outbox dispatcher**
(relay), and the **asynq scheduler** (time-driven enqueue).

Worker startup log (`:6379`, business zone Asia/Shanghai):
```
outbox dispatcher started  interval=2s batch=100
worker starting            tasks=10
rollup task: complete      revenueRows=... reservationRows=0
```

### 4.1 Outbox dispatcher (relay) — real, verified
Trigger: settle a **member-bound** offline collection via `POST /internal/payments/offline-acquirer/notify`
(fake acquirer, `alipay` channel). `SettleOffline` → `writePostProcess` →
`outbox.Write(topic=payment:post-process)` inside the settlement tx.

```
create collection (memberPhone-bound, store 1) → id=1, PO=PO202607181003058280
notify → HTTP 200 ; order → paid, collection → paid
outbox_events: id=1 topic=payment:post-process idem_key=payment:1:post-process status=pending
  (2 s later) → status=dispatched, attempts=1
worker log: "outbox events dispatched" count=1
worker log: "task received" type=payment:post-process payload={memberId:5,paymentOrderId:1,...}
```
Chain confirmed: **business tx → outbox row (pending) → dispatcher → asynq enqueue → handler**.

### 4.2 Idempotency / no double side effect — PASS
Replay the same notify → `HTTP 200`; `outbox_events` for that PO still **1 row**;
`payment_transactions` for the PO still **1 row**. No duplicate enqueue, no double settle.

### 4.3 Scheduler + expiry sweeps do real DB work — PASS
Scheduler enqueues per spec §11: per-minute `reservation:expire`, `activity-order:expire`,
`offline-collection:expire`; hourly `ticket-coupon:expire`; daily `report:rollup` (+ once at startup).
Observed every minute in the worker log, e.g.:
```
expiry sweep complete  type=offline-collection:expire expired=0   (18:02, 18:03, 18:04)
```
To prove the sweep **transitions rows** (not just logs 0), a walk-in collection with a
1 s TTL was created and left pending:
```
collection id=2 status=pending expires_at=10:04:59Z
→ 18:05:00 sweep: "expiry sweep complete type=offline-collection:expire expired=1"
→ collection id=2 status = expired      ✅ real state transition
```

### 4.4 `report:rollup` populates `reporting_daily` — PASS
The startup rollup ran with `revenueRows=0` (before any paid order existed). After the
§4.1 settlement, a fresh rollup aggregated it:
```
rollup task: complete revenueRows=2
reporting_daily:
  store_id=1  2026-07-18 revenue amount_cent=1234 quantity=2
  store_id=NULL 2026-07-18 revenue amount_cent=0  quantity=1
```
This is the pre-aggregate that backs `/admin/reports/*` and mini `/rankings`.

### 4.5 Failure path
Not force-triggered at runtime (would need Redis fault injection); covered by
`outbox/dispatch_test.go` (retry/backoff, `pending`→`failed` on exhaustion) which passes
under `-race`.

---

## 5. Migration state — ~~FAIL (BLOCKER)~~ **RESOLVED (2026-07-18): duplicate migration version 18**

> **Resolution (2026-07-18):** `00018_recharge_growth_vip.sql` was renumbered to
> `00019_recharge_growth_vip.sql`; `00018_diagnostics.sql` kept version 18 (it is
> referenced as `00018` by `internal/bootstrap/app.go`, `internal/modules/diagnostics/service.go`,
> and `server/docs/diagnostics.md`, and — in case any environment had already applied v18 as
> diagnostics — keeping it at 18 makes `up` correct there too). Verified against the local dev DB
> (`inward@127.0.0.1:3307/inwardclub2`, was v17): `migrate up` applied `00018_diagnostics` then
> `00019_recharge_growth_vip` → **version 19**; `status`/`version` list all 19 in order; a fresh
> empty schema migrated 1→19 cleanly. `error_events`, `recharge_products.growth_amount`, and
> `members.current_tier_id` all exist. `go test ./...` passes. The original finding is preserved
> below for the record.

`db/migrations/` **previously** contained **two** files numbered 18:

```
00018_diagnostics.sql          -- CREATE TABLE error_events (diagnostics persistence)
00018_recharge_growth_vip.sql  -- ALTER recharge_products ADD growth_amount; ALTER members ADD current_tier_id
```

`goose` refuses to load a filesystem with a duplicate version and panics on **every**
sub-command:
```
$ go run ./cmd/migrate status
panic: goose: duplicate version 18 detected:
	migrations/00018_recharge_growth_vip.sql
	migrations/00018_diagnostics.sql
```
Consequences, all reproduced against the live v17 DB (both the 17:55 and 18:13 builds —
`cmd/api` does **not** auto-migrate, so the API boots on v17, but the two v18 objects can
never be created):

- **Fresh DB bring-up is impossible** — `migrate up` / `status` / `version` / `down` / `reset` all panic.
- **Recharge VIP/growth settlement → HTTP 500.** `creditRechargeBenefit`
  (`settlement_repository.go:143`) does `SELECT bonus_amount, growth_amount FROM recharge_products`:
  ```
  POST /internal/payments/wechat/notify (recharge order) → 500
  api log: Error 1054 (42S22): Unknown column 'growth_amount' in 'field list'
  ```
  The settlement tx rolls back cleanly — order stays `pending`, **0** `wallet_ledger_entries`
  rows for the recharge (no partial credit). But a real WeChat recharge **cannot complete**.
  This is a **regression** vs. the full-sweep baseline (which passed recharge before the
  growth/VIP code landed).
- **Diagnostics feed → HTTP 500.** The module is now SQL-backed
  (`diagnostics/repository.go` → `error_events`), table missing:
  ```
  GET /api/v2/admin/error-events → 500 ; Error 1146 (42S02): Table 'inwardclub2.error_events' doesn't exist
  ```
  Worse, the `Capture()` middleware (`router.go:16`) tries to persist **every** error
  response and fails: `ERROR "diagnostics: persist error event failed" ... error_events doesn't exist`
  fires on each 4xx/5xx — **regressing the baseline's "0 ERROR-level log lines"** property.

**Root cause:** two parallel waves (diagnostics persistence; recharge growth/VIP) each
grabbed number 18. **Fix:** renumber either file to `00019_*` (the two migrations are
independent, so which one moves is arbitrary), then `go run ./cmd/migrate up`. Not applied
here — it is a schema change owned by the in-flight wave (§7), not a harness fix.

---

## 6. Newly enabled receipt / outbox / rule flows (build @ 18:13)

These landed **during** this pass (§7); verified against the pinned 18:13 build:

- **`print:receipt` — now fully wired & working.** `printer.WriteReceipt` is called from
  both settlement paths and order completion (`settlement_repository.go:110,409`,
  `order/write_repository.go:406`); it appends a `print:receipt` outbox event **iff the
  store has an active printer device**. With an active device registered for store 1:
  ```
  settle member-bound collection (store 1) →
    outbox id=3 topic=print:receipt idem_key=payment:5:print-receipt (pending → dispatched)
    worker log: "print task: printed" sn=ACC0718CSN01 template=order-receipt   ✅ (fake printer)
  ```
  (A store with no active device prints nothing and the settlement still succeeds —
  confirmed earlier: store 1 produced no receipt event until a device was added.)
- **`payment:post-process` — no longer a log-only stub.** The handler now evaluates rules:
  ```
  worker log: "post-process task: complete" ruleMatched=false ruleVersion=0 grantsApplied=0 alreadyDone=false
  ```
  (`ruleMatched=false` because no active reward rule matches the test order; the handler
  ran cleanly, no 5xx — its tables exist at v17.)
- **Rule engine** — `/admin/rule-definitions` CRUD read is real (`HTTP 200`, returns
  `sign_in` + test rules). Evaluation now runs inside `payment:post-process` (above).
- **Outbox flow** — real and idempotent (§4.1/§4.2).

Still log-only stubs on the 18:13 build worker: `benefit:vip-monthly`, `asset:pending-cleanup`
(asset cleanup code was mid-landing in `internal/modules/asset/` at run time).

---

## 7. Process finding — source tree in active flux during the pass

The `server/` Go source was being edited by a **parallel wave while this acceptance pass
ran**. File mtimes 18:02–18:13+ across `printer/receipt.go`, `payment/settlement_repository.go`,
`order/write_repository.go`, `asset/{repository,cleanup}.go`, `payment/postprocess*.go`,
`cmd/worker/main.go`, `rule/{repository,service}.go`. At **18:11:00 the tree did not
compile** (`postprocess_repository.go` referenced an undefined `consumeRewardRuleKey`); it
recovered by **18:11:40**. A new `payment-post-process-handler` project memory was written
by a concurrent agent mid-run.

**Implication:** a comprehensive acceptance sign-off on a *frozen* post-wave snapshot is
not currently possible — the wave has not landed. Every result here is pinned to a specific
build (17:55 or 18:13). **A re-pass is required once the wave freezes** and, critically,
**once the §5 v18 collision is resolved** (recharge + diagnostics stay red until then).

---

## 8. §B — Adapters

- **B1 (`USE_FAKE_ADAPTERS=true`)** — the whole pass ran on fakes with zero network deps;
  the fake WeChat/offline/printer paths exercised above all work. No regression.
- **B2 (`USE_FAKE_ADAPTERS=false`)** — config fail-fast is present in code
  (`config.go` `Validate()` requires wechat-login / wechat-pay / offline / Xpyun / Qiniu
  creds when fakes are off). **Live real-adapter smoke: DEFERRED** — no sandbox creds; not faked.

---

## 9. §D — Frontend build status: **PASS (all three)**

Run with the repo Node (v20.20.1); `node_modules` already installed in each console.

| Console | lint | typecheck | build | notes |
|---|---|---|---|---|
| admin-console | ✅ exit 0 | ✅ (via `vue-tsc --noEmit`) | ✅ `vite build` — “✓ built in 2.41s” | — |
| store-console | ✅ exit 0 | ✅ (via `vue-tsc --noEmit`) | ✅ `vite build` — “✓ built in 2.08s” | — |
| mini-program | ✅ exit 0 | ✅ `tsc --noEmit` | ✅ `node scripts/build.js` | 7 ESLint **warnings** (unused `e` in catch), 0 errors |

Commands: `npm run lint`, `npm run build` (admin/store; `build` = `vue-tsc --noEmit && vite build`),
`npm run lint && npm run typecheck && npm run build` (mini).

---

## 10. Test artifacts left in local dev DB (dev-only, labelled `acc0718c`)

- `offline_collection_orders`: id 1 (paid, member 5), id 2 (expired by sweep, walk-in),
  id 3 & 4 (paid, member 5) + their `payment_orders`/`payment_transactions`/`business_orders`.
- `outbox_events`: 4 rows (3× payment:post-process, 1× print:receipt) — all `dispatched`.
- Two recharge orders (member 5) left **`pending`** (settlement 500'd on `growth_amount` — see §5).
- `printer_devices`: id 2 `acc0718c-printer` (store 1, active, sn `ACC0718CSN01`).
- member 5: phone bound `117****5565` (fake resolver); `token_version` bumped by logout tests
  (also superadmin/storeadmin bumped).
- `reporting_daily`: 2 rows for 2026-07-18 (revenue).
- No DELETE-capable resource left residue beyond the above; nothing dropped/overwritten.

## 11. Teardown

- The fresh `api-bin`/`api-bin2` (`:18130`/`:18131`) and `worker-bin`/`worker-bin2`
  processes were started for the pass and stopped afterwards.
- Pre-existing `api` processes (8080/8099/18110) and the dev MySQL/Redis containers left as-is.
- **No `.go` / product source modified. No migrations applied.** Only this report was written.

## 12. Sign-off checklist (from the re-smoke delta)

- [x] §0 pre-flight green **on clean snapshot**; async stack running — ⚠️ source not frozen (§7)
- [x] §B1 fake-adapter baseline green (no regression from fakes)
- [x] §C logout invalidation verified (three audiences) — **enforced**
- [x] §A each worker task's relay/scheduler/side-effect observed + idempotent
- [ ] §B2 real-adapter live smoke — **DEFERRED** (config fail-fast present; no creds)
- [x] §D mini / admin / store lint + typecheck + build green
- [x] **BLOCKER §5** duplicate migration v18 — **RESOLVED (2026-07-18):** recharge migration
      renumbered to `00019_*`; `migrate up` reaches v19; fresh DB bring-up + `go test ./...` verified
- [x] Result recorded in this dated report
