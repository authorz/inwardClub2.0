# Post-Wave Acceptance — Follow-up after v18 migration-collision fix (2026-07-18)

Re-run of the post-wave acceptance pass focused on the paths the previous report
(`postwave-acceptance-2026-07-18-workers-logout-frontend.md`) left **blocked** by the
duplicate migration version 18: **recharge VIP/growth settlement**, **diagnostics
error-events persistence**, the **worker chain**, and the **receipt / outbox / rule**
flows. Local dev DB/Redis only; the remote `server/.env` `MYSQL_DSN` was never
exported or touched. No product code or migration file was modified by this pass.

---

## 0. Headline

| Area | Prior report | This follow-up |
|---|---|---|
| Migration state / v18 collision | **FAIL — BLOCKER** | **RESOLVED** — goose at v19, `migrate status` no longer panics, fresh DB `1→19` clean |
| Recharge VIP/growth settlement | **FAIL (500, `growth_amount` missing)** | **PASS** — settles 200, coins+growth credited, VIP tier advances, idempotent |
| Diagnostics error-events persistence | **FAIL (500, `error_events` missing)** | **PASS** — events persist, admin feed 200, zero persist-fail ERROR logs |
| Worker chain (relay/scheduler/expiry/rollup) | PASS (at v17) | **PASS** — re-verified end-to-end at v19 incl. real expiry transition |
| Receipt / outbox / rule flows | PASS (build @18:13) | **PASS** — `print:receipt` + `payment:post-process` relay & idempotent |
| Runtime log health | 1 regression (ERROR per error) | **PASS** — 0 ERROR/WARN, 0 5xx, 0 panics |

**Bottom line:** the single BLOCKER from the prior pass is cleared and every
previously-blocked runtime path now passes.

> **Note on the fix itself.** The collision was **already resolved when this pass
> began** — by a concurrent actor/user, not by this pass. On disk,
> `00018_recharge_growth_vip.sql` had been renamed to `00019_recharge_growth_vip.sql`
> (diagnostics kept `00018`), and goose had applied v18+v19 to the dev DB at
> `2026-07-18 10:37:33Z`. The rename matches file creation order (diagnostics mtime
> 17:20 → v18, recharge 17:23 → v19) and is exactly the fix the prior report §5
> prescribed. This pass **validated** that fix rather than performing it.

---

## 1. Environment & method

- **Binaries:** rebuilt fresh from current source at 18:39 into the job tmp dir and run
  on a private port so other jobs' processes (8080/8099/18110) were left untouched.
  `api-bin` on `HTTP_ADDR=:18140`; `worker-bin` (asynq server + outbox dispatcher +
  scheduler). Both stopped at teardown.
- **DB/Redis:** local dev docker `inward:inward@tcp(127.0.0.1:3307)/inwardclub2`
  (`?parseTime=true`) + Redis `127.0.0.1:6379` (`deploy/docker-compose.dev.yaml`,
  up ~3 days). Applied goose version = **v19**. `USE_FAKE_ADAPTERS=true`, private
  `JWT_SIGNING_KEY`.
- **Why local, not `.env`:** `config.Load()` does not read `.env`; `server/.env`'s
  `MYSQL_DSN` points at a remote host. The local dev DSN was used exclusively.
- **Seed accounts:** `superadmin`/`password` (admin), `storeadmin`/`password`
  (store 1); fresh mini members created per-test via fake `wechat/login` (distinct codes).
- **Toolchain:** `go1.25.7 darwin/arm64`.
- **Test-data label:** everything created by this pass is tagged `acc0718d`.

### 1.1 Pre-flight — static checks (clean tree)
```
go vet ./...   → clean (exit 0)
go build ./... → clean (exit 0)
go test ./...  → ok — every module + platform package, no FAIL, no panic
```
The source tree was frozen for this pass (no `.go` mtime newer than the prior report's
18:17 sign-off).

---

## 2. Migration collision — RESOLVED (was the BLOCKER)

`db/migrations/` now holds **distinct** versions 18 and 19:
```
00018_diagnostics.sql          -- CREATE TABLE error_events
00019_recharge_growth_vip.sql  -- ALTER recharge_products ADD growth_amount; ALTER members ADD current_tier_id
```

- **`go run ./cmd/migrate status` → no panic**, lists all migrations through
  `00019_recharge_growth_vip.sql` (applied `Sat Jul 18 10:37:33 2026`). The prior
  `panic: goose: duplicate version 18 detected` is gone.
- **Dev DB objects present:** `error_events` table exists; `recharge_products.growth_amount`
  (default 0) and `members.current_tier_id` (nullable) exist. `goose_db_version` max = 19.
