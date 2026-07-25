# InwardClub v1 完整接口盘点与 v2 映射

## 1. 基线与使用规则

本清单以 2026-07-14 在当前 Laravel 项目执行 `php artisan route:list --path=api --json` 的结果为准，而非旧接口文档。实际注册路由共 233 条：总后台 101 条、门店后台 63 条、小程序 59 条，其余为支付回调、测试、系统任务和 Swagger 路由。

本文件解决两个问题：

1. 保留所有现有已暴露的业务能力，避免 Go 2.0 因“按领域概述”漏实现。
2. 不复制 v1 混乱路径。v2 以 REST 资源路径重新设计；每项标注 `重建`、`合并`、`废弃` 或 `待确认`。

`GET|HEAD` 在下表统一写成 `GET`。除特别说明外，`/api` 是 v1 公共前缀，v2 统一使用 `/api/v2`。

## 2. 路由身份与范围

| v1 范围 | 实际中间件 | v2 身份 | 说明 |
| --- | --- | --- | --- |
| `/mini` 公开路由 | `api` | 匿名/会员 | 门店、活动、商品浏览可匿名 |
| `/mini` 受保护路由 | `miniprogram.jwt` | `member` 或 `staff` | 员工审核能力必须额外校验 `staff` 及门店归属 |
| `/store` | `store.jwt` | `store_admin`、`cashier` | 强制 token 的 `store_id` 数据范围 |
| `/admin` | `admin.jwt` | `super_admin` | 全局权限，敏感操作审计 |
| 回调 | `api` | 服务商签名 | 微信支付、聚合收单、七牛各自独立验签 |

## 3. v1 公开、回调、系统和测试接口

| v1 方法与路径 | 现有功能 | v2 处理 |
| --- | --- | --- |
| `ANY /payment/notify` | 通用微信支付通知 | 合并为 `POST /internal/payments/wechat/notify`，重建 |
| `ANY /activity/notify` | 活动订单微信通知 | 合并进统一微信通知，以支付单区分业务类型 |
| `ANY /mini/recharge/notify` | 充值微信通知 | 合并进统一微信通知 |
| `ANY /mini/food-order/notify` | 点餐微信通知 | 合并进统一微信通知 |
| `GET /systems/auto/clear/seat/hasdhoio21322` | 释放/清理座位预约 | 废弃隐藏 GET；改为预约过期 worker 与内部受限管理命令 |
| `GET /test`、`/printer/test`、`/password/test`、`/test/audit`、`/test/unreverse/{id}/{status}` | 测试、打印、密码、审计、积分撤销修复 | 不进入 v2 对外 API；分别以单测、受限 CLI、审计修复脚本替代 |
| `/documentation`、`/oauth2-callback` | Swagger UI 基础设施 | v2 使用 OpenAPI UI，可保留但不属于业务兼容范围 |

## 4. 小程序端完整接口清单

### 4.1 认证、账户和会员资产

| v1 方法与路径 | 功能 | v2 路径/状态 |
| --- | --- | --- |
| `POST /mini/auth/login` | 微信小程序登录 | `POST /mini/auth/wechat/login`，重建 |
| `POST /mini/auth/refresh` | 刷新 JWT | `POST /mini/auth/refresh`，重建 |
| `GET /mini/auth/me` | 当前用户 | `GET /mini/me`，重建 |
| `PUT /mini/auth/update` | 更新会员资料 | `PATCH /mini/me`，重建 |
| `POST /mini/auth/logout` | 退出登录 | `POST /mini/auth/logout`，重建 |
| `POST /mini/user/get/phone` | 微信手机号授权与绑定 | `POST /mini/me/phone-bindings`，重建 |
| `POST /mini/user/avatar/upload` | 头像上传 | 合并为七牛资产凭证 + `PATCH /mini/me` 的 `assetId` |
| `GET /mini/user/points` | 积分流水 | `GET /mini/wallet/ledger?assetType=points`，合并 |
| `POST /mini/user/points/save` | 用户存积分申请 | `POST /mini/point-savings`，重建审批流 |
| `POST /mini/user/points/withdrawal` | 用户积分提现/取分申请，可能打印 | `POST /mini/point-withdrawals`，重建审批流和打印任务 |
| `GET /mini/user/transactions` | 用户交易记录 | `GET /mini/wallet/ledger`，合并 |
| `GET /mini/vip/list` | VIP 等级 | `GET /mini/membership-tiers`，重建 |
| `GET /mini/ranking` | 排行榜 | `GET /mini/rankings`，重建为周/月/总榜快照 |

