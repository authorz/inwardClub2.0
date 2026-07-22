# InwardClub 2.0 总后台架构设计

## 1. 定位

总后台是系统级管理站点，只服务 `super_admin` 及后续可能拆分出的总部运营、财务、审计角色。总后台拥有全局数据视角，可以跨门店查看和管理资源，但所有跨店写操作必须审计。

总后台必须与门店后台独立：

- 独立域名。
- 独立前端站点。
- 独立登录入口。
- 独立账号表或至少独立账号类型。
- 独立 token audience。
- 独立菜单、路由和权限码。
- 独立部署、监控和错误追踪项目。

总后台不能复用门店后台登录态，门店后台账号也不能登录总后台。这样做的目标是降低水平越权和错配权限的风险。

## 2. 推荐开源底座

推荐使用 `Vue Vben Admin 5` 快速创建总后台。

理由：

- Vue 3、Vite、TypeScript、Pinia、Vue Router，适合长期维护。
- 内置后台布局、动态路由、权限菜单、表格、表单、Mock 等基础能力。
- 国内中后台开发习惯匹配，二次开发成本低。

备选：

- `SoybeanAdmin`：更轻量、风格更现代，适合团队偏好 Naive UI 时选择。
- `Ant Design Pro`：适合团队明确偏 React/Ant Design 时选择。

本项目建议总后台定版为：

```text
Vue Vben Admin 5
Vue 3
TypeScript
Pinia
Vue Router
OpenAPI 类型生成
```

## 3. 工程与部署

总后台作为独立站点：

```text
admin-web/
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
https://admin.inwardclub.cn
```

总后台前端只调用 `/api/v2/admin/*`、公共资产上传凭证接口和必要的内部只读配置接口。禁止调用 `/api/v2/store/*`。

环境变量示例：

```dotenv
VITE_APP_ID=inwardclub-admin-console
VITE_API_BASE_URL=https://api.inwardclub.cn/api/v2
VITE_AUTH_AUDIENCE=admin
VITE_ASSET_PUBLIC_DOMAIN=
```

## 4. 账号与认证

总后台账号只来自 `admin_accounts` 或等效总部账号表。

认证接口：

```text
POST /api/v2/admin/auth/login
POST /api/v2/admin/auth/refresh
GET  /api/v2/admin/auth/me
POST /api/v2/admin/auth/logout
```

总后台 JWT 必须包含：

```json
{
  "subject_type": "admin",
  "role": "super_admin",
  "aud": "admin",
  "store_id": null,
  "token_version": 1
}
```

总后台前端收到 `aud != admin` 或 `subject_type` 不匹配的 token，必须立即清除登录态并返回登录页。服务端同样必须拒绝错误 audience。

## 5. 权限模型

第一期可以使用固定角色：

```text
super_admin
finance_admin
ops_admin
audit_admin
support_admin
```

权限码示例：

```text
admin.store.read
admin.store.write
admin.catalog.global.write
admin.catalog.store_override.write
admin.activity.global.write
admin.coupon.global.write
admin.member.read
admin.member.wallet_adjust
admin.payment.read
admin.refund.approve
admin.rule.publish
admin.audit.read
admin.system.payment_settings.write
```

权限分层：

1. 页面权限：菜单和路由。
2. 操作权限：新增、编辑、删除、发布、退款、调账、导出。
3. 数据权限：由服务端强制。前端可展示门店筛选器，但不能绕过服务端权限。

员工账号绑定规则：

- 总后台可以创建或维护工作人员账号，并为其指定唯一绑定门店。
- 同一个微信用户/手机号/工作人员账号不得绑定多个门店；如需调整门店，必须走受审计的解绑/迁移流程，而不是新增第二个绑定。
- 小程序工作人员端和门店后台均不提供员工自行选择门店的能力。

## 6. API Client

总后台只生成和使用 admin API client：

```text
docs/openapi/v2.yaml
  -> admin-api-client
  -> admin domain services
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

- 退款
- 人工调账
- 批量发券
- 规则发布
- 商品/活动批量上下架
- 库存调整
- 支付配置修改

## 7. 总后台菜单

建议菜单：

```text
工作台
门店管理
账号与权限
  总后台账号
  门店管理员
  员工
商品与积分商城
  全局分类
  全局商品
  门店覆盖
活动管理
  全局活动
  场次
  票档
  投放门店
券管理
  券模板
  适用商品
  发券记录
订单与支付
  统一订单
  支付单
  支付流水
  退款单
会员管理
  会员列表
  钱包账本
  人工调账
规则中心
  签到规则
  邀请规则
  低消规则
  VIP 福利
  充值产品
运营资源
  Banner
  资产库
  默认桌座背景
设备与打印
  打印机
  打印任务
报表
  总览
  收款
  商品
  活动
  券
审计与系统
  审计日志
  登录日志
  错误事件
  支付配置
```

## 8. 页面原则

总后台页面默认有门店筛选器，但写操作必须明确展示影响范围。

跨店写入必须要求：

- 选择目标门店或投放范围。
- 填写操作原因。
- 二次确认。
- 写入审计日志。

规则发布必须支持：

- 草稿。
- 预览。
- 发布。
- 禁用。
- 版本历史。
- 命中记录查看。

## 9. 重点模块

### 9.1 商品

总后台维护全局分类、全局商品模板和跨店投放。

必须支持：

- 全局商品创建、编辑、上下架。
- 商品规格。
- 支付方式白名单。
- 投放门店。
- 查看门店覆盖价格、库存、状态。

### 9.2 活动

总后台维护全局活动模板、场次、票档和投放。

必须支持：

- 活动草稿、发布、下架。
- 多场次、多票档。
- 票档库存。
- 活动分享资产生成。
- 门店采用和覆盖情况查看。

### 9.3 券

总后台维护全局券模板和发券规则。

必须支持：

- 券模板。
- 适用商品/分类。
- 有效期规则。
- 跨店发券。
- 发券记录、核销记录、作废。

### 9.4 财务与审计

支付、退款、人工调账、规则发放都必须可追溯。

总后台必须提供：

- 支付单。
- 支付流水。
- 退款单。
- 钱包账本。
- 福利发放记录。
- 审计日志。

## 10. 资产上传

总后台只提交 `assetId`，不接受任意 URL。

组件：

```text
AdminAssetPicker
AdminAssetUpload
AdminAssetPreview
```

上传流程严格遵守 `QINIU_ASSET_SERVICE_SPEC.md`。

## 11. 验收要求

总后台前端验收：

- 门店后台账号不能登录总后台。
- 错误 audience token 会被清除。
- 所有 `/api/v2/admin/*` 请求带 admin token。
- 不出现 `/api/v2/store/*` 请求。
- 跨店写操作必须填写原因并二次确认。
- 退款、调账、发券、发布等操作携带 `Idempotency-Key`。
- 资产上传只提交 `assetId`。

## 12. 第一阶段落地

1. 使用 Vben 初始化 `admin-web`。
2. 删除无关示例页面。
3. 接入总后台登录、refresh、me、logout。
4. 建立 admin-only API client。
5. 建立总后台菜单和权限码。
6. 先做门店、账号、商品、活动、券、订单列表。
7. 再做退款、调账、规则、报表和审计。
