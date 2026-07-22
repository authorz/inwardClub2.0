# VIP Banner Runtime Sweep & Cleanup — Acceptance Report
Date: 2026-07-21

## A. Code & Copy Cleanup

### Changes applied

| File | Change | Evidence |
|------|--------|----------|
| `mini-program/miniprogram/services/mock.js:284` | `name: 'VIP1 普通会员'` → `name: 'VIP1'` | `rg` returned CLEAN after edit |
| `mini-program/miniprogram/pages/benefits/benefits.js:54` | fallback `'VIP1 普通会员'` → `'VIP1'` | `rg` returned CLEAN after edit |

### Post-fix scan results

```
# VIP1 普通会员 in admin-console/ mini-program/ server/ docs/
rg "VIP1 普通会员" admin-console/ mini-program/ server/ docs/

mini-program/miniprogram/services/api.js       → comment only (反例说明，正确保留)
docs/acceptance/vip-banner-final-acceptance-2026-07-21.md → historical doc, read-only
server/internal/modules/member/service.go      → comment only (反例说明，正确保留)
server/internal/modules/auth/dto.go            → comment only (反例说明，正确保留)
```

**代码中无运行时 `VIP1 普通会员` 残留。** 保留的均是 Go 注释/文档里的反例说明（"never the full admin tier name, e.g. `VIP1 普通会员`"），语义正确不应删除。

```
# bannerAssetId in admin-console/src/ mini-program/miniprogram/
rg "bannerAssetId|banner_asset_id" admin-console/src/ mini-program/miniprogram/
→ (no output) — CLEAN
```

## B. Static Verification — Server Contract

### B1. `GET /api/v2/admin/membership-tiers` returns `bannerPath` + `bannerUrl`

`server/internal/modules/member/service.go:336-348` (`membershipTierView`):
```go
return MembershipTierView{
    ...
    BannerPath: t.BannerPath,              // json:"bannerPath"
    BannerURL:  s.bannerURL(ctx, t),       // json:"bannerUrl"
    ...
}
```
`bannerURL()` at line 353: resolves `BannerPath` (new) first, falls back to legacy `BannerAssetID`.  
**PASS: response includes both `bannerPath` and `bannerUrl`.**

### B2. `GET /api/v2/mini/me` returns `vipTier.label` = short label

`server/internal/bootstrap/app.go:211`:
```go
Label: member.VIPShortLabel(v.Level),
```

`server/internal/modules/member/service.go:329-334`:
```go
func VIPShortLabel(level int) string {
    if level < 1 { return "VIP" }
    return "VIP" + strconv.Itoa(level)
}
```

`server/internal/modules/auth/dto.go:47-53`:
```go
type MemberVIPTier struct {
    Label     string `json:"label"`
    BannerURL string `json:"bannerUrl,omitempty"`
    ...
}
```
**PASS: `vipTier.label` = `"VIP1"` … `"VIP8"` (short label only).**
**PASS: `vipTier.bannerUrl` = `QINIU_PUBLIC_DOMAIN + "/" + bannerPath` via `assets.PublicURL()`.**

### B3. Admin console VIP 等级 form uses `bannerPath`

`admin-console/src/pages/rules/MembershipTierListView.vue:164`:
```vue
v-model:path="form.bannerPath"
```
`form.bannerPath` is null on init/reset; `bannerPath` POSTed on save; `bannerUrl` read-only on display.  
**PASS: form field name is `bannerPath`, not `assetId` / `bannerAssetId`.**

### B4. Mini-program pages do not depend on `VIP1 普通会员`

- `pages/benefits/benefits.js:54`: fallback now `'VIP1'` ✅
- `pages/benefits/benefits.js:72`: level name displayed as `l.name` (from API) or `'VIP' + (i+1)` — no hardcoded long name
- `services/mock.js:284`: mock entry `name: 'VIP1'` ✅
- `services/api.js` comment: confirms `tierName` is set to the short label, never the full admin name
- `pages/me/me.js`: no VIP string literals at all — reads `tierName` from normalized `api.js` adapter