### 4.2 门店、内容、餐品和桌座浏览

| v1 方法与路径 | 功能 | v2 路径/状态 |
| --- | --- | --- |
| `GET /mini/stores`、`GET /mini/stores/{id}` | 门店列表、详情 | `GET /mini/stores`、`GET /mini/stores/{storeId}`，重建；支持定位排序 |
| `GET /mini/banner/list/{storeId}` | 门店 Banner | `GET /mini/stores/{storeId}/banners`，重建 |
| `GET /mini/{storeId}/tables/list` | 门店桌台与状态 | `GET /mini/stores/{storeId}/tables`，重建 |
| `GET /mini/{storeId}/products/categories` | 商品分类 | `GET /mini/stores/{storeId}/catalog/categories`，重建 |
| `GET /mini/{storeId}/products/products/{categoryId}` | 分类商品 | `GET /mini/stores/{storeId}/catalog/items?categoryId=`，重建 |
| `GET /mini/activities/{storeId}` | 门店活动列表 | `GET /mini/stores/{storeId}/activities`，重建 |
| `GET /mini/activities/{storeId}/{id}` | 活动详情 | `GET /mini/activities/{activityId}`，重建 |
| `POST /mini/seats/reserved` | 座位预约 | 合并进 `POST /mini/reservations`；v1 与 `user/reservations` 重复 |

### 4.3 小程序预约、订单、支付和充值

| v1 方法与路径 | 功能 | v2 路径/状态 |
| --- | --- | --- |
| `GET|POST /mini/user/reservations` | 我的预约列表、创建 | `GET|POST /mini/reservations`，重建 |
| `GET /mini/user/reservations/{id}` | 预约详情 | `GET /mini/reservations/{reservationId}`，重建 |
| `DELETE /mini/user/reservations/cancel/{reservationId}` | 取消预约 | `POST /mini/reservations/{reservationId}/cancel`，重建状态机 |
| `GET|POST /mini/user/orders`、`GET /mini/user/orders/{id}` | 旧通用订单列表/创建/详情 | 合并到统一订单读模型；确认 v1 数据用途后迁移，不新增该泛化写接口 |
| `GET|POST /mini/user/activity-orders`、`GET /mini/user/activity-orders/{id}` | 旧活动订单列表/创建/详情 | 读接口合并至 `GET /mini/activity-orders`；写入由活动下单接口处理 |
| `GET /mini/orders/activity`、`/activity/info/{id}` | 新活动订单列表、详情 | `GET /mini/activity-orders`、`GET /mini/activity-orders/{id}`，重建 |
| `GET /mini/orders/recharge`、`/recharge/info/{id}` | 充值订单列表、详情 | `GET /mini/recharge-orders`、`GET /mini/recharge-orders/{id}`，重建 |
| `GET /mini/orders/food`、`/food/info/{id}` | 点餐订单列表、详情 | `GET /mini/food-orders`、`GET /mini/food-orders/{id}`，重建 |
| `POST /mini/order/create`、`GET /mini/order/query`、`POST /mini/order/refund` | 旧通用微信下单/查询/退款 | 合并进 `payment_orders`；会员端不开放任意退款，使用受控退款申请 |
| `GET /mini/activities/purchase/{activityId}`、`POST /mini/activity/buy` | 活动购买，存在 GET 写操作 | 统一 `POST /mini/activity-orders`，重建；删除 GET 写入 |
| `POST /mini/recharge/add`、`POST /mini/recharge/query` | 充值下单、查询支付状态 | `POST /mini/recharge-orders`、`GET /mini/recharge-orders/{id}`，重建 |
| `POST /mini/food/add`、`POST /mini/food/query` | 点餐下单、查询支付状态 | `POST /mini/food-orders`、`GET /mini/food-orders/{id}`，重建 |
| `GET /mini/coin/list` | 快捷充值产品 | `GET /mini/recharge-products`，重建 |

### 4.4 邀请、券和员工能力

