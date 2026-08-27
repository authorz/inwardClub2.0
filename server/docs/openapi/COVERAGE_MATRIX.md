# V1 → V2 覆盖矩阵

依据 `docs/V1_API_INVENTORY_AND_V2_MAPPING.md`（v1 实际注册路由 233 条：总后台 101、门店后台 63、小程序 59，其余为回调/测试/系统/Swagger）与 `docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md` §8.8。

状态含义：`重建`=新 REST 资源重写；`合并`=多条 v1 路径并入一个 v2 资源/读模型；`废弃`=不进入 v2 对外 API（给出替代）；`待确认`=保留禁用态/feature flag，不猜测上线。

里程碑：`M0`=本次交付已实现；`M1`+ 为后续里程碑（路由已挂 `NOT_IMPLEMENTED` 骨架）。

> 同步更新（2026-07-18）：下表 M1–M5 多个能力域的**对外 HTTP 路由现已实现并挂载**（见各行"已实现"备注），
> `M1`+「路由已挂 NOT_IMPLEMENTED 骨架」仅适用于历史。**剩余骨架工作已不在路由层**，集中于异步 worker 副作用、
> 真实第三方适配器等，详见 `docs/acceptance/server-not-implemented-audit-2026-07-18.md`。
> **更新（2026-07-18）**：审计 §1.C「充值 VIP/growth 结算」的同步部分已落地——微信充值结算现发放
> coins（不变）+ growth 累积 + VIP 升级（仅升不降），见"充值订单"行；剩余为降级/月度福利/阈值口径确认等业务待定项。
> **更新（2026-07-18，Wave 2）**：审计 §1.A 异步 worker 已全部接线；`internal/platform/outbox` 的 outbox→asynq **派发器已实现**并在 `cmd/worker` 内运行（见 `docs/outbox-dispatch.md`）。当前有两个 outbox 生产者，均在结算事务内写入：`payment:post-process`（会员绑定订单结算）与 `print:receipt`（门店绑定订单结算），两者**生产端均已接线**（见"打印""门店线下聚合收款""审计/幂等/outbox"各行）；剩余为真实第三方适配器与 config-gated 规则/VIP 权益的业务确认。

