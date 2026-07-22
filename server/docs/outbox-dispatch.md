# 事务性 outbox → 异步任务分发 as-built 使用说明

面向后端实现者。本文描述 `internal/platform/outbox` 已落地的**分发管道**：业务
写入的 `outbox_events` 行如何被真正投递（enqueue）到 asynq 任务队列，交由
`cmd/worker` 的任务处理器执行。`Write`（在业务事务内追加事件）此前已存在；本次
补齐的是缺失的 relay 半环，让持久化的事件不再只是躺在表里。

## 1. 组件与落地位置

| 文件 | 职责 |
| --- | --- |
| `outbox.go` | `Write`：在调用方事务内追加一条 `pending` 事件（既有）。`Event` 领域类型。 |
| `dispatch.go` | `Dispatcher`：轮询 relay 主体。`Store` / `Enqueuer` 端口、`Result` 状态机、退避与重试策略。不依赖任何具体传输或数据库。 |
| `sqlstore.go` | `SQLStore`：`Store` 的 MySQL 实现。用 `FOR UPDATE SKIP LOCKED` 领取到期事件，并在同一事务内写回终态。 |
| `enqueuer.go` | `AsynqEnqueuer`：`Enqueuer` 的 asynq 实现。topic → 任务类型，payload 原样透传。 |
| `dispatch_test.go` | 用内存 fake（`memStore` / `recordingEnqueuer`）覆盖 `Dispatcher` 的判定逻辑，无需真实 MySQL/Redis。 |

组合根是 `cmd/worker/main.go`：worker 进程既是 asynq **消费端**（`asynq.Server` +
`ServeMux`），又托管 outbox **生产端**（`Dispatcher.Run` goroutine）。同一进程内轮询
DB → enqueue → 被同进程的任务处理器消费。

## 2. 分发流程（一个批次）

1. `Dispatcher` 每 `interval`（默认 2s）触发一次，并在启动时立即 drain 一轮。
2. `SQLStore.Dispatch` 在一个事务内领取到期待发事件：
   ```sql
   SELECT id, topic, payload, idem_key, attempts, available_at
     FROM outbox_events
    WHERE status = 'pending' AND available_at <= ?
    ORDER BY id LIMIT ?
    FOR UPDATE SKIP LOCKED
   ```
   `SKIP LOCKED` 让多个 worker 实例领取互不相交的批次，不会重复分发。
3. 对每条事件调用 `AsynqEnqueuer.Enqueue`：以 `topic` 为 asynq 任务类型、`payload`
   为任务体投递；`idem_key` 非空时作为 asynq `TaskID`。
4. 按 enqueue 结果在**同一事务内**写回终态（`attempts` 始终 +1）：
   - 成功 → `status='dispatched'`, `dispatched_at=now`, `last_error=NULL`。
   - 失败且未超限 → 保持 `pending`，`available_at=now+backoff`（指数退避，封顶 5m），`last_error` 记录原因。
   - 失败且达到 `maxAttempts`（默认 10）→ `status='failed'`。
5. 领取与写回同事务提交：进程若在提交前崩溃，被领取的行随事务回滚仍为 `pending`，
   下一轮重新投递。

topic 必须与 `cmd/worker` 注册的任务类型一致（如 `payment:post-process` ==
`TaskPaymentPostProcess`）。当前实际写入的 topic 有两个，均在结算事务内写入：
- `payment:post-process`（`TaskPaymentPostProcess`）—— 会员绑定订单结算后的权益评估。
- `print:receipt`（`TaskPrint`）—— 门店绑定订单结算后的小票打印，payload 为 `printer.Job`，
  `idem_key` 取 `payment:{paymentOrderId}:print-receipt`（见 `docs/adapters.md` §3.1）。

## 3. 投递语义（caveats）

- **至少一次（at-least-once）。** enqueue 成功但事务提交前崩溃时，行仍为 `pending`
  并会被重投；因此下游任务处理器必须幂等。
- **`idem_key` 去重窗口。** enqueue 时把 `idem_key` 作为 asynq `TaskID`：重投期间
  只要上一个同 ID 任务还在队列（pending/active/retry/scheduled/retention 未过期），
  重复 enqueue 会返回 `ErrTaskIDConflict`，被 `AsynqEnqueuer` 视为成功、正常落
  `dispatched`——这把上述崩溃窗口收敛为近似一次。`idem_key` 为空的 topic 无此去重，
  纯至少一次。支付结算的 `idem_key` 为 `payment:{paymentOrderId}:post-process`，唯一。
- **relay 领取期间持有行锁跨 Redis 调用。** 一个批次的领取→enqueue→写回都在一个事务
  内，enqueue 的网络往返期间持有被领取行的 `FOR UPDATE` 锁。因此批次刻意保持较小
  （默认 100）：Redis 变慢只会拖慢 relay，不影响 API（API 只 `INSERT` outbox 行，从不
  分发）。`SKIP LOCKED` 保证其它 relay/写入者不会被这些锁阻塞。
- **批次内 DB 写回失败会回滚整批。** 某条事件写回 UPDATE 出错会回滚整个事务，含本批
  已成功 enqueue 的事件——它们保持 `pending` 下轮重投（由上面的 `idem_key` 去重兜底）。
- **未注册 topic。** 若分发了 worker 未注册处理器的 topic，asynq 端 `handler not
  found`，任务重试后归档；outbox 行仍标记 `dispatched`（已成功入队）。这在对应里程碑
  处理器落地前是预期行为。
- **两类重试相互独立。** outbox 的 `attempts`/退避只覆盖“把任务放进队列”；任务**运行
  时**的重试是 asynq 自己的预算（`workerMaxRetry`，默认 25）。
- **`failed` 行是终态，不自动复活。** 达到 `maxAttempts` 后需人工介入（排障后手动将
  `status` 置回 `pending`）。

## 4. 配置与运行

worker 现在同时需要 `REDIS_ADDR` 与 `MYSQL_DSN`（后者用于 outbox 轮询）；缺任一即
启动失败。其余按既有默认。

```bash
export MYSQL_DSN='...'; export REDIS_ADDR='127.0.0.1:6379'
go run ./cmd/worker   # 启动后日志: "outbox dispatcher started"
```

`Dispatcher` 的调优项（batch/interval/maxAttempts/退避）目前为 `NewDispatcher` 内的
生产默认值；测试通过白盒直接设置这些未导出字段。
