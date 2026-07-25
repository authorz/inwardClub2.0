# 总后台 admin-console

InwardClub 2.0 总后台独立站点，只服务总部 / 总管理角色。

技术栈：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Naive UI + Axios。

## 业务边界

- 只调用 `/api/v2/admin/*` 与必要公共资产接口；HTTP client 运行时拦截并拒绝 `/store/*`、`/mini/*`。
- 独立总后台账号、独立登录入口、独立 token audience：`admin`。
- 独立 auth store 与 permission store，不复用门店后台登录态。
- 可查看 / 管理跨门店资源，但资产、钱包、退款、规则、跨店写操作均有二次确认 + 审计提示 + 幂等键。
- 业务规则数值（积分、金币、VIP 门槛、充值档位等）不写死，一律以接口 / 表单配置为准。

---

## 一、本地启动方式

```bash
cd admin-console
pnpm install
pnpm dev        # http://localhost:5180
```

其它脚本：

```bash
pnpm lint        # eslint . --max-warnings 0
pnpm lint:fix    # 自动修复
pnpm typecheck   # vue-tsc --noEmit
pnpm build       # vue-tsc --noEmit + vite build
pnpm preview     # 预览 dist
```

环境变量（`.env.development` / `.env.production`，以 `.env.example` 为模板）：

| 变量 | 说明 | 默认（dev） |
| --- | --- | --- |
| `VITE_APP_ID` | 应用标识（错误监控 / 埋点区分） | `inwardclub-admin-console` |
| `VITE_API_BASE_URL` | API 根地址，必须指向 `/api/v2` 网关 | `http://localhost:8081/api/v2` |
| `VITE_AUTH_AUDIENCE` | token audience，固定 `admin` | `admin` |
| `VITE_ASSET_PUBLIC_DOMAIN` | 资产公共访问域名（可选，展示 assetId 图片） | 空 |

### 关于 pnpm 构建脚本审批

本仓库使用 pnpm 11。`pnpm-workspace.yaml` 已通过 `allowBuilds` 批准 `esbuild`、`vue-demi` 的安装脚本；
`.npmrc` 额外设置 `verify-deps-before-run=false`。因此 `pnpm lint/typecheck/build/dev` 均可直接运行，
无需交互式 `pnpm approve-builds`。

---

## 二、验收命令结果

| 命令 | 结果 |
| --- | --- |
| `pnpm install` | ✅ 成功（Done，无 ignored builds 报错） |
| `pnpm lint` | ✅ exit 0（0 error / 0 warning，`--max-warnings 0`） |
| `pnpm typecheck` | ✅ exit 0（vue-tsc 无错误） |
| `pnpm build` | ✅ exit 0（vite 产物输出到 `dist/`） |
| `pnpm dev` | ✅ 启动成功，`http://localhost:5180/` 返回 HTTP 200 |

---

## 三、页面列表

