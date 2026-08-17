# 后端验收清单（Backend Acceptance Checklist）

范围：`server/` Go 服务，`/api/v2/*` 与 `/internal/*`。依据 `server/docs/openapi/v2.yaml`
（契约）与 `server/docs/openapi/COVERAGE_MATRIX.md`（v1→v2 覆盖矩阵，M0 = 已实现）。
仅收录当前已实现（矩阵中标注"已实现"）的能力域；骨架/`NOT_IMPLEMENTED` 路由不在本清单。

用途：人类或 Agent 在每次交付/回归后按顺序跑一遍，快速判断已交付能力是否仍然可用。

## 1. 前置条件

```bash
cd server
docker compose -f deploy/docker-compose.dev.yaml up -d   # MySQL + Redis
export MYSQL_DSN='inward:inward@tcp(127.0.0.1:3307)/inwardclub2?parseTime=true&loc=UTC&charset=utf8mb4'
export REDIS_ADDR='127.0.0.1:6379'
export JWT_SIGNING_KEY='dev-signing-key'
export HTTP_ADDR=':8099'
go run ./cmd/migrate up
go run ./cmd/api
```

接口验收前，应在隔离的测试环境中准备测试账号和业务数据；项目不内置默认账号或示例数据。

`USE_FAKE_ADAPTERS=true`（默认）：微信登录/支付、七牛、打印机、线下聚合收单全部走 fake，
无需外网依赖。

## 2. 静态检查（先于接口验收）

```bash
go vet ./...
go test ./...
go test -race ./...
```

全部通过后再进行下面的接口级验收。

## 3. 覆盖的已交付能力域

| 能力域 | 三端 | 关键路径 |
| --- | --- | --- |
| 健康检查 | - | `/internal/health`,`/internal/ready` |
| 认证（三套 audience 隔离） | mini/admin/store | `*/auth/login`(或 `wechat/login`),`/refresh`,`/me`,`/logout` |
| 资产上传（合并至七牛） | mini/admin/store | `*/assets/upload-credentials`,`/internal/qiniu/upload-callback` |
| 门店公开信息 | mini | `/mini/stores`,`/{id}` |
| 分类/商品公开读取 | mini | `/mini/stores/{id}/catalog/categories`,`/items` |
| 活动公开读取 | mini | `/mini/activities`,`/{id}`,`/mini/stores/{id}/activities` |
| 钱包/账本（只读+只追加） | mini/admin/store | `/mini/wallet`,`/wallet/ledger`,`/admin/wallet-ledger`,`/store/wallet-ledger` |
| 支付流水/支付单查询 | admin/store | `/{admin,store}/payment-transactions`,`/payment-orders` |
| 退款（查询 + 发起） | admin/store | `/{admin,store}/refunds`,`/refund-orders`（只读别名） |
| 会员详情/钱包手动调整 | admin/store | `/{admin,store}/members/{memberID}`,`/wallet-adjustments` |
| 员工/收银员账号写操作 | admin/store | `/admin/staff-accounts/*`；`/store/cashiers/*`,`/store/staff-accounts/*` |

未在此表出现的能力域（会员等级/充值档位、签到/积分提现/邀请、点餐下单支付、活动购票核销、
券、预约排队、线下聚合收款、后台商品/活动 CRUD 等）在矩阵中仍是骨架或 M1+ 里程碑，
**不纳入本清单**，避免验收虚构接口。

## 4. 推荐冒烟顺序

按依赖顺序执行，每步失败即停止并定位：

1. **健康检查**
   `GET /internal/health` → 200；`GET /internal/ready` → 200（探活 MySQL）。
2. **三端登录**（验证 audience 隔离）
   - `POST /api/v2/mini/auth/wechat/login` `{code}`（fake adapter 任意 code）→ 返回
     `token.accessToken/refreshToken` + `profile`。
   - 使用测试环境的总后台账号调用 `POST /api/v2/admin/auth/login` → 200；
     用同一 token 调用 `POST /api/v2/store/auth/*` 应被拒绝（audience 不匹配）。
   - 使用测试环境的门店账号调用 `POST /api/v2/store/auth/login` → 200，
     token 携带 store scope。
3. **鉴权 `/me`**
   `GET /api/v2/mini/me`、`/api/v2/admin/auth/me`、`/api/v2/store/auth/me` 各带对应 token
   → 200；不带 token 或带错 audience token → 401/403。
