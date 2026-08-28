# 线下聚合收单 & 云打印真实适配器 as-built

面向后端实现者。本文描述两个"真实适配器路径"的落地：门店线下聚合收款
（`payment.OfflineAcquirer`）与芯烨云云打印（`printer.Printer`）。二者都藏在既有
接口 + 配置之后，**默认仍是 fake**；仅当 `USE_FAKE_ADAPTERS=false` 且相应凭证齐全时
才切换到真实实现。第三方 SDK / 协议细节只在这里封装，业务代码不受影响。

## 1. 选择开关

| 适配器 | 接口 | 默认（`USE_FAKE_ADAPTERS=true`） | 真实实现（`false`） | 选择位置 |
| --- | --- | --- | --- | --- |
| 线下聚合收单 | `payment.OfflineAcquirer` | `FakeOfflineAcquirer` | `payment.HTTPAcquirer` | `bootstrap.buildOfflineAcquirer` |
| 云打印 | `printer.Printer` | `FakePrinter` | `printer.XpyunPrinter` | `printer.Select`（worker 消费） |

`config.Validate()` 在 `USE_FAKE_ADAPTERS=false` 时强制校验两者所需的凭证（连同
七牛 / 微信），因此配置缺失会在启动时快速失败，而不是在首个请求时才炸。

## 2. 线下聚合收单 `payment.HTTPAcquirer`

尚未选定具体收单机构，因此采用一套**清晰、可映射**的通用 HMAC 签名 JSON 协议：
每个出站请求体用商户 `apiKey` 做 `HMAC-SHA256`（小写 hex）签名，放在 `X-Sign` 头；
每个入站回调用同样方式验签。选定具体机构后，只需改本文件里的字段映射，业务代码不动。

| 方法 | HTTP | 说明 |
| --- | --- | --- |
| `CreateDynamicQR` | `POST {baseUrl}/collect/qr` | 开动态收款码。请求 `{merchantId,outTradeNo,amountCent,subject,expiresAt,notifyUrl,nonce,timestamp}`；响应 `{code,msg,data:{acquirerOrderNo,qrContent,expiresAt?}}`，`code==0` 为成功。 |
| `VerifyNotification` | 收单机构回调 | 读原始 body → 与 `X-Sign` 比对（`hmac.Equal`）→ 解析 `{outTradeNo,acquirerOrderNo,externalTradeNo,channel,amountCent,status}`；`status=="success"` 记为已付。验签失败返回 `UNAUTHENTICATED`，handler 回非 2xx 触发机构重试。付款渠道（wechat/alipay）由机构决定，服务端不猜。 |
| `Refund` | `POST {baseUrl}/refund` | 退款。请求 `{merchantId,acquirerOrderNo,outRefundNo,amountCent,nonce,timestamp}`；响应 `{code,msg,data:{refundNo}}`。 |

- 入口回调仍是既有 `POST /internal/payments/offline-acquirer/notify`（`payment.Handler.OfflineNotify`）。
- `CreateDynamicQR` 已被 `StoreService.CreateCollectionOrder` 实时调用；切到真实实现即真实开码。
- **仍待补**：`Refund` 目前无实时调用方——退款下发走 outbox worker，尚未接线（见 §4）。

## 3. 云打印 `printer.XpyunPrinter`

芯烨云开放平台（`https://open.xpyun.net/api/openapi/xprinter`）。账户级鉴权用配置里的
`user + ukey`，每次请求签名 `SHA1(user + ukey + timestamp)`（协议要求）。

| 方法 | HTTP | 说明 |
| --- | --- | --- |
| `Print` | `POST {base}/print` | 请求 `{user,timestamp,sign,sn,content,copies,voice,mode}`；`sn` 取 `Job.DeviceSN`，`content` 取 `Job.Content`。响应 `{code,msg,data}`，`code==0` 为受理。`Job.Template` 由上游渲染进 `Content`，print 接口本身不吃模板。 |

- 消费方是 worker 的 `print:receipt` 处理器：解出 `printer.Job` → 调 `Printer.Print`。
  无法解码的 payload 直接丢弃（返回 nil，避免 asynq 无意义重试）；打印失败返回 error 以走
  重试预算。
- **生产端已接线**（见 §3.1）：真实结算成功时会在同一事务内写 `print:receipt` outbox 事件，
  因此该路径不再只由测试驱动。

### 3.1 小票生产端 `printer.WriteReceipt`

生产端落在 `internal/modules/printer/receipt.go`，由各结算完成点在**结算事务内**调用，
遵循既有事务性 outbox 约定（事件随结算一起提交，回滚则不发）：

| 完成点 | 位置 | 触发的业务流 |
| --- | --- | --- |
| 微信支付回调结算 | `payment.settlementSQLRepository.SettleWeChat` | 微信支付的餐饮 / 活动订单（门店绑定） |
| 线下聚合收款结算 | `payment.settlementSQLRepository.SettleOffline` | 门店柜台线下收款（始终门店绑定） |
| 金币（coin）支付结算 | `order.sqlRepository.SettleByCoin` | 金币支付的餐饮 / 活动订单（门店绑定） |

- **打印规则**：`WriteReceipt` 先查 `printer_devices` 里该门店首个 `active` 设备，取其
  `device_sn`。**只有餐饮、活动等可打印订单绑定门店且门店有在用打印机时**才写事件；
  充值订单的门店归属只用于经营统计，仍不打印。门店没有配置打印机时结算照常成功、只是不出小票。
- **payload 就是 `printer.Job`**（`DeviceSN` / `Template` / `Content`），worker 原样消费，
  无需改动处理器。`Content` 由生产端预渲染（单号、类型抬头、金额 `¥X.XX`、结算时间）；
  `Template` 固定为 `order-receipt`（芯烨云 print 接口不吃模板，仅日志/未来分支用）。
- **恰好一次**：outbox `idem_key` 取 `payment:{paymentOrderId}:print-receipt`，作为 asynq
  `TaskID` 去重；结算本身幂等（重复回调只结算一次），故每个已付订单只出一张小票。

## 4. 仍延后到 worker / 生产端的部分

- ~~`print:receipt` 的**生产端**~~ 已实现（见 §3.1）：三个结算完成点（微信 / 线下 / 余额）
  在结算事务内写 `print:receipt` 事件。
- 线下收单**退款下发**（worker 调 `OfflineAcquirer.Refund` 并回写退款状态机）尚未接线；
  当前门店/总后台退款只落 `pending`。
- 会员权益发放 `rule:post-process` 仍为 no-op（既有延后项）。

## 5. 环境变量

真实线下收单（`USE_FAKE_ADAPTERS=false` 时必填）：

```
OFFLINE_ACQUIRER_PROVIDER      # 机构标识
OFFLINE_ACQUIRER_MERCHANT_ID   # 商户号
OFFLINE_ACQUIRER_API_KEY       # HMAC 签名密钥
OFFLINE_ACQUIRER_BASE_URL      # 机构 HTTP API 根地址
OFFLINE_ACQUIRER_NOTIFY_URL    # 机构回调地址（/internal/payments/offline-acquirer/notify）
```

真实云打印（`USE_FAKE_ADAPTERS=false` 时必填）：

```
XPYUN_USER
XPYUN_UKEY
```