- **Fresh-DB bring-up (independent check):** migrated a throwaway scratch DB from empty:
  ```
  OK 00018_diagnostics.sql
  OK 00019_recharge_growth_vip.sql
  goose: successfully migrated database to version: 19
  ```
  `MAX(version_id)=19`, both v18/v19 objects present. The prior "fresh DB bring-up is
  impossible" consequence is resolved. Scratch DB dropped afterward.

---

## 3. Recharge VIP/growth settlement — PASS (was 500)

Full HTTP flow on fakes: admin creates a growth-bearing product → member recharges →
internal WeChat notify settles.

**Setup (admin API):** `POST /admin/recharge-products`
`{amount:7777, bonusAmount:300, growthAmount:5000, status:active}` → product id 6; the
API **persisted and echoed `growthAmount:5000`** (admin write path handles the new field).

**Settlement:** fresh member id 6, `POST /mini/recharge-orders {amountCent:7777,
payMethod:"wechat"}` → payment order 7 (`PO2026071810454257…`) → `POST
/internal/payments/wechat/notify {success:true}`:

| Assertion | Expected | Observed |
|---|---|---|
| notify HTTP | 200 (was **500** `Unknown column 'growth_amount'`) | **200** `{"data":{}}` |
| `payment_orders.status` | paid | **paid** |
| `payment_transactions` rows | 1 | **1** |
| coins ledger (`recharge_order:7`) | credit 7777+300=8077 | **8077** |
| growth ledger (`recharge_growth:7`) | credit 5000 | **5000** |
| `wallet_accounts` coins / growth_value | 8077 / 5000 | **8077 / 5000** |
| `members.current_tier_id` | 3 (threshold 5000 crossed, level 3) | **3** |

**Idempotency (replay same notify):** HTTP 200; ledger rows stay 2; coins 8077, growth
5000, tier 3 all unchanged; `payment_transactions` stays 1. No double credit.

**Stepwise tier progression (member id 7, two recharges):**
```
initial current_tier_id = NULL
recharge growth 1200  → growth balance 1200 → current_tier_id = 2 (threshold 1000)
recharge growth 4000  → growth balance 5200 → current_tier_id = 3 (threshold 5000)
```
Growth accrues across top-ups and the tier advances **upgrade-only** to the highest
qualifying **active** tier (the disabled level-9 tier at threshold 9999 is correctly
ignored).

---

## 4. Diagnostics error-events persistence — PASS (was 500)

The module is SQL-backed (`diagnostics/repository.go → error_events`) and the `Capture()`
middleware persists any 5xx or handler-attached (`c.Errors`) response.

Generated three tagged failures (unique `X-Request-ID` each) and read them back:

| Trigger | HTTP | Persisted row (method / path / status) |
|---|---|---|
| bad bearer token → `GET /admin/error-events` | 401 | `GET /api/v2/admin/error-events 401 UNAUTHENTICATED` |
| missing record → `GET /mini/recharge-orders/999999` | 404 | `GET /api/v2/mini/recharge-orders/:orderID 404 NOT_FOUND` |
| invalid body → `POST /mini/recharge-orders` | 400 | `POST /api/v2/mini/recharge-orders 400 INVALID_ARGUMENT` |

- `error_events` count **0 → 3**; all three tagged rows persisted with correct
  method/path/status/message and stored **newest-first** (id desc).
- **`GET /api/v2/admin/error-events` → 200**, `meta.total=3`, 3 items returned, all three
  request-ids present on the first page (was **500** `Table 'error_events' doesn't exist`).
- **Prior regression fixed:** `grep "persist error event failed"` → **0**; api.log had
  **0 ERROR-level lines** for the whole pass. The retention cap (`retentionMaxEvents=500`,
  pruned on each write) is in code and unexercised at this volume.

---

## 5. Worker chain + receipt / outbox / rule — PASS (re-verified at v19)

Trigger: a **member-bound** offline counter collection (store 1) settled via
`POST /internal/payments/offline-acquirer/notify` (fake `alipay`). Setup: store 1 has an
active printer device; member id 10 bound by fake phone; collection id 6 → payment order 11.

**Outbox relay (business tx → outbox → dispatcher → asynq → handler):**
```
immediately after notify:  outbox_events
  id=5 print:receipt        payment:11:print-receipt      status=pending
  id=6 payment:post-process payment:11:post-process       status=pending
+5s (dispatcher interval=2s):
  both → status=dispatched, attempts=1
worker log: "outbox events dispatched" count=2
            "print task: printed" template=order-receipt
            "post-process task: complete" ruleMatched=false ruleVersion=0 grantsApplied=0
```
- `offline_collection_orders.status = paid`, `payment_orders.status = paid`,
  `payment_transactions` = 1 row.