4. **刷新与登出**
   `POST */auth/refresh` 用 refreshToken 换新 token pair；`POST */auth/logout` 后旧
   accessToken 应失效（token version bump）。
5. **资产上传凭证**
   `POST /api/v2/mini/assets/upload-credentials`（带 mini token）→ 返回七牛上传凭证；
   同理验证 admin/store 端点各自的 scope。
6. **小程序公开读**
   `GET /mini/stores` → 返回测试环境的真实数据；`GET /mini/stores/{id}`、
   `/catalog/categories`、`/catalog/items`、`/mini/activities` → 200，字段与 `v2.yaml`
   schema 一致。
7. **钱包只读+账本只追加**
   `GET /mini/wallet`、`/mini/wallet/ledger`（mini token）→ 200；`GET /admin/wallet-ledger`
   与 `/store/wallet-ledger` 各自只能看到管理范围内数据（admin 跨店，store 限本店）。
8. **支付/退款查询**
   `GET /admin/payment-transactions`、`/admin/payment-orders`、`/admin/refunds` 分页参数
   生效；`store` 对应端点验证结果被 token 中的 store scope 过滤（不接受请求参数指定
   storeId 越权）。
9. **会员详情与钱包调整**
   `GET /{admin,store}/members/{memberID}` → 200；store 端对不属于本店的 memberID
   → 404。`POST .../wallet-adjustments` 写入后账本追加一条记录，不可修改历史记录。
10. **员工/收银员账号写操作**
    `POST /admin/staff-accounts` 建号 → `PATCH /{staffID}` 改资料 → `POST /{staffID}/disable`
    禁用；`/store/cashiers` 同样验证增/改/禁用/密码重置全流程。禁用后旧 token 应无法再登录。
11. **回归收尾**
    `go run ./cmd/reconcile --target-dsn "$MYSQL_DSN" --report ./tmp/reconciliation.json`
    确认对账框架仍可运行且报告生成。

## 5. 阻断项（出现即视为不通过）

- 任一 audience 的 token 可访问其他 audience 的路由。
- `store/*` 接口的数据范围可被请求参数（如 `storeId`）改变，而非仅由 token scope 决定。
- 钱包账本出现 `UPDATE`/`DELETE`（应只追加）。
- 小程序相关 DTO 出现支付宝字段。
- 微信支付回调未走统一 `/internal/payments/wechat/notify`（如涉及）。

## 6. 已知文档缺口

- `V1_API_INVENTORY_AND_V2_MAPPING.md` 与 `COVERAGE_MATRIX.md` 未标注具体的错误码/校验规则，
  冒烟测试目前只能靠 `v2.yaml` 的 response schema 粗略核对。
- 尚无自动化的契约测试（如 schemathesis/Dredd）针对 `v2.yaml` 跑一致性检查，本清单第 4 节
  仍需人工或脚本手动调用。
- `server/docs/openapi/v2.yaml` 里未实现的骨架路径（M1+）没有单独文档标注"预期 404/
  NOT_IMPLEMENTED 行为"，回归时若误触会缺少参照标准。

## 7. 冒烟测试记录（2026-07-18）

针对本地已运行的 `go run ./cmd/api` 实例（`127.0.0.1:18110`）执行只读+代表性写操作冒烟测试，
使用测试环境单独准备的总后台与门店账号。

**通过：**

- `store/cashiers` 增/改/禁用/密码重置全流程 200/201。
- `store/staff-accounts` 增/改/禁用全流程 200/201。
- `store/members/{id}` 详情读取、`wallet-adjustments` 写入（`direction` 取值为
  `credit`/`debit`，非 `increase`/`decrease`）200/201，余额正确累加且账本追加。
- `admin/refunds`、`admin/refund-orders`（别名）、`store/refunds` 只读 200。

**未通过 / 无法验证：**

- `admin/payment-orders`、`store/payment-orders`、`admin/rule-definitions`
  的 POST/PATCH/publish/disable 系列写接口均返回 `404 page not found`，
  尽管路由已在 `internal/bootstrap/router.go` 中定义（如 121/261/143-146 行）。
  根因非代码缺陷：当前存活的两个 `go run` 进程（PID 7217 于 09:19 启动、PID 15971
  于 07:00 启动）编译时间早于 `router.go` 最后一次修改时间（当日 12:07），属于
  **运行中二进制落后于源码**，并非路由未实现。建议下次验收前重启 `go run ./cmd/api`
  重新编译，再复测这三类写接口。