| 分组 | 页面 | 文件 | 能力 |
| --- | --- | --- | --- |
| — | 登录 | `pages/LoginView.vue` | 独立登录，登录后校验 token audience |
| — | 工作台 | `pages/DashboardView.vue` | 运营指标概览 + 快捷入口（非装饰大屏） |
| 门店 | 门店管理 | `pages/stores/StoreListView.vue` | 列表 + 新增/编辑（审计提示） |
| 账号 | 总后台账号 | `pages/accounts/AdminAccountsView.vue` | 复用 `AccountListView` |
| 账号 | 门店管理员 | `pages/accounts/StoreAdminAccountsView.vue` | 复用 `AccountListView`（含绑定门店） |
| 账号 | 员工 | `pages/accounts/StaffAccountsView.vue` | 复用 `AccountListView`（单店绑定） |
| 商品 | 全局分类 | `pages/catalog/CategoryListView.vue` | 列表 + 新增/编辑 |
| 商品 | 全局商品 | `pages/catalog/ItemListView.vue` | 列表 + 发布/下架 |
| 活动 | 全局活动 | `pages/activities/ActivityListView.vue` | 列表 + 发布/下架 |
| 券 | 券模板 | `pages/coupons/CouponTemplateListView.vue` | 列表 + 发布/下架 |
| 运营 | Banner 管理 | `pages/banners/BannerListView.vue` | 列表 + 新增/编辑/删除 |
| 运营 | 快捷充值 | `pages/recharge/RechargeProductListView.vue` | 列表 + 新增/编辑/禁用（支付金额、到账金币、赠送积分） |
| 规则 | VIP 等级 | `pages/rules/MembershipTierListView.vue` | 列表 + 新增/编辑 |
| 规则 | 规则中心 | `pages/rules/RuleDefinitionListView.vue` | 列表 + 发布/禁用（高风险） |
| 订单 | 订单中心 | `pages/orders/OrderListView.vue` | 统一订单只读列表 |
| 支付 | 支付单/流水 | `pages/payments/PaymentOrderListView.vue` | 只读列表 |
| 支付 | 退款单 | `pages/payments/RefundListView.vue` | 列表 + 退款审批（高风险 + 幂等） |
| 会员 | 会员列表 | `pages/members/MemberListView.vue` | 列表 + 人工调账（高风险 + 幂等 + 原因） |
| 会员 | 钱包账本 | `pages/members/WalletLedgerView.vue` | 只读流水 |
| 报表 | 报表 | `pages/reports/ReportsView.vue` | 分项标签骨架 |
| 审计 | 审计日志 | `pages/audit/AuditLogView.vue` | 只读 |
| 审计 | 登录日志 | `pages/audit/LoginEventView.vue` | 只读 |
| 审计 | 错误事件 | `pages/audit/ErrorEventView.vue` | 只读 |
| 系统 | 支付配置 | `pages/system/PaymentSettingsView.vue` | 读写表单（高风险 + 幂等） |
| — | 404 | `pages/NotFoundView.vue` | 兜底 |

---

## 四、路由列表

菜单与路由从单一配置 `layouts/menu.ts` 派生（`flattenMenu`），标题 / 权限 / 面包屑不重复维护。

| Path | 页面 | 页面权限码 |
| --- | --- | --- |
| `/login` | 登录 | 无（公开） |
| `/dashboard` | 工作台 | `admin.report.read` |
| `/stores` | 门店管理 | `admin.store.read` |
| `/accounts/admins` | 总后台账号 | `admin.account.read` |
| `/accounts/store-admins` | 门店管理员 | `admin.account.read` |
| `/accounts/staff` | 员工 | `admin.account.read` |
| `/catalog/categories` | 全局分类 | `admin.catalog.read` |
| `/catalog/items` | 全局商品 | `admin.catalog.read` |
| `/activities` | 全局活动 | `admin.activity.read` |
| `/coupons` | 券模板 | `admin.coupon.read` |
| `/banners` | Banner 管理 | `admin.banner.read` |
| `/recharge-products` | 快捷充值 | `admin.recharge.read` |
| `/rules/membership-tiers` | VIP 等级 | `admin.rule.read` |
| `/rules/definitions` | 规则中心 | `admin.rule.read` |
| `/orders` | 订单中心 | `admin.order.read` |
| `/payments/orders` | 支付单/流水 | `admin.payment.read` |
| `/payments/refunds` | 退款单 | `admin.payment.read` |
| `/members` | 会员列表 | `admin.member.read` |
| `/members/wallet-ledger` | 钱包账本 | `admin.member.read` |
| `/reports` | 报表 | `admin.report.read` |
| `/audit/logs` | 审计日志 | `admin.audit.read` |
| `/audit/login-events` | 登录日志 | `admin.audit.read` |
| `/audit/error-events` | 错误事件 | `admin.audit.read` |
| `/system/payment-settings` | 支付配置 | `admin.system.payment_settings.write` |
| `/:pathMatch(.*)*` | 404 | 无 |

路由守卫（`router/index.ts`）：未登录跳登录页（带 redirect）；已登录访问登录页跳工作台；
无页面权限跳工作台并提示；首次进入 `bootstrap` 恢复会话并校验 token audience。

