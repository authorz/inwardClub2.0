# VIP Banner 闭环验收报告 — 2026-07-21

**任务来源：** `tasks/claude-handoffs/09-final-vip-banner-acceptance-and-release.md`
**验收日期：** 2026-07-21
**验收环境：** 本地 docker dev stack（`inwardclub2-mysql` 127.0.0.1:3307，DB v21，`USE_FAKE_ADAPTERS=true`）
**执行者：** Claude（自动化 curl 验收，非浏览器 GUI）

---

## 结论：**全部通过 ✅**

| # | 验收标准 | 状态 |
|---|---|---|
| 1 | `vip_banner` 上传凭证不报 `unsupported upload purpose` | **PASS** |
| 2 | 总后台 POST tier 接收 `bannerPath`，响应含 `bannerPath` + `bannerUrl` | **PASS** |
| 3 | GET `/admin/membership-tiers` 列表 + 详情均返回 `bannerPath` + `bannerUrl` | **PASS** |
| 4 | `/mini/me` 返回 `vipTier.label`（短标，无空格）+ `vipTier.bannerUrl` | **PASS** |
| 5 | 小程序首页/个人中心不再出现 `VIP1 普通会员` | **PASS（代码路径已验证）** |
| 6 | 所有构建绿色（server/admin-console/mini-program） | **PASS** |

---

## 1. 环境

| 项目 | 值 |
|---|---|
| 服务端口 | `:18477` |
| DB 版本 | goose v21（含 00020 banner_asset_id + 00021 banner_path） |
| 假适配器 | `USE_FAKE_ADAPTERS=true`（假 Qiniu，假微信登录） |
| 公开域（fake） | `https://cdn.dev.example.com` |
| 管理员账号 | `superadmin` / `password`（seed）|
| 小程序测试用户 | `seed-user-001`（示例会员 id=2，current_tier_id→tier#6 VIP3 白金会员） |

---

## 2. 关键请求与返回

### 2.1 CHECK 1 — 上传凭证（`vip_banner`）

```
POST /api/v2/admin/assets/upload-credentials
Authorization: Bearer <super_admin token>
{"purpose":"vip_banner","filename":"poster.png","contentType":"image/png","sizeBytes":102400}
```
```json
{
  "data": {
    "assetId": 6,
    "objectKey": "inwardclub/development/vip_banner/2026/07/6-2e650f70faf57539.png",
    "uploadToken": "fake-token:inwardclub/development/vip_banner/2026/07/6-2e650f70faf57539.png",
    "uploadUrl": "https://up.fake.qiniup.com",
    "expiresAt": "2026-07-21T00:00:50.402089Z",
    "maxSizeBytes": 10485760
  }
}
```
无 `unsupported upload purpose` 错误。✅

### 2.2 CHECK 2 — 创建 tier（bannerPath 写入）

```
POST /api/v2/admin/membership-tiers
{"name":"VIP9 验收测试","level":9,"threshold":9000,"bannerPath":"inwardclub/development/vip_banner/2026/07/acceptance-test-1784591463.png","status":"active"}
```
```json
{
  "data": {
    "id": 7,
    "name": "VIP9 验收测试",
    "level": 9,
    "threshold": 9000,
    "bannerPath": "inwardclub/development/vip_banner/2026/07/acceptance-test-1784591463.png",
    "bannerUrl": "https://cdn.dev.example.com/inwardclub/development/vip_banner/2026/07/acceptance-test-1784591463.png",
    "status": "active"
  }
}
```
`bannerPath`（对象键）+ `bannerUrl`（域名+路径）均出现在响应中。✅

### 2.3 CHECK 3 — 列表与详情均含 bannerPath + bannerUrl

`GET /api/v2/admin/membership-tiers` 列表确认：
- id=6 VIP3 白金会员：`bannerPath=inwardclub/development/vip_banner/2026/07/5-66d856...` `bannerUrl=SET`
- id=7 VIP9 验收测试：`bannerPath=inwardclub/development/vip_banner/2026/07/acceptance-test-...` `bannerUrl=SET`
- 其余无 bannerPath 的 tier：`bannerPath=(none)` `bannerUrl=NONE`（正确的 omitempty 行为）

`GET /api/v2/admin/membership-tiers/7` 详情：
```
bannerPath=inwardclub/development/vip_banner/2026/07/acceptance-test-1784591463.png
bannerUrl=https://cdn.dev.example.com/inwardclub/development/vip_banner/2026/07/acceptance-test-1784591463.png
```
✅

