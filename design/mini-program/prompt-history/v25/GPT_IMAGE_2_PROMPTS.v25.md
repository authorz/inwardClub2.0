# GPT Image 2 Prompts v25

## 金币 icon

用途：

- 小程序支付方式组件中的金币支付 icon
- 个人中心资产条中的金币 icon

接口：

- `api_base`: `image-key.json`
- model: `gpt-image-2`
- size: `1024x1024`

最终落地资产：

- `/mini-program/miniprogram/assets/payment/coin.png`

提示词：

```text
Use case: logo-brand
Asset type: mobile app icon
Primary request: a premium minimalist gold coin icon for a wallet/recharge feature
Scene/backdrop: perfectly flat solid chroma-key green background for easy cutout
Subject: a round embossed gold coin, centered, with concentric rings and subtle metallic shine, no text, no letters, no suit symbols, no currency symbol, no emblem inside, just a refined coin silhouette
Style/medium: clean 3D icon, app-friendly, polished but simple
Composition/framing: centered, square crop, icon fills about 70 percent of the frame
Lighting/mood: soft studio lighting, premium, crisp, balanced
Color palette: gold, warm highlights, deep amber shadows, chroma-key green background
Materials/textures: metallic coin edges, brushed gold surface, slightly raised rim
Constraints: no text, no watermark, no extra objects
Avoid: club symbol, poker symbol, card suit, logo mark, glow blobs, scattered coins, background gradients, shadows on background
```

后处理：

- 使用 chroma-key 去除纯色背景。
- 裁切透明边缘。
- 输出为 256x256 RGBA PNG。

## 微信支付 icon

微信支付 icon 不使用 AI 生成，必须使用微信官方品牌素材。

官方来源：

- 微信设计资源库：`https://wechat.design/tool/brand/`
- 官方包：`微信支付Emblem_V2.0_20200529_liueliu.zip`

最终落地资产：

- `/mini-program/miniprogram/assets/payment/wechat-pay.png`

处理规则：

- 从官方全彩中文竖版标志中裁出微信支付 emblem。
- 不改颜色、不重绘、不替换为自制图标。
- 仅做透明背景裁切和尺寸压缩，适配小程序支付方式行内 icon。
