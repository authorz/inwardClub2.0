# InwardClub 2.0 七牛云文件与图片服务开发规格

本文定义 InwardClub 2.0 所有文件、图片和未来音视频资源的唯一接入方式。实现者不得使用本地 `public/storage`、业务表 URL 字段直写、客户端自定义对象 key，或其他对象存储服务。

官方依据：七牛对象存储 Go SDK 文档 <https://developer.qiniu.com/kodo/1238/go>，页面最近更新时间为 2025-04-02。该文档确认 SDK 模块为 `github.com/qiniu/go-sdk/v7`，支持服务端签发客户端上传凭证、服务端直传、回调验签、私有空间下载签名和对象删除。

## 1. 必须实现的架构

```text
小程序 / 管理后台
  -> POST /api/v2/assets/upload-credentials
  <- 临时 token + 固定 objectKey + 上传地址
  -> 直传七牛 Kodo
七牛 Kodo
  -> POST /internal/qiniu/upload-callback
  -> 验证签名、校验 key、创建/更新 assets 行
业务端
  -> POST /api/v2/assets/{assetID}/bind 或业务写接口引用 asset_id
  -> GET 公开 CDN URL；私有资源由 API 生成短时签名 URL
```

文件二进制不经过 Go API。Go API 只做鉴权、对象 key 分配、上传策略签名、回调验签、资产元数据落库和资源授权。

## 2. 环境配置

所有密钥只存在部署环境或密钥管理服务，绝不提交到仓库、日志、接口响应或前端配置。

```dotenv
QINIU_ACCESS_KEY=
QINIU_SECRET_KEY=
QINIU_BUCKET=
QINIU_REGION=
QINIU_PUBLIC_DOMAIN=
QINIU_PRIVATE_DOMAIN=
QINIU_UPLOAD_CALLBACK_URL=
QINIU_TOKEN_TTL_SECONDS=600
QINIU_PRIVATE_URL_TTL_SECONDS=300
```

开发环境必须使用独立 bucket 与域名。生产 bucket 禁止匿名写入；公开读取只允许专用 CDN 域名。

## 3. Go 依赖与配置接口

在 Go module 中固定使用官方 v7 SDK：

```go
require github.com/qiniu/go-sdk/v7 v7.25.3
```

对 SDK 作唯一封装（as-built 落地于 `server/internal/modules/asset/qiniu_client.go`，使用说明见 `server/docs/asset-service.md`）。其公开接口如下，业务模块不得直接引用七牛 SDK：

```go
type UploadPurpose string

const (
    UploadPurposeAvatar       UploadPurpose = "avatar"
    UploadPurposeStoreLogo    UploadPurpose = "store_logo"
    UploadPurposeBanner       UploadPurpose = "banner"
    UploadPurposeCategory     UploadPurpose = "category"
    UploadPurposeProduct      UploadPurpose = "product"
    UploadPurposeActivity     UploadPurpose = "activity"
    UploadPurposeTableLayout  UploadPurpose = "table_layout"
    UploadPurposeSeatLayout   UploadPurpose = "seat_layout"
    UploadPurposeVipIcon      UploadPurpose = "vip_icon"
    UploadPurposeRichContent  UploadPurpose = "rich_content"
)

type UploadCredential struct {
    AssetID      int64     `json:"assetId"`
    ObjectKey    string    `json:"objectKey"`
    UploadToken  string    `json:"uploadToken"`
    UploadURL    string    `json:"uploadUrl"`
    ExpiresAt    time.Time `json:"expiresAt"`
    MaxSizeBytes int64     `json:"maxSizeBytes"`
}

type ObjectStore interface {
    CreateUploadCredential(ctx context.Context, input CreateCredentialInput) (UploadCredential, error)
    VerifyUploadCallback(req *http.Request) (CallbackPayload, error)
    PublicURL(objectKey string) string
    PrivateURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
    Delete(ctx context.Context, objectKey string) error
}
```

## 4. 资产表和数据库迁移

创建 `assets` 表；所有图片/文件字段逐步迁移为 `asset_id`，保留旧 URL 只用于历史兼容读取。