- **`print:receipt`** — active printer device present → receipt event emitted, relayed,
  printed by the fake Xpyun handler. (A store with no active device emits none; the
  settlement still succeeds.)
- **`payment:post-process`** — no longer a log-only stub: the handler evaluates reward
  rules (here `ruleMatched=false` for the test order) and completes cleanly, no 5xx.

**Idempotency (replay notify):** HTTP 200; outbox rows for this PO stay **2**;
`payment_transactions` stays **1**. No duplicate dispatch, no double settle.

**Scheduler + expiry sweeps do real DB work:** per-minute `reservation:expire`,
`offline-collection:expire`, `activity-order:expire` observed every minute. Proven with a
real transition — a walk-in collection (id 7, 1 s TTL) left pending was flipped by the
`18:55:00` sweep:
```
[18:54:48] status=pending
[18:55:04] status=expired   ✅ real pending→expired transition
```

**`report:rollup` populates `reporting_daily`:** the startup rollup logged
`rollup task: complete revenueRows=2`; `reporting_daily` holds `report_date=2026-07-18`
rows (e.g. store 1 `revenue amount_cent=2899 quantity=4`). This is the pre-aggregate
backing `/admin/reports/*` and mini `/rankings`.

---

## 6. Runtime health

- api.log: **0 ERROR**, **0 WARN**, **0 responses with status 5xx** across the whole pass
  (the only non-2xx were the three intentional 4xx diagnostics triggers).
- worker.log: **0 ERROR**.
- No `panic`, no `duplicate version`, no `persist error event failed` in either log.

---

## 7. Concurrency caveat (shared dev infra)

A **parallel actor was active on the same dev MySQL/Redis** during this pass: it applied
the v18/v19 migration at 10:37:33Z, re-created the prior pass's printer device
`ACC0718CSN01` on store 1 (so `activeDeviceSN`'s lowest-id pick printed with that SN, not
this pass's `ACC0718D-SN-…`), and wrote the "RESOLVED" note to project memory. All checks
here are additive and label test data `acc0718d`; no destructive operations were run, so
the interleaving does not affect any verdict — but a future frozen sign-off should run when
the dev DB is quiescent.

---

## 8. Test artifacts left in local dev DB (dev-only, labelled `acc0718d`)

- `recharge_products`: id 6 (`acc0718d-vip7777`, amount 7777, growth 5000) + two
  `acc0718d-tier-*` products used by the progression test.
- Members: three fresh fake-login members (recharge / tier / worker tests); ids in the
  6–10 range at run time; two ranked to tier 2/3, one bound to a fake phone.
- `wallet_accounts` / `wallet_ledger_entries`: coins + growth_value credits for the above.
- `offline_collection_orders`: one paid member-bound (id 6), plus short-TTL walk-ins used
  for the expiry proof (id 7 expired; an earlier mis-scripted id 5 left pending — harmless,
  will be swept). Their `payment_orders`/`payment_transactions`/`business_orders`.
- `outbox_events`: 2 rows for the settled collection (`print:receipt`,
  `payment:post-process`), both dispatched.
- `printer_devices`: two active `acc0718d-printer` devices on store 1 (ids 3, 4).
- `error_events`: 3 rows (401/404/400) from the diagnostics test.
- Scratch DB `inwardclub2_acc0718d` was created for the fresh-migrate check and **dropped**.

---

## 9. Teardown

- The private `api-bin` (`:18140`) and `worker-bin` started for this pass were **stopped**.
- Dev MySQL/Redis containers and other jobs' processes (8080/8099/18110) left as-is.
- **No `.go` / product source modified. No migration file modified. No migration applied
  to the dev DB by this pass** (it was already at v19). Only this report was written and
  the (now-dropped) scratch DB was migrated.

---

## 10. Sign-off checklist

- [x] Pre-flight `vet` / `build` / `test` green on the frozen tree
- [x] **v18 collision resolved** — `migrate status` no panic; dev DB at v19; fresh DB `1→19` clean
- [x] **Recharge VIP/growth** settles 200; coins+growth credited; VIP tier advances (NULL→2→3); idempotent
- [x] **Diagnostics** events persist; admin feed 200 newest-first; 0 persist-fail ERROR logs
- [x] **Worker chain** relay/scheduler/expiry(real transition)/rollup verified; idempotent
- [x] **Receipt / outbox / rule** — `print:receipt` + `payment:post-process` emitted, relayed, handled
- [x] Runtime health: 0 ERROR/WARN, 0 5xx, 0 panics
- [ ] §B2 real-adapter live smoke — **still DEFERRED** (config fail-fast present; no sandbox creds)
- [x] Result recorded in this dated report
```
