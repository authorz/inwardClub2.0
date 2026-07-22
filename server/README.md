# 服务端 server

InwardClub 2.0 Go 模块化单体服务端。

## 边界

- 提供 `/api/v2/mini/*`、`/api/v2/admin/*`、`/api/v2/store/*` 和 `/internal/*`。
- 强制 RBAC、store scope、幂等、审计、钱包账本和支付回调验签。
- 总后台和门店后台复用领域 service，但认证中间件、权限码、token audience、数据 scope 分开。
- 小程序 API/DTO 不出现支付宝；支付宝仅存在于门店线下聚合收款渠道。

## 技术栈

Go 1.25 · Gin · MySQL 8 · Redis · goose migrations · database/sql · asynq ·
`golang-jwt/jwt/v5` · `log/slog` JSON · 七牛 `go-sdk/v7`。

## 本地启动

```bash
# 1. 起 MySQL + Redis
docker compose -f deploy/docker-compose.dev.yaml up -d

# 2. 环境变量
export MYSQL_DSN='inward:inward@tcp(127.0.0.1:3307)/inwardclub2?parseTime=true&loc=UTC&charset=utf8mb4'
export REDIS_ADDR='127.0.0.1:6379'
export JWT_SIGNING_KEY='dev-signing-key'
export HTTP_ADDR=':8099'            # 8080 可能被 Docker 占用

# 3. 迁移 + 开发种子（superadmin / storeadmin，密码 password）
go run ./cmd/migrate up
go run ./cmd/migrate seed

# 4. 运行
go run ./cmd/api        # HTTP API
go run ./cmd/worker     # 异步任务（需 Redis）
```

默认 `USE_FAKE_ADAPTERS=true`，微信登录/支付、七牛、打印机、线下聚合收单均使用
fake adapter，本地开发与测试不依赖外网。

## 验收命令

```bash
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/migrate up
go run ./cmd/api
go run ./cmd/worker
go run ./cmd/reconcile --target-dsn "$MYSQL_DSN" --report ./tmp/reconciliation.json
```

## 目录

- `cmd/{api,worker,migrate,reconcile}` — 四个入口
- `internal/platform/*` — config、logger、errors、httpx、db、redis、authn、rbac、
  storescope、idempotency、outbox、audit（平台公共能力）
- `internal/modules/*` — auth、asset、store、catalog、activity、wallet、payment、printer
- `internal/bootstrap` — 组合根与路由
- `db/migrations` — goose 迁移（0001–0018）
- `docs/openapi/v2.yaml` — 接口契约；`docs/openapi/COVERAGE_MATRIX.md` — v1→v2 覆盖矩阵
- `docs/asset-service.md` — 七牛资产服务 as-built 使用说明（三端上传凭证 + 回调复用）
- `docs/adapters.md` — 线下聚合收单 + 芯烨云打印真实适配器 as-built（选择开关、协议映射、待补项）
- `docs/outbox-dispatch.md` — 事务性 outbox → asynq 分发管道 as-built（投递语义与 caveats）
- `docs/diagnostics.md` — 错误事件流 as-built 说明（error_events 持久化、保留与查询语义）
- `docs/wechat-auth.md` — 微信登录 + 手机号解析 as-built 使用说明（真实客户端 + fake 切换）

开发任务由 Claude 执行；Codex 负责拆任务、验收和合并标准。
