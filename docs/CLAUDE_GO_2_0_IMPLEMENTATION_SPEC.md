# InwardClub 2.0 Go 开发执行规格

## 0. 使用方式

把本文件、`GO_REWRITE_2_0_PLAN.md`、`V1_API_INVENTORY_AND_V2_MAPPING.md`、`QINIU_ASSET_SERVICE_SPEC.md`、`inwardclub.sql` 放入新 Go 项目根目录的 `docs/`。Codex 收到任务后必须先阅读五份文件，然后直接按本文件的里程碑实现和验证；不得重写需求、替换技术栈、在小程序接入支付宝、使用本地文件存储，或修改生产旧库结构。

本文件的规定优先于旧 Laravel 代码和旧接口文档；`V1_API_INVENTORY_AND_V2_MAPPING.md` 作为 v1 已暴露能力的覆盖检查表，优先于旧 OpenAPI 注释和历史接口文档。若业务规则在第 13 节列为“待业务确认”，实现者应建立可配置规则和禁用态，不能自行填入猜测值。

## 1. 固定范围

### 1.1 必须交付

- 微信小程序用户端、门店后台、总后台的 Go API。
- MySQL 8、Redis、异步任务、微信支付、芯烨云打印、七牛 Kodo 资产服务。
- 多门店数据隔离、角色权限、审计日志、幂等处理、支付/钱包账本、数据迁移和对账工具。
- 现网历史数据可迁移、可查询、可追溯；新旧系统可灰度切换。

### 1.2 明确不做

- 不在小程序开放支付宝支付，也不为小程序订单生成支付宝支付参数或入口。
- 不使用本地磁盘、Laravel storage、裸 URL 字段或其他对象存储上传业务文件。
- 不把积分、金币、余额、VIP、低消、邀请等运营数值写死在 Go 或前端。
- 不重写或直接 ALTER 当前正在运营的 v1 数据库；2.0 使用新 MySQL 数据库。
- 不拆微服务；2.0 是模块化单体。

### 1.3 支付规则

小程序支付方式只有 `wechat` 与 `coin`。门店后台可以创建线下聚合收款码，供用户以微信或支付宝扫码支付；支付宝仅存在于该收单渠道回调和门店收款记录，绝不出现在小程序 API、DTO 或 UI。创建收款码时操作员可选填已注册小程序会员手机号，系统解析并锁定会员归属；付款成功后才触发该会员的返金币、积分、成长值、低消等规则。未填写、未匹配或付款失败的收款单不发放会员权益。后台也可保留 `cash`、`card` 作为人工线下收款记录。

聚合收单服务商尚未指定，必须以 `OfflineAcquirer` 接口实现，不能把任意第三方 SDK 写进业务模块。配置服务商后，动态收款码必须对应唯一的门店、收款人、金额、业务用途和有效期；禁止使用无金额/无订单的通用静态码。微信和支付宝退款均只通过原收单渠道的退款能力执行。

## 2. 技术约束

| 项目 | 固定选择 |
| --- | --- |
| Go | 当前稳定版 Go，模块名由项目初始化时确定 |
| HTTP | `github.com/gin-gonic/gin` |
| MySQL | MySQL 8，`utf8mb4`，时区统一 UTC |
| 数据访问 | `sqlc` + `database/sql`；资金、支付、库存禁止 ORM 自动写入 |
| 数据库迁移 | `pressly/goose/v3`，仅追加 migration |
| Redis/任务 | Redis + `hibiken/asynq` |
| JWT | `github.com/golang-jwt/jwt/v5` |
| 日志 | `log/slog` JSON |
| OpenAPI | `docs/openapi/v2.yaml` 为接口唯一契约 |
| 文件 | 七牛 Kodo，实施细节见 `QINIU_ASSET_SERVICE_SPEC.md` |
| 测试 | 标准库 `testing`、`testcontainers-go`（集成测试） |

不能因为方便而新增 GORM、全局单例数据库、业务层直接 SQL 拼接、浮点金额、或无版本的 SQL 文件。

## 3. 项目目录与模块边界

初始化后必须创建以下结构。每个领域模块只访问自己的 repository；跨模块只调用 service 接口或通过 outbox 事件。

```text
cmd/api/main.go
cmd/worker/main.go
cmd/migrate/main.go
cmd/reconcile/main.go
configs/config.example.yaml
deploy/docker-compose.dev.yaml
db/migrations/
db/queries/
docs/openapi/v2.yaml
internal/bootstrap/
internal/platform/{config,db,redis,httpx,authn,rbac,logger,errors,idempotency,outbox}
internal/modules/
  auth/{handler,service,repository,model,dto,errors}.go
  member/{handler,service,repository,model,dto,errors}.go
  store/{handler,service,repository,model,dto,errors}.go
  catalog/{handler,service,repository,model,dto,errors}.go
  reservation/{handler,service,repository,model,dto,errors}.go
  activity/{handler,service,repository,model,dto,errors}.go
  order/{handler,service,repository,model,dto,errors}.go
  payment/{handler,service,repository,model,dto,errors}.go
  wallet/{handler,service,repository,model,dto,errors}.go
  coupon/{handler,service,repository,model,dto,errors}.go
  asset/{handler,service,repository,model,dto,errors,qiniu_client}.go
  printer/{handler,service,repository,model,dto,errors}.go
  reporting/{handler,service,repository,model,dto,errors}.go
  admin/{handler,service,repository,model,dto,errors}.go
  migration/{importer,reconciler,legacy_repository}.go
```

每个模块必须有 handler、service、repository、model、dto、errors 六个文件；复杂模块可额外增加明确职责文件。handler 不写业务逻辑，service 不依赖 HTTP，repository 不返回 HTTP DTO。

## 4. 启动与运行要求

### 4.1 环境变量

实现 `.env.example`，只列变量名。最低变量：

```dotenv
APP_ENV=development
HTTP_ADDR=:8081
MYSQL_DSN=
REDIS_ADDR=
JWT_SIGNING_KEY=
WECHAT_MINI_APP_ID=
WECHAT_MINI_APP_SECRET=
WECHAT_PAY_MCH_ID=
WECHAT_PAY_MCH_CERT_SERIAL_NO=
WECHAT_PAY_PRIVATE_KEY_PATH=
WECHAT_PAY_API_V3_KEY=
WECHAT_PAY_NOTIFY_URL=
OFFLINE_ACQUIRER_PROVIDER=
OFFLINE_ACQUIRER_MERCHANT_ID=
OFFLINE_ACQUIRER_API_KEY=
OFFLINE_ACQUIRER_NOTIFY_URL=
QINIU_ACCESS_KEY=
QINIU_SECRET_KEY=
QINIU_BUCKET=
QINIU_PUBLIC_DOMAIN=
QINIU_PRIVATE_DOMAIN=
QINIU_UPLOAD_CALLBACK_URL=
XPYUN_USER=
XPYUN_UKEY=
```