- 未重启共享的本地 dev server（可能被其他并行任务占用），因此以上三项本轮未能验证通过/失败，仅记录为阻断项。

**待办：** 重新编译后补测 `admin/payment-orders` 读接口、`admin/rule-definitions`
create/update/publish/disable 全流程。

### 复测（2026-07-18，全量重编译全表面扫描）

见 `docs/acceptance/backend-smoke-2026-07-18-full-sweep.md`。用当前源码**新编译**
的二进制（`go build ./cmd/api`，端口 18120）对 `router.go` 中**全部**已注册路由
（三端 + `/internal`）做了一轮扫描，解决了上文"运行中二进制落后于源码"的阻断项。

- **静态检查全绿：** `go vet` / `go build` / `go test ./...` / `go test -race ./...`。
- **194 项请求级检查：0 个 5xx、0 个 `NOT_IMPLEMENTED`。** `stub()` 骨架处理器已不再
  挂载，整套文档化接口面均为真实实现（覆盖矩阵里的"骨架/M1+"标注已过时）。
- 上一条"待办"全部复测通过：`admin/payment-orders` 读（列表 200 / 未知 id 404 信封）、
  `admin/rule-definitions` create→publish→disable 均 200。
- **§5 阻断项全部 PASS：** audience 隔离（跨端 token 一律 401）、门店 scope 不可被
  请求参数改变（`storeId:999` 被强制回落到 token 的门店 1；越权会员 404）、钱包账本
  仅 `INSERT`、mini DTO 无支付宝字段、微信回调单一入口。
- **幂等：** 重复 `Idempotency-Key` 返回 `409 CONFLICT`（claim-and-reject，非重放
  原响应），同日重复签到不重复发放（余额仅追加一条）。
- **发现一项（非 §5 阻断，功能缺口 vs §4.4）：** `POST */auth/logout` 会 bump
  `token_version` 并使 refresh 失效，但 access-token 中间件（`authn/middleware.go`
  `RequireAuth`）是无状态的、不校验 `token_version`，故**登出后旧 accessToken 仍可用
  至自然过期（2h）**。三端一致。需产品决策：放宽 §4.4 表述，或在中间件补 token_version
  校验。详见 full-sweep 报告 F1。
- 复测未改动任何 `.go` 源码；仅新增验收报告与本记录。

### 修复（2026-07-18，§4.4 access-token 失效）

上条 F1 采用"在中间件补 token_version 校验"方案，已落地：

- `RequireAuth` 现按 audience 注入 `TokenVersionChecker`（mini 读 `members`、
  admin/store 读 `admin_accounts`），在验签/audience/subject/scope 全部通过后，
  以 `SubjectID` 查当前 `token_version` 并与 access token 内声明比对；不一致（登出
  /禁用/改密均已 bump）即 401 `session expired`。三端一致，**§4.4 现真正通过**。
- 同一处理器过去只在 refresh 路径校验 `token_version`（`admin/repository.go` 的
  禁用/改密注释也写"下次 refresh 才失效"）；此改动把同一失效原语扩展到 access token，
  所以禁用账号、重置密码后旧 access token 也**立即**失效，而非等到刷新。
- **性能取舍：** 每个"已认证请求"新增 1 次主键 `SELECT`（members/admin_accounts）。
  与业务处理已有的 DB 往返相比属边际成本；换取的是登出后立即失效。**未加缓存**是
  刻意的——短 TTL 缓存会重新引入一段"登出后仍可用"的窗口，令 §4.4 复现失败；若日后
  QPS 需要，横向扩展路径是 Redis 版本缓存 + 登出时失效（跨实例），而非进程内 TTL。
- 新增测试：`authn/middleware_test.go`（mini/admin/store 三端过期 token→401、当前
  版本→200、subject 不存在→401、查库失败→500 fail-closed）与
  `auth/service_test.go`（登出后 checker 反映 bump 后的版本）。
- 改动文件：`authn/middleware.go`、`auth/tokenversion.go`（新增）、`bootstrap/app.go`、
  `bootstrap/router.go`、`docs/openapi/v2.yaml`（三处 logout 描述）。