---

## 五、API client 范围

- 唯一 axios 实例：`api/http.ts`。所有请求必须经此 client，页面 / 服务不得散落 axios / fetch。
- **白名单**：`ALLOWED_PATH_PREFIXES = ['/admin/', '/assets']`（`constants/api-paths.ts`）。
- **黑名单**：`FORBIDDEN_PATH_PREFIXES = ['/store/', '/mini/']`，命中直接抛错（越界防护）。
- 统一注入：`Authorization: Bearer <token>`、`X-Request-ID`、`X-Admin-App`；高风险写操作（`idempotent: true`）注入 `Idempotency-Key`。
- 统一处理：401 单飞刷新并重放 / 刷新失败清空登录态；403/409/422/网络错误归一化为 `NormalizedError`（422 映射字段错误）。
- 统一解包响应信封 `{ data, meta }`；列表返回 `{ items, meta }`。
- 所有接口路径集中在 `constants/api-paths.ts`（单一事实来源），覆盖规格中全部 `/api/v2/admin/*` 端点。

高风险操作强制携带 `Idempotency-Key`：退款审批、人工调账、发券、规则发布、商品/活动发布下架、快捷充值写、支付配置写。

---

## 六、auth / audience 防护说明

- 独立存储命名空间（`utils/storage.ts` 以 `VITE_APP_ID` 为前缀），与门店后台同域调试也不会互相覆盖或复用登录态。
- 登录 / refresh 拿到 token 后，`stores/auth.ts` 用 `utils/jwt.ts` 解析并校验：
  - `aud` 必须包含 `admin`（`EXPECTED_AUDIENCE`）；
  - `subject_type` 必须为 `admin`（`EXPECTED_SUBJECT_TYPE`）；
  - 过期直接判定无效。
  - 任一不满足：清空 access/refresh/user 并提示「登录凭证不属于总后台」。
- `/admin/auth/me` 返回的 `audience` 再次校验，不匹配同样清空登录态。
- HTTP 401 → 单飞 refresh；refresh 失败 → `onUnauthorized` 清空登录态，路由守卫跳回登录页。
- 前端权限仅用于菜单 / 路由 / 按钮显隐；最终授权由服务端强制（`stores/permission.ts` 以 `/admin/auth/me` 的 `permissions` 为真值，`super_admin` 默认放行）。

---

## 七、严格开发模式：公共能力清单

### 公共组件（`src/components/`）

| 组件 | 职责 |
| --- | --- |
| `ResourceListView.vue` | **配置驱动列表视图**：页头 + 筛选 + 工具栏 + 分页表格 + 空状态一体化，列表页只写配置 |
| `DataTable.vue` | 表格：分页 / loading / 空状态 / 行 key；对 NDataTable 不变型行做单点类型收口 |
| `FilterBar.vue` | schema 驱动筛选区（input / select / daterange） |
| `FormDrawer.vue` | 新增/编辑表单抽屉，内置审计提示位与提交 loading |
| `ConfirmDialog`（二次确认） | 由 `composables/useAuditedAction` + Naive UI dialog 统一实现，见下 |
| `AuditRiskAlert.vue` | 审计风险提示条（含跨店写操作提示） |
| `PermissionButton.vue` | 权限按钮（无权限隐藏或禁用+提示） |
| `StatusTag.vue` | 状态标签（tone 映射 + 始终展示文本，不依赖颜色单独表达状态） |
| `PageHeader.vue` | 面包屑 + 页标题 + 描述 + 操作区 |
| `AssetImage.vue` | 只按 assetId 展示图片，不接受任意 URL |
| `ui-types.ts` | 共享 UI 类型：`FilterField`、`ToolbarAction`、`TableColumnList<T>`、`ResourceListInstance` |

### 公共工具（`src/utils/`、`src/directives/`）

