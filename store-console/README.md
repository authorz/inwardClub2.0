# 门店后台 store-console

门店后台独立站点，只服务门店管理员、收银员、门店运营等门店角色。

技术栈：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Naive UI + Axios。

## 站点边界（硬约束）

- 只调用 `/api/v2/store/*`，`src/api/http.ts` 在请求拦截器里**硬拦截** `/admin/*` 与任何非 `/store` 前缀。
- 独立门店账号、独立登录入口、独立 token audience：`store`。
- 独立登录态：token 存储命名空间为 `icsc:`，与总后台物理隔离，不复用总后台登录态。
- 不展示门店选择器，不允许切换门店。
- 请求 body/query/path 不传 `storeId` 决定数据范围；当前门店只来自服务端 token scope 与 `/store/auth/me`。

---

## 本地启动 / 验收命令

命令名称与任务书一致：

```bash
cd store-console
pnpm install      # 安装依赖（首次会执行 esbuild/vue-demi 构建脚本，已在 pnpm-workspace.yaml 放行）
pnpm lint         # eslint . --max-warnings 0
pnpm typecheck    # vue-tsc --noEmit
pnpm build        # vue-tsc --noEmit && vite build
pnpm dev          # vite，默认端口 5183
```

> 关于 `pnpm install`：pnpm 10+ 默认忽略依赖的安装脚本。本项目已在 `pnpm-workspace.yaml`
> 中通过 `onlyBuiltDependencies`（及 codex 运行时的 `allowBuilds`）放行 `esbuild`、`vue-demi`，
> 因此 `pnpm install` 不需要额外 `pnpm approve-builds`。

### 本次交付实测结果

| 命令 | 结果 |
| --- | --- |
| `pnpm install` | ✅ 通过，esbuild/vue-demi 构建脚本正常执行 |
| `pnpm lint` | ✅ 0 error / 0 warning |
| `pnpm typecheck` | ✅ 通过（vue-tsc 无错误） |
| `pnpm build` | ✅ 通过，产物输出至 `dist/` |
| `pnpm dev` | ✅ 通过，`http://localhost:5183/` 返回 200 |

环境变量（`.env.development` / `.env.production` 已就绪）：

```dotenv
VITE_APP_ID=inwardclub-store-console
VITE_API_BASE_URL=<https://api.inwardclub.cn/api/v2 或本地网关>
VITE_AUTH_AUDIENCE=store
VITE_ASSET_PUBLIC_DOMAIN=
```

---

## 页面列表（第一阶段骨架，15 个业务页 + 登录 + 错误页）

| 页面 | 路由 name | 路径 | 权限码（命中其一可见） |
| --- | --- | --- | --- |
| 今日概览 | `dashboard` | `/dashboard` | 登录即可 |
| 本店订单 | `orders` | `/orders` | `store.order.read` |
| 点餐订单处理 | `food-orders` | `/food-orders` | `store.order.read` |
| 活动核销 | `activity-verify` | `/activity-verify` | `store.ticket.verify` |
| 票券核销 | `ticket-verify` | `/ticket-verify` | `store.ticket.verify` |
| 积分审核 | `point-review` | `/point-review` | `store.point.review` |
| 核销记录 | `verifications` | `/verifications` | `store.ticket.verify` |
| 本店商品/库存/价格覆盖 | `catalog` | `/catalog` | `store.catalog.read` / `store.catalog.write` |
| 本店活动 | `activities` | `/activities` | `store.activity.read` / `store.activity.write` |
| 预约/桌台视图 | `reservations` | `/reservations` | `store.reservation.write` |
| 线下聚合收款 | `collection` | `/collection` | `store.collection.create` |
| 收款记录 | `collection-records` | `/collection-records` | `store.collection.read` |
| 打印机管理 | `printers` | `/printers` | `store.printer.write` |
| 本店报表 | `reports` | `/reports` | `store.report.read` |
| 设置 | `settings` | `/settings` | 登录即可 |
| 登录 | `login` | `/login` | 公开 |
| 无权限 | `forbidden` | `/403` | 需登录 |
| 未找到 | `not-found` | `/:pathMatch(.*)*` | 公开 |

路由由 `src/constants/menu.ts` 的菜单配置驱动（`src/router/index.ts` + `src/router/pages.ts`），
菜单、路由、权限三处共用同一份 name/permission 定义。

---

## API client 范围

