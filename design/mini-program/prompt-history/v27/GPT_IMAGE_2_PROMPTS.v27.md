# GPT Image 2 Prompts v27

## 深色页面无数据空状态插画

用途：

- 活动列表无数据状态插画
- 后续深色页面空状态可复用的视觉基准

参考图：

- `/design-demo/微信图片_20260714061142_28_188.png`

接口：

- `api_base`: `image-key.json`
- model: `gpt-image-2`
- size: `1024x1024`

最终落地资产：

- `/mini-program/miniprogram/assets/empty/activity-empty.png`

验收预览：

- `/design/mini-program/final/icon-assets/empty-state-soft-v27-preview.png`

提示词：

```text
Use case: logo-brand
Asset type: WeChat mini program empty-state illustration asset
Primary request: Create a soft, comfortable empty-state illustration inspired by the provided reference screen: a quiet folded paper / invitation sheet floating gently, with one small loose curl line above it. It should feel calm, premium, understated, and suitable for "no data" states in InwardClub pages.
Scene/backdrop: perfectly flat solid chroma-key green background (#00ff00) for background removal
Subject: one simple folded paper or ticket-like sheet, dark charcoal gray, low contrast, soft curved silhouette, two short pale gray diagonal strokes on the paper, and one small thin pale gray spiral/curl line above the sheet. No text.
Style/medium: minimalist 2.5D illustration, not a hard icon, soft edges, relaxed proportions, premium black-white-gray club UI feeling
Composition/framing: centered square crop, subject occupies about 48 percent of the frame, generous empty space around it, paper sits slightly below center, curl line above center
Lighting/mood: very soft diffuse lighting, no obvious cast shadow, calm and comfortable
Color palette: charcoal #202020, deep gray #2b2b2b, muted gray highlights #8a8a8a, chroma-key green background only
Constraints: no words, no logos, no QR code, no people, no decorative stars, no colorful accents, no white card background, no heavy shadow, no hard black outline
Avoid: generic coupon icon, shopping voucher, envelope mail icon, 3D shiny object, overly sharp paper, high-contrast white illustration, marketplace promotion style, red/orange color, blue/purple gradients
```

后处理：

- 使用 chroma-key 去除纯色背景。
- 保留宽松透明边缘，不裁得过紧，让页面空状态有呼吸感。
- 输出为 RGBA PNG。

设计规则：

- 插画 icon 必须单独提取为透明 PNG 资产，不允许只存在于整屏 mock 设计图中。
- 小程序所有无数据空状态默认使用该插画，包括入场券、优惠券、订单、预约、流水、门店、工作人员空记录等，不再按业务类型单独换图标。
- 空状态插画必须轻、暗、低对比，不要做成亮色功能 icon。
- 深色页面空状态的图标大小应克制，文案使用中灰色，不要白色高亮。
- 空状态插画和文案之间留足间距，页面整体保持大面积留白。
