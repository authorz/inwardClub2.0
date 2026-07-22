# 资产服务（七牛）as-built 使用说明

面向后端实现者。本文描述 `internal/modules/asset` 已落地的共享资产服务，以及
小程序 / 总后台 / 门店后台三端如何复用上传凭证与回调，以及后续上传相关任务应
如何调用本服务。设计规格（约束与安全要求的权威来源）见
[`docs/QINIU_ASSET_SERVICE_SPEC.md`](../../docs/QINIU_ASSET_SERVICE_SPEC.md)。

## 1. 组件与落地位置

七牛 SDK 只在本包封装一次；业务模块不得直接 import `github.com/qiniu/go-sdk/v7`。

| 文件 | 职责 |
| --- | --- |
| `objectstore.go` | `ObjectStore` 端口 + 领域类型（`UploadPurpose`、`UploadCredential`、`CallbackPayload`…）。业务代码只依赖这里。 |
| `qiniu_client.go` | `QiniuObjectStore`：唯一的七牛 v7 SDK 封装（签发凭证、服务端直传、回调验签、私有签名、删除）。 |
| `fake_client.go` | `FakeObjectStore`：`USE_FAKE_ADAPTERS=true`（默认）时使用，不触网，回调用 HMAC-SHA256 模拟验签。 |
| `service.go` | `Service`：校验 purpose/MIME/大小/调用者权限、落 `assets(pending)` 行、生成 object key、签发凭证、处理回调（幂等）。 |
| `handler.go` | `Handler`：HTTP 入口 `UploadCredentials` 与 `Callback`。三端共用同一个 handler。 |
| `objectkey.go` | 服务端生成固定格式 object key，客户端文件名绝不参与路径。 |
| `model.go` | `purpose → {可接受 MIME, 最大大小}` 策略表。 |
| `repository.go` | `assets` 表读写（`Repository` 端口 + MySQL 实现）。 |

组合根 `internal/bootstrap/app.go` 里按 `USE_FAKE_ADAPTERS` 选择 `FakeObjectStore`
或 `QiniuObjectStore`（`buildObjectStore`），再注入 `asset.NewService`。

## 2. 路由（三端复用）

同一个 `Handler` 挂在三个控制台组和内部回调组，无需为每端重写：

| 方法 & 路径 | 认证 | 说明 |
| --- | --- | --- |
| `POST /api/v2/mini/assets/upload-credentials` | 小程序 JWT（member/staff） | `router.go` `registerMini` |
| `POST /api/v2/admin/assets/upload-credentials` | 总后台 JWT（super_admin） | `router.go` `registerAdmin` |
| `POST /api/v2/store/assets/upload-credentials` | 门店后台 JWT（store_admin/cashier） | `router.go` `registerStore` |
| `POST /internal/qiniu/upload-callback` | 无 JWT，靠七牛签名验签 | `router.go` `registerInternal` |

调用者身份来自 JWT claims，权限由 `Service.callerAllowed` 决定：`avatar` 仅本人
（member/staff）；其余运营类 purpose 需后台角色（super_admin / store_admin /
cashier）。

## 3. 环境契约

密钥只在部署环境注入，绝不入库/日志/响应。所有变量见 `.env.example`，其中：

```dotenv
QINIU_ACCESS_KEY=
QINIU_SECRET_KEY=
QINIU_BUCKET=
QINIU_REGION=
QINIU_PUBLIC_DOMAIN=            # 公共读 CDN 域名；PublicURL = 该域名 + "/" + objectKey
QINIU_PRIVATE_DOMAIN=           # 私有资源签名下载域名
QINIU_UPLOAD_CALLBACK_URL=      # 七牛回调打到 /internal/qiniu/upload-callback 的公网地址
QINIU_TOKEN_TTL_SECONDS=600     # 上传 token 有效期（默认 10 分钟）
QINIU_PRIVATE_URL_TTL_SECONDS=300
```

`QINIU_PUBLIC_DOMAIN` 无 scheme 时按 `https://` 处理（见 `QiniuObjectStore.PublicURL`）。
`USE_FAKE_ADAPTERS=false` 时 `config.Validate` 强制要求 AK/SK/Bucket 非空。

## 4. 客户端直传流程（三端 UI 的标准调用方式）

文件二进制不经过 Go API。前端流程固定为：

