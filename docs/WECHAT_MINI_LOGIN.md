# 微信小程序登录接入文档

本文描述 InwardClub 2.0 当前实现，可作为其他项目的接入蓝本。流程适用于微信小程序，不适用于公众号网页 OAuth。

## 1. 总体流程

1. 小程序调用 `wx.login()` 获得一次性 `code`。
2. 客户端将 `code` 发送到服务端 `POST /api/v2/mini/auth/wechat/login`。
3. 服务端调用微信 `GET https://api.weixin.qq.com/sns/jscode2session`，使用 `appid`、`secret`、`js_code`、`grant_type=authorization_code` 换取 `openid`（以及仅服务端使用的 `session_key`）。
4. 已完成资料的会员直接返回 access/refresh JWT；首次登录或资料未完成的会员只返回短期 `registerTicket`。
5. 新用户通过微信手机号按钮获得 `phoneCode`，服务端调用 `getuserphonenumber`，把已验证手机号写入新的注册票据。
6. 客户端提交头像、昵称、性别和注册票据到 `/register`；服务端创建或补全会员后签发正式 JWT。

服务端永远不向客户端返回 `appSecret` 或 `session_key`。`code` 和手机号 `phoneCode` 均按一次性凭证处理，不能缓存后重复使用。

## 2. 接口契约

以下路径均以 `/api/v2/mini` 为前缀。

### 登录

`POST /auth/wechat/login`

```json
{"code":"wx.login 返回的 code"}
```

老用户响应（`data` 外层为项目统一响应包装）：

```json
{"token":{"accessToken":"...","refreshToken":"...","expiresAt":"...","tokenType":"Bearer"},"profile":{},"isNew":false,"subjectType":"member"}
```

新用户响应：`{"isNew":true,"registerTicket":"..."}`，此时 `token` 为空，不能把注册票据当作 access token 调用业务接口。

### 手机号授权

`POST /auth/wechat/phone-mask`

```json
{"registerTicket":"...","phoneCode":"wx.getPhoneNumber 回调 detail.code"}
```

返回 `phoneMasked` 和新的 `registerTicket`。服务端只解密一次手机号；客户端必须用新票据覆盖旧票据。旧票据可能仍在有效期内，但不应继续使用。

### 注册完成

`POST /auth/wechat/register`

```json
{"registerTicket":"...","avatarUrl":"https://...","nickname":"昵称","gender":"male","inviterCode":"可选"}
```

`gender` 只能是 `male`、`female`、`other`；昵称长度为 1–30 个字符。成功后返回正式 token pair 和会员资料。手机号必须已嵌入注册票据，否则请求失败。

### 注册头像（可选但当前客户端使用）

`POST /auth/wechat/register-avatar`，multipart 字段：`file`、`registerTicket`。仅接受 JPEG/PNG/WebP，返回 HTTPS `avatarUrl`，再提交给 `/register`。

### 刷新与业务会话

- `POST /auth/refresh`：`{"refreshToken":"..."}`，返回新的 token pair。
- 需要登录的请求带 `Authorization: Bearer <accessToken>`。
- `GET /me` 获取当前会员；`POST /auth/logout` 注销。

JWT 的 audience 为 `mini`，服务端应校验签名、issuer、audience、过期时间、token 类型和会员 `tokenVersion`。注册票据 audience 独立为 `register`，有效期 10 分钟，仅允许注册相关接口使用。

## 3. 小程序端伪代码

```js
const login = await wxLogin();
const r = await post('/mini/auth/wechat/login', { code: login.code });
if (r.data.isNew) {
  let ticket = r.data.registerTicket;
  const phone = await wxGetPhoneNumber();
  const p = await post('/mini/auth/wechat/phone-mask', { registerTicket: ticket, phoneCode: phone.code });
  ticket = p.data.registerTicket;
  const avatarUrl = await uploadAvatar(ticket);
  const done = await post('/mini/auth/wechat/register', {
    registerTicket: ticket, avatarUrl, nickname, gender
  });
  saveTokens(done.data.token);
} else {
  saveTokens(r.data.token);
}
```

手机号按钮必须使用微信开放能力（`open-type="getPhoneNumber"`）；不要在客户端自行解密 `encryptedData`，也不要把 `session_key` 存储到本地。

## 4. 服务端实现要点

- 用数据库唯一索引约束 `wechat_openid` 和手机号，注册补全必须幂等并处理并发冲突。
- 未完成注册的用户可先建立受限身份（如项目的 `pre_member`），但不能获得完整会员权限。
- 会员停用、staff 绑定状态和 tokenVersion 必须在签发及鉴权时检查。
- 所有微信上游调用设置超时；`getuserphonenumber` 使用服务端 app access token，并在过期前刷新、遇到 token 失效时重试一次。
- 对 `code`、票据和用户输入做长度、字符和 URL 校验；日志禁止打印 secret、session_key、手机号原文和完整 token。

## 5. 配置与上线检查

```env
USE_FAKE_ADAPTERS=false
WECHAT_MINI_APP_ID=微信小程序 AppID
WECHAT_MINI_APP_SECRET=微信小程序 AppSecret
JWT_SIGNING_KEY=所有 API 实例共享的强随机密钥
JWT_ISSUER=inwardclub
```

微信公众平台需配置小程序 AppID、服务器域名（API 与对象存储 HTTPS 域名），并确保后端出口可访问 `api.weixin.qq.com`。开发环境可使用 fake adapter；生产必须关闭 fake 并通过配置校验。不要把 `.env`、AppSecret 或 JWT 密钥提交到仓库。

## 6. 常见错误

- `400 invalid request body`：缺少 `code`、票据或手机号凭证。
- `401 微信身份获取失败`：`code` 过期/重复、AppID 与小程序不匹配，或微信接口不可达。
- `401 invalid or expired register ticket`：超过 10 分钟、签名密钥不一致，或把 access token 当注册票据使用。
- `手机号已绑定其他会员账号`：业务唯一约束生效，应提示用户使用已有账号登录。
- 真机无响应：检查微信后台 request/uploadFile 合法域名、HTTPS 证书和服务端超时日志。

## 7. 迁移到其他项目的最小清单

实现五个公开接口（login、phone-mask、register、register-avatar、refresh），一个服务端微信客户端（jscode2session + getuserphonenumber），注册票据 JWT、会员表唯一约束、access/refresh 会话和小程序登录页。先用 fake adapter 覆盖成功、重复 code、过期票据、手机号重复、停用会员等测试，再切换真实凭证做真机验证。