### 2.4 CHECK 4 — /mini/me vipTier.label + bannerUrl

```
GET /api/v2/mini/me
Authorization: Bearer <member#2 token>
```
```json
{
  "vipTier": {
    "id": 6,
    "label": "VIP3",
    "level": 3,
    "threshold": 3000,
    "bannerUrl": "https://cdn.dev.example.com/inwardclub/development/vip_banner/2026/07/5-66d856018b161568.png"
  }
}
```
- `label: "VIP3"` — 短标，无空格，不含全名 "VIP3 白金会员"。✅
- `bannerUrl` 已解析为完整 URL。✅

---

## 3. 小程序展示说明（代码路径验证）

WeChat DevTools GUI 在此无头环境中无法运行（headless 限制），以代码路径替代：

| 组件 | 文件 | 渲染表达式 | 数据来源 |
|---|---|---|---|
| 首页会员卡 tier 名称 | `pages/index/index.wxml:33` | `{{me.tierName}}` | `normalizeMe → vipTier.label → tierName` |
| 首页 VIP 海报 | `pages/index/index.wxml:54-56` | `<image src="{{me.tierBannerUrl}}">` | `normalizeMe → vipTier.bannerUrl → tierBannerUrl` |
| 个人中心 badge | `pages/home/home.wxml:13` | `{{me.tierShort}}` | 同上 `tierShort` |
| 个人中心等级行 | `pages/profile/profile.wxml:26` | `{{me.tierName\|\|'VIP1'}}` | 同上（fallback 为 'VIP1'，非 'VIP1 普通会员'） |

`normalizeMe`（`services/api.js:168-182`）：`tierShort = vipTier.label || tierShortOf(...)` — 第一优先级为 server 短标，全名不透传。

"VIP1 普通会员" 字符串残留：
- `services/mock.js:284` — mock 数据，生产不走此路径
- `pages/benefits/benefits.js:54` — 权益梯级页 fallback，该页面使用全名（tier 说明卡），与首页/个人中心的 `vipTier.label` 语义不同，属于已知设计意图

✅ 首页/个人中心无遗留 `VIP1 普通会员`。

---

## 4. 构建状态

| 端 | 命令 | 结果 |
|---|---|---|
| server | `go test ./...` | 全绿（所有包 ok/cached） |
| admin-console | `pnpm build` | ✓ built in 2.26s |
| mini-program | `node scripts/build.js` | build OK — 33 pages |

---

## 5. 实际使用的数据

| 项目 | 值 |
|---|---|
| 验收 VIP 等级 | id=6，VIP3 白金会员，level=3 |
| 测试用海报路径 | `inwardclub/development/vip_banner/2026/07/acceptance-test-1784591463.png` |
| 生产语义 | bannerUrl = `QINIU_PUBLIC_DOMAIN + "/" + bannerPath` |
| 当前 fake 公开域 | `https://cdn.dev.example.com` |
| 生产公开域 | `assets.inwardclub.com`（`server/.env` QINIU_PUBLIC_DOMAIN） |

---

## 6. 发现的问题

**无阻塞问题。**

非阻塞观察：
- 部分老 tier（id=1 普通会员、id=2 白银会员等）无 `bannerPath`/`bannerUrl` — 正常，未上传海报，`omitempty` 正确省略
- `/mini/membership-tiers`（公开权益梯级接口）返回 tier 全名（如"VIP3 白金会员"）— 属于权益说明页设计意图，不影响首页/个人中心短标

---

## 7. 发布准备清单

| 项目 | 状态 |
|---|---|
| 当前可上线风险 | **低**。bannerPath 路径写入已迁移（00021），服务端读写均有测试，旧 banner_asset_id 读兼容保留 |
| 回滚点 | goose down 回 v20（去掉 banner_path 列）即可回退；admin-console 和 mini-program 均可回滚到上一 tag |
| 数据库初始化数据 | 不需要额外 seed；现有 tier 无海报属正常状态，运营方可在总后台补充上传 |
| 文档 | `docs/openapi/v2.yaml` 已含 `bannerPath`/`bannerUrl` 字段；无需额外文档更新 |
| WeChat DevTools 验证 | 建议发布前由运营/测试在 DevTools 中走一次首页 banner 视觉确认（见 task 07 acceptance §6 handoff 步骤） |

---

## 8. 是否可以进入下一阶段

**可以。** VIP 海报全链路已验收，所有阻塞项已清除，三端构建绿色，无遗留 `VIP1 普通会员` 泄漏。
