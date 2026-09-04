# InwardClub 2.0 门店后台架构设计

## 1. 定位

门店后台是单店运营站点，只服务 `store_admin`、`cashier` 以及后续可能拆分出的门店运营角色。门店后台只能访问当前账号绑定门店的数据，不提供跨门店数据视图。

门店后台必须与总后台独立：

- 独立域名。
- 独立前端站点。
- 独立登录入口。
- 独立门店账号体系。
- 独立 token audience。
- 独立菜单、路由和权限码。
- 独立部署、监控和错误追踪项目。

门店后台账号不能登录总后台。总后台账号也不应直接登录门店后台；如总部需要代运营门店，必须通过总后台的受审计代操作机制或单独发放门店账号，不能共享登录态。

## 2. 推荐开源底座

推荐使用 `Vue Vben Admin 5` 快速创建门店后台。

门店后台可以和总后台使用同一个开源模板版本，但必须是独立站点、独立工程配置、独立 API client、独立登录态。允许复制基础组件代码，禁止共享运行时权限上下文。

备选：

- `SoybeanAdmin`：适合更轻量的门店后台。
- `Ant Design Pro`：适合团队选择 React 技术栈时。

本项目建议门店后台定版为：

```text
Vue Vben Admin 5
Vue 3
TypeScript
Pinia
Vue Router
OpenAPI 类型生成
```

## 3. 工程与部署

门店后台作为独立站点：

```text
store-web/
  src/
    api/
    auth/
    router/
    stores/
    layouts/
    pages/
    components/
    domain/
```

部署建议：

```text
https://store.inwardclub.cn
```

门店后台前端只调用 `/api/v2/store/*`、公共资产上传凭证接口和必要的公开读取接口。禁止调用 `/api/v2/admin/*`。

环境变量示例：

```dotenv
VITE_APP_ID=inwardclub-store-console
VITE_API_BASE_URL=https://api.inwardclub.cn/api/v2
VITE_AUTH_AUDIENCE=store
VITE_ASSET_PUBLIC_DOMAIN=
```

## 4. 账号与认证

门店后台账号来自 `store_admin_accounts`、`cashier_accounts` 或等效门店账号表。

认证接口：

```text
POST /api/v2/store/auth/login
POST /api/v2/store/auth/refresh
GET  /api/v2/store/auth/me
POST /api/v2/store/auth/logout
```

门店后台 JWT 必须包含：

```json
{
  "subject_type": "store_admin",
  "role": "store_admin",
  "aud": "store",
  "store_id": 1,
  "token_version": 1
}
```

收银员示例：

```json
{
  "subject_type": "cashier",
  "role": "cashier",
  "aud": "store",
  "store_id": 1,
  "token_version": 1
}
```

门店后台前端收到 `aud != store`、`store_id` 为空或 `subject_type` 不匹配的 token，必须立即清除登录态并返回登录页。服务端同样必须拒绝错误 audience。

## 5. 水平越权防护

门店后台设计原则：

1. 前端不展示门店选择器。
2. 前端不允许用户输入或切换 `storeId`。
3. API 请求 body、query、path 中不使用 `storeId` 决定数据范围。
4. 当前门店只从服务端 JWT scope 和 `/store/auth/me` 返回。
5. 服务端 repository 必须强制注入 `store_id`。
6. 若接口返回了其他门店数据，前端应报警并阻断展示。

门店后台页面可以展示当前门店名称，但只是展示，不参与请求参数。

员工绑定规则：

- 门店后台创建或维护工作人员时，只能绑定当前 token 的门店。
- 工作人员账号可以同时绑定多个门店，但门店后台不能替其他门店新增或修改绑定。
- 如果同一个微信用户/手机号已经绑定当前门店，服务端必须拒绝重复绑定并返回明确错误；其他门店的绑定不受影响。
- 员工列表、核销、积分审核等页面不得出现门店选择器、门店下拉或跨店操作。

## 6. 权限模型

第一期角色：

```text
store_admin
cashier
store_operator
```

权限码示例：

```text
store.profile.read
store.profile.write
store.catalog.write
store.activity.write
store.coupon.write
store.order.read
store.order.status_write
store.collection.create
store.refund.request
store.member.read
store.member.wallet_adjust_request
store.reservation.write
store.ticket.verify
store.staff.write
store.printer.write
store.report.read
store.audit.read
```

收银员默认只开放：

```text
store.order.read
store.collection.create
store.collection.read
store.refund.request
store.member.read_limited
```

## 7. API Client

门店后台只生成和使用 store API client：