| v1 方法与路径 | 功能 | v2 路径/状态 |
| --- | --- | --- |
| `GET /mini/user/invitations` | 我的邀请关系/记录 | `GET /mini/invitations`，重建 |
| `POST /mini/user/invite` | 邀请或绑定邀请关系 | `POST /mini/invitations/bind`，重建；只允许一次绑定 |
| `GET /mini/coupon/list` | 我的券 | `GET /mini/coupons`，重建 |
| `POST /mini/coupon/exchange` | 使用券兑换商品，可能打印 | `POST /mini/coupon-redemptions`，重建为订单+券状态机 |
| `GET /mini/management/save/points/list` | 员工待审核存积分列表 | `GET /store/point-savings?status=pending`，迁移到门店后台 |
| `POST /mini/management/audit/points` | 员工审核存积分 | `POST /store/point-savings/{id}/review`，迁移到门店后台 |
| `POST /mini/management/audit/activity` | 员工活动核销 | `POST /store/tickets/verify`，重建 |
| `GET /mini/management/today/activity` | 今日活动/核销数据 | `GET /store/activities/today`，迁移到门店后台 |
| `GET /mini/management/audit/activity/history` | 活动核销历史 | `GET /store/verifications`，迁移到门店后台 |

## 5. 门店后台完整接口清单

除下列特殊接口外，所有 `/store` 接口都限定当前 token 的门店。v2 对相同资源统一 REST 语义，不保留 v1 的 URL 差异。

| v1 资源和方法 | 实际功能 | v2 处理 |
| --- | --- | --- |
| `GET|POST /store/users`，`GET|PUT|DELETE /store/users/{user}` | 本店会员查看、创建、详情、编辑、删除 | `GET /store/members`、`GET|PATCH /store/members/{id}`；禁止随意物理删除会员，删除改停用/匿名化 |
| `GET|POST /store/tables`，`GET|PUT|DELETE /store/tables/{table}` | 本店桌台 CRUD | `GET|POST /store/tables`、`GET|PATCH|DELETE /store/tables/{id}`，重建 |
| `GET|POST /store/categories`，`GET|PUT|DELETE /store/categories/{category}`，`POST /store/categories/batch-sort` | 本店分类 CRUD、排序 | `catalog/categories` 资源与批量排序，重建 |
| `GET|POST /store/products`，`GET|PUT|DELETE /store/products/{product}`，`POST /batch-status`、`POST /batch-sort`、`PUT /{product}/stock`、`GET /products/categories` | 本店商品 CRUD、上下架、排序、库存、分类选择器 | `catalog/items`、批量状态/排序、库存调整，重建 |
| `GET|POST /store/activities`，`GET|PUT|DELETE /store/activities/{activity}` | 本店活动 CRUD | `activities`、场次、票档、库存，扩展重建 |
| `GET|POST /store/reservations`，`GET|PUT|DELETE /store/reservations/{reservation}` | 本店预约 CRUD | `reservations` + 到店/取消/过期状态机，重建 |
| `GET|POST /store/orders`，`GET|PUT|DELETE /store/orders/{order}` | 门店旧通用订单 CRUD | 统一订单读模型；不允许后台任意创建/删除已支付订单，按订单类型受控操作 |
| `GET|POST /store/staff`，`GET|PUT|DELETE /store/staff/{staff}` | 本店员工 CRUD | `staff` + 微信绑定/角色，重建；工作人员只能绑定一个门店，门店后台创建时自动使用当前 token 门店，不允许选择其他门店 |
| `POST /store/payments/cash`、`/card`、`/alipay` | 现金、刷卡、支付宝线下收款记录 | 合并为 `POST /store/offline-collection-orders`；动态聚合码支持微信/支付宝，人工现金/刷卡单独记录 |
| `GET|PUT /store/settings/info`，`POST /store/settings/logo` | 门店资料、Logo | `GET|PATCH /store/profile`，Logo 改七牛 `assetId` |
| `GET|POST /store/cashiers`，`GET|PUT|DELETE /store/cashiers/{cashier}`，`PUT /store/cashiers/password/{id}` | 收银员 CRUD、改密码 | `cashiers` 资源；密码重置单独审计，重建 |
| `POST /store/images/upload`、`upload-multiple`、`info`，`DELETE /images/delete`，`GET /images/config` | 本地图片上传、删除、配置 | 全部合并至七牛资产服务；不兼容本地文件路径 |
| `GET /store/dashboard/stats` | 门店 Dashboard 汇总 | `GET /store/reports/dashboard`，重建 |