### 4.2 必须可执行的命令

```bash
go run ./cmd/migrate up
go run ./cmd/api
go run ./cmd/worker
go test ./...
go vet ./...
go run ./cmd/reconcile --source-dsn "$V1_MYSQL_DSN" --target-dsn "$MYSQL_DSN" --report ./tmp/reconciliation.json
```

`docker-compose.dev.yaml` 必须提供 MySQL、Redis 和开发依赖；真实微信、七牛、打印机使用 fake adapter 或环境开关，测试不依赖外网。

## 5. 统一约定

### 5.1 API

- 路由前缀：`/api/v2/mini`、`/api/v2/store`、`/api/v2/admin`；内部回调位于 `/internal`。
- JSON 响应固定为 `{ "data": ..., "meta": ... }` 或 `{ "error": { "code", "message", "details" } }`。
- 列表请求统一 `page`、`pageSize`，最大 100；响应 meta 包含 `page`、`pageSize`、`total`。
- 每个写请求可接受 `Idempotency-Key`，支付、钱包、兑换、核销、签到、人工调账必须要求该头。
- 时间以 RFC3339 UTC 输出；金额字段均为整数分，命名 `amountCent`、`priceCent`，不返回浮点数。
- 所有后台写操作写 `audit_logs`，记录 actor、角色、store scope、对象、前后差异、request ID。

### 5.2 认证与权限

JWT claim 必须包含 `sub`、`subject_type`、`role`、`store_id`、`token_version`、`exp`。身份如下：

| 身份 | 说明 | 数据范围 |
| --- | --- | --- |
| `member` | 小程序会员 | 自己的数据；门店公开资源 |
| `staff` | 已绑定微信的门店工作人员 | 当前所选已绑定门店的审核与核销 |
| `store_admin` | 门店后台管理员 | 指定门店 |
| `cashier` | 门店收银角色 | 指定门店的受限收款操作 |
| `super_admin` | 总后台管理员 | 全局 |

`store_id` 不可取自后台请求参数。store middleware 从 JWT 注入 scope；repository 所有门店查询使用该 scope。总后台跨门店查询必须有显式权限和审计记录。

工作人员绑定规则：

- 一个工作人员身份可以绑定多个门店；同一工作人员与同一门店的组合必须唯一。
- 小程序 staff 页面可以在服务端返回的启用绑定中选择门店；切换后必须由服务端重新签发 staff token，客户端不得自行指定授权范围。
- v2 数据模型通过 `(member_id, store_id)` 与 `(wechat_openid, store_id)` 联合唯一键防止同店重复绑定，同时保留跨店绑定。
- 所有 staff 核销、积分审核、今日活动、核销历史查询都只使用当前 staff token 的 `store_id` scope，业务接口中客户端传入的 `storeId` 不参与授权范围计算。

### 5.3 事务与并发

- 金额、库存、票券库存、钱包余额、核销都在同一 MySQL 事务中处理。
- 对钱包账户、库存记录、票券实例使用 `SELECT ... FOR UPDATE`。
- 每条外部副作用（微信调用、打印、删除七牛对象、通知）先写 outbox/job，事务提交后异步执行。
- MySQL deadlock 可有限重试；业务幂等键唯一索引是最终保护，不依赖 Redis 锁。

## 6. 目标表与迁移顺序

每个 migration 只做一个逻辑变更，文件按以下顺序建立。历史迁移在新库执行，不修改 `docs/inwardclub.sql`。

| Migration 组 | 必须包含的表 | 要点 |
| --- | --- | --- |
| `0001_identity` | `members`、`admin_accounts`、`staff_accounts`、`roles`、`role_permissions` | 保留 legacy ID、openid、手机号、邀请码和 token version；`staff_accounts` 必须用全局唯一约束保证同一微信用户/员工账号只能绑定一个 `store_id` |
| `0002_store` | `stores`、`store_settings`、`store_rules`、`tables`、`seats` | 全局默认规则可被门店覆盖 |
| `0003_assets` | `assets` | 完整按七牛规格实现 |
| `0004_catalog` | `catalog_categories`、`catalog_items`、`catalog_variants`、`store_item_overrides` | 价格分、库存、支付方式白名单、asset ID |
| `0005_wallet` | `wallet_accounts`、`wallet_ledger_entries`、`wallet_holds` | points/coins/cash_balance/growth_value 分账户 |
| `0006_payment` | `business_orders`、`payment_orders`、`payment_transactions`、`refund_orders`、`offline_collection_orders` | 统一支付与退款；外部交易号唯一；聚合码订单独立保存、锁定可空 `member_id`、手机号掩码快照和绑定操作人 |
| `0007_food` | `food_orders`、`food_order_items` | 下单时保存商品、价格、支付规则快照 |
| `0008_activity` | `activities`、`activity_sessions`、`activity_ticket_types`、`activity_orders`、`tickets`、`verifications` | 一单多票、独立票实例、核销记录 |
| `0009_reservation` | `reservations`、`waitlist_entries`、`arrival_records` | 预约、排队、到店三种独立状态机 |
| `0010_coupon` | `coupon_templates`、`coupon_entitlements`、`coupon_redemptions` | 复用商品库，券不复制商品 |
| `0011_rules` | `rule_definitions`、`rule_executions`、`benefit_grants` | 版本、生效期、幂等发放键 |
| `0012_ops` | `banners`、`printer_devices`、`print_jobs`、`audit_logs`、`outbox_events`、`idempotency_keys`、`reporting_daily` | 支撑运营和可观测性 |
| `0013_migration` | `legacy_id_maps`、`migration_runs`、`reconciliation_results` | 支撑可重跑迁移和对账 |

所有金额列采用 `BIGINT` 分；业务资产数量采用 `BIGINT`；状态使用 `VARCHAR(32)` + 代码常量，避免 MySQL ENUM 演进困难。

## 7. 核心不变量