```text
docs/openapi/v2.yaml
  -> store-api-client
  -> store domain services
  -> pages
```

请求层统一处理：

- `Authorization`
- `X-Request-ID`
- `Idempotency-Key`
- 401 refresh
- 403 权限不足
- 409 状态冲突
- 422 表单错误映射

高风险操作必须生成 `Idempotency-Key`：

- 创建聚合收款码。
- 退款申请。
- 核销票/券。
- 发券。
- 库存调整。
- 人工调账申请。
- 订单状态流转。

## 8. 门店后台菜单

建议菜单：

```text
工作台
门店资料
本店商品
  分类
  商品
  采用全局商品
本店活动
  活动
  场次
  票档
  采用全局活动
本店券
  券模板
  发券
  核销
订单
  点餐订单
  活动订单
  充值记录
  兑换订单
收款
  聚合收款码
  收款记录
  退款申请/退款单
会员
  本店会员视图
  钱包流水
预约与桌座
  预约
  排队
  桌台
  座位
员工与收银员
打印机
报表
  今日总览
  收款趋势
  热门商品
  活动核销
审计日志
```

收银员菜单建议：

```text
工作台
收款
  创建收款码
  收款记录
订单
  点餐订单
  活动订单
会员查询
退款申请
```

## 9. 重点模块

### 9.1 本店商品

门店后台支持两种商品来源：

- 本店自建商品。
- 采用总后台投放的全局商品。

门店可维护：

- 本店分类。
- 本店商品价格。
- 本店库存。
- 本店支付方式。
- 本店商品图片。
- 本店上下架。

门店不可修改全局商品模板，不可影响其他门店。

### 9.2 本店活动

门店后台支持：

- 本店自建活动。
- 采用总后台投放的全局活动。

门店可维护：

- 本店场次。
- 本店票档库存。
- 本店售卖时间。
- 本店支付方式。
- 本店上下架。
- 本店核销记录。

门店不可修改全局活动模板，不可影响其他门店。

### 9.3 本店券

门店后台支持：

- 本店券模板。
- 本店发券。
- 本店券核销。
- 本店兑换记录。

券适用商品必须引用商品库或分类，不复制第二套餐品库。

### 9.4 收款

门店后台创建动态聚合收款码：

```text
POST /api/v2/store/offline-collection-orders
```

要求：

- 固定金额。
- 固定门店。
- 固定收款人。
- 固定有效期。
- 可选会员手机号匹配。
- 支付成功后才触发权益发放。

门店后台不得展示或创建无金额通用静态码。

### 9.5 会员视图

门店后台的会员列表是本店业务视图，不代表会员归属。

可展示：

- 会员昵称、手机号掩码、等级。
- 本店消费/核销/预约记录。
- 钱包摘要。

不可做：

- 跨店会员数据查询。
- 任意修改会员全局身份。
- 直接改余额。

人工调账只能作为受控申请或有权限的门店操作，并必须写账本和审计。

## 10. 资产上传

门店后台只提交 `assetId`，不接受任意 URL。

组件：

```text
StoreAssetPicker
StoreAssetUpload
StoreAssetPreview
```

上传 purpose 必须受门店权限限制：

- `store_logo`
- `category`
- `product`
- `activity`
- `seat_layout`
- `rich_content`

## 11. 页面原则

门店后台页面默认不出现：

- 门店选择器。
- 全局模板编辑入口。
- 跨店报表。
- 跨店用户查询。
- 支付渠道配置。

危险操作必须二次确认：

- 退款申请。
- 核销。
- 批量下架。
- 库存调整。
- 发券。
- 员工/收银员禁用。

## 12. 验收要求

门店后台前端验收：

- 总后台账号不能登录门店后台。
- 错误 audience token 会被清除。
- `store_id` 为空的 token 会被清除。
- 所有 `/api/v2/store/*` 请求带 store token。
- 不出现 `/api/v2/admin/*` 请求。
- 页面不展示门店选择器。
- 请求不传 `storeId` 作为数据范围。
- 退款、核销、发券、库存调整等操作携带 `Idempotency-Key`。
- 资产上传只提交 `assetId`。

## 13. 第一阶段落地

1. 使用 Vben 初始化 `store-web`。
2. 删除无关示例页面。
3. 接入门店后台登录、refresh、me、logout。
4. 建立 store-only API client。
5. 建立门店后台菜单和权限码。
6. 先做门店资料、商品、活动、券、订单、预约、桌座。
7. 再做聚合收款、退款申请、核销、会员视图、报表和审计。