下列控制器方法存在但没有 v1 路由：订单 `confirm/prepare/ready/complete/cancel`、支付记录/统计、门店营业状态、趋势/热门商品/用户活跃统计。v2 必须保留为正式能力，接口为 `POST /store/food-orders/{id}/confirm|prepare|ready|complete|cancel`、`GET /store/payment-orders`、`GET /store/reports/*`、`PATCH /store/profile/status`；这些不是 v1 兼容路径。

## 6. 总后台完整接口清单

| v1 资源和方法 | 实际功能 | v2 处理 |
| --- | --- | --- |
| `POST /admin/auth/login`、`refresh`；`GET /admin/auth/me`；`POST /admin/auth/logout` | 总后台认证 | `admin/auth/*`，重建 |
| `GET|POST /admin/stores`，`GET|PUT|DELETE /admin/stores/{store}`，`PUT /admin/stores/admin/{storeId}` | 门店 CRUD、绑定门店管理员 | `admin/stores` + `admin/stores/{id}/admin-account`，重建 |
| `GET|POST /admin/admins`，`GET|PUT|DELETE /admin/admins/{admin}` | 总管理员 CRUD | `admin/admin-accounts`，重建 |
| `GET|POST /admin/store-admins`，`GET|PUT|DELETE /admin/store-admins/{storeAdmin}` | 门店管理员 CRUD | `admin/store-admin-accounts`，重建 |
| `GET /admin/store-admins/batch-status` | 返回固定 `1` 的占位路由 | 废弃 |
| 分类 CRUD + `POST /admin/categories/batch-sort` | 全局分类管理、排序 | `admin/catalog/categories`，重建 |
| 商品列表、`add/info/update/delete`、`batch-status`、`batch-sort`、`{product}/stock` | 全局商品、状态、排序、库存 | `admin/catalog/items`，重建 |
| 全局活动列表、`add/info/update/delete`、`assign-stores/{activityId}` | 全局活动 CRUD、投放门店 | `admin/activities` + `assignments`，重建 |
| 桌台 `list/add/info/update/delete` | 所有门店桌台管理 | `admin/tables`，重建 |
| 座位 `list/add/info/update/delete` | 所有门店座位管理 | `admin/seats`，重建 |
| `settings/member-levels`、`invite-rules`、`point-rules` 的 GET/PUT | 会员、邀请、积分规则 | 合并为版本化 `admin/rules`，重建 |
| `GET /admin/logs/operations`、`/logins` | 操作、登录日志 | `admin/audit-logs`、`admin/login-events`，重建 |
| `images/upload`、`upload-multiple`、`info`、`delete`、`config` | 总后台本地图片管理 | 合并至七牛资产服务 |
| `vip/list/add/update/delete` | VIP 等级 CRUD | `admin/membership-tiers`，重建 |
| `orders/activity|recharge|food` 的 `list/info`；`GET /orders/food/update/{id}` | 三类订单读取，点餐订单以 GET 更新 | 统一 `admin/orders` 与按类型过滤；状态更新改 `PATCH`，删除 GET 写入 |
| `members/list/info/update/delete`、`POST /members/recharge` | 会员列表、详情、编辑、删除、后台充值 | `admin/members`、`POST /admin/members/{id}/wallet-adjustments`，重建并审计 |
| `banner/list/info/add/update/delete` | Banner CRUD | `admin/banners`，重建 |
| `quick/recharge/list/info/add/update/delete` | 快捷充币产品 CRUD | `admin/recharge-products`，重建 |
| `global-settings/list/add/update/delete` | KV 全局配置 | 仅保留非业务规则配置；业务规则迁移至 `admin/rules` |
| `transactions/list` | 交易流水查询 | `admin/wallet-ledger`、`admin/payment-transactions`，拆分重建 |
| `staff/list/add/update/delete` | 全局员工 CRUD | `admin/staff`，重建；总后台可为员工指定唯一绑定门店，但同一微信用户/员工账号不得绑定多个门店 |
| `dashboard/stats`、`all-record/stats` | 总后台统计、全记录统计 | `admin/reports/dashboard`、`admin/reports/records`，重建 |

