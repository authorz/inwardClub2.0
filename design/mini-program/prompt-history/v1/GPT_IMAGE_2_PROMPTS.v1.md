# gpt-image-2 小程序设计图提示词

## 1. 总提示词约束

以下提示词用于生成 InwardClub 2.0 小程序页面设计图。所有页面必须遵守：

```text
Use case: ui-mockup
Asset type: WeChat mini program mobile screen design
Style: high-fidelity premium product UI, black and white, clean, spacious, restrained, modern private club feeling
Palette: pure black, white, grayscale, subtle black-to-white gradients allowed
Typography: confident Chinese UI typography, strong hierarchy, readable body text
Layout: not card-based; use full-width sections, editorial spacing, dividers, bottom sheets, segmented controls, fixed bottom action bars
Device: modern phone screen, single vertical screen, 9:16 composition
Avoid: colorful restaurant app style, coupon marketplace style, card-heavy layout, nested cards, beige/cream palette, purple or blue gradients, excessive icons, decorative blobs, glassmorphism, noisy shadows, Alipay entry, sign-in/check-in UI
```

出图要求：

- 每条 prompt 生成 1 张图。
- 每张图只展示 1 个页面。
- 文件命名使用 `NN-page-slug.png`。
- 输出保存到 `design/mini-program/generated/`。