- 唯一 axios 实例：`src/api/http.ts`。全站禁止另建实例或直接 `fetch`。
- 允许前缀：`/store/*`、`/assets/*`、`/internal/assets/*`（资产读取）。命中 `/admin/*` 或其他前缀直接抛 `FORBIDDEN_SCOPE`。
- 所有接口路径集中在 `src/constants/apiPaths.ts`，服务层不硬编码路径字符串。
- 领域服务（`src/api/services/`）：auth、orders、catalog、activities、verification、pointSavings、reservations、collection、printers、reports、profile。
- 覆盖的 `/api/v2/store/*` 能力：auth（login/refresh/me/logout）、profile/settings/status、reports、orders/food-orders 状态机/payment-orders/refund-orders/refunds、catalog（items/categories/global-items/stock/payment-rules/publish）、activities（含 today/global-activities/publish）、tickets/verifications、coupon-entitlements/redemptions（核销/作废）、point-savings/review、reservations/arrive、tables、offline-collection-orders（创建/查看/取消）、printer-devices。

### 请求层统一处理

- `Authorization: Bearer <access>`、`X-Request-ID`（每请求 UUID）。
- 高风险写操作自动附加 `Idempotency-Key`（创建收款码、退款、票/券核销、发券作废、库存调整、订单状态流转、积分审核、到店确认、门店营业状态变更）。
- 401 单飞刷新并重试一次，失败即清空登录态回登录页。
- 403/409/422/网络错误统一归一化为 `ApiError`（`src/api/error.ts`），422 映射字段级错误。
- 列表统一解包 `{ data, meta }`，并做跨店数据兜底：返回行若带非本店 `storeId` 抛 `CROSS_STORE_DATA` 阻断展示。

---

## auth / audience / store scope 防护说明

实现于 `src/stores/auth.ts`（独立 Pinia store）+ `src/utils/jwt.ts` + `src/router/guards.ts`：

1. 登录成功与每次 refresh 后都对 token claims 做校验 `validateClaims`：
   - `aud` 必须为 `store`（`VITE_AUTH_AUDIENCE`），否则清空登录态。
   - `subject_type` 必须在门店白名单 `store_admin` / `cashier` / `store_operator`，否则清空。
   - `store_id` 必须非空（非 `null`/空串/0），否则清空。
2. token 过期（带 30s 容差）在启动 bootstrap 时尝试刷新，刷新失败清空。
3. 当前门店 `storeId` 只从 token claims / `me` 读取；页面只读展示门店名，不参与任何请求参数。
4. http 层 `getStoreId()` 用于跨店返回数据兜底阻断。
5. 前端权限守卫仅收敛入口体验，最终权限由服务端强制。

---

## 严格开发模式：公共能力清单

### 公共组件（`src/components/common/`）

| 组件 | 职责 | 复用点 |
| --- | --- | --- |
| `DataTable` | 表格 + 远程分页 + loading + 空状态 | 全部 8 个列表页 |
| `StatusFilterBar` | 状态筛选 + 关键字搜索 + 额外筛选/操作插槽 | 全部列表页筛选区 |
| `StatusTag` | 按集中枚举字典渲染状态标签与色调 | 所有状态展示 |
| `ReviewDialog` | 审核弹窗（通过/驳回 + 备注） | 积分审核（可扩展其它审批） |
| `VerifyDialog` | 核销弹窗（扫码/输入码 + 确认） | 活动核销、票券核销 |
| `CollectionCodeDialog` | 收款码弹窗（金额/二维码/倒计时/掩码会员/取消） | 线下收款、收款记录 |
| `PrintStatusIndicator` | 打印机/打印任务状态圆点提示 | 打印机管理（可扩展订单打印） |
| `PermissionButton` | 按权限码隐藏/禁用的按钮 | 所有写操作入口 |
| `PageHeader` | 页面标题 + 描述 + 操作插槽 | 所有页面头部 |
| `EmptyState` | 统一空状态 | DataTable、报表分项 |
| `MetricTile` | 指标块（通栏轻背景，非重卡片） | 工作台、报表 |
| `AssetImage` | 只按 assetId/url 展示资产图片 | 设置页等 |
| `AppIcon` | 内联 SVG 图标集（免图标库依赖） | 侧边菜单、顶栏 |

### 公共 composables / 工具（`src/composables/`, `src/utils/`）

| 工具 | 职责 |
| --- | --- |
| `useAsyncList` | 列表 loading/分页/筛选/空态/错误编排（所有列表页） |
| `useAsyncAction` | 写操作 loading + 二次确认 + 成功/失败提示 + 回调 |
| `useConfirm` | 统一二次确认弹窗 |
| `utils/columns.ts` | 文本/金额/时间/手机号掩码/状态/操作列构造器 |
| `utils/format.ts` | 金额(整数分)、手机号掩码、日期时间、倒计时格式化 |
| `utils/jwt.ts` | JWT payload 解码、过期判断、audience 判断 |
| `utils/id.ts` | X-Request-ID / Idempotency-Key 生成 |
| `utils/storage.ts` | `icsc:` 命名空间的 token 存储（与总后台隔离） |
| `utils/feedback.ts` | Naive UI discrete API（message/dialog/notification）单实例 |