下列控制器方法存在但未挂 v1 路由：支付设置读写、错误日志、日志统计/清理、会员创建、活动订单退款、详细 Dashboard 分项。v2 必须根据权限重新暴露为：`admin/payment-channel-settings`、`admin/error-events`、`admin/audit-log-maintenance`、`admin/members`、`admin/refunds`、`admin/reports/revenue|members|reservations|stores`；不得保留未鉴权维护接口。

## 7. v1 重复、缺陷和不可直接兼容项

| 项目 | 证据 | v2 决策 |
| --- | --- | --- |
| 同一业务多入口 | 活动下单、预约、订单分别同时存在通用与专用路径 | v2 每个业务只保留一个写入口和一个读模型 |
| GET 产生写操作 | 活动 purchase、点餐订单 update、隐藏座位清理 | 全部改为 POST/PATCH 或 worker |
| 路由顺序歧义 | `/store/products/categories` 位于 `/{product}` 之后；`/admin/store-admins/batch-status` 可能被参数路由吞掉 | v2 固定静态路径优先，OpenAPI 测试覆盖 |
| 控制器与路由不一致 | 多个 Controller public method 未暴露；个别路由指向不存在或重复能力 | 以本清单为迁移基线，逐项以业务确认决定重建 |
| 本地上传 | mini/store/admin 三套上传接口 | 统一迁移至 `QINIU_ASSET_SERVICE_SPEC.md` |
| 支付分散 | 通用、活动、充值、餐品各有回调 | 统一 payment order、回调和退款；门店线下聚合收款单单列 |
| 旧通用 `orders/order_items` 与实际 SQL | 路由/模型仍引用，`inwardclub.sql` 未导出对应业务表 | 历史导入前从生产库核验；v2 不以其为新写模型 |

## 8. 代码复核后的实际业务行为

本节来自对路由实际指向的 Controller、`WechatPayService`、`ConsumptionRecordService`、`User`、`FoodOrder`、`Recharge`、`ActivityOrder`、`SavePoints` 等实现的逐段复核。它优先于 v1 的 OpenAPI 注释、方法名和旧文档。

### 8.1 已实现且必须保留的业务行为

| 领域 | 代码实际行为 | 2.0 必须承接的能力 |
| --- | --- | --- |
| 微信登录/会员 | 用微信 `code` 换 openid；首次登录创建会员、邀请码；资料更新可修改昵称、头像、性别、手机号和一次性邀请人 | 微信身份、手机号绑定、邀请码、会员资料和 token 生命周期 |
| 手机号获取 | `/mini/user/get/phone` 仅调用微信 API 并返回手机号，**不落库** | v2 必须设计“获取并确认绑定”两阶段，不能误认为 v1 已完成绑定 |
| 点餐 | 创建 `food_orders`/明细、下单即扣库存；微信或金币支付；支付成功后按商品 `points` 发积分，`type=2` 商品发 30 天用户券，并向所有启用打印机同步打印 | 商品快照、库存、微信/金币支付、商品赠分、购券发券、打印任务 |
| 充值 | 充值产品或 ID `999` 自定义金额；微信成功后增加 `balance`、`all_balance`、`points`，并按 `all_balance` 更新 VIP | 充值档位、自定义充值是否保留、余额/成长值/积分三账本和升级触发 |
| 活动 | 单活动单订单，微信或金币支付；生成本地二维码 PNG 与 10 位核销码；核销按核销码一次性置 used | 活动购买、票/核销码、二维码资产、微信/金币支付、核销记录 |
| 预约 | 用户预约、取消；门店可 CRUD；旧状态含 pending/confirmed/seated/completed/cancelled | 桌座预约状态机、门店到店和自动释放 |
| 积分存入审核 | 用户提交 `save_points`，员工审核后按营业时段（17:00 至次日 10:00）、本时段取分基数，以 1:1/1:5 计算实际积分，并按条件发余额；同时写积分/交易记录 | 规则引擎必须能表达时段、基数、审批人、计算快照、积分与余额双发放 |
| 积分取出 | 创建取分记录后立即扣积分、写交易记录并同步打印 | 取分申请、审批/履约状态、账本扣减和打印任务 |
| 用户券 | `product.type=2` 的餐品购买后发券；券默认 30 天；兑换会把券直接置 used、打印兑换小票 | 券模板、用户券、过期、兑换订单、打印 |
| 排行榜 | 按当月审核通过的 `save_points.save_points` 求和，取前 20 名，不是营收榜 | 需要保留“存分月榜”；新的营收榜应是新增榜单而非替换 |
| 邀请 | 支持 openid 指定被邀请人，也支持资料更新时填写邀请码；关系可写入 `invited_by` | 单次绑定、邀请记录和奖励规则；两个 v1 写入口必须合并 |
| 门店收款 | 现金、刷卡、支付宝接口均要求 `user_id`，只是即时写积分记录并增加 `total_points`；没有真实支付、没有收款单、没有回调 | 用已设计的动态聚合收款码和会员手机号绑定替代，同时保留手工现金/刷卡收款记录 |
| 报表 | 现有总后台有营收、用户、预约、门店统计分项；门店有营收趋势、热门商品、用户活跃分项，部分未挂路由 | 总/门店报表体系、按日汇总任务和权限隔离 |

