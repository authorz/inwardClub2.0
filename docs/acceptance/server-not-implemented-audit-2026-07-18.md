# Server NOT_IMPLEMENTED / Skeleton Audit (2026-07-18)

> **更新（2026-07-18，Wave 0 已落地）**：§1.D 门店预约路由（`GET /store/reservations`、
> `GET /store/reservations/{id}`、`POST /store/reservations/{id}/arrive`）已实现并挂载（新增
> store-scoped service/repo/handler + 作用域测试）；§1.G 死代码 `bootstrap/stub.go` 已删除；
> `COVERAGE_MATRIX.md` 过期"骨架"标注与 `wallet/points.go` 过期注释已同步。**剩余最大块仍为
> §1.A worker 副作用 与 §1.B 真实适配器。**

> **更新（2026-07-18，Wave 2 已落地）**：§1.A 异步 worker 10 个任务已**全部接线为真实 handler**
> （`logHandler` 仅兜底、已无任务命中）；`internal/platform/outbox` 的 **outbox→asynq 派发器已实现**
> 并在 `cmd/worker` 内运行（见 `server/docs/outbox-dispatch.md`）；`print:receipt` 与 `payment:post-process`
> **生产端均已在结算事务内接线**；§1.C 充值 growth/VIP 同步结算已落地。**剩余最大块 = §1.B 真实适配器**，
> 其次为 §1.C/F config-gated 规则/VIP 权益的业务确认（启用后再落地发放）、§1.E 诊断持久化、线下退款下发。

范围：`server/` Go 服务 `/api/v2/*`、`/internal/*` 路由，以及其支撑的
service/repository/worker/adapter 层。核对对象为**实际代码**（`internal/bootstrap/router.go`、
`internal/bootstrap/app.go`、各 module 的 repository），而非 `COVERAGE_MATRIX.md` 的里程碑标注
（矩阵多处已过期，见 §6）。

## 0. 结论（TL;DR）

**同步 HTTP 路由面已基本全部实现。** `router.go` 中每一条挂载的路由都指向真实
handler → 真实 service → 真实 SQL repository（事务 / `FOR UPDATE` / 追加账本 / 幂等声明齐全）。
在默认 `USE_FAKE_ADAPTERS=true` 下运行时，**没有任何已挂载路由返回 `NOT_IMPLEMENTED`（501）**。

因此"剩余的 skeleton 工作"不在路由层。**截至 Wave 2，异步 worker 10 个任务已全部接线**（`logHandler` 兜底分支已无任务命中；其中
`rule:post-process`、`benefit:vip-monthly` 为 config-gated、默认禁用的安全 no-op，待业务确认后发放），
**outbox→asynq 派发器已实现**，**充值 growth/VIP 同步结算已落地**，**门店预约到店路由已挂载**。
剩余缺口集中于：真实第三方适配器（§1.B）、config-gated 规则/VIP 权益的业务确认（§1.C/F）、
诊断错误事件持久化（§1.E）、线下退款下发（`docs/adapters.md` §4）。

代码中仅存的 4 个 `apperr.NotImplemented(...)` 调用点全部是**不可达的 nil-guard**
（对应 provider 在 `app.go` 已注入，见 §5），不构成运行时缺口。

---

## 1. 真正剩余的缺口（按域 + 前端影响）

### A. 异步 Worker —— 原为 10 个 log-only no-op，现已全部接线　【原最大一块】
（原始问题）`cmd/worker/main.go` 的 `logHandler` 只打印 payload 后 `return nil`、无任何副作用。**现状：10 个任务已全部接线为真实 handler，`logHandler` 仅作兜底、已无任务命中；下方 ✅ 更新逐批记录。**