| 能力域 | v1 来源（表/控制器） | v2 资源 | 状态 | 里程碑 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 健康检查 | - | `/internal/health`,`/ready` | 重建 | M0 | 已实现 |
| 小程序登录 | `users`,微信登录 | `/mini/auth/wechat/login`,`/refresh`,`/logout` | 重建 | M0 | 已实现，fake 微信 code2session |
| 会员资料 | `users` | `/mini/me` | 重建 | M0/M1 | 读、`PATCH /me`（资料更新）、`POST /me/phone-bindings` 手机号绑定均已实现；手机号绑定经微信手机号换取（当前 fake，真实性取决于适配器，见 acceptance 审计 §1.B） |
| 总后台登录 | `admins` | `/admin/auth/*` | 重建 | M0 | 已实现，`aud=admin`，super_admin |
| 门店后台登录 | `admins`(门店)、`staff` | `/store/auth/*` | 重建 | M0 | 已实现，`aud=store`，store scope 来自 token |
| 资产/图片上传 | 各本地上传接口 | `/{mini,admin,store}/assets/upload-credentials`,`/internal/qiniu/upload-callback` | 合并 | M0 | 已实现；所有本地上传合并到七牛资产服务 |
| 门店公开信息 | `stores`,`settings` | `/mini/stores`,`/{id}` | 重建/合并 | M0 | 已实现（列表/详情） |
| 分类/商品公开读取 | `categories`,`products` | `/mini/stores/{id}/catalog/categories`,`/items` | 重建 | M0 | 已实现（读） |
| 活动公开读取 | `activities` | `/mini/activities`,`/{id}`,`/mini/stores/{id}/activities` | 重建 | M0 | 已实现（读）；详情 `/mini/activities/{id}` 现返回在售票档 `ticketTypes`（`stock`=-1 表示不限量），供小程序活动详情购票弹层使用 |
| 钱包/账本 | `user_points`,`transaction_records`,`balance_consumption_records`,`points_consumption_records` | `/mini/wallet`,`/wallet/ledger` | 合并 | M0 | 读已实现；账本只追加模型已建表 |
| 会员等级/快捷充值/排行 | `vip_level`,`coin_product`,`recharge` | `/mini/membership-tiers`,`/recharge-products`,`/rankings` | 重建 | M1 | `/mini/*` 读（membership-tiers/recharge-products/rankings）已实现并文档化（`v2.yaml`）；`/rankings` 按 approved `point_savings` 汇总、支持 `week/month/all` 窗口（默认 all）；总后台 `/admin/membership-tiers`,`/admin/recharge-products` 增/改/禁用已实现（`v2.yaml`）；快捷充值显式配置 `amountCent`、`coinAmount`、`pointsAmount`，结算同时发放金币与积分，成长值仍作为内部结算配置驱动 VIP 升级（见“充值订单”行） |
| 签到/存取积分/提现/邀请 | `sign_in`,`save_points`,`points_withdrawal`,`invitations` | `/mini/sign-ins`,`/point-savings`,`/point-withdrawals`,`/invitations/*` | 重建 | M1 | 已实现并文档化（`v2.yaml`）：签到即时入账，梯度默认 day1..7=100..700、day7+ 封顶，由总后台 `/admin/rule-definitions`（ruleKey=`sign_in`，已种子化启用）可改值/禁用，无启用规则时回落服务端默认；存积分落 `pending` 请求待门店审核；取积分立即扣减积分、记录已通过提取单并向所选门店打印小票，无需审核；邀请绑定/列表已实现。其余积分规则值待业务确认（§13） |
| 点餐下单/支付 | `food_orders`,`food_order_items`,点餐回调 | `/mini/food-orders`,`/payment-orders/*` | 重建/合并 | M3 | 已实现：下单/支付单/微信 JSAPI/币支付 + 账本落库；微信支付网关当前为 fake，支付成功后处理依赖 worker（见 acceptance 审计） |
| 支付流水/支付单查询（总后台/门店） | 各后台支付相关控制器 | `/admin/payment-transactions`,`/admin/payment-orders`,`/store/payment-transactions`,`/store/payment-orders` | 重建 | M3 | 已实现（列表+详情，`v2.yaml`）；门店端按 token 门店范围过滤 |
| 退款（总后台/门店） | 各后台退款相关控制器 | `/admin/refunds`,`/admin/refund-orders`,`/store/refunds`,`/store/refund-orders` | 重建 | M3 | 已实现（读 + `POST` 发起退款，`v2.yaml`）；`refund-orders` 为 `refunds` 只读别名 |
| 会员详情/钱包手动调整（总后台/门店） | `users`,`user_points` 等 | `/admin/members/{memberID}`,`/admin/members/{memberID}/wallet-adjustments`,`/store/members/{memberID}`,`/store/members/{memberID}/wallet-adjustments` | 重建 | M2/M3 | 已实现（详情读 + 调整写，`v2.yaml`）；门店端 404 若会员不属于本店 |
| 钱包账本（总后台） | `transaction_records` 等 | `/admin/wallet-ledger` | 合并 | M3 | 已实现（读，`v2.yaml`），对齐已交付的 `/store/wallet-ledger`（见门店后台交付记录） |
| 员工账号写操作（总后台） | `staff` | `/admin/staff-accounts`,`/{staffID}`,`/{staffID}/disable` | 重建 | M2 | 已实现（增/改/禁用，`v2.yaml`） |
| 收银员/员工账号写操作（门店后台） | `staff` | `/store/cashiers`,`/{cashierID}`,`/{cashierID}/disable`,`/{cashierID}/password-reset`,`/store/staff-accounts`,`/{staffID}`,`/{staffID}/disable` | 重建 | M2 | 已实现（增/改/禁用/重置密码，`v2.yaml`） |
| 充值订单 | `recharge` | `/mini/recharge-orders` | 重建 | M3 | 已实现：下单 + 结算入账；微信快捷充值结算按档位在同一事务发放 `coin_amount` 与 `points_amount`，自定义金额按 1 元 = 1 金币入账；同时保留内部 growth 累积与 VIP 升级（按 `membership_tiers.threshold` 对成长值取最高档，仅升不降，写 `members.current_tier_id`）；各权益独立幂等键，重复通知不重复入账。待确认/待接线：降级/月度福利/补发、阈值口径（growth_value vs points）、线下聚合与消费渠道的 growth（见 acceptance 审计 §1.C 与 spec §13） |
| 活动购票/核销 | `activity_orders`,票据 | `/mini/activity-orders`,`/store/tickets/verify` | 重建 | M3/M4 | 已实现：一单多票下单 + 门店 `/tickets/verify` 核销；一单多票模型已建表 |
| 券 | `user_coupon`,`coupon_order_items` | `/*/coupon-templates`,`/coupon-entitlements`,`/mini/coupon-redemptions` | 重建/合并 | M4 | 已实现：模板 CRUD + 发放/作废/核销 + mini 兑换；券复用商品库 |
| 预约/排队/到店 | `reservations`,`tables`,`seats` | `/mini/reservations`,`/waitlist-entries`,`/store/.../arrive` | 重建 | M4 | 已实现：mini 预约/排队/取消/查询；门店 `/store/reservations` 列表/详情 + `/arrive` 到店核销；三状态机已建表 |
| 门店线下聚合收款 | 无（新增） | `/store/offline-collection-orders/*`,`/internal/payments/offline-acquirer/notify` | 重建 | M3 | 已实现并文档化（`v2.yaml`）：收款单开单（含 `expiresInSeconds` 必填、可选 `memberPhone` 一次性精确匹配注册会员并固化 `member_id`+掩码手机号快照+绑定操作人，未匹配返回受控 `MEMBER_NOT_FOUND`，原始手机号绝不落库/日志/回显）/取消/查询；收单回调结算后对绑定会员的收款单在同一事务写 `payment:post-process` outbox（散客不写），会员权益发放由 worker `payment:post-process` 评估 `low_spend_reward` 规则落地（发放管道已实现、经 outbox 派发器消费；规则 config-gated、默认禁用，无启用规则时安全 no-op，见 acceptance 审计 §1.A/F 与 `docs/payment-post-process.md`）；线下聚合渠道默认 fake，`USE_FAKE_ADAPTERS=false` 时切换真实 HMAC 签名 HTTP 适配器（`payment.HTTPAcquirer`：开码/回调验签/退款请求响应映射，见 `docs/adapters.md`），退款下发仍待 worker；支付宝仅存在于此渠道 |
| 微信支付回调（通用/活动/充值/点餐 4 套） | 4 个回调控制器 | `/internal/payments/wechat/notify` | 合并 | M3 | 已实现并文档化（`v2.yaml`）：单一回调 + 结算，按支付单区分业务；会员充值单结算在同一事务发放 coins/growth 并升级 VIP 档位（见"充值订单"行）；门店绑定订单结算会在同一事务写 `print:receipt` outbox（出小票）；邀请 `invite_reward` 已在微信结算事务内落地，餐品、活动、金币充值及绑定会员的门店聚合微信收款按后台比例累计整数金币，余数结转，金币支付、支付宝与现金不计，成功退款按原比例冲正；低消 `low_spend_reward` 仍由 worker `payment:post-process` 承接线下聚合收款权益。 |
| 打印 | `printer_device` + 芯烨云 | `printer_devices` + worker `print` 任务 | 重建 | M3 | 门店/总后台 `printer-devices` 设备 CRUD 已实现；Printer 默认 fake，`USE_FAKE_ADAPTERS=false` 时切换真实芯烨云适配器（`printer.XpyunPrinter`，SHA1 签名，见 `docs/adapters.md`）；worker `print:receipt` 由所选 Printer 执行打印；**生产端已接线**——三个结算完成点（`SettleWeChat`/`SettleOffline`/`SettleByCoin`）在结算事务内按「门店绑定 + 门店有在用打印机」规则写 `print:receipt` outbox 事件（payload 为 `printer.Job`），经 outbox 派发器投递给 worker（充值订单无门店、不打印），见 `docs/adapters.md` §3.1 |
| 后台商品/券/活动 CRUD | `products`,`categories`,`activities` 等 | `/admin/*`,`/store/*` | 重建 | M2 | 已实现：admin/store 商品/分类/规格/券/活动/会话/票种 CRUD；商品/分类/规格/活动/会话/票种 已补 openapi（`v2.yaml`，含 scope 归属与写操作幂等语义），券 CRUD 已补 openapi（`v2.yaml`，含 mini 券/兑换读 + admin/store 模板 CRUD + 发放/作废/核销）；scope_type/store_id 模型已建表 |
| 后台订单/支付/退款/报表 | 各后台控制器 | `/admin/*`,`/store/*` reports/orders/refunds | 重建/合并 | M5 | 已实现并文档化（`v2.yaml`）：订单/支付/退款读 + `/reports/*` 报表读模型——总后台 9 项（overview/revenue/catalog-items/activities/coupons/records/members/reservations/stores，跨店聚合），门店 5 项（overview/revenue/catalog-items/activities/coupons，store scope 来自 token，忽略请求 storeId）；`revenue` 与 `reservations` 已改由 `report:rollup` worker 预聚合的 `reporting_daily` 读模型提供（每日 + worker 启动各全量汇总一次；跨店求和含无门店散客桶、门店维度按 store_id 过滤，语义与原实时查询一致），其余（overview/catalog-items/activities/coupons/records/members/stores）仍为按需实时聚合 |
| 后台用户/员工/门店/运营配置 | `admins`,`staff`,`stores`,`settings` | `/admin/*`,`/store/*` | 重建 | M2 | 已实现：admin/store 账号（管理员/员工/收银员）、门店资料/设置、打印机等运营配置；staff 单门店唯一约束已建表 |
| 规则引擎 | 分散写死逻辑 | `rule_definitions`,`rule_executions`,`benefit_grants` | 重建 | M2/M6 | CRUD 已实现并文档化（`/admin/rule-definitions/*`，`v2.yaml`）；签到、门店预约低消和邀请奖励均由后台配置。`invite_reward` 已按正式口径启用：首次微信低消奖励与微信支付返金币在结算事务内幂等发放，退款冲正；`benefit:vip-monthly` 仍保持待业务确认的禁用状态。 |
| 审计/幂等/outbox/报表汇总 | 无 | `audit_logs`,`idempotency_keys`,`outbox_events`,`reporting_daily` | 重建 | M0 | 平台能力已实现（中间件+表）；**outbox→asynq 派发器已实现**（`internal/platform/outbox` 的 `Dispatcher`/`SQLStore`/`AsynqEnqueuer`，在 `cmd/worker` 内轮询、`FOR UPDATE SKIP LOCKED` 领取→enqueue→同事务写回终态，至少一次投递 + `idem_key` 去重，见 `docs/outbox-dispatch.md`）；当前有两个 outbox 生产者，均在结算事务内写入：`payment:post-process`（会员绑定订单结算）与 `print:receipt`（门店绑定订单结算）；`report:rollup` 与各过期扫描由 scheduler 定时 enqueue、不经 outbox |
| 数据迁移/对账 | 全部 v1 表 | `legacy_id_maps`,`migration_runs`,`reconciliation_results`,`cmd/reconcile` | 重建 | M0 | 对账框架（表发现+行数+映射）已实现 |
| 测试/Swagger/隐藏 GET 写/座位清理 GET | v1 隐藏路由 | — | 废弃 | — | 不进入 v2 对外 API；改由 worker/CLI/单测 |

## 门禁自检（§8.8）

1. v1 测试/Swagger/隐藏 GET 写接口不进入 v2：✅ 未暴露。
2. 本地上传全部合并到七牛资产服务：✅ 仅 `assetId` 写入。
3. 微信 4 套回调合并为单一 `/internal/payments/wechat/notify`：✅ 路由已合并（结算 M3）。
4. v1 小程序员工管理迁移到 `/store/*` 并校验 staff 身份与门店归属：✅ `/store/staff-accounts`、`/store/cashiers` 增/改/禁用/重置密码已实现并按 token 门店归属校验；`staff_accounts` 单门店唯一约束已建表。
5. v1 未挂路由能力按 §8.3 进入正式接口或 worker：⏳ 随对应里程碑。
6. 切流前用 v1 访问日志复核端点调用量：⏳ 迁移阶段执行（需 v1 访问日志）。