1. 钱包账本只追加，不能 UPDATE/DELETE；修正用相反方向的新账本行。
2. `wallet_accounts.available_amount` 等于已生效账本行合计；每种资产一账户。
3. 支付回调以 `(provider, out_trade_no)` 和外部交易号去重；重复回调只返回成功。
4. 业务订单、支付订单、退款单、券核销、活动票、打印任务都有不可重复的业务编号。
5. 订单创建时写商品/活动/规则快照；后续商品和规则修改不影响历史。
6. 券/票核销只能从 `active` 到 `used` 一次；退款后必须不可用。
7. 资产必须先 `uploaded` 后才能被绑定；业务表只存 `asset_id`。
8. 任何门店操作都不能读写其他门店数据。

## 8. API 实现清单

实现 OpenAPI 后再写 handler。以下为最小资源契约，字段以 DTO 统一为准。

### 8.1 M1 公开读取

```text
GET  /api/v2/mini/stores?lat=&lng=
GET  /api/v2/mini/stores/{storeID}
GET  /api/v2/mini/stores/{storeID}/banners
GET  /api/v2/mini/stores/{storeID}/catalog/categories
GET  /api/v2/mini/stores/{storeID}/catalog/items
GET  /api/v2/mini/stores/{storeID}/activities
GET  /api/v2/mini/activities
GET  /api/v2/mini/activities/{activityID}
GET  /api/v2/mini/membership-tiers
GET  /api/v2/mini/recharge-products
GET  /api/v2/mini/rankings
```

### 8.2 M2 会员与资产

```text
POST /api/v2/mini/auth/wechat/login
POST /api/v2/mini/auth/refresh
POST /api/v2/mini/auth/logout
GET  /api/v2/mini/me
PATCH /api/v2/mini/me
POST /api/v2/mini/me/phone-bindings
POST /api/v2/mini/assets/upload-credentials
POST /api/v2/mini/sign-ins
GET  /api/v2/mini/wallet
GET  /api/v2/mini/wallet/ledger
POST /api/v2/mini/point-savings
POST /api/v2/mini/point-withdrawals
GET  /api/v2/mini/invitations
POST /api/v2/mini/invitations/bind
GET  /api/v2/mini/coupons
```

`/mini/me/phone-bindings` 必须把 v1 “仅获取手机号但不落库”的行为升级为可审计的绑定流程：服务端调用微信接口取得手机号，返回待确认手机号掩码；会员确认后写入会员手机号。不得把“获取手机号”等同于已经绑定。

### 8.3 M3 订单与支付

```text
GET  /api/v2/mini/food-orders
POST /api/v2/mini/food-orders
GET  /api/v2/mini/food-orders/{orderID}
GET  /api/v2/mini/recharge-orders
POST /api/v2/mini/recharge-orders
GET  /api/v2/mini/recharge-orders/{orderID}
GET  /api/v2/mini/activity-orders
POST /api/v2/mini/activity-orders
GET  /api/v2/mini/activity-orders/{orderID}
POST /api/v2/mini/payment-orders/{paymentOrderID}/wechat-jsapi
POST /api/v2/mini/payment-orders/{paymentOrderID}/pay-by-coin
POST /internal/payments/wechat/notify
POST /api/v2/admin/refunds
POST /api/v2/store/refunds
```

`pay-by-coin` 仅在业务规则允许、可用金币足够且订单未支付时执行；它创建钱包账本和支付交易，并把业务订单置为 paid。微信支付创建 JSAPI 参数，回调才可把支付单置为 paid。v1 的通用 `/mini/order/create|query|refund` 不进入 v2 新写模型；读能力合并到订单读模型，会员端退款只能做受控退款申请或由后台发起，不能开放任意退款。

### 8.4 门店线下聚合收款

```text
POST /api/v2/store/offline-collection-orders
GET  /api/v2/store/offline-collection-orders/{collectionOrderID}
POST /api/v2/store/offline-collection-orders/{collectionOrderID}/cancel
POST /internal/payments/offline-acquirer/notify
```

创建请求必须包含 `amountCent`、`subject`、`businessType`、`expiresInSeconds`，可选 `memberPhone`。服务端以规范化手机号精确查找已注册会员；只返回掩码昵称/手机号供操作员确认，并把 `member_id`、掩码手机号快照、绑定操作人、绑定时间写入收款单。原始手机号只用于本次匹配，不写收款单、审计日志、错误响应或任务 payload。手机号未匹配时返回受控 `MEMBER_NOT_FOUND`，操作员可改为不绑定会员继续创建。不得在付款回调时重新按手机号查找，也不得在收款码创建后更换会员归属。

成功响应返回收单服务商提供的动态二维码内容/展示 URL、收款单号、过期时间和可选的掩码会员信息。后端不得尝试识别付款人；只以合规收单回调中的渠道 `wechat` 或 `alipay`、外部交易号和实际金额确认支付。门店后台只能查看本店收款单。

### 8.5 M4 预约、活动履约与券

```text
GET  /api/v2/mini/stores/{storeID}/tables
GET  /api/v2/mini/stores/{storeID}/seats
GET  /api/v2/mini/reservations
POST /api/v2/mini/reservations
GET  /api/v2/mini/reservations/{reservationID}
POST /api/v2/mini/reservations/{reservationID}/cancel
POST /api/v2/mini/waitlist-entries
POST /api/v2/store/reservations/{reservationID}/arrive
POST /api/v2/store/tickets/verify
GET  /api/v2/store/point-savings
POST /api/v2/store/point-savings/{requestID}/review
GET  /api/v2/store/activities/today
GET  /api/v2/store/verifications
POST /api/v2/mini/coupon-redemptions
```

### 8.6 后台通用资源

总后台使用 `/api/v2/admin/*`，门店后台使用 `/api/v2/store/*`。两者是独立站点、独立账号、独立登录入口、独立 token audience；后端可以复用领域 service，但认证中间件、权限码、数据 scope 和审计上下文必须分开。实现以下 CRUD：门店、门店规则、员工、分类、商品、活动/票档/场次、券模板、Banner、桌台、座位、打印机、充值产品、VIP/权益规则、订单、支付、退款、用户、报表、审计日志。门店后台不能创建全局模板，也不能跨店查询。

后台认证接口必须包含：

```text
POST /api/v2/admin/auth/login
POST /api/v2/admin/auth/refresh
GET  /api/v2/admin/auth/me
POST /api/v2/admin/auth/logout

POST /api/v2/store/auth/login
POST /api/v2/store/auth/refresh
GET  /api/v2/store/auth/me
POST /api/v2/store/auth/logout
```

