# payment:post-process 支付后处理 as-built 使用说明

面向后端实现者。本文描述 `payment:post-process` 任务处理器的**已落地**实现：结算
写入的 `payment:post-process` outbox 事件，如何被 worker 消费并把已启用规则的权益
（金币/积分/成长值）**幂等地**发放给会员。此前该处理器只是 `logHandler` 骨架（只打
印 payload），本次补齐的是真正的规则解析与发放半环。

对应规格：spec §9.3.4（聚合收款支付后处理）、§11（任务表 `rule:{ruleVersion}:{order}`）、
§13（低消/VIP 等规则的业务待确认项）、§1（运营数值不得写死）。

## 1. 触发链路

```text
门店聚合收款回调 → SettleOffline（结算事务内）
  └─ 已支付且 member_id 存在 → writePostProcess()
        └─ outbox_events(topic=payment:post-process, idem=payment:{id}:post-process)
             └─ Dispatcher relay → asynq 任务 payment:post-process（TaskID=idem）
                  └─ worker: postProcessHandler → payment.PostProcessService.Process
```

- 事件只由**会员绑定**的线下聚合收款结算写入（散客不写，spec §9.3.4）。
- payload 即 `postProcessPayload`（`source/paymentOrderId/businessOrderId/memberId/
  storeId/amountCent/businessType`），绝不含明文手机号。
- outbox idem_key = `payment:{id}:post-process`，同时作为 asynq `TaskID`，队列层已去重。

## 2. 组件与落地位置

| 文件 | 职责 |
| --- | --- |
| `internal/modules/payment/postprocess.go` | 纯逻辑（无 DB）：`low_spend_reward` 规则的 config схема、`computeGrants` 权益计算、payload 解码与校验、`PostProcessResult`。可单测。 |
| `internal/modules/payment/postprocess_repository.go` | `PostProcessService` + `PostProcessRepository`（SQL 实现）：加载已启用规则、在一个事务内幂等写 `rule_executions` + `benefit_grants` + 钱包账本，成长值发放后复用 `applyTierUpgrade` 复算 VIP 等级。 |
| `internal/modules/payment/postprocess_test.go` | 纯计算单测 + 用内存 fake 覆盖幂等/发放契约，无需真实 MySQL。 |
| `cmd/worker/main.go` | 组合根：`postProcessHandler` 绑定到 `TaskPaymentPostProcess`。 |

发放复用既有充值结算的成熟机制：钱包入账沿用 `creditGrowthValue` 的“账户 upsert +
幂等账本”形状；成长值发放后走与充值同一条 `applyTierUpgrade` / `resolveTier` 升级路径。

## 3. 处理流程（一条事件）

1. `postProcessHandler` 收到任务，调用 `svc.Process(ctx, t.Payload())`。
2. 解码 payload；解码失败或缺 `memberId/paymentOrderId` → 返回 `ErrUndecodablePostProcess`，
   处理器映射为 `asynq.SkipRetry`（无法靠重试修复，直接丢弃）。
3. 加载**已启用、已发布、当前生效**的 `low_spend_reward` 规则（`rule_definitions`）：
   门店级规则优先于全局，其次版本号最大。沿用 `wallet.signInLadder` 的解析式并扩展了
   表本身就建模的 `scope_type/store_id`。
4. **无启用规则**（今日线上现状，见 §5）→ 记 `RuleMatched=false`，确认事件（不发放）。
5. 有规则 → 用 config 计算权益 `computeGrants(cfg, amountCent)`，随后在**一个事务**内：
   1. 写 `rule_executions`（idem_key=`rule:{version}:{paymentOrderId}`）——这就是 spec
      §9.3.4 的 `member_id + payment_order_id + rule_version` 幂等标记；命中重复键说明该
      事件已处理，直接 `AlreadyDone` 返回，不发放。
   2. 逐条权益写 `benefit_grants`（idem_key=`rule:{version}:{paymentOrderId}:{asset}`）
      并入账钱包（账本 idem_key 相同，二次防御）；`growth_value` 权益额外复算 VIP 等级。
