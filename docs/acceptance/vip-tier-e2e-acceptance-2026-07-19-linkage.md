# VIP Tier End-to-End Linkage — Acceptance (2026-07-19)

Closed-loop acceptance for task `tasks/claude-handoffs/07-vip-tier-end-to-end-linkage.md`: admin
configures a VIP tier + VIP poster → server persists `bannerPath` and returns `bannerUrl`
(`QINIU_PUBLIC_DOMAIN + path`) → a test member is linked to that tier → `GET /api/v2/mini/me`
returns the short `vipTier.label` + `bannerUrl` → the mini home / personal-center pages consume it.

Run against the **local dev stack only** (docker MySQL `127.0.0.1:3307`, `USE_FAKE_ADAPTERS=true`).
The remote `server/.env` `MYSQL_DSN` (a `119.29.35.118` host) was **never exported or touched**.
No product code was modified for this acceptance — only two mini-program tails from task 05 were
already in place. The one server-side observation found is documented under Findings (non-blocking).

## 0. Headline

| # | 验收标准 (task §验收标准) | Status |
|---|---|---|
| 1 | 总后台成功保存 `bannerPath` | **PASS** — DB `membership_tiers.banner_path` = the uploaded objectKey |
| 2 | 服务端正确返回 `bannerUrl` | **PASS** — `bannerUrl = https://cdn.dev.example.com/<objectKey>` on admin create + list |
| 3 | `/api/v2/mini/me` 返回 `vipTier.label` | **PASS** — `label: "VIP3"` (short, derived from level; full name not leaked) |
| 4 | 小程序首页成功展示 VIP banner | **PASS (data + code)** — live `vipTier.bannerUrl` present; `normalizeMe` → `tierBannerUrl`; `index.wxml` renders it. Pixel render needs WeChat DevTools GUI (see §6 handoff). |

**Bottom line:** the VIP-tier closed loop is verified end-to-end at the API + data-contract level with
real (non-faked) server data. Criteria 1–3 are fully server-verified via curl; criterion 4's server
prerequisite and the client consume path are proven — only the final visual render in WeChat DevTools
is a manual GUI step this headless environment cannot perform (procedure provided in §6).

## 1. Environment & method

- **Server:** `go run ./cmd/api`, `env=development`, `HTTP_ADDR=:18477`, `USE_FAKE_ADAPTERS=true`
  (in-process fake Qiniu + fake WeChat; no real external creds). Health `GET /internal/health` → `200 {"status":"ok"}`.
- **DB / infra:** docker `inwardclub2-mysql` (MySQL 8) on `127.0.0.1:3307`, db `inwardclub2`, user `inward/inward`;
  docker `inwardclub2-redis` on `6379` (not required by the API for this flow).
  `MYSQL_DSN='inward:inward@tcp(127.0.0.1:3307)/inwardclub2?parseTime=true&loc=UTC&charset=utf8mb4'`, `JWT_SIGNING_KEY='dev-signing-key'`.
- **Why local, not `.env`:** `config.Load()` does not read `server/.env`, and its `MYSQL_DSN` points at a remote
  (possibly production) host that is also under-migrated. Local docker DB was used throughout.
- **Migrations / seed:** DB was at goose **v19** (banner columns absent); ran `go run ./cmd/migrate up` →
  **v21** (applies `00020_membership_tier_banner.sql`, `00021_membership_tier_banner_path.sql`) then
  `go run ./cmd/migrate seed` (idempotent).
- **Fake-adapter resolution:** `NewFakeObjectStore` defaults the public domain to `https://cdn.dev.example.com`
  when `QINIU_PUBLIC_DOMAIN` is empty, so `bannerUrl = https://cdn.dev.example.com/<objectKey>`. In production
  this is `QINIU_PUBLIC_DOMAIN + objectKey` — same code path (`asset.Service.PublicURL`).

### 1.1 Test accounts used (§必须产出 1)

| Role | Credential | Resolves to |
|---|---|---|
| Super admin | `POST /admin/auth/login {"username":"superadmin","password":"password"}` | `admin_accounts#1` (role `super_admin`) — seed |
| Test member | `POST /mini/auth/wechat/login {"code":"seed-user-001"}` (fake WeChat) | `members#2` "示例会员", openid `fake_openid_9cb72405ead00a32` — seed |