总后台 token 必须使用 `aud=admin`，门店后台 token 必须使用 `aud=store`。任一后台收到错误 audience、错误 subject type 或缺失必要 scope 的 token 都必须拒绝。门店后台 token 必须包含非空 `store_id`，总后台 token 不得通过伪造 `store_id` 降级为门店账号。

### 8.7 后台商品、券、活动接口补充

商品、券、活动在 2.0 必须同时支持总后台全局运营和门店独立运营。总后台负责全局模板、跨店投放、审核和全局查询；门店后台负责本店自有资源、本店上下架、本店价格/库存/支付方式覆盖和本店订单履约。两端不得复制两套业务逻辑，必须共用 service，通过 `scope_type`、`store_id` 和 RBAC 控制行为。

#### 8.7.1 资源归属模型

| 资源 | 总后台能力 | 门店后台能力 | 关键字段 |
| --- | --- | --- | --- |
| 商品分类 | 创建全局分类、查看全部门店分类、禁用违规分类 | 创建/维护本店分类；可引用全局分类作为父级或展示分组 | `scope_type=global/store`、`store_id`、`parent_id`、`asset_id`、`sort_order` |
| 商品/积分商城商品 | 创建全局商品模板、投放门店、审核门店商品、全局上下架 | 创建本店商品；覆盖全局商品的价格、库存、支付方式、图片、状态 | `scope_type`、`store_id`、`source_item_id`、`item_type`、`pay_channels`、`asset_id` |
| 商品规格 | 管理全局规格模板 | 管理本店商品规格；覆盖全局规格价格/库存/状态 | `item_id`、`sku_code`、`price_cent`、`stock_quantity`、`status` |
| 券模板 | 创建全局券模板、跨店投放、审核门店券 | 创建本店券模板；设置本店适用商品/分类、有效期和库存 | `scope_type`、`store_id`、`coupon_type`、`value_cent`、`validity_rule`、`applicable_scope` |
| 券发放/核销 | 全局发券、查看全部券实例和兑换记录 | 仅给本店会员视图发券；仅核销本店可用券 | `coupon_template_id`、`member_id`、`store_id`、`status`、`expires_at` |
| 活动 | 创建全局活动、分配门店、审核门店活动、全局报表 | 创建本店活动；维护本店场次、票档、库存、支付方式和上下架 | `scope_type`、`store_id`、`assigned_store_ids`、`asset_id`、`status` |
| 活动票档/场次 | 配置全局活动的标准票档/场次 | 配置本店活动的票档/场次，或覆盖被投放活动的库存与售卖时间 | `activity_id`、`session_id`、`ticket_type_id`、`price_cent`、`stock_quantity` |

`scope_type=global` 的资源 `store_id` 必须为空，只能由总后台创建。`scope_type=store` 的资源 `store_id` 必须来自 JWT 注入或总后台显式传入。门店后台提交的 `storeId` 参数一律忽略或拒绝，不能用请求参数决定门店范围。

#### 8.7.2 商品与分类接口

总后台接口：

```text
GET    /api/v2/admin/catalog/categories
POST   /api/v2/admin/catalog/categories
GET    /api/v2/admin/catalog/categories/{categoryID}
PATCH  /api/v2/admin/catalog/categories/{categoryID}
DELETE /api/v2/admin/catalog/categories/{categoryID}

GET    /api/v2/admin/catalog/items
POST   /api/v2/admin/catalog/items
GET    /api/v2/admin/catalog/items/{itemID}
PATCH  /api/v2/admin/catalog/items/{itemID}
POST   /api/v2/admin/catalog/items/{itemID}/publish
POST   /api/v2/admin/catalog/items/{itemID}/unpublish
POST   /api/v2/admin/catalog/items/{itemID}/assign-stores
GET    /api/v2/admin/catalog/items/{itemID}/store-overrides
PATCH  /api/v2/admin/catalog/items/{itemID}/store-overrides/{storeID}

GET    /api/v2/admin/catalog/items/{itemID}/variants
POST   /api/v2/admin/catalog/items/{itemID}/variants
PATCH  /api/v2/admin/catalog/variants/{variantID}
DELETE /api/v2/admin/catalog/variants/{variantID}
```

门店后台接口：

```text
GET    /api/v2/store/catalog/categories
POST   /api/v2/store/catalog/categories
GET    /api/v2/store/catalog/categories/{categoryID}
PATCH  /api/v2/store/catalog/categories/{categoryID}
DELETE /api/v2/store/catalog/categories/{categoryID}

GET    /api/v2/store/catalog/global-items
POST   /api/v2/store/catalog/global-items/{itemID}/adopt
GET    /api/v2/store/catalog/items
POST   /api/v2/store/catalog/items
GET    /api/v2/store/catalog/items/{itemID}
PATCH  /api/v2/store/catalog/items/{itemID}
POST   /api/v2/store/catalog/items/{itemID}/publish
POST   /api/v2/store/catalog/items/{itemID}/unpublish
PATCH  /api/v2/store/catalog/items/{itemID}/stock
PATCH  /api/v2/store/catalog/items/{itemID}/payment-rules

GET    /api/v2/store/catalog/items/{itemID}/variants
POST   /api/v2/store/catalog/items/{itemID}/variants
PATCH  /api/v2/store/catalog/variants/{variantID}
DELETE /api/v2/store/catalog/variants/{variantID}
```

商品列表筛选至少支持：`storeID`（仅总后台）、`scopeType`、`categoryID`、`itemType=food/coupon/redeemable/physical`、`status`、`payChannel=wechat/coin`、`keyword`、`createdFrom`、`createdTo`、`page`、`pageSize`。门店后台的 `global-items` 只返回可被本店采用、未被禁用、且在投放范围内的全局商品。

商品创建/更新 DTO 至少包含：

```json
{
  "scopeType": "store",
  "storeId": 1,
  "categoryId": 10,
  "name": "小吃套餐",
  "description": "可选描述",
  "assetId": 123,
  "itemType": "food",
  "priceCent": 8800,
  "stockQuantity": 100,
  "payChannels": ["wechat", "coin"],
  "pointsReward": 0,
  "sortOrder": 10,
  "status": "draft"
}
```

门店采用全局商品时，写入 `store_item_overrides`，请求 DTO 至少包含：

```json
{
  "categoryId": 10,
  "priceCent": 8800,
  "stockQuantity": 50,
  "payChannels": ["wechat"],
  "assetId": 123,
  "status": "active"
}
```

门店覆盖只能影响本店展示、价格、库存、支付方式、排序和状态，不能修改全局商品模板的名称、历史订单快照或其他门店配置。商品被订单、券模板或活动引用后不得硬删除，只能 `archived`。