```sql
CREATE TABLE assets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  bucket VARCHAR(100) NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  etag VARCHAR(128) NULL,
  original_filename VARCHAR(255) NOT NULL,
  content_type VARCHAR(100) NOT NULL,
  size_bytes BIGINT UNSIGNED NOT NULL,
  width INT UNSIGNED NULL,
  height INT UNSIGNED NULL,
  purpose VARCHAR(32) NOT NULL,
  visibility VARCHAR(16) NOT NULL DEFAULT 'public',
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  uploaded_by_type VARCHAR(20) NOT NULL,
  uploaded_by_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL,
  uploaded_at DATETIME NULL,
  deleted_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_assets_object_key (object_key),
  KEY idx_assets_owner_purpose (uploaded_by_type, uploaded_by_id, purpose),
  KEY idx_assets_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

对象 key 由后端生成，格式固定为：`inwardclub/{environment}/{purpose}/{yyyy}/{mm}/{assetID}-{random}.{ext}`。随机段至少 16 个安全随机十六进制字符。客户端传入的原始文件名只能用于展示，绝不参与路径。

## 5. 上传凭证接口

### `POST /api/v2/assets/upload-credentials`

认证：小程序用户、门店后台、总后台均可调用，但由权限决定允许的 `purpose` 和关联对象。请求：

```json
{
  "purpose": "product",
  "filename": "menu.jpg",
  "contentType": "image/jpeg",
  "sizeBytes": 1048576,
  "visibility": "public"
}
```

成功响应：

```json
{
  "data": {
    "assetId": 123,
    "objectKey": "inwardclub/prod/product/2026/07/123-a1b2c3d4e5f60708.jpg",
    "uploadToken": "...",
    "uploadUrl": "https://up-<region>.qiniup.com",
    "expiresAt": "2026-07-14T12:10:00Z",
    "maxSizeBytes": 5242880
  }
}
```

实现规则：

1. 在创建 token 前验证 purpose、MIME、扩展名、文件大小和调用者权限。
2. 先写入 `assets(status=pending)`，再以该 `asset_id` 生成 object key。
3. 使用 `credentials.NewCredentials`、`uptoken.NewPutPolicyWithKey`、`uptoken.NewSigner(...).GetUpToken` 创建 token；token 过期时间固定 10 分钟。
4. 上传策略必须限制 bucket 和唯一 object key，设置 `SetFsizeLimit(maxSize)`，设置 JSON `returnBody`，并设置回调 URL/body/body type。
5. 策略回调 body 至少包含 `assetId`（自定义变量）、`key`、`etag`、`fsize`、`mimeType`、`bucket`。客户端上传时必须带 `x:assetId`，后端仍须验证 object key 与待上传资产一致。
6. 禁止生成覆盖上传 token；修改资源一律新建 asset，再由业务实体原子切换引用。

## 6. 类型、大小和访问控制

| 用途 | 可接受 MIME | 最大大小 | 可见性 | 调用者 |
| --- | --- | ---: | --- | --- |
| `avatar` | JPEG、PNG、WebP | 5 MiB | public | 本人 |
| `store_logo`、`banner`、`category`、`product`、`activity`、`table_layout`、`seat_layout`、`vip_icon` | JPEG、PNG、WebP | 10 MiB | public | 有对应后台权限者 |
| `rich_content` | JPEG、PNG、WebP、MP4 | 图片 10 MiB；视频 200 MiB | public | 总后台/门店后台内容权限者 |
| 私有凭证/内部附件（后续启用） | 由专用用途白名单定义 | 20 MiB | private | 仅获授权角色 |

服务端在回调后对图片执行 `image.DecodeConfig` 复核宽高和格式；不信任请求头 MIME。无法解析、大小不一致、purpose 不匹配或 key 不匹配时，把资产标为 `failed` 并异步删除七牛对象。

## 7. 七牛回调接口

### `POST /internal/qiniu/upload-callback`

此接口不使用 JWT，只接受七牛回调。实现顺序必须如下：

1. 限制来源网络，并设置严格请求体上限。
2. 使用官方 `credentials.VerifyCallback(mac, request)` 验证 `Authorization` 签名；验证失败返回 `401`。
3. 解析 JSON body，锁定 `assets` 行；检查状态仍为 `pending`、asset ID/key/bucket 均一致。
4. 校验回传尺寸、MIME 和 object key 规则；更新 `etag`、`size_bytes`、`status=uploaded`、`uploaded_at`。
5. 返回 JSON `{ "assetId": 123, "status": "uploaded" }`；重复回调返回同一成功结果，不重复创建资产。

回调处理必须幂等。回调收到后不可立刻信任“已被业务使用”；只有业务实体将 `asset_id` 持久化后把资产标记为 `bound`。

## 8. 业务绑定、展示和删除

- 所有创建/更新门店、商品、分类、活动、Banner、桌台、座位、VIP、头像的 API 接收 `assetId`，不再接收任意 URL。
- 写业务实体时验证资产状态为 `uploaded` 或已由当前实体绑定，验证 purpose 和调用者归属，再在同一数据库事务内更新实体并标记 `assets.status=bound`。
- 公共资源 URL 按 `QINIU_PUBLIC_DOMAIN + "/" + object_key` 生成；响应可包含 `assetId`、`url`、`width`、`height`。
- 私有资源只能由授权 API 以官方签名下载 URL 生成器返回，TTL 固定 5 分钟；不把私有 bucket 域名直接返回。
- 业务删除资源时先解除引用并软删除 asset；异步任务调用对象删除。七牛删除失败可重试，DB 删除状态必须可追踪。
- 每小时清理超过 24 小时仍为 `pending` 的资产与七牛孤儿对象；只删除由当前环境 key 前缀产生的对象。

## 9. 必须覆盖的测试

1. 未授权角色不能为不允许的 purpose 申请 token。
2. token 的对象 key 不能被客户端替换，也不能覆盖已有资源。
3. 伪造、过期或篡改的回调被拒绝；重复回调保持幂等。
4. 错误 MIME、超限大小、损坏图片、asset ID/key 不一致均不能绑定业务实体。
5. 门店管理员不能绑定其他门店或其他管理员创建的资源。
6. 删除业务实体后对象删除任务可重试且不会误删仍被引用的资源。
7. 私有资源 URL 在有效期后失效，公共资源不需要服务端签名。

## 10. 实现完成标准

- 仓库内只存在一个七牛 SDK 封装包，所有上传入口都使用该包。
- 数据库中所有新业务实体只存 `asset_id`，不新增裸 URL 字段。
- `.env.example` 仅列变量名，不含真实 AK/SK、bucket、域名或回调地址。
- 日志只能记录 asset ID、object key、bucket 和错误类别，不能记录 token、AK、SK 或完整私有签名 URL。
