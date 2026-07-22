# gpt-image-2 小程序设计图提示词 v2

## 1. 本次修改来源

参考图目录：

```text
design-demo/
```

本次用户反馈：

1. 配色参考 `design-demo/`：黑白、浅灰、深灰到黑渐变、大面积留白、黑色胶囊按钮。
2. 活动列表必须是卡片形式，可左右滑动展示上一个/下一个活动，点击后展开活动详情。
3. 选购商品页面左侧为分类栏，右侧商品一行展示两个商品，商品可选择数量，需要步进器。
4. 预约页面以桌子为单位，每张桌子 9 个座位，点击座位弹出预约选项框确认是否预约。
5. 每一次修改设计稿的提示词都必须保留，用于后续 Claude 高保真还原。

## 2. 全局提示词约束 v2

```text
Use case: ui-mockup
Asset type: WeChat mini program mobile screen design
Style: high-fidelity premium product UI, black and white, clean, spacious, restrained, modern club feeling
Palette: pure black, white, cool light gray, dark gray to black gradients, black capsule buttons, subtle gray dividers
Typography: confident Chinese UI typography, strong hierarchy, readable body text
Reference visual language: large whitespace, minimal black/gray interface, iOS mini-program chrome, black rounded capsule primary buttons, subtle empty-state illustration style, light gray page background when needed
Device: modern phone screen, single vertical screen, 9:16 composition
Avoid: colorful restaurant app style, coupon marketplace style, beige/cream palette, purple or blue gradients, excessive icons, glassmorphism, noisy shadows, Alipay entry, sign-in/check-in UI
Important: Preserve prompt text and output version for developer handoff.
```

## 3. 活动列表卡片轮播 v2

```text
Use case: ui-mockup
Asset type: WeChat mini program activity list screen
Primary request: Design an InwardClub activity list page where current activities are shown as large swipeable cards. The user can swipe left/right to view previous and next activity. Tapping a card expands or enters the activity detail. Include historical activities below as a simpler list.
Style/medium: high-fidelity mobile UI mockup, black-white premium club style, inspired by dark gray to black gradient reference screens
Composition/framing: single vertical phone screen, top navigation title "活动", main carousel with one large centered event card and partial previous/next card edges visible, card can be used here because activity carousel is a required interaction, below it a concise history list with dividers
Color palette: black, white, grayscale, dark gray to black gradients, no color accents except subtle status marks
Text language: Chinese
Required visible text: "活动", "近期活动", "黑桃周赛", "左右滑动", "查看详情", "历史活动", "已结束"
Interaction cues: visible pagination dots or side peeks to imply swipeable cards
Constraints: no colorful marketing poster, no sign-in/check-in, no Alipay
Avoid: flat list-only activity page, neon concert style, purple/blue gradients
```

## 4. 点餐商品选择 v2

```text
Use case: ui-mockup
Asset type: WeChat mini program ordering screen
Primary request: Design the ordering page. The left side is a vertical category rail. The right side shows products in a two-column grid, two products per row. Each product has image, name, price, stock/payment hint and a quantity stepper with minus, quantity, plus. Include a fixed bottom cart bar.
Style/medium: high-fidelity mobile UI mockup, black-white clean ordering interface, light gray background with white product cells and subtle dividers
Composition/framing: single vertical phone screen, left category sidebar about 22% width, right two-column product grid, fixed bottom cart bar with total and checkout button
Color palette: white, black, light gray, dark gray, black capsule checkout button
Text language: Chinese
Required visible text: "点餐", "酒水", "小吃", "饮品", "微信", "金币", "¥", "去结算"
Required UI: quantity stepper on each product: "− 1 +", selected products may show quantity greater than 0
Constraints: no Alipay, no colorful food delivery style, no single-column product list
Avoid: heavy card shadows, red/orange food promotion, coupon badges
```

## 5. 预约桌子九座 v2

```text
Use case: ui-mockup
Asset type: WeChat mini program reservation screen
Primary request: Design the reservation page based on table units. Each table module represents one table with exactly 9 seats. Each seat is clickable. Show a reservation confirmation bottom sheet after selecting one available seat.
Style/medium: high-fidelity mobile UI mockup, black-white and light gray operational UI, closely referencing the provided reservation screenshot structure
Composition/framing: single vertical phone screen, top title "预约", current store row, rules link, scrollable table modules. Each table module contains table name, update time, status dot, base points, seat count "0/9", central table graphic and 9 seat buttons around it. Show bottom sheet/modal asking to confirm reservation for selected seat.
Color palette: white, light gray, black text, gray dividers, small status dot, optional orange/blue numeric emphasis only if needed by reference, otherwise grayscale
Text language: Chinese
Required visible text: "预约", "预约规则与牌桌礼仪", "1号桌", "基础积分", "50/100", "座位 0/9", "可预定", "确认预约", "取消"
Required UI: exactly 9 seat controls around one table; selected seat state; bottom confirmation sheet
Constraints: status must use text and icon, not color alone; no Alipay; no sign-in/check-in
Avoid: abstract floor map with many unrelated tables, dark-only map, colorful seating chart
```

## 6. 订单确认 v2 修正

```text
Use case: ui-mockup
Asset type: WeChat mini program order confirmation screen
Primary request: Redesign the food order confirmation page for offline store ordering, not delivery. Show current store, optional table/seat, selected items, remark, payment method selection for WeChat and coins, coin balance hint, total amount, and fixed bottom confirm payment action.
Style/medium: high-fidelity black-white mobile checkout UI, calm and trustworthy, reference light gray/white screens with black capsule button
Composition/framing: single vertical screen, top title, store/table context band, item rows with dividers, payment method rows, fixed bottom bar
Color palette: white, black, light gray, subtle dividers, black primary button
Text language: Chinese
Required visible text: "确认订单", "当前门店", "桌台", "商品明细", "备注", "微信支付", "金币支付", "金币余额", "合计", "确认支付"
Constraints: no delivery, no express shipping, no buyer message wording, no Alipay, no card-heavy checkout
Avoid: e-commerce checkout, shipping address, logistics icons
```

## 7. 工作人员核销 v2 修正

```text
Use case: ui-mockup
Asset type: WeChat mini program staff verification screen
Primary request: Redesign the staff verification page. Staff must be fixed to the current store. Show current store only, no switch store action. Include scan verification, manual verification code input, today's activity summary, verification result, and recent verification history.
Style/medium: black-white professional operations UI, fast and clear, light gray/white reference style with black primary button
Composition/framing: single vertical screen, top title "工作人员", current store display row without switching, two large utility actions, stats row, result area, history list, fixed or prominent confirm button
Color palette: white, black, light gray, grayscale icons
Text language: Chinese
Required visible text: "工作人员", "当前门店", "扫码核销", "手输核销码", "今日活动", "核销结果", "核销历史", "确认核销"
Constraints: no switch store, no cross-store selector, no Alipay, no colorful scanner app style
Avoid: "切换门店", store dropdown, colorful scan frame
```