> ✅ 已解决（第一批纯 DB worker，轨道 2）：`reservation:expire`、`activity-order:expire`、
> `ticket-coupon:expire`、`offline-collection:expire` 四个纯 DB 过期扫描已实现并接线——
> 每个 domain 模块新增 `ExpiryService`（仅依赖各自 repository）+ repository 幂等 sweep，
> `cmd/worker` 用真实 handler 替换这四个 log stub 并在既有 scheduler 上按 spec §11 节奏注册
> （前三者每分钟、`ticket/coupon` 每小时）。要点：
> - `reservation:expire`：`booked → expired`，`reserved_at` 过 `noShowGrace`（默认 2h）未到店的空放预约。
> - `activity-order:expire`：未支付活动订单过 `activityPayWindow`（默认 15m）→ 逐单加锁复核，释放票档
>   `sold_quantity`、`pending` 票 / 活动订单 / 支付单 / 业务单一并置 `expired`；`payment_orders` 离开 `pending`
>   后结算回调按既有 `status != pending` 守卫安全拒绝，绝不复活为 paid。
> - `ticket-coupon:expire`：`tickets` `active → expired`（场次否则活动结束时间已过）+ `coupon_entitlements`
>   `active → expired`（`expires_at < now`，NULL 永不过期）。
> - `offline-collection:expire`：`pending → expired`，与 `SettleOffline` 一样加锁 collection+payment order，
>   连带 payment/business order 关闭；`spec §9.3.5` 单终态、幂等。
> 幂等性由 `status=...` 守卫提供（重跑影响 0 行），不写多余 outbox（无外部副作用、无消费者）。
> `noShowGrace` / `activityPayWindow` 为占位默认，待业务确认（spec §13），可在各 `expiry.go` 调整。
> 单测：各模块 `expiry_test.go`（faithful fake + 固定 now，覆盖边界与幂等重跑），`go test -race ./...` 通过。
> 此批交付时，表中 `payment:post-process`、`benefit:vip-monthly`、`rule:post-process`、`asset:pending-cleanup` 仍为 stub、
> `print:receipt`/`report:rollup` 已由其它轨道实现；**这四个 stub 已在下方 Wave 2 更新中全部接线。**
>
> ✅ 更新（Wave 2）：`print:receipt` **生产端已接线**——三个结算完成点（`SettleWeChat` /
> `SettleOffline` / `SettleByCoin`）在结算事务内按「门店绑定 + 门店有在用打印机」规则写
> `print:receipt` outbox 事件（payload 为 `printer.Job`）。故「小票永不打印」已不成立：
> 餐饮 / 活动（门店绑定）及门店线下收款结算后会真实出小票；充值订单无门店、不打印。
> 详见 `server/docs/adapters.md` §3.1。
>
> ✅ 更新（Wave 2，asset/rule worker 批次）：表中最后三个 stub 已接线，`logHandler`
> 兜底分支现已无任务命中：
> - `asset:pending-cleanup`：**DB 侧完整实现**。`asset.CleanupService.SweepPending`
>   把仍 `pending` 且 `created_at` 早于 24h（`pendingTTL`）的资产置 `failed`；worker
>   按 `@every 1h` 调度，`status='pending'` 守卫保证幂等。删除对应七牛对象与环境前缀
>   孤儿对象仍待办（需把 `ObjectStore.Delete` 接入扫描 + bucket 列举），见
>   `server/docs/asset-service.md` §7。
> - `benefit:vip-monthly`（每日调度）与 `rule:post-process`（邀请奖励 `invite_reward`）：
>   新增 `internal/modules/rule` 包，读取「enabled + published + 生效窗口」的
>   `rule_definitions`（复用 `wallet.signInLadder` 的解析约定）。按 spec §13，VIP
>   日/月福利与邀请奖励的权益值/资格/补发/过期口径仍待业务确认，故规则默认禁用、
>   两个 evaluator 今天恒为安全 no-op（发放 0）；启用后再补 `benefit_grants` 落地，
>   幂等键沿用 `benefit:{ruleVersion}:{member}:{period}` / `rule:{ruleVersion}:{order}`。
>   低消奖励（`low_spend_reward`）由 `payment:post-process` 负责，rule 包不触碰以避免重复发放。
> - 单测：`asset/cleanup_test.go`、`internal/modules/rule/service_test.go`、
>   `cmd/worker/rule_handlers_test.go`；`go test ./...`、`go vet ./...` 与相关包 `-race` 通过。

| 任务 | 期望副作用 | 前端影响 |
| --- | --- | --- |
| `payment:post-process` | 支付成功后处理 | mini/store（下单支付闭环） |
| `offline-collection:expire` | 线下聚合收款单超时关闭 | store-console |
| `print:receipt` | 芯烨云小票打印 | store-console |
| `reservation:expire` | 预约超时状态机流转 | mini + store-console |
| `activity-order:expire` | 未支付活动订单过期释放库存 | mini |
| `ticket-coupon:expire` | 票 / 券过期 | mini |
| `benefit:vip-monthly` | VIP 月度权益发放 | mini（依赖规则/档位映射，见 C/F） |
| `rule:post-process` | 规则引擎评估 | 三端（间接，见 F） |
| `asset:pending-cleanup` | pending 资产 GC | 三端（间接） |
| `report:rollup` | `reporting_daily` 汇总 | admin `/reports/*` 与 mini `/rankings` 新鲜度 |

影响（**下列为上表原始 M1 缺口口径；Wave 2 现状见本节 ✅ 更新**）：原「过期订单/预约永不自动关闭、
库存不释放、小票永不打印、报表与排行陈旧」已随各 worker + 生产端接线而解决；仅 **VIP / 规则化权益**
因 config-gated 默认禁用而暂不发放（待业务确认后启用即可发放，非缺实现）。