| 工具 | 职责 |
| --- | --- |
| `columns.ts` | 表格列工厂：`textColumn`/`statusColumn`/`moneyColumn`/`dateTimeColumn`/`renderColumn`/`actionsColumn` |
| `format.ts` | 金额（整数分↔元）、时间（RFC3339→本地）、手机号掩码 |
| `id.ts` | `X-Request-ID` 与 `Idempotency-Key` 生成 |
| `jwt.ts` | JWT payload 解析、过期判定、audience 匹配 |
| `storage.ts` | 命名空间化本地存储（隔离门店后台） |
| `feedback.ts` | 全局 message / dialog 句柄，供非组件环境统一弹提示 |
| `directives/permission.ts` | `v-permission` 指令 |

### 公共状态 / composables（`src/composables/`、`src/stores/`）

| 名称 | 职责 |
| --- | --- |
| `useDataTable.ts` | 列表分页 / 筛选 / loading / 空状态 / 刷新统一封装 |
| `useAuditedAction.ts` | 高风险 / 跨店写操作统一二次确认 + 审计提示 + 成功失败反馈 |
| `usePublishableActions.ts` | 商品/活动/券共用的发布 / 下架动作（含幂等与审计） |
| `stores/auth.ts` | 独立 auth store（登录态 + audience 防护 + 401 刷新装配） |
| `stores/permission.ts` | 独立权限 store |

### 公共服务 / API client（`src/api/`、`src/constants/`）

| 名称 | 职责 |
| --- | --- |
| `api/http.ts` | 唯一 axios 实例 + 白/黑名单 + 头注入 + 错误归一化 + 信封解包 |
| `api/resource.ts` | 通用 REST 资源工厂 `createResource`（list/get/create/update/remove/action） |
| `api/services/index.ts` | 各领域服务实例（门店/账号/分类/商品/活动/券/Banner/充值/等级/规则/订单/会员）+ 只读列表集合 + 系统/报表服务 |
| `api/services/auth.ts` | 认证服务（login/refresh/me/logout） |
| `api/types.ts` / `api/models.ts` | 响应信封类型、分页 meta、后台实体类型 |
| `constants/api-paths.ts` | 全部 `/admin/*` 接口路径（单一事实来源） |
| `constants/enums.ts` | 状态 / 支付渠道 / 订单类型 / 资产类型 / 核销状态等枚举 + 展示映射 |
| `constants/permissions.ts` / `roles.ts` | 权限码 / 角色常量 |
| `styles/tokens.css` / `styles/theme.ts` | 颜色 / 间距 / 字号 / 圆角 token 与 Naive UI 黑白灰主题覆写 |

### 已复用的重复模式说明

- **列表页**：23 个列表页全部复用 `ResourceListView`（配置驱动），不存在复制粘贴的筛选/表格/分页实现。
- **账号页**：总后台账号 / 门店管理员 / 员工三页结构一致，抽象为单一 `AccountListView`，三个路由页仅传配置（约 10 行）。
- **发布类资源**：商品 / 活动 / 券的发布-下架逻辑抽象为 `usePublishableActions`，三处共用。
- **CRUD 服务**：绝大多数模块用 `createResource` 生成标准服务，服务层几乎零重复；接口路径全部集中在 `api-paths.ts`。
- **表格列**：状态标签 / 金额 / 时间 / 自定义 / 操作列由 `columns.ts` 工厂统一生成，页面不手写 render。
- **状态枚举 / 支付方式 / 订单类型 / 资产类型 / 核销状态**：集中在 `enums.ts`，配合 `StatusTag` 复用。
- **二次确认 + 审计**：所有高风险操作走 `useAuditedAction`，弹窗文案与幂等一致。
- **错误 / loading / 分页 / 空状态 / 权限判断**：分别集中在 http.ts / useDataTable / DataTable / permission store。

### 未抽象复用的原因

- 各列表页的 **columns / filter fields / toolbar 配置**属于业务差异（字段、权限码、动作不同），按设计以「配置」而非「复制代码」区分，已是复用后的最小差异面，无需再抽象。
- 门店 / 分类 / Banner / 充值 / VIP 等 **新增-编辑表单字段**各不相同，表单容器（`FormDrawer`）已复用，字段本体属于业务差异，未强行套用同一 schema，避免过度抽象。