6. DB 错误原样返回 → asynq 按 dispatcher 预算重试；因步骤 5 全程幂等，重试/重复投递不会
   重复发放。

## 4. 规则 config схема（`rule_definitions.config_json`，rule_key=`low_spend_reward`）

```json
{
  "grants": [
    { "asset": "points",       "mode": "fixed",    "value": 100 },
    { "asset": "coins",        "mode": "permille", "value": 10  },
    { "asset": "growth_value", "mode": "permille", "value": 5   }
  ]
}
```

- `asset`：可发放资产仅限 `points | coins | growth_value`；`cash_balance` 只由充值/退款
  移动，规则不得发放，故被排除。
- `mode=fixed`：发放 `value` 个单位（与金额无关）。
- `mode=permille`：发放 `floor(amountCent * value / 1000)` 个单位（每 1000 分结算额发
  `value` 个）。
- 任一非法资产/非法 mode/计算结果 ≤ 0 的条目被**跳过**，既不会写零账本，也不会因单条坏
  配置卡住队列。
- 金额（value）全部来自 admin config，**代码不写死任何运营数值**（spec §1），与充值
  `recharge_products` 的做法一致。

## 5. 仍待业务确认的边界（precise seam，spec §13「低消奖励」行）

处理器已把**发放管道**打通，但**低消奖励的资格判定与金额口径仍属业务待确认**，因此
`low_spend_reward` 规则**默认禁用、无 seed**，开发者不得自行启用（spec §13）。今日任何
部署都无启用规则，处理器对每条事件走 §3 步骤 4 的安全 no-op（确认但不发放）。

具体未实现、需业务正式输入后再补的判定（对应 spec §13）：

| 待确认项 | 现状 |
| --- | --- |
| 合格订单类型 / 支付方式 | **未实现**：当前 config 无资格谓词，规则一旦启用会对**任意**会员绑定收款无条件发放。 |
| 金额口径（`amountCent` 是否即计奖基数） | 暂以“结算 `amount_cent`”为基数，属**临时建模选择**，已在代码/本文标注。 |
| 到店判定 / 20:30 条件 | **未实现**。 |
| 退款后的权益冲正 | **未实现**：`benefit_grants` 已留 `status`/`source`/idem，冲正任务待 §9.3.6 退款链路补。 |
| 邀请奖励（`invite_reward`） | 不在本处理器内；由 rule 包的 `rule:post-process` 评估器承接（见 §6）。 |

补齐方式：在 §4 的 config схема 增加资格谓词字段，并在 `computeGrants` 前置一个
`qualifies(payload, cfg)` 判定；判定确认前，规则保持禁用即可保证零副作用。

## 6. 与 `internal/modules/rule` 包的关系

`internal/modules/rule` 另建了规则**解析**层（`ActiveRule`）与两个 config-gated 评估器
（`benefit:vip-monthly`、`rule:post-process` 的邀请奖励），它们按 §13 **只解析不发放**。
本处理器与之**互补**：

- 本处理器负责 `payment:post-process`——结算真正产生的 topic——并**落地发放**（低消奖励）。
- rule 包负责 `benefit:vip-monthly` 与 `rule:post-process`（其 topic 目前无生产者）。
- 两者共用同一 `rule_key` 词表（`low_spend_reward` == `rule.KeyLowSpendReward`）；本处理器
  为保持自足，未 import rule 包，而是复用本地常量。若后续合并两条实现，词表已一致。

## 7. 测试

- 纯计算：`fixed`/`permille` 计算、permille 向下取整与 ≤0 丢弃、非法资产/mode 跳过、空
  config、malformed config、payload 解码/校验。
- 契约（内存 fake，逐分支镜像 SQL 语义）：无启用规则不发放；同一事件重复投递只发放一次
  （`AlreadyDone`）；成长值发放触发一次 VIP 复算；不同订单各自独立发放。
- 命令：`go test ./internal/modules/payment/ ./cmd/worker/`（含 `-race`）。
