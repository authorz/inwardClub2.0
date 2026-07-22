# 诊断 / 错误事件流（error-events）as-built 使用说明

面向后端实现者。本文描述 `internal/modules/diagnostics` 已落地的服务端错误事件流：
它把请求链路中产生的 5xx 响应与 handler 挂载的错误持久化到 `error_events` 表，供总
后台 `GET /api/v2/admin/error-events` 读取。此前该流是进程内环形缓冲（重启即丢），
现已改为持久化存储。

## 1. 组件与落地位置

| 文件 | 职责 |
| --- | --- |
| `service.go` | `Service`：`Record`（截断消息、落库、按保留上限剪枝，best-effort）与 `List`（分页读）。`ErrorEvent` 领域类型。 |
| `repository.go` | `Repository` 端口 + `sqlRepository`：`Insert` / `List` / `Prune`，唯一触达 `error_events` 表的地方。 |
| `middleware.go` | `Capture()`：请求结束后判定 5xx / `c.Errors`，用独立的带超时 background context 调 `Record`。 |
| `handler.go` | `Handler`：HTTP 入口 `ListErrorEvents`，读失败时走 `httpx.Fail`。 |
| `db/migrations/00018_diagnostics.sql` | `error_events` 表建表 / 回滚。 |

装配在组合根 `internal/bootstrap/app.go`：
`diagnostics.NewService(diagnostics.NewRepository(database), log)`；
`Capture()` 注册于全局中间件链（`internal/bootstrap/router.go`）。

## 2. 表结构（`error_events`）

| 列 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `BIGINT UNSIGNED AUTO_INCREMENT` | 主键，单调递增，兼作“新→旧”排序键。 |
| `request_id` | `VARCHAR(64)` | 请求 ID（来自 `httpx.RequestIDFromContext`），可为空串。 |
| `method` | `VARCHAR(16)` | HTTP 方法。 |
| `path` | `VARCHAR(255)` | 路由模板 `c.FullPath()`；未命中路由时为空串。 |
| `status` | `INT` | 响应状态码（≥500，或带 `c.Errors` 的其它码）。 |
| `message` | `VARCHAR(1024)` | `c.Errors.Last()` 文案，超长按 rune 截断到 1024 字。 |
| `created_at` | `DATETIME` | UTC 记录时间。 |

索引：主键 `id`；`idx_error_events_created (created_at)`（预留按时间查询/清理）。

## 3. 捕获语义（写路径）

- `Capture()` 在 `c.Next()` 之后运行：当 `status >= 500` **或** `len(c.Errors) > 0`
  时记录；`httpx.Recovery` 把 panic 转成的 500 也会被捕获到。
- 记录发生在响应已写出之后，因此使用**独立的 background context（5s 超时）**，避免
  客户端断连（取消请求 context）导致诊断丢失，也避免写库卡住 handler。
- **Best-effort 持久化**：`Record` 落库或剪枝失败只记日志、吞掉错误，绝不反向影响
  已写出的响应或掩盖原始错误。因此错误路径上即便数据库不可用也不会二次崩溃。

## 4. 保留策略（retention）

- 保留上限 `retentionMaxEvents = 500`（`service.go` 常量，沿用旧环形缓冲的界）。
- **每次写入后剪枝**：`Prune(keep)` 只保留最新的 `keep` 条——定位“新→旧”排序中第
  `keep` 位的 `id`，删除所有 `id <= 该阈值` 的行；当总行数 ≤ `keep` 时子查询返回
  NULL，不删除任何行。
- 表大小因此被硬性限制在 ~500 行，重启后仍在；无独立定时清理任务。
- 消息长度上限 `maxMessageLen = 1024`，写入前按 rune 边界截断，防止溢出列并避免拆断
  多字节字符产生非法 UTF-8。

## 5. 查询语义（读路径）

- `GET /api/v2/admin/error-events`（audience=admin，`super_admin`）。
- 分页：复用 `httpx.ParsePage`（`page` / `pageSize`，`pageSize` 上限 100）。
- 排序：`ORDER BY id DESC`——**最新在前**。
- 响应：标准 List 信封 `{ data: ErrorEvent[], meta: { page, pageSize, total } }`；
  `total` 为 `error_events` 全表行数（受保留上限约束，≤500）。
- 读失败（如数据库不可用）返回标准错误信封（`INTERNAL`，500）。

## 6. 验收

```bash
go test ./internal/modules/diagnostics/...
go run ./cmd/migrate up      # 应用 00018_diagnostics
```