#### 8.7.3 券接口

总后台接口：

```text
GET    /api/v2/admin/coupon-templates
POST   /api/v2/admin/coupon-templates
GET    /api/v2/admin/coupon-templates/{templateID}
PATCH  /api/v2/admin/coupon-templates/{templateID}
POST   /api/v2/admin/coupon-templates/{templateID}/publish
POST   /api/v2/admin/coupon-templates/{templateID}/unpublish
POST   /api/v2/admin/coupon-templates/{templateID}/assign-stores
GET    /api/v2/admin/coupon-templates/{templateID}/applicable-items
PUT    /api/v2/admin/coupon-templates/{templateID}/applicable-items

POST   /api/v2/admin/coupon-entitlements/grant
GET    /api/v2/admin/coupon-entitlements
GET    /api/v2/admin/coupon-redemptions
POST   /api/v2/admin/coupon-entitlements/{entitlementID}/void
```

门店后台接口：

```text
GET    /api/v2/store/coupon-templates
POST   /api/v2/store/coupon-templates
GET    /api/v2/store/coupon-templates/{templateID}
PATCH  /api/v2/store/coupon-templates/{templateID}
POST   /api/v2/store/coupon-templates/{templateID}/publish
POST   /api/v2/store/coupon-templates/{templateID}/unpublish
GET    /api/v2/store/coupon-templates/{templateID}/applicable-items
PUT    /api/v2/store/coupon-templates/{templateID}/applicable-items

POST   /api/v2/store/coupon-entitlements/grant
GET    /api/v2/store/coupon-entitlements
GET    /api/v2/store/coupon-redemptions
POST   /api/v2/store/coupon-entitlements/{entitlementID}/void
POST   /api/v2/store/coupon-entitlements/{entitlementID}/verify
```

券模板 DTO 至少包含：

```json
{
  "scopeType": "store",
  "storeId": 1,
  "name": "酒水券",
  "description": "可兑换指定酒水",
  "couponType": "exchange",
  "valueCent": 0,
  "pointsPrice": 0,
  "stockQuantity": 100,
  "validityRule": {
    "type": "relative_days",
    "days": 30
  },
  "applicableScope": {
    "type": "items",
    "categoryIds": [],
    "itemIds": [1001, 1002]
  },
  "perMemberLimit": 1,
  "status": "draft"
}
```

券必须复用商品库，不建立第二套餐品库。`applicableScope` 可以按商品、分类或活动票档配置，但核销时必须把命中的规则版本、商品快照和门店范围写入兑换/核销记录。门店后台只能发放、作废、核销本店范围内的券；总后台跨店发券必须写审计日志并记录发券原因。发券、作废、核销、兑换均要求 `Idempotency-Key`。

#### 8.7.4 活动接口

总后台接口：

```text
GET    /api/v2/admin/activities
POST   /api/v2/admin/activities
GET    /api/v2/admin/activities/{activityID}
PATCH  /api/v2/admin/activities/{activityID}
POST   /api/v2/admin/activities/{activityID}/publish
POST   /api/v2/admin/activities/{activityID}/unpublish
POST   /api/v2/admin/activities/{activityID}/assign-stores
POST   /api/v2/admin/activities/{activityID}/generate-share-assets

GET    /api/v2/admin/activities/{activityID}/sessions
POST   /api/v2/admin/activities/{activityID}/sessions
PATCH  /api/v2/admin/activity-sessions/{sessionID}
DELETE /api/v2/admin/activity-sessions/{sessionID}

GET    /api/v2/admin/activities/{activityID}/ticket-types
POST   /api/v2/admin/activities/{activityID}/ticket-types
PATCH  /api/v2/admin/activity-ticket-types/{ticketTypeID}
DELETE /api/v2/admin/activity-ticket-types/{ticketTypeID}

GET    /api/v2/admin/activity-orders
GET    /api/v2/admin/tickets
GET    /api/v2/admin/verifications
```

门店后台接口：

```text
GET    /api/v2/store/activities/global-activities
POST   /api/v2/store/activities/global-activities/{activityID}/adopt
GET    /api/v2/store/activities
POST   /api/v2/store/activities
GET    /api/v2/store/activities/{activityID}
PATCH  /api/v2/store/activities/{activityID}
POST   /api/v2/store/activities/{activityID}/publish
POST   /api/v2/store/activities/{activityID}/unpublish
POST   /api/v2/store/activities/{activityID}/generate-share-assets

GET    /api/v2/store/activities/{activityID}/sessions
POST   /api/v2/store/activities/{activityID}/sessions
PATCH  /api/v2/store/activity-sessions/{sessionID}
DELETE /api/v2/store/activity-sessions/{sessionID}

GET    /api/v2/store/activities/{activityID}/ticket-types
POST   /api/v2/store/activities/{activityID}/ticket-types
PATCH  /api/v2/store/activity-ticket-types/{ticketTypeID}
DELETE /api/v2/store/activity-ticket-types/{ticketTypeID}

GET    /api/v2/store/activity-orders
GET    /api/v2/store/tickets
GET    /api/v2/store/verifications
POST   /api/v2/store/tickets/verify
```

活动 DTO 至少包含：

```json
{
  "scopeType": "store",
  "storeId": 1,
  "title": "周赛活动",
  "description": "列表描述",
  "content": "详情内容",
  "assetId": 123,
  "startAt": "2026-07-14T12:00:00Z",
  "endAt": "2026-07-14T16:00:00Z",
  "payChannels": ["wechat", "coin"],
  "purchaseLimitPerMember": 4,
  "status": "draft"
}
```

票档 DTO 至少包含：

```json
{
  "name": "早鸟票",
  "priceCent": 8800,
  "stockQuantity": 100,
  "saleStartAt": "2026-07-01T00:00:00Z",
  "saleEndAt": "2026-07-10T23:59:59Z",
  "payChannels": ["wechat", "coin"],
  "maxTicketsPerOrder": 4,
  "status": "active"
}
```

活动支持一活动多场次、多票档、多库存。小程序购买多张票时，服务端必须生成多张 `tickets`，每张票一个核销码；订单聚合码只能作为展示或跳转入口。门店采用全局活动时，可以覆盖本店场次、票档库存、售卖时间、支付方式和上下架状态，但不能修改全局活动模板或其他门店活动。

#### 8.7.5 后台订单、支付、退款与报表接口

总后台接口：

