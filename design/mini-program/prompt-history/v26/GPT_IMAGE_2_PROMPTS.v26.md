# GPT Image 2 Prompts v26

## 入场券 icon

用途：

- 我的入场券页面空状态 icon
- 后续可用于票夹、票码相关入口的视觉资产

接口：

- `api_base`: `image-key.json`
- model: `gpt-image-2`
- size: `1024x1024`

最终落地资产：

- `/mini-program/miniprogram/assets/icons/ticket-gpt.png`

提示词：

```text
Use case: logo-brand
Asset type: mobile app feature icon
Primary request: a premium entry ticket icon for InwardClub admission tickets
Scene/backdrop: perfectly flat solid chroma-key green background for easy cutout
Subject: one horizontal admission ticket, black enamel center with refined gold metallic border, subtle inner line details, a small QR-code detail in one corner, centered crown or premium club emblem, no readable text
Style/medium: polished 3D app icon, luxury club feeling, crisp silhouette, high-end but simple
Composition/framing: centered square crop, ticket fills about 78 percent of the frame, generous padding, no tilt
Lighting/mood: soft studio lighting, premium, calm, not flashy
Color palette: black, deep charcoal, gold metallic highlights, chroma-key green background
Materials/textures: satin black surface, polished gold rim, subtle bevel
Constraints: no Chinese text, no English text, no watermark, no extra objects, no background shadow
Avoid: generic coupon marketplace icon, colorful gradient, red/orange promotion style, cluttered typography, paper receipt, lottery ticket
```

后处理：

- 使用 chroma-key 去除纯色背景。
- 裁切透明边缘。
- 输出为 RGBA PNG。

## 优惠券 icon

用途：

- 我的券页面空状态 icon
- 个人中心资产条的券 icon
- 优惠券列表中的券类型视觉标识

接口：

- `api_base`: `image-key.json`
- model: `gpt-image-2`
- size: `1024x1024`

最终落地资产：

- `/mini-program/miniprogram/assets/icons/coupon-gpt.png`

提示词：

```text
Use case: logo-brand
Asset type: mobile app feature icon
Primary request: a premium coupon icon for InwardClub benefits and vouchers
Scene/backdrop: perfectly flat solid chroma-key green background for easy cutout
Subject: one clean horizontal coupon voucher, white satin paper body with side notches, center perforation line, a small black price-tag shape on the right with one small gold dot accent, no readable text
Style/medium: polished 3D app icon, minimalist premium voucher, distinct from the admission ticket icon
Composition/framing: centered square crop, coupon fills about 76 percent of the frame, straight-on view, generous padding
Lighting/mood: soft studio lighting, refined, clean, calm
Color palette: white, soft gray, black, small gold accent, chroma-key green background
Materials/textures: satin paper, slight bevel, soft realistic depth
Constraints: no Chinese text, no English text, no watermark, no extra objects, no background shadow
Avoid: red coupon, shopping marketplace style, sale badge, percentage symbol, QR code, crown emblem, colorful gradients
```

后处理：

- 使用 chroma-key 去除纯色背景。
- 裁切透明边缘。
- 输出为 RGBA PNG。