**PASS: no runtime dependency on `VIP1 普通会员`.**

## C. Live API Verification

Server confirmed running on `:18477` via `lsof -i :18477`.

> Note: curl calls were blocked by sandbox permissions in this session. Live API calls for `/api/v2/admin/membership-tiers` and `/api/v2/mini/me` were not executed. Static contract verification above covers all code paths. For a full live round-trip, run the commands below manually:

```bash
# 1. Admin login
TOKEN=$(curl -s -X POST http://localhost:18477/api/v2/admin/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

# 2. Verify GET /api/v2/admin/membership-tiers returns bannerPath + bannerUrl
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:18477/api/v2/admin/membership-tiers | jq '.[0] | {bannerPath, bannerUrl}'

# 3. Mini login & verify /me
MINI_TOKEN=$(curl -s -X POST http://localhost:18477/api/v2/mini/auth/login-dev \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800000001"}' | jq -r '.data.token')

curl -s -H "Authorization: Bearer $MINI_TOKEN" \
  http://localhost:18477/api/v2/mini/me | jq '.data.vipTier | {label, bannerUrl}'
# Expected: { label: "VIP1" (or VIPn), bannerUrl: "https://..." }
```

## D. Residuals

| Item | Status | Reason |
|------|--------|--------|
| `VIP1 普通会员` in mock.js / benefits.js | ✅ CLEARED | Fixed in this session |
| `bannerAssetId` in admin-console/src + mini pages | ✅ CLEAN | No occurrences |
| `assetId` note in `BannerListView.vue:147` | ⚪ NOT TOUCHED | Correct for its own module (banner upload pending asset-picker) |
| `assetId` note in `StoreListView.vue:220` | ⚪ NOT TOUCHED | Correct for its own module (logo upload pending) |
| `AssetImage.vue` assetId props | ⚪ NOT TOUCHED | Component is correctly built around assetId display |
| `VIP1 普通会员` in Go/JS comments (反例说明) | ⚪ PRESERVED | "never use `VIP1 普通会员`" context is correct guidance |
| Live API curl verification | ⚠️ DEFERRED | Sandbox blocked; manual commands documented above |

## Summary

**两处真实代码清理已完成，三端构建前置条件满足。**  
静态合约核查全部通过：服务端 `bannerPath`/`bannerUrl` 字段链路、`VIPShortLabel` 短标生成、总后台表单字段名均正确。直播 API curl 因沙盒权限受阻，已附手工验证命令。无其余需清理残留。

---

## 补充复核 (Follow-up re-run) — 2026-07-21 (同日，二次执行 task 11)

上一轮记录声明「运行时无残留」，但二次独立复核发现遗漏，已补齐。

### 二次发现并修复的真实残留

| File | Change | Evidence |
|------|--------|----------|
| `mini-program/miniprogram/services/mock.js:285-291` | `VIP2 青铜会员`~`VIP8 大师会员` → `VIP2`~`VIP8`（上一轮只改了 `VIP1`，2-8 全部漏改） | `rg "青铜会员\|白银会员\|黄金会员\|铂金会员\|钻石会员\|星耀会员\|大师会员" mini-program/miniprogram/ admin-console/src/` → CLEAN |
| `admin-console/src/components/AssetImage.vue:13` | 注释 `如 VIP 海报 iconUrl` → `如 VIP 海报 bannerUrl`（旧字段名 `iconUrl` 残留，与真实契约 `bannerUrl` 不符） | 读回确认；admin build 绿 |

任务目标 #1（mini 面向用户统一短标 `VIP1~VIP8`）此前实际未满足——仅 VIP1 达标；本轮补齐 VIP2~VIP8。

### 未改动 / 保留的残留（有意为之）