### 8.2 代码证实的高风险遗留，禁止原样迁移

| 现象 | 代码证据 | 2.0 处理 |
| --- | --- | --- |
| 支付回调验签不足 | `WechatPayService` 解密 resource 后直接执行业务回调；没有验证微信请求签名 | 使用官方验证器验签、校验商户号/应用 ID/金额/订单状态，回调幂等 |
| 食品/充值金额校验被注释 | 食品和充值通知中的金额比较代码被注释；活动通知才校验金额 | 所有支付通知强制以整数分校验订单金额 |
| 同一资产有多套字段 | `users` 的 `total_points/used_points`、`points`、`balance`、`all_balance` 被不同代码写入 | 数据迁移必须逐字段对账；v2 分为积分、可消费余额/金币、成长值账本 |
| v1 旧通用订单与真实餐品订单并存 | `OrderController` 使用 `orders` 且允许小程序 alipay/cash；实际 SQL 导出和新点餐流程使用 `food_orders` | 仅迁移真实历史表；小程序不保留 alipay/cash；旧订单先做数据存在性审计 |
| 无门店范围或所有权检查 | 例如餐品订单详情只按 ID 查询；部分门店后台查询未过滤 store_id | 所有读写在 repository 层注入成员所有权或 store scope |
| 同步外部副作用 | 下单/兑换/取分循环同步调用全部启用打印机 | 事务提交后写 `print_jobs`，worker 异步打印、重试、按门店路由设备 |
| 规则和全局活动存在空实现 | `SettingController` 的更新逻辑为注释；全局活动 assign-stores 未执行分配 | v2 规则必须版本化持久化；活动投放必须有 assignment 表和实际写入 |
| 本地文件写入 | 活动二维码、Logo、三套图片上传均写本地 storage | 统一七牛资产服务；二维码也要成为 asset 或可重新生成资源 |
| 手工收款不可靠 | 现金/刷卡/支付宝直接按请求 user_id 加积分，收款记录实际复用积分记录 | `offline_collection_orders`、支付交易、签名回调和权益 outbox 取代 |

### 8.3 已确认但未在 v1 路由暴露的功能

以下方法在 Controller 中有实现或明确代码，但未由 `routes/api.php` 暴露。2.0 应按业务价值重建，不应将它们误认为已上线 API：门店餐品订单的确认/备餐/待取/完成/取消状态流转；门店收款记录和统计；门店营业状态；门店营收趋势/热门商品/用户活跃；总后台营收/用户/预约/门店报表分项；活动订单退款；错误日志、日志统计和清理；全局支付设置。它们的 v2 权限和路由以 `CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md` 为准。

## 9. Claude 实施要求

1. 将本文件作为 v1 覆盖检查表，而非要求逐路径复制。
2. 每实现一个 v2 模块，在 OpenAPI 中列出对应 v1 路径、迁移状态和测试；未覆盖的 v1 业务路由不得标记模块完成。
3. 所有 `废弃` 项必须在迁移报告中明确记录替代措施，不能悄悄消失。
4. 所有“控制器存在但未挂路由”的项目先实现为权限明确的 v2 API 或 worker；业务不确认时保持 feature flag 关闭。
5. 在切流前，从 v1 访问日志校验本清单中的端点实际调用量；零调用且无业务确认的接口才可不提供兼容层。