### 1.2 Chosen VIP tier (§必须产出 2) & uploaded poster path (§必须产出 3)

- Tier **id 6** — name **"VIP3 白金会员"**, `level 3`, `threshold 3000`, `status active`.
  (A full admin name at level 3 was chosen deliberately, to prove `/mini/me` returns the *short* label
  "VIP3" and never leaks the full name.)
- Uploaded poster object path (the `bannerPath`): **`inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png`**
  — minted by `POST /admin/assets/upload-credentials {"purpose":"vip_banner",...}` (fake Qiniu, `assetId 5`).

## 2. Admin side — save `bannerPath`, return `bannerUrl` — PASS

**Upload credential** (`POST /api/v2/admin/assets/upload-credentials`, Bearer super_admin):
```json
{ "assetId": 5,
  "objectKey": "inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png",
  "uploadUrl": "https://up.fake.qiniup.com",
  "uploadToken": "fake-token:inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png" }
```

**Create tier** (`POST /api/v2/admin/membership-tiers`, Bearer super_admin):
```
request  {"name":"VIP3 白金会员","level":3,"threshold":3000,"benefits":"专属客服 / 生日礼遇",
          "bannerPath":"inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png","status":"active"}
```
```json
response.data
{ "id": 6, "name": "VIP3 白金会员", "level": 3, "threshold": 3000,
  "benefits": "专属客服 / 生日礼遇",
  "bannerUrl": "https://cdn.dev.example.com/inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png",
  "status": "active" }
```

**DB persistence proof** (`banner_path`, not asset id, not URL):
```
membership_tiers WHERE id=6 →  level=3  status=active  banner_asset_id=NULL
  banner_path = inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png
```

| Assertion | Expected | Observed |
|---|---|---|
| tier persists `bannerPath` | `banner_path` = the objectKey | ✅ exact match |
| server returns `bannerUrl` | `QINIU_PUBLIC_DOMAIN + path` | ✅ `https://cdn.dev.example.com/…/5-66d856018b161568.png` |

## 3. `GET /api/v2/admin/membership-tiers` (必跑验证 #1) — PASS

```bash
curl -s -X GET http://127.0.0.1:18477/api/v2/admin/membership-tiers -H "Authorization: Bearer <ADMIN>"
```
Created tier as returned by the list:
```json
{ "id": 6, "name": "VIP3 白金会员", "level": 3, "threshold": 3000,
  "benefits": "专属客服 / 生日礼遇",
  "bannerUrl": "https://cdn.dev.example.com/inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png",
  "status": "active" }
```
Field-presence probe on the list item: `{ has_bannerPath: false, has_bannerUrl: true }`.
The admin read view returns the resolved `bannerUrl` but not a separate raw `bannerPath` field — see **Finding F1** (non-blocking; the raw path is fully round-tripped and is embedded verbatim inside `bannerUrl`).

## 4. Member linkage — set `current_tier_id` — PASS

There is **no** admin/service API to set a member's current tier (it is only advanced by the paid-recharge /
worker settlement path, upgrade-only). Per task §关键前提 (SQL allowed), linked via one UPDATE on the plain
nullable, no-FK column:
```sql
UPDATE members SET current_tier_id = 6 WHERE wechat_openid = 'fake_openid_9cb72405ead00a32';
-- verify → members#2 current_tier_id = 6
```

## 5. `GET /api/v2/mini/me` (必跑验证 #2, §必须产出 5) — PASS

```bash
curl -s -X GET http://127.0.0.1:18477/api/v2/mini/me -H "Authorization: Bearer <MEMBER>"
```
```json
response.data
{ "id": 2, "nickname": "示例会员", "inviteCode": "SEED0001", "status": "active",
  "vipTier": {
    "id": 6, "label": "VIP3", "level": 3, "threshold": 3000,
    "bannerUrl": "https://cdn.dev.example.com/inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png"
  } }
```

| Assertion | Expected | Observed |
|---|---|---|
| `vipTier.label` present & short | `VIP1..VIP8` from level | ✅ `"VIP3"` |
| full admin name NOT leaked | no `"VIP3 白金会员"` / `"普通会员"` | ✅ only `"VIP3"` |
| `vipTier.bannerUrl` correct | `QINIU_PUBLIC_DOMAIN + path` | ✅ matches the stored `banner_path` |

## 6. Mini-program display (§必须产出 6) — data + code PASS; visual handoff