### 公共服务 / API client（`src/api/`）

`http.ts`（唯一实例 + 拦截器）→ `request.ts`（解包/分页/跨店兜底）→ `services/*`（领域服务）→ 页面。
`authBridge.ts` 打破 http 与 auth store 的循环依赖。

### 集中定义

- `constants/enums.ts`：订单类型、支付渠道、支付状态、点餐状态机（含可执行动作 `FOOD_ORDER_TRANSITIONS`）、核销状态、审核状态、上下架、来源 scope、预约状态、收款单状态、打印状态、资产 purpose，附带 label/tone 展示映射。
- `constants/apiPaths.ts`：接口路径。
- `constants/permissions.ts`：门店权限码与 subject_type 白名单。
- `constants/menu.ts`：菜单（驱动路由与权限）。
- `styles/tokens.css` + `theme.ts`：颜色/间距/字号/圆角/分割线 token 与 Naive UI 主题覆盖。

### 已复用的重复模式

- 8 个列表页全部由 `useAsyncList` + `DataTable` + `StatusFilterBar` + `columns.ts` 组合而成，无复制粘贴。
- 两个核销页共用 `VerifyDialog`；两个收款页共用 `CollectionCodeDialog`；状态展示全部走 `StatusTag` + 枚举字典。
- 所有写操作走 `useAsyncAction` + `PermissionButton`，统一 loading/确认/提示/权限。

### 未抽象复用的原因

- 各页面 `columns` 定义与业务字段强相关（列标题、宽度、行内动作不同），保留在页面内更可读；共性部分已下沉到 `columns.ts` 构造器。
- 商品覆盖编辑、打印机编辑、收款创建表单字段差异大且各自单一使用，暂未抽象为通用 FormDrawer；如后续出现第二个同构表单再抽象。

---

## 设计规则遵守声明

遵守 `design/GLOBAL_DESIGN_RULES.md`：黑白浅灰运营工作台风格，主按钮为黑，无渐变/彩色促销风/金色奢华风/装饰元素；列表用通栏 + 分割线 + 轻背景区块，不做重阴影卡片堆叠、不做框中框；无门店选择器、无跨店筛选、无数据大屏。金额整数分、手机号掩码、核销/退款等危险操作二次确认。

---

## 未实现内容（第一阶段范围外 / 待后续）

- 商品/活动/券的**新建与完整表单**（当前提供列表、上下架、价格/库存/支付方式覆盖骨架）；场次、票档、券模板编辑抽屉待第二阶段。
- 会员本店视图与钱包流水页（服务已预留路径 `members`，未做页面，非第一阶段 15 页范围）。
- 员工/收银员管理页（路径已在 `apiPaths` 预留，非第一阶段页面清单）。
- 资产上传组件 `StoreAssetPicker/Upload/Preview`（当前仅 `AssetImage` 展示；上传待七牛资产服务对接）。
- 报表趋势图表（`reports/*` 分项以占位空状态呈现，待服务端报表接口）。
- 审计日志页（权限码 `store.audit.read` 已定义，页面待后续）。
- 打印任务实时状态流（当前展示设备状态，任务级 `PrintStatusIndicator kind="job"` 已就绪待接入）。

## 需要服务端配合的接口（关键契约）

- 统一响应包体 `{ data, meta }` / `{ error: { code, message, details } }`；列表 `meta` 含 `page/pageSize/total`。
- `POST /store/auth/login` 返回 `{ accessToken, refreshToken, expiresIn }`；token claims 含 `aud=store`、`subject_type`、`role`、非空 `store_id`、`token_version`。
- `GET /store/auth/me` 返回 `{ account: { id,name,role,subjectType,permissions[] }, store: { id,name,status,logoUrl } }`。
- `POST /store/auth/refresh` 接受 `{ refreshToken }`，返回同 login 结构。
- 高风险写接口需接受并要求 `Idempotency-Key` 头（见 API 范围）。
- 列表行如含门店归属请使用 `storeId` 字段命名，便于前端跨店兜底校验（服务端仍须强制 scope）。
- 线下聚合收款：`POST /store/offline-collection-orders` 接受 `{ amountCent, subject, businessType, expiresInSeconds, memberPhone? }`，返回 `{ collectionOrderNo, amountCent, qrContent|qrDisplayUrl, expiresAt, status, memberNickname?, memberPhoneMasked? }`；原始手机号仅用于匹配、不回写。
- 点餐状态流转：`POST /store/food-orders/{id}/{confirm|prepare|ready|complete|cancel}`。
- 金额字段整数分、以 `Cent` 结尾；资产字段只接受 `assetId`。

---

开发任务由 Claude 执行；Codex 负责拆任务、验收和合并标准。