```text
GET  /api/v2/admin/orders
GET  /api/v2/admin/orders/{businessOrderID}
GET  /api/v2/admin/payment-orders
GET  /api/v2/admin/payment-transactions
GET  /api/v2/admin/refund-orders
POST /api/v2/admin/refunds
GET  /api/v2/admin/reports/overview
GET  /api/v2/admin/reports/revenue
GET  /api/v2/admin/reports/catalog-items
GET  /api/v2/admin/reports/activities
GET  /api/v2/admin/reports/coupons
GET  /api/v2/admin/reports/records
```

门店后台接口：

```text
GET  /api/v2/store/orders
GET  /api/v2/store/orders/{businessOrderID}
POST /api/v2/store/food-orders/{orderID}/confirm
POST /api/v2/store/food-orders/{orderID}/prepare
POST /api/v2/store/food-orders/{orderID}/ready
POST /api/v2/store/food-orders/{orderID}/complete
POST /api/v2/store/food-orders/{orderID}/cancel
GET  /api/v2/store/payment-orders
GET  /api/v2/store/payment-transactions
GET  /api/v2/store/refund-orders
POST /api/v2/store/refunds
GET  /api/v2/store/reports/overview
GET  /api/v2/store/reports/revenue
GET  /api/v2/store/reports/catalog-items
GET  /api/v2/store/reports/activities
GET  /api/v2/store/reports/coupons
```

订单筛选至少支持：`orderType=food/activity/recharge/coupon/offline_collection`、`paymentStatus`、`orderStatus`、`payChannel`、`memberPhone`、`keyword`、`storeID`（仅总后台）、`createdFrom`、`createdTo`、`page`、`pageSize`。门店后台退款只能操作本店订单，且必须校验角色权限、订单终态、可退金额和原支付渠道能力。退款请求必须要求 `Idempotency-Key`。

#### 8.7.6 后台用户、员工、门店、运营配置接口

总后台接口：

```text
GET    /api/v2/admin/stores
POST   /api/v2/admin/stores
GET    /api/v2/admin/stores/{storeID}
PATCH  /api/v2/admin/stores/{storeID}
GET    /api/v2/admin/stores/{storeID}/settings
PUT    /api/v2/admin/stores/{storeID}/settings

GET    /api/v2/admin/admin-accounts
POST   /api/v2/admin/admin-accounts
PATCH  /api/v2/admin/admin-accounts/{accountID}
POST   /api/v2/admin/admin-accounts/{accountID}/disable

GET    /api/v2/admin/store-admin-accounts
POST   /api/v2/admin/store-admin-accounts
PATCH  /api/v2/admin/store-admin-accounts/{accountID}
POST   /api/v2/admin/store-admin-accounts/{accountID}/disable

GET    /api/v2/admin/staff-accounts
POST   /api/v2/admin/staff-accounts
PATCH  /api/v2/admin/staff-accounts/{staffID}
POST   /api/v2/admin/staff-accounts/{staffID}/disable

GET    /api/v2/admin/members
GET    /api/v2/admin/members/{memberID}
PATCH  /api/v2/admin/members/{memberID}
POST   /api/v2/admin/members/{memberID}/wallet-adjustments

GET    /api/v2/admin/wallet-ledger
GET    /api/v2/admin/membership-tiers
POST   /api/v2/admin/membership-tiers
PATCH  /api/v2/admin/membership-tiers/{tierID}
POST   /api/v2/admin/membership-tiers/{tierID}/disable

GET    /api/v2/admin/recharge-products
POST   /api/v2/admin/recharge-products
PATCH  /api/v2/admin/recharge-products/{productID}
POST   /api/v2/admin/recharge-products/{productID}/disable

GET    /api/v2/admin/banners
POST   /api/v2/admin/banners
PATCH  /api/v2/admin/banners/{bannerID}
DELETE /api/v2/admin/banners/{bannerID}

GET    /api/v2/admin/rule-definitions
POST   /api/v2/admin/rule-definitions
PATCH  /api/v2/admin/rule-definitions/{ruleID}
POST   /api/v2/admin/rule-definitions/{ruleID}/publish
POST   /api/v2/admin/rule-definitions/{ruleID}/disable

GET    /api/v2/admin/audit-logs
GET    /api/v2/admin/login-events
GET    /api/v2/admin/error-events
POST   /api/v2/admin/audit-log-maintenance/cleanup
GET    /api/v2/admin/payment-channel-settings
PUT    /api/v2/admin/payment-channel-settings
```

门店后台接口：

```text
GET    /api/v2/store/profile
PATCH  /api/v2/store/profile
PATCH  /api/v2/store/profile/status
GET    /api/v2/store/settings
PUT    /api/v2/store/settings

GET    /api/v2/store/cashiers
POST   /api/v2/store/cashiers
PATCH  /api/v2/store/cashiers/{cashierID}
POST   /api/v2/store/cashiers/{cashierID}/disable
POST   /api/v2/store/cashiers/{cashierID}/password-reset

GET    /api/v2/store/staff-accounts
POST   /api/v2/store/staff-accounts
PATCH  /api/v2/store/staff-accounts/{staffID}
POST   /api/v2/store/staff-accounts/{staffID}/disable

GET    /api/v2/store/members
GET    /api/v2/store/members/{memberID}
POST   /api/v2/store/members/{memberID}/wallet-adjustments
GET    /api/v2/store/wallet-ledger

GET    /api/v2/store/banners
POST   /api/v2/store/banners
PATCH  /api/v2/store/banners/{bannerID}
DELETE /api/v2/store/banners/{bannerID}

GET    /api/v2/store/tables
POST   /api/v2/store/tables
PATCH  /api/v2/store/tables/{tableID}
DELETE /api/v2/store/tables/{tableID}
GET    /api/v2/store/seats
POST   /api/v2/store/seats
PATCH  /api/v2/store/seats/{seatID}
DELETE /api/v2/store/seats/{seatID}

GET    /api/v2/store/printer-devices
POST   /api/v2/store/printer-devices
PATCH  /api/v2/store/printer-devices/{deviceID}
DELETE /api/v2/store/printer-devices/{deviceID}
```

人工调账、会员等级调整、规则发布、员工禁用、门店设置、Banner 发布、桌座状态变更都必须写 `audit_logs`。门店后台的会员列表是本店业务视图，不代表会员归属门店；不得因此限制会员在其他门店消费。

#### 8.7.7 后台 DTO、状态和权限缺口补齐