The mini-program is a **native WeChat app** (requires WeChat DevTools GUI) and in this build is hard-wired to
mock data (`config/env.js useMock:true`). It has no headless/automation harness, so a pixel-level screenshot
of the home / personal-center VIP display cannot be produced in this environment. Per task §禁止, no fake banner
or fake `vipTier` was introduced on the frontend.

**What was verified headlessly against the LIVE server payload:** the exact `normalizeMe` logic
(verbatim from `services/api.js:161-182`) was run over §5's real `/mini/me` `.data`:
```
me.tierShort     = "VIP3"    → home.wxml:13  {{me.tierShort}}         (personal-center badge)
me.tierName      = "VIP3"    → index.wxml:34 {{me.tierName}}          (home VIP name)
me.tierBannerUrl = "https://cdn.dev.example.com/…/5-66d856018b161568.png"
                            → index.wxml:54-56 <image src="{{me.tierBannerUrl}}"> under
                              wx:if="{{loggedIn && me.tierBannerUrl}}"  (home VIP banner)
RESULT: PASS — pages render the short label + live banner; no full-name leak.
```
(The rendering wiring itself was code-verified in task 05; this run proves the data feeding it comes
through correctly from the live server.)

**Manual visual confirmation (2-minute GUI step, for the reviewer):** in `mini-program/miniprogram/config/env.js`
set `useMock:false` and `apiBaseUrl:'http://127.0.0.1:18477/api/v2'`; open the project in WeChat DevTools
(`project.config.json urlCheck:false` already permits a local `http://` host); log in so the client exchanges
`seed-user-001`; the home ident block shows the "VIP3" short label + the VIP banner image, and the 我的 page shows
"VIP3". (Revert `env.js` afterwards — it must ship `useMock:true` / prod base URL.)

## Findings

### F1 — admin tier read view returns `bannerUrl` but not a raw `bannerPath` field ✅ RESOLVED (task 10)

- **Original severity:** low (non-blocking; all four §验收标准 passed).
- **Resolution (2026-07-21, task 10):** `BannerPath string \`json:"bannerPath,omitempty"\`` added to
  `MembershipTierView` (`server/internal/modules/member/dto.go`) and populated in `membershipTierView()`
  (`service.go`). `GET /admin/membership-tiers` and `GET /admin/membership-tiers/:tierID` now return
  **both** `bannerPath` (raw Qiniu objectKey) and `bannerUrl` (resolved public URL).
  `GET /mini/membership-tiers` and `GET /mini/me` are unchanged — `MemberVIPTier` (auth/dto.go) is a
  separate struct and was not modified; the mini surface is not polluted.
- **OpenAPI:** `MembershipTierView` schema updated to document both fields.
- **Test:** `TestCreateMembershipTierBannerPath` (service_test.go) extended to assert `view.BannerPath`.

## Test artifacts left in the local dev DB (dev-only)

- `membership_tiers#6` — "VIP3 白金会员", `banner_path=inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png`.
- `assets#5` — pending `vip_banner` asset from the upload-credentials call (no bytes uploaded; fake store).
- `members#2` — `current_tier_id` set to `6` (was NULL).

These live only in the local docker `inwardclub2` DB. To reset the member: `UPDATE members SET current_tier_id=NULL WHERE id=2;`.

## Teardown

- API server (`:18477`) stopped after the run.
- Local docker MySQL/Redis left running (shared dev infra). Remote `server/.env` DB untouched throughout.

## Sign-off checklist

- [x] 1 · 总后台成功保存 `bannerPath` (DB `banner_path` = objectKey)
- [x] 2 · 服务端正确返回 `bannerUrl` (admin create + list)
- [x] 3 · `/api/v2/mini/me` 返回 `vipTier.label` = "VIP3" (short, no full-name leak)
- [x] 4 · VIP banner data path proven end-to-end (live `vipTier.bannerUrl` → `normalizeMe` → `index.wxml` render)
- [ ] 4b · Final pixel render in WeChat DevTools — manual GUI step (procedure in §6); not performable headlessly
- [x] `curl GET /api/v2/admin/membership-tiers` executed with response captured (§3)
- [x] `curl GET /api/v2/mini/me` executed with response captured (§5)
- [x] F1 resolved (task 10, 2026-07-21) · admin read now returns both `bannerPath` and `bannerUrl`; mini surface unchanged
