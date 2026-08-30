# 微信登录 + 手机号解析 as-built 使用说明

面向后端实现者。本文描述 `internal/modules/auth` 已落地的微信小程序登录与手机号
解析适配层：接口边界、真实客户端实现、`USE_FAKE_ADAPTERS` 切换、以及生产所需的
环境变量与运行时依赖。支付相关的微信能力（WeChat Pay）不在本文范围。

## 1. 组件与落地位置

微信开放接口只在本包封装；业务代码（`auth.Service`、`member.Service`）只依赖接口
`WeChatClient` / `member.PhoneResolver`，不感知 HTTP 细节。

| 文件 | 职责 |
| --- | --- |
| `wechat.go` | `WeChatClient` 端口 + 领域类型（`WeChatSession`）；`FakeWeChatClient`：`USE_FAKE_ADAPTERS=true`（默认）时使用，不触网，openid/手机号由 code 确定性派生。 |
| `wechat_real.go` | `WeChatHTTPClient`：唯一的微信开放接口封装（jscode2session 登录、getuserphonenumber 手机号、stable_token 令牌缓存）。仅 `USE_FAKE_ADAPTERS=false` 时使用。 |
| `service.go` | `Service.MiniLogin`：用 `Code2Session` 换 openid，首登建号，签发 mini audience token。 |
| `wechat_real_test.go` | 用 `httptest` 打桩微信三个端点，覆盖成功/错误码/令牌缓存/令牌失效重试。 |

`member.Service.BindPhone`（`internal/modules/member/service.go`）通过
`PhoneResolver` 端口调用 `WeChatHTTPClient.GetPhoneNumber`，只向前端返回打码手机号。

组合根 `internal/bootstrap/app.go` 按 `USE_FAKE_ADAPTERS` 选择 `FakeWeChatClient`
或 `WeChatHTTPClient`（`buildWeChatClient`），同一个实例同时供登录（`auth.Service`）
与手机号绑定（`phoneResolverAdapter`）使用。

## 2. 真实客户端调用的微信端点

| 用途 | 方法 & 路径 | 说明 |
| --- | --- | --- |
| 登录换 openid | `GET /sns/jscode2session` | 参数 `appid/secret/js_code/grant_type=authorization_code`；返回 `openid/unionid/session_key`。 |
| 手机号解析 | `POST /wxa/business/getuserphonenumber?access_token=...` | body `{"code": <phoneCode>}`；使用微信验证后的 `phone_info.countryCode`、`purePhoneNumber` 与 `phoneNumber`。 |
| 应用令牌 | `POST /cgi-bin/stable_token` | body `{grant_type:client_credential, appid, secret, force_refresh}`；返回 `access_token/expires_in`。 |

Host 默认 `https://api.weixin.qq.com`，测试用 `httptest` 覆盖（`baseURL` 字段）。

### 令牌缓存与容错

- `access_token` 通过 **stable_token**（多实例安全）获取并在进程内缓存，过期前
  `tokenRefreshMargin`（5 分钟）主动刷新，避免与微信端过期竞争。
- 缓存令牌被外部作废时（errcode `40001/40014/42001`），`GetPhoneNumber` 会
  `force_refresh` 令牌并重试一次。
- 微信逻辑错误统一走 HTTP 200 + `errcode` 语义；本客户端把 `errcode != 0` 转成
  Go error，由上层（`auth.Service` / `member.Service`）归一为 `UNAUTHENTICATED`。
- 每次外呼 10s 超时，防止上游卡死阻塞登录/绑定请求。

## 3. 环境变量与运行时

真实登录/手机号解析所需（`USE_FAKE_ADAPTERS=false` 时由 `config.Validate` 强制）：

| 变量 | 说明 |
| --- | --- |
| `USE_FAKE_ADAPTERS` | 置 `false` 启用真实客户端；默认 `true` 走 fake。 |
| `WECHAT_MINI_APP_ID` | 小程序 AppID。 |
| `WECHAT_MINI_APP_SECRET` | 小程序 AppSecret。 |

运行时依赖：出网可达 `api.weixin.qq.com`；小程序后台需把服务器出口 IP 加入
**IP 白名单**（否则 stable_token 返回 errcode 40164）。手机号能力需小程序已开通
「手机号快速验证」权限。缺少任一 mini 凭证时 `Validate` 直接失败，服务不启动。

fake 默认下（本地/测试）无需任何微信凭证，也不触网。

## 4. 契约要点

- `WeChatClient` 接口签名不变；真实与 fake 均实现 `Code2Session` /
  `GetPhoneNumber`，切换只在组合根一处。
- 手机号只能通过微信一次性 `phoneCode` 在服务端换取。注册 DTO 不接收手机号，换绑 DTO
  也只接收 `code`，客户端不能手工提交号码完成注册或换绑。
- 中国大陆 `countryCode=86` 继续保存 11 位 `purePhoneNumber`，兼容已有会员；海外号码
  规范化为 E.164（如 `+85291234567`），以“国家码 + 本地号码”参与唯一约束。
- 国家码缺失且无法明确判断为大陆号码时拒绝绑定，避免把不同国家的同号号码错误合并。
- 大陆与海外号码均按保留前 3、后 4 的规则掩码展示。
- `session_key` 已随 `WeChatSession` 返回；当前登录只用 `openid`，如未来改用
  加密数据解密手机号可直接复用。