1. `POST /api/v2/{mini,admin,store}/assets/upload-credentials`

   ```json
   { "purpose": "product", "filename": "menu.jpg",
     "contentType": "image/jpeg", "sizeBytes": 1048576, "visibility": "public" }
   ```

   响应（`{"data": …}` 信封）：

   ```json
   { "data": {
       "assetId": 123,
       "objectKey": "inwardclub/prod/product/2026/07/123-a1b2c3d4e5f60708.jpg",
       "uploadToken": "…", "uploadUrl": "https://up-<region>.qiniup.com",
       "expiresAt": "2026-07-18T12:10:00Z", "maxSizeBytes": 10485760 } }
   ```

2. 前端用 `uploadToken` 直传七牛 Kodo（`uploadUrl`），表单必须带自定义变量
   `x:assetId`（值为上面的 `assetId`），object key 用服务端返回的 `objectKey`。
3. 七牛回调 `/internal/qiniu/upload-callback`；服务端验签、核对 key/bucket、把
   `assets` 行置为 `uploaded`（幂等）。回调响应 `{ "assetId": 123, "status": "uploaded" }`。
4. 业务写接口只收 `assetId`（不收裸 URL），落库时校验资产归属再引用。

## 5. 后续上传任务应如何调用本服务

- **新增前端上传入口（新页面 / 新 purpose）**：无需改 handler。若是全新 purpose，
  在 `objectstore.go` 增常量、在 `model.go` `policies` 增一行（MIME + 大小），必要时
  在 `service.go` `callerAllowed` 调整角色，即可复用现有三条路由。
- **服务端直传（种子数据、系统生成图、定时任务）**：调用
  `ObjectStore.UploadPublicObject(ctx, PublicUploadInput{...})`，或直接用 CLI
  `go run ./cmd/qiniu-upload -file ./logo.png -key seed/logo.png`（同一封装，
  从 `.env` 读配置，不打印密钥）。同一 object key 覆盖写，可重复执行。
- **读取展示 URL**：业务模块只存 `asset_id`，通过窄接口
  `PublicURLByID(ctx, id) (string, error)` 取公共 URL —— 各模块（store / catalog /
  activity / member / order）已用自己的 `AssetResolver` 接口内嵌 `asset.Service`
  这一个方法即可，不要新拉整份 service。私有资源用
  `ObjectStore.PrivateURL(ctx, objectKey, ttl)` 返回 5 分钟签名 URL。
- **删除**：`ObjectStore.Delete(ctx, objectKey)`；业务侧应先解引用再软删 asset，
  再异步删对象（详见规格 §8）。

## 6. 测试

- `service_test.go` —— object key 格式、MIME/大小拒绝、purpose 权限、回调幸福路径 /
  幂等 / 伪造 / key 不匹配。
- `objectstore_test.go` —— `PublicURL` 默认 https、区域上传主机、Fake 直传确定性。
- `handler_test.go` —— HTTP 层冒烟：一个 handler 为 mini/admin/store 三端签发凭证
  （复用性）、禁止的 purpose 返回 403、凭证→回调 全链路（签名成功 / 伪造 401）。

均使用 `FakeObjectStore` + 内存仓储，不触网、不做破坏性外部操作。运行：

```bash
go test ./internal/modules/asset/...
```

## 7. 已知待办（不在本次范围）

- 回调后用 `image.DecodeConfig` 复核宽高/格式（规格 §6）——需拉取对象，尚未实现。
- 业务实体绑定后把 `assets.status` 置为 `bound`（规格 §8）——绑定流程随各业务写接口落地。
- 每小时清理超过 24h 的 `pending` 资产（规格 §8）——**DB 侧已落地**：
  `asset.CleanupService.SweepPending`（`cleanup.go`）把仍为 `pending` 且
  `created_at` 早于 24h（`pendingTTL`）的资产置 `failed`；worker 任务
  `asset:pending-cleanup` 按 `@every 1h` 调度，`status='pending'` 守卫保证幂等
  （重跑/漏跑/重复 tick 均安全）。**仍待办**：删除对应七牛对象（被弃 `pending`
  对象与当前环境 key 前缀下的孤儿对象），需把 `ObjectStore.Delete` 接入扫描并新增
  bucket 列举能力，暂缓（与其余纯 DB 扫描一致，不在扫描里触碰适配器）。