所有后台列表响应统一：

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "pageSize": 20,
    "total": 100
  }
}
```

所有后台详情响应必须包含：`id`、`scopeType`、`storeId`、`status`、`createdAt`、`updatedAt`、`createdBy`、`updatedBy`；可被门店覆盖的资源还必须包含 `sourceId`、`isInherited`、`overrideFields`。

后台写接口统一要求：

1. `POST`、`PATCH`、`DELETE`、发布、上下架、发券、核销、退款、调账必须支持或要求 `Idempotency-Key`；资金、库存、发券、核销、退款、调账必须要求该头。
2. 所有资产字段只接受 `assetId`，不能接受图片 URL。
3. 所有金额字段使用整数分，字段名以 `Cent` 结尾。
4. 所有状态使用字符串常量，禁止直接暴露旧库 `INT` 或 `ENUM` 语义。
5. 总后台跨店写入必须记录目标 `store_id`、原因、操作者和前后差异。
6. 门店后台 repository 查询必须强制带 JWT 注入的 `store_id`；不能从 URL、query 或 body 读取门店范围。

后台最小权限矩阵：

| 权限 | super_admin | store_admin | cashier | staff |
| --- | --- | --- | --- | --- |
| 全局商品/券/活动模板 | 读写 | 不可写，可读已投放 | 不可写 | 不可写 |
| 本店商品/券/活动 | 读写全部门店 | 读写本店 | 只读商品和活动，不能改价格库存 | 只读 |
| 订单与退款 | 全局读写，退款审批 | 本店读写，按权限退款 | 本店收款和受限退款申请 | 只读核销相关 |
| 聚合收款 | 全局配置和报表 | 本店创建和查看 | 本店创建和查看 | 无 |
| 券/票核销 | 全局查看 | 本店查看和核销 | 本店核销 | 本店核销 |
| 用户与钱包 | 全局查看和审批调账 | 本店业务视图和受限调账申请 | 只读会员匹配 | 无 |
| 规则发布 | 全局发布 | 本店覆盖规则草稿/申请 | 无 | 无 |
| 审计日志 | 全局查看 | 本店查看 | 无 | 无 |

### 8.8 v1 接口覆盖整理规则

`docs/V1_API_INVENTORY_AND_V2_MAPPING.md` 是 v2 OpenAPI 的覆盖检查表。该文件确认 v1 实际注册 API 路由共 233 条：总后台 101 条、门店后台 63 条、小程序 59 条，其余为支付回调、测试、系统任务和 Swagger 路由。v2 不复制 v1 混乱路径，但每个业务能力必须被明确处理。

实现任一模块时，OpenAPI 和模块完成报告必须列出对应 v1 路径及迁移状态：

| 状态 | 含义 | 要求 |
| --- | --- | --- |
| `重建` | v2 使用新的 REST 资源路径重新实现 | 必须有 OpenAPI、handler、service 测试和权限测试 |
| `合并` | 多个 v1 路径合并到一个 v2 资源或读模型 | 必须列出所有被合并的 v1 路径，避免漏功能 |
| `废弃` | 测试、隐藏 GET 写操作、本地上传等不进入 v2 | 必须写替代措施，如 worker、CLI、七牛资产服务或单测 |
| `待确认` | v1 存在路由或代码，但业务用途不清 | 保留 feature flag、禁用态或只读迁移，不得自行猜测上线 |

覆盖门禁：

1. v1 的测试接口、Swagger、隐藏座位清理 GET、GET 写操作不进入 v2 对外 API。
2. v1 的本地上传接口全部合并到七牛资产服务；活动二维码、头像、Logo、商品图、Banner 均不得回退到本地 storage。
3. v1 的通用、活动、充值、点餐四套微信回调合并为 `POST /internal/payments/wechat/notify`；以支付单区分业务类型。
4. v1 小程序员工管理接口迁移到 `/api/v2/store/*`，并校验 `staff` 身份和门店归属。
5. v1 控制器中存在但未挂路由的能力，按 `V1_API_INVENTORY_AND_V2_MAPPING.md` 第 8.3 节进入 v2 正式接口或 worker，不能因路由未暴露而忽略。
6. 切流前必须用 v1 访问日志复核端点调用量；零调用且无业务确认的接口才可不提供兼容层。

## 9. 关键流程

### 9.1 微信支付

1. 创建业务订单和 `payment_orders(status=pending)`，落库幂等键。
2. 调微信 JSAPI 创建预支付，保存商户订单号与请求摘要，不把密钥写库。
3. 收到通知后验证微信签名、解密资源，以商户订单号加锁。
4. 未支付时写 `payment_transactions`、标记支付单 paid、推进业务订单、写 outbox；已处理时直接成功。
5. worker 异步发积分/成长值、打印、低消规则评估、统计汇总。每项使用独立业务幂等键。

### 9.2 金币支付

1. 锁定 payment order 与 member 的 `coins` 钱包账户。
2. 检查订单允许金币支付和余额充足；写 debit 账本、payment transaction、业务订单 paid。
3. 退款写对应 credit 账本；不得直接修改余额字段。

### 9.3 门店聚合收款码

1. 门店管理员/收银员创建 `offline_collection_orders` 和关联 `payment_orders(status=pending)`，记录门店、金额、用途、有效期、发起人和幂等键。请求含 `memberPhone` 时，在创建事务中锁定匹配会员并固化 `member_id`、掩码手机号快照和绑定审计信息。
2. `OfflineAcquirer` 向已配置收单服务商请求动态聚合码；保存渠道订单号和二维码内容/URL，返回门店后台展示。
3. 收单回调验证服务商签名、来源、商户号、订单号、金额和渠道；锁定支付单后写 `payment_transactions`，记录 `channel=wechat|alipay`，再置支付单和收款单 paid。
4. 已支付且存在 `member_id` 的收款单写支付后处理 outbox；规则 worker 以 `member_id + payment_order_id + rule_version` 作为幂等键发放金币、积分、成长值或其他已启用权益。无 `member_id` 的散客收款不写权益发放任务。
5. 超时收款单由定时任务关闭；取消、支付、过期只允许一个终态。重复回调和轮询必须幂等。
6. 退款由原收单服务商执行，退款结果写 `refund_orders`；对已发放权益写同一来源单的冲正账本/福利撤销任务。若服务商不支持原路退款，接口返回受控错误，禁止人工伪造已退款状态。

### 9.4 活动一单多票与核销

1. 下单锁定票档库存，创建活动订单和 N 张 `tickets`；每张票有唯一短核销码。
2. 订单支付成功才激活票；支付失败/超时释放库存。
3. 工作人员扫码或手输核销码，锁定 ticket，检查活动/门店/场次/状态；创建 `verifications` 并置 ticket used。
4. 聚合码只是跳转或展示工具，服务端最终按单票核销，不能因一张码使 N 张票重复使用。

### 9.5 七牛文件上传

严格按 `QINIU_ASSET_SERVICE_SPEC.md` 完成：签发限定 key 的短时 token、七牛回调验签、资产落库、业务绑定、异步删除。此项在所有业务 CRUD 开始前完成。

## 10. 数据迁移执行规格

### 10.1 来源与写入原则

来源为 v1 只读数据库，结构基线为 `docs/inwardclub.sql`。新库导入不覆盖已有 2.0 写入；所有导入表必须有 `(source_system, source_table, source_id)` 唯一映射或等效 `legacy_id_maps` 记录。

### 10.2 导入顺序

```text
stores -> admins/users/staff/vip_level -> categories/products/banner/settings/printer_device
-> tables/seats/reservations -> activities/activity_orders -> food_orders/food_order_items
-> recharge/coin_product -> user_points/save_points/points_withdrawal/*_consumption_records/transaction_records
-> user_coupon/coupon_order_items -> invitations/sign_in
```

历史 `users` 资产字段存在 `total_points`、`used_points`、`points`、`balance`、`all_balance` 语义歧义。导入器必须先建立成员与来源映射，再把经业务确认的展示余额写为 `migration_opening_balance` 账本，随后以各流水回放核对。不能因无法完全回放而丢弃记录。

### 10.3 对账 CLI

`cmd/reconcile` 输出 JSON 和可读 Markdown，至少包含：表行数、主键映射数、每门店/日已支付餐品/活动/充值金额、每用户各资产余额、未使用活动票、未过期券、支付交易号冲突、孤儿外键、无法导入行。非零差异必须有批准的差异原因，不能静默通过。

## 11. 任务与定时作业

| 任务 | 触发 | 幂等键 |
| --- | --- | --- |
| 微信支付后处理 | 支付 outbox | `payment:{id}:post-process` |
| 聚合收款单过期 | 每分钟 | `offline-collection:{id}:expire` |
| 打印小票/券 | 已支付订单或兑换 | `print:{businessOrderNo}:{template}` |
| 预约超时释放 | 每分钟 | `reservation:{id}:expire` |
| 活动订单超时关闭 | 每分钟 | `activity-order:{id}:expire` |
| 票券/用户券过期 | 每小时 | `ticket/coupon:{id}:expire` |
| VIP 月度福利 | 每日 | `benefit:{ruleVersion}:{member}:{period}` |
| 低消/邀请奖励 | 支付后处理 | `rule:{ruleVersion}:{order}` |
| 资产 pending 清理 | 每小时 | `asset:{id}:expire` |
| 报表汇总 | 每日 | `report:{store}:{date}` |

## 12. 测试与验收门禁

每个模块至少有 service 单测和 repository 集成测试。CI 必须执行：

```bash
go test ./...
go test -race ./...
go vet ./...
goose -dir db/migrations mysql "$MYSQL_DSN" status
```

必须覆盖的端到端场景：微信支付重复通知、金币并发支付、微信退款/金币退款、聚合收款码微信/支付宝回调、聚合收款手机号匹配/未匹配/未填写、收款码创建后会员不可替换、聚合收款支付后权益只发一次、退款权益冲正、聚合收款金额或签名篡改、收款码过期、跨门店越权、库存超卖、活动多票核销、券重复兑换、签到重复领取、规则重复发放、七牛回调伪造/重放、历史数据重复导入、资产账本对账。任何一个失败不得发布。

## 13. 需业务确认的规则输入

开发者建立规则模型、后台编辑和禁用态，但不得自行启用以下规则：

| 规则 | 需要的正式输入 |
| --- | --- |
| 积分/金币/余额 | 各资产定义、可消费范围、可否互转或提现、退款冲正规则 |
| 签到 | 时间窗口、连续规则、每级奖励、是否需低消、当日补签规则 |
| 低消奖励 | 合格订单类型/支付方式、金额口径、到店判定、20:30 条件、退款后的处理 |
| 邀请 | 绑定时机、首笔有效消费定义、奖励比例/上限、跨店和退款规则 |
| VIP | 升级条件、降级、日/月福利、补发与过期规则 |
| 券 | 面值、适用商品/分类、叠加、有效期、退款/退券规则 |
| 存取积分 | 水上/水下公式、带入证据、审核角色、审批流、异常处理 |
| 聚合收款 | 收单服务商、商户号/门店映射、动态码有效期、支持渠道、退款能力与回调 IP/签名规范；会员手机号绑定是否需要二次确认及可触发的权益规则 |

## 14. Codex 工作协议

1. 先创建空项目、基础设施、migration 和 OpenAPI，不实现临时 mock 业务来替代正式表。
2. 按 M0 -> M6 顺序，一个里程碑完成测试、OpenAPI、migration、日志和审计后再进入下一个。
3. 每次提交只包含一个明确领域或 migration；不改动已稳定模块的无关代码。
4. 遇到第 13 节未确认规则：实现配置数据结构和默认禁用规则，记录阻塞项；继续不依赖该数值的工作。
5. 任何外部服务必须通过 interface + fake adapter 测试；真实密钥只在部署环境使用。
6. 每个里程碑结束输出：变更文件、迁移版本、已实现 API、测试命令和结果、未决规则、数据迁移影响。

## 15. 首次执行任务

Codex 的第一轮工作必须完成 M0 的以下产物，而不是直接写订单或后台页面：

1. 初始化 Go module、目录、配置加载、slog、request ID、错误响应和健康检查。
2. 用 Docker Compose 启动 MySQL 与 Redis；加入 goose 和最小 `0001_identity`、`0002_store`、`0003_assets` migration。
3. 写 `docs/openapi/v2.yaml` 的健康检查、认证、资产上传凭证和门店公开读取接口。
4. 实现 RBAC/store scope、idempotency/outbox 基础表与中间件。
5. 按七牛规格实现 asset 模块的 token 签发、回调验签接口和 fake adapter 单测。
6. 建立 `docs/openapi/v2.yaml` 与 `V1_API_INVENTORY_AND_V2_MAPPING.md` 的覆盖矩阵，至少标注 M0/M1 已覆盖、后续里程碑、废弃和待确认接口。
7. 实现 `cmd/reconcile` 的框架、v1 SQL 表发现和行数/主键映射报告。
8. 运行所有命令，修复失败后提交 M0 完成报告。