### B. 真实第三方适配器 —— 生产路径全部是 fake
`app.go` L88-93、L146：除对象存储外，所有对外集成均注入 fake。

| 适配器 | 现状 | 前端影响 |
| --- | --- | --- |
| 微信登录（code2session + 手机号） | fake | mini 登录 / `POST /mini/me/phone-bindings` |
| 微信支付 JSAPI 网关 | fake | mini 下单支付、admin/store 结算 |
| 线下聚合收单（支付宝渠道） | fake | store-console `offline-collection-orders` |
| 打印机（芯烨云） | fake | store-console（配合 A 的 `print:receipt`） |
| 七牛对象存储 | **有真实实现**（`USE_FAKE_ADAPTERS=false` 即启用） | 三端资产上传（唯一已可切真实的适配器） |

### C. 充值权益结算 —— 部分实现
> ✅ 同步部分已解决（2026-07-18，Wave 2 起步）：`creditRechargeBenefit` 现在同一结算事务里除
> coins（`amount + bonus_amount`，行为不变）外，还发放 **growth_value 累积** 与 **VIP 等级升级**：
> - 新增 `recharge_products.growth_amount`（migration `00019`，默认 0，管理员经 `/admin/recharge-products` 可配），
>   避免把成长值写死在 Go（spec §1）；产品未配置即 0，历史行为逐字节保持。
> - 新增 `members.current_tier_id`；结算按 `membership_tiers.threshold` 对会员成长值余额取最高合格档位，
>   **仅升不降**（`payment.resolveTier` 纯函数 + `applyTierUpgrade`），锁会员行避免并发充值丢更新。
> - growth 用独立幂等键 `recharge_growth:<paymentID>`，与 coins 的 `recharge_order:<paymentID>` 互不影响，重复通知不重复入账。
>
> 仍待业务确认/接线（spec §13 VIP 行）：**降级 / 月度福利 / 补发 / 过期**；**阈值口径**（`threshold` 计 growth_value 还是 points，
> migration `00014` 注释写的是 points）；**非充值渠道的 growth**（线下聚合 `payment:post-process`、消费/低消）。
> mini 会员成长/等级读、admin/store 会员详情读（暴露 `currentTierId`/growth）仍为前端契约待办，属另一轨。

以下为修复前描述，保留备查。
`internal/modules/payment/settlement_repository.go:88`：`creditRechargeBenefit` 已把
`amount + bonus_amount` 记入币钱包并追加账本；但 **VIP 等级升级 与 growth_value 累积未接线**
（`recharge_products` schema 只有 coin amount/bonus，需等规则/档位映射，见 F）。
影响：mini 会员成长/等级、admin/store 会员详情。

### D. 已实现但未挂载的路由（store-console 预约管理缺失）
> ✅ 已解决（Wave 0，2026-07-18）：`GET /store/reservations`、`GET /store/reservations/{id}`、
> `POST /store/reservations/{id}/arrive` 现已在 `router.go`（`registerStore`）挂载。以下为修复前描述，保留备查。
`reservation.Handler.Arrive` 与 `service.ArriveReservation` 已完整实现
（`UPDATE reservations` + `INSERT arrival_records`），但 `POST /store/reservations/{id}/arrive`
**未在 `router.go` 挂载**；且门店端**完全没有预约管理路由**（list/detail/arrive）。
mini 端预约创建/取消/查询齐全。影响：store-console 无法查看或核销到店。**修复成本极小（仅挂路由）。**

### E. 诊断错误事件仅存内存
`internal/modules/diagnostics/service.go`：`Record`/`List` 基于内存切片、进程重启即丢失，
无持久化 schema。`GET /admin/error-events` 可用但数据易失。影响：admin-console。

### F. 规则引擎 —— CRUD 已实现，评估未实现
> ✅ 更新（Wave 2）：评估**读取侧已接线**。`payment:post-process` 评估 `low_spend_reward`
> 并可真实发放（config-gated，默认禁用）；`internal/modules/rule` 包为 `benefit:vip-monthly`
> 与 `rule:post-process`（`invite_reward`）提供评估 harness，读取 enabled 规则、幂等、
> 默认禁用即恒 no-op。仍待业务确认的是**具体权益值/资格/补发/过期口径**（spec §13），
> 确认后 rule 包补 `benefit_grants` 落地即可。详见 §A 的 Wave 2 更新。以下为修复前描述，保留备查。
`/admin/rule-definitions` 的增/改/禁用/发布已实现并挂载；但**规则评估 = worker `rule:post-process`
（即 A 中的 no-op）**；默认禁用，具体权益值待业务确认。影响：三端（间接）。与 C、`benefit:vip-monthly` 同源。