---

## 八、未实现内容（第一阶段范围外 / 待后续）

- 复杂详情页与子资源编辑：商品规格/门店覆盖、活动场次/票档/投放、券适用商品/发券记录/核销记录 —— 当前提供列表与发布/下架骨架，详情表单待服务端 DTO 明确后补充。
- 资产上传组件 `AdminAssetPicker / AdminAssetUpload / AdminAssetPreview`：当前 `AssetImage` 只做 assetId 展示，上传流程待接入 `QINIU_ASSET_SERVICE_SPEC.md`。
- 报表图表与导出：当前为分项标签 + 空状态骨架。
- 规则的草稿 / 预览 / 版本历史 / 命中记录、门店设置详情、打印机与打印任务、座位/桌台管理页面。
- 账号密码策略、角色-权限分配 UI；门店选择器（跨店发券/投放的目标门店选择，当前以说明占位）。
- OpenAPI 类型自动生成（当前实体类型按规格手写于 `api/models.ts`，待 `docs/openapi/v2.yaml` 就绪后切换）。
- 单元测试 / E2E：本阶段以 lint + typecheck + build 作为质量门禁，未编写测试用例。

## 九、需要服务端配合的接口

前端已按 `docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md` 集中登记全部 `/api/v2/admin/*` 路径于 `constants/api-paths.ts`，
均需服务端提供并遵守约定（响应信封 `{data,meta}`、整数分 `*Cent`、资产只收 `assetId`、写操作支持/要求 `Idempotency-Key`、跨店写审计）。重点：

- 认证：`POST /admin/auth/login`、`POST /admin/auth/refresh`、`GET /admin/auth/me`、`POST /admin/auth/logout`；
  token 需含 `aud=admin`、`subject_type=admin`、`role`、`permissions`；`me` 返回 `permissions` 数组作为前端权限真值。
- 列表：统一 `page`/`pageSize`（≤100），响应 `meta{page,pageSize,total}`；筛选参数按规格 8.7 各资源约定。
- 门店 / 账号（admin / store-admin / staff）/ 分类 / 商品（含 publish/unpublish/assign-stores/store-overrides/variants）。
- 活动（含 sessions/ticket-types/publish/assign-stores/generate-share-assets）/ 券模板（含 publish/assign-stores/applicable-items/grant/void）。
- 订单只读 / 支付单 / 支付流水 / 退款单 / `POST /admin/refunds`（退款审批，需幂等）。
- 会员 / `POST /admin/members/{id}/wallet-adjustments`（人工调账，需幂等 + 原因）/ 钱包账本。
- Banner / 快捷充值 / VIP 等级 / 规则定义（含 publish/disable）。
- 报表 `/admin/reports/*`、审计 `/admin/audit-logs`、登录日志、错误事件、支付渠道配置读写。

## 十、修改文件列表

本任务仅在 `admin-console/` 内新增/修改，未触碰 `server/`、`store-console/`、`mini-program/`、`design/`、`tasks/acceptance/`、`image-key.json`。

工程配置：`package.json`、`pnpm-workspace.yaml`、`.npmrc`、`vite.config.ts`、`tsconfig.json`、`tsconfig.node.json`、
`env.d.ts`、`eslint.config.js`、`index.html`、`.gitignore`、`.env.example`、`.env.development`、`.env.production`、`README.md`。

源码 `src/`：`main.ts`、`App.vue`、`AppShell.vue`；
`api/`（http、resource、types、models、services/auth、services/index）；
`constants/`（api-paths、enums、permissions、roles）；
`utils/`（columns、format、id、jwt、storage、feedback）；
`composables/`（useDataTable、useAuditedAction、usePublishableActions）；
`stores/`（auth、permission）；`directives/permission`；
`layouts/`（DefaultLayout、menu）；`router/`（index、routes）；
`styles/`（tokens.css、theme.ts）；`components/`（10 个公共组件 + ui-types）；
`pages/`（25 个页面，见第三节）。