## 2. 首页

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Primary request: Design the InwardClub mini program home page, excluding sign-in/check-in. The page shows current store context, premium black-white club identity, quick entry to ordering, reservation, activity ticket folder, coupons, member wallet summary, and a full-width monochrome banner area.
Style/medium: high-fidelity mobile UI mockup, black and white premium product interface, modern private club feeling
Composition/framing: single vertical phone screen, strong top brand area, full-width sections with dividers, no card grid
Color palette: black, white, grayscale, subtle black-to-white gradients
Text language: Chinese
Required visible text: "Inward Club", "当前门店", "点餐", "桌游预约", "活动票夹", "我的券", "金币", "积分"
Constraints: no sign-in/check-in module, no colorful food promotion, no card-based layout, no Alipay
Avoid: beige, purple, blue gradients, nested cards, heavy shadows
```

## 3. 门店选择

```text
Use case: ui-mockup
Asset type: WeChat mini program store selection screen
Primary request: Design a store selection page showing nearby stores sorted by distance, with store name, business status, distance, address, phone and navigation action.
Style/medium: high-fidelity mobile UI mockup, black-white editorial list design
Composition/framing: single vertical screen, list with fine dividers, large store names, fixed top title, no cards
Color palette: black, white, grayscale
Text language: Chinese
Required visible text: "选择门店", "距离最近", "营业中", "导航", "电话"
Constraints: no card containers; use dividers and clean spacing
Avoid: map-heavy screenshot, colorful pins, beige background
```

## 4. 点餐首页

```text
Use case: ui-mockup
Asset type: WeChat mini program ordering screen
Primary request: Design an ordering page with category navigation, product list, product image area, price, stock/payment channel hints, and a fixed bottom cart bar.
Style/medium: premium black-white product UI, clean food ordering without colorful restaurant style
Composition/framing: single vertical phone screen, category rail or segmented top categories, full-width product rows separated by dividers, fixed bottom cart bar
Color palette: black, white, grayscale, subtle gradients only
Text language: Chinese
Required visible text: "点餐", "酒水", "小吃", "微信", "金币", "加入", "去结算"
Constraints: only WeChat and coin payment; no Alipay; no card grid
Avoid: colorful food delivery app style, coupon badges, heavy shadows
```

## 5. 订单确认

```text
Use case: ui-mockup
Asset type: WeChat mini program order confirmation screen
Primary request: Design a food order confirmation page with selected items, total amount, payment method selection for WeChat and coins, coin balance hint, remarks, and fixed bottom payment action.
Style/medium: high-fidelity black-white mobile checkout UI, calm and trustworthy
Composition/framing: single vertical screen, section bands and dividers, bottom action bar
Color palette: black, white, grayscale
Text language: Chinese
Required visible text: "确认订单", "支付方式", "微信支付", "金币支付", "金币余额", "合计", "确认支付"
Constraints: no Alipay; no card-based layout; money states must be clear
Avoid: colorful checkout buttons except black primary action
```

## 6. 预约桌台/座位

```text
Use case: ui-mockup
Asset type: WeChat mini program reservation screen
Primary request: Design a table and seat reservation screen for a board-game club. Show current store, table list or table layout, seat states, reservation and waitlist actions.
Style/medium: premium black-white operational UI, spacious and clear
Composition/framing: single vertical screen, flat layout/map feeling, status legend, no cards
Color palette: black, white, grayscale, subtle gradient header
Text language: Chinese
Required visible text: "桌游预约", "可预约", "已预约", "游戏中", "维护中", "排队等候", "确认预约"
Constraints: status must use text plus visual mark, not color only; no card grid
Avoid: playful cartoon board game style, colorful seating map
```

## 7. 活动列表

```text
Use case: ui-mockup
Asset type: WeChat mini program event list screen
Primary request: Design an activity list page with active events displayed as monochrome poster-like rows and historical events as concise list entries with ended labels.
Style/medium: black-white premium event UI, editorial poster composition
Composition/framing: single vertical screen, horizontal poster area or stacked full-width poster bands, no cards
Color palette: black, white, grayscale gradients
Text language: Chinese
Required visible text: "活动", "进行中", "历史活动", "已结束", "查看详情"
Constraints: no colorful event marketing style, no card layout
Avoid: neon, purple gradients, loud badges
```

## 8. 活动详情/票档

```text
Use case: ui-mockup
Asset type: WeChat mini program event detail and ticket type screen
Primary request: Design an activity detail page with monochrome hero image area, store distance, event time, location, ticket types, stock and purchase action.
Style/medium: premium black-white event detail UI, cinematic but readable
Composition/framing: single vertical screen, full-width hero, details as clean rows, ticket type selector, fixed bottom buy bar
Color palette: black, white, grayscale, subtle dark gradient over hero
Text language: Chinese
Required visible text: "活动详情", "早鸟票", "预售票", "剩余", "微信", "金币", "立即购票"
Constraints: no Alipay; ticket status must be clear; no nested cards
Avoid: colorful concert poster style, noisy decorations
```

## 9. 我的首页

```text
Use case: ui-mockup
Asset type: WeChat mini program member center screen
Primary request: Design the member center home page with profile, member level, invite code, wallet summary, order entries, coupons, invitation, ranking, membership benefits, and optional staff entry.
Style/medium: premium black-white member UI, elegant and restrained
Composition/framing: single vertical screen, strong profile header, divided action rows, no cards
Color palette: black, white, grayscale
Text language: Chinese
Required visible text: "我的", "会员等级", "邀请码", "金币", "积分", "我的券", "订单", "邀请", "排行榜", "会员权益"
Constraints: no sign-in/check-in, no card-heavy wallet
Avoid: colorful badges, cute icons, beige background
```

## 10. 我的券/票夹

```text
Use case: ui-mockup
Asset type: WeChat mini program coupons and ticket folder screen
Primary request: Design a coupons and activity ticket folder page with segmented tabs for coupons, activity tickets, used and expired states. Show validity, applicable items, verification code entry and clear status.
Style/medium: black-white operational UI, precise and premium
Composition/framing: single vertical screen, segmented control, list rows with strong dividers, bottom action for verify/show code
Color palette: black, white, grayscale
Text language: Chinese
Required visible text: "活动票夹", "我的券", "可使用", "已使用", "已过期", "核销码", "去使用"
Constraints: status must be explicit; no card coupons; no colorful coupon marketplace style
Avoid: red coupon design, perforated coupon card visuals
```

## 11. 工作人员核销

```text
Use case: ui-mockup
Asset type: WeChat mini program staff verification screen
Primary request: Design a staff verification page for activity tickets and coupons. It should show current store, scan/manual code input, today's activity summary, verification result area, and recent verification history.
Style/medium: black-white professional operations UI, fast and clear
Composition/framing: single vertical screen, utility-first layout, large scan/action area, clear input, no cards
Color palette: black, white, grayscale, high contrast
Text language: Chinese
Required visible text: "工作人员", "当前门店", "扫码核销", "手输核销码", "今日活动", "核销历史", "确认核销"
Constraints: staff-only feeling, success/failure states must be unambiguous, no card layout
Avoid: colorful scanner app style, decorative icons
```