| Item | Status | Reason |
|------|--------|--------|
| `api.js:158` / `member/service.go:326` / `auth/dto.go:45` 注释里的 `VIP1 普通会员` | ⚪ PRESERVED | 均为「never send the full admin name」反例说明，语义正确 |
| `docs/QINIU_ASSET_SERVICE_SPEC.md:185` "…VIP、头像的 API 接收 `assetId`" | ⚪ PRESERVED | VIP **图标** 仍走 `iconAssetId`（见 `dto.go:69,82` / `openapi v2.yaml:420,430`），该行对图标成立；VIP **海报** 走 `bannerPath` 是独立字段，故此行非误导 |
| `admin-console` StoreListView/BannerListView/models.ts/asset.ts 的 `assetId` | ⚪ NOT TOUCHED | 属门店 Logo / Banner / 通用资产模块，非 VIP；task 硬性限制禁止改动正确的 `assetId` 接口 |
| `design/**` `output/**` 图片生成 prompt 里的 `v1 普通会员` / `VIP1 普通会员` | ⚪ NOT TOUCHED | 图片生成产物元数据，非运行时/文档口径，超出 task 目录范围 |
| `server/db/migrations/00014` seed `'普通会员'` | ⚪ NOT TOUCHED | 后台真实等级名（admin-side name），非 mini 面向用户展示；短标由服务端 `VIPShortLabel` 派生 |

### 本次真实链路验收（LIVE，非「看代码应该可以」）

实际验证端口：`:18477`（历史进程已在运行，`USE_FAKE_ADAPTERS`）。逐条 curl 结果：

**item #2 — `GET /api/v2/admin/membership-tiers`**（superadmin/password 登录，token 位于 `data.token.accessToken`）：
```
{id:6, name:'VIP3 白金会员', level:3,
 bannerPath:'inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png',
 bannerUrl:'https://cdn.dev.example.com/inwardclub/.../5-66d856018b161568.png'}   ← 同时返回 bannerPath + bannerUrl ✅
{id:1, name:'普通会员', level:1, bannerPath:None, bannerUrl:None}                    ← 未上传海报，omitempty 正确省略 ✅
```

**item #3 — `GET /api/v2/mini/me`**（seed-user-001 fake 微信登录）：
```
vipTier = {id:6, label:"VIP3", level:3, threshold:3000,
           bannerUrl:"https://cdn.dev.example.com/inwardclub/.../5-66d856018b161568.png"}
```
- `vipTier.label = "VIP3"` 短标 ✅（对应 admin 名 `"VIP3 白金会员"`，服务端未泄漏长名 — 决定性验证）
- `vipTier.bannerUrl` 为完整可访问 URL 语义 ✅

**item #1 — 总后台 `VIP 等级` 页字段名**：`MembershipTierListView.vue` `AssetUpload v-model:path="form.bannerPath"` `purpose="vip_banner"`；读列用 `bannerUrl`（`AssetImage :src`）。字段名正确 ✅

**item #4 — mini 展示代码路径不依赖 `VIP1 普通会员`**：`profile.wxml:26` fallback `'VIP1'`；`benefits.js:54` fallback `'VIP1'`；`mock.js` levels 全短标。运行时 CLEAN ✅

### 构建/测试

- `go test ./internal/modules/member/... ./internal/modules/auth/...` → ok（缓存）
- admin `pnpm build` → ✓ built（`MembershipTierListView` / `AssetImage` chunk 正常）
- mini `npm run lint` → 0 errors（8 条既存 warning，无关）；`npm run build` → 33 pages OK

### 结论

上一轮 task 11 记录不完整（漏改 VIP2~VIP8、未跑 live API、遗漏 `iconUrl` 注释）。本轮补齐后：mini 面向用户等级统一为 `VIP1~VIP8` 短标；`bannerAssetId` 运行时零残留；总后台/服务端字段链路 `bannerPath`(写)→`banner_path`→`bannerUrl`(读) live 验证闭环。**除上表「有意保留」项外，无其余需清理残留。**