### G. 死代码
> ✅ 已解决（Wave 0，2026-07-18）：`internal/bootstrap/stub.go` 已删除，`stub()` 不再存在。
`internal/bootstrap/stub.go` 的 `stub()` 早期骨架期产物，现已无任何引用，可删除。

---

## 2. 按前端归纳

- **mini-program**：路由 100% 实现。受影响项 = 真实微信登录/支付/手机号适配器（B）、
  充值 VIP/growth 结算（C）、其订单/预约的过期扫描（A）、`/rankings` 依赖报表汇总（A）。
- **admin-console**：路由 100% 实现。受影响项 = 诊断持久化（E）、`/reports/*` 依赖 `report:rollup`（A）、
  规则引擎评估（A/F）。
- **store-console**：路由 100% 实现（D 门店预约管理已于 Wave 0 挂载，不再是例外）。受影响项 = 真实线下收单 +
  打印适配器（B）、`print:receipt` / `offline-collection:expire` worker（A）。

---

## 3. 推荐实现顺序（最大化并行）

以「无依赖、可立刻开工」优先，适配器与 worker 任务天然可拆给不同人并行。

**Wave 0 — 立即并行（小、无依赖）**
- D：挂载 `POST /store/reservations/{id}/arrive` + 门店预约 list/detail 只读路由（1 人，半天）。
- G：删除 `bootstrap/stub.go`。
- E：诊断错误事件落库（新增 schema + repository swap，与其它互不影响）。
- 文档同步：更新 `COVERAGE_MATRIX.md` 过期的"骨架"标注（见 §6）。

**Wave 1 — 主力并行（互不依赖，高价值）**
- 轨道 1《适配器》可拆 3-4 人各领一个：微信登录/手机号、微信支付、线下聚合收单、打印机。
- 轨道 2《纯 DB worker》可再拆 2-3 人：`reservation:expire`、`activity-order:expire`、
  `ticket-coupon:expire`、`offline-collection:expire`、`report:rollup`、`asset:pending-cleanup`
  —— 全是纯 DB 扫描/汇总，彼此独立，**不等适配器即可完成**。

**Wave 2 — 依赖 Wave 1 / 业务输入**
- ~~`print:receipt`（依赖轨道 1 打印机适配器）~~ ✅ 生产端已接线（结算完成点写 outbox 事件，见上）。
- `payment:post-process`（依赖微信支付适配器 + 结算路径）。
- C 充值 VIP/growth 结算 + `benefit:vip-monthly` + `rule:post-process`（F）——
  共同依赖「规则/档位映射」schema 与业务确认，建议由同一人/小组顺序推进。

关键路径 = 规则/档位映射决策（阻塞 C + F + VIP 月度权益）。建议 Wave 0 阶段即向业务方发起确认，
使其在 Wave 2 前就位。

---

## 4. 最大的下一块（Biggest next chunks）

1. **Worker 副作用（A）**—— 10 个 no-op 任务，是唯一「已挂钩但完全无实现」的整片区域；
   其中 6 个纯 DB 任务可立即并行，价值最高、解锁成本最低。
2. **真实适配器（B）**—— 4 个 fake→real 替换，决定 mini 支付闭环与 store 收单/打印能否上生产。
3. **规则/档位映射（C+F+VIP）**—— 需业务确认后一并落地，属关键路径。
4. **store-console 预约管理路由（D）**—— 成本最小、收益直接，应最先做。

---

## 5. 非缺口（已澄清，勿误列）

以下 `apperr.NotImplemented` 调用点在生产 `app.go` 装配下**不可达**，非运行时缺口：
- `admin/service.go:174,216`（wallet provider nil-guard）—— `app.go:120` 已注入 `walletSvc`。
- `admin/service.go:735`（store profile provider nil-guard）—— 已注入 `storeProfileAdapter`。
- `member/service.go:59`（phone resolver nil-guard）—— `app.go:116` 已注入 `phoneResolverAdapter`
  （经 fake 微信客户端可返回 fake 手机号，真实性取决于 B）。
- `wallet/points.go` 顶部"controlled NOT_IMPLEMENTED"注释**已过期**：`points_repository.go` 的
  `SavePoints`/`WithdrawPoints`/`RecordSignIn` 均已实现 SQL 写入。

## 6. 文档同步建议

`server/docs/openapi/COVERAGE_MATRIX.md` 第 22-39 行仍把下列域标为「骨架」，但代码已实现并挂载，
建议同步为「已实现」：签到/存取积分/提现/邀请（M1）、点餐下单/支付、充值订单、活动购票/核销、
券、预约/排队（mini 端）、线下聚合收款、后台商品/券/活动 CRUD、后台用户/员工/门店/运营配置。
真正应保留为「未完成」的只剩本文件 §1 列出的 A–G。
