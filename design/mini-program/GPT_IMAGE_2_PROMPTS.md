# gpt-image-2 小程序设计图提示词汇总

## v28 个人中心会员/工作人员菜单 icon

输出文件：

```text
design/mini-program/generated/v28/mine-menu-icons-v28.png
output/imagegen/mine-icons/mine-menu-icons-v28-crops-preview.png
mini-program/miniprogram/assets/mine-menu/*.png
```

提示词：

```text
Use case: logo-brand
Asset type: monochrome menu icon design sheet for WeChat mini program personal center
Primary request: Design a cohesive premium monochrome icon set for InwardClub member center menus and staff operation menus. Create a clean 4-column grid contact sheet of 13 standalone icons, each icon centered in its own square cell with generous padding. The icons must be easy to crop into individual PNG assets later.
Icon subjects, in this exact order: 我的券, 存积分, 邀请有礼, 取积分, 充金币, 交易记录, 排行榜, 会员权益, 加入社群, 活动核销, 积分审核, 核销记录, 今日活动.
Style/medium: high-end vector-like line icons, black and dark gray strokes, consistent stroke weight, rounded line caps, simple geometric forms, premium private club app feeling.
Composition/framing: square contact sheet, 4 columns x 4 rows, first 13 cells filled and remaining cells blank. No phone mockup. No UI screen. No labels inside the cells. Each icon should occupy the same visual area, about 60% of its cell, with identical optical weight.
Color palette: pure black #111111 and neutral gray only on a pure white background.
Constraints: no text, no Chinese characters, no colored icons, no emoji, no filled colorful badges, no shadows, no gradients, no 3D, no hand-drawn sketch style. Keep every icon visually balanced and consistent.
Avoid: childish icon set, stock app icons, inconsistent sizes, tiny details, broken line art, decorative background.
```

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

## 8. v3 修改来源

基于 v2 初验：

- `05-reservation-v2.png` 有预约确认弹层，但只生成了 8 个座位，不满足每桌 9 座硬要求。
- `03-ordering-v2.png` 结构正确，但商品图和支付图标偏彩色；为了贴近参考图黑白配色，补一张更严格灰阶版本。

## 9. 点餐商品选择 v3：严格灰阶

```text
Mobile WeChat mini program ordering screen, strict black-white grayscale UI. Left vertical category rail. Right side two-column product grid, exactly two products per row. Use monochrome grayscale product photos or grayscale placeholders, no colored packaging, no colored WeChat or coin icons. Each product cell has name, price, payment hints as black/gray text, and a quantity stepper with minus, number, plus. Fixed bottom cart bar with black capsule checkout button. Chinese text: 点餐, 酒水, 小吃, 饮品, 微信, 金币, 去结算. No Alipay. Avoid colorful food delivery style.
```

## 10. 预约桌子九座 v3：强制 9 个座位

```text
Mobile WeChat mini program reservation page, closely matching the provided reservation reference. White and light gray UI with black text. Top title 预约, current store row, 预约规则与牌桌礼仪. Show one table module: 1号桌, 更新时间 06:16:10, 基础积分 50/100, 座位 0/9, status 预约中. Inside the table module, draw one central poker table graphic and EXACTLY NINE separate seat buttons labeled 可预定. Seat placement must be: four seats along the top edge, one seat on the left side, one seat on the right side, and three seats along the bottom edge. After one seat is selected, show a bottom confirmation sheet with 已选择 1号桌 · 3号座位, 取消, 确认预约. Do not draw eight seats. Do not draw more than nine seats. Chinese UI, no Alipay, no sign-in.
```

## 11. v4 修改来源

用户对活动列表提出新的视觉和交互要求：

- 活动列表的大图轮播需要有相册式堆叠感觉。
- 主活动图像居中突出，前后活动像相册一样在左右或后方露出层叠边缘。
- 可以左滑右滑切换上一个/下一个活动。
- 历史活动入口不需要大面积列表，只保留简单文字入口。

## 12. 活动列表相册式堆叠轮播 v4

```text
Use case: ui-mockup
Asset type: WeChat mini program activity list screen
Primary request: Redesign the InwardClub activity list page as a premium photo-album style stacked carousel. The active event should be a large centered visual card, while previous and next event cards are partially visible behind it with layered offsets, like a physical photo stack or album cover carousel. The user should clearly understand they can swipe left or right to browse events. Tapping the active stacked card enters or expands the activity detail. Historical activities should not be a large list; show only a small understated text link entry.
Style/medium: high-fidelity mobile UI mockup, premium black-white grayscale club style, clean and spacious, restrained, modern WeChat mini program UI
Composition/framing: single vertical phone screen. Top title "活动". Main area dominated by one stacked album carousel: centered large event image/card, left and right/back cards peeking with depth and slight offset, subtle pagination dots or swipe cue. Under carousel show concise event meta and a black capsule button. Near the bottom only a small text link "历史活动 >" or "查看历史活动", not a big history section.
Color palette: pure black, white, cool gray, dark gray to black gradients, subtle gray dividers. No colorful poster style.
Text language: Chinese
Required visible text: "活动", "近期活动", "黑桃周赛", "左右滑动", "查看详情", "历史活动"
Required UI: album-like stacked carousel, visible previous/next card edges, swipe affordance, active card detail CTA, small text-only history entry
Constraints: no sign-in/check-in, no Alipay, no large historical activity list, no colorful marketing poster, no coupon marketplace feeling
Avoid: flat single card with no stack, full-width history list, red/orange food promotion, purple/blue gradients, noisy shadows, card grid
```

## 13. v5 修改来源

用户对首页提出新的视觉要求：

- 首页需要重新设计。
- 首页最好以白色为主。
- 之前 v1 首页太复杂，需要更简洁、更轻。

## 14. 首页白底极简版 v5

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Primary request: Redesign the InwardClub mini program home page to be much simpler than the previous dark poster-like version. The page should be mostly white, quiet, premium, and tool-like. Keep only the necessary home information: current store, four main actions, a compact member asset summary, and one lightweight current activity or recommendation entry. No large dark hero poster. No complex marketing layout.
Style/medium: high-fidelity mobile UI mockup, premium minimalist WeChat mini program UI, black-white-gray palette, mostly white background, large whitespace, restrained typography
Composition/framing: single vertical phone screen. Top navigation with title "Inward Club" or "首页". A clean current store row near top with store name, distance/status, and small navigation/phone affordances. Four core action entries arranged as simple icon+text shortcuts: 点餐, 预约, 活动, 我的券. Compact member asset row showing 金币, 积分, 券. One understated activity/recommendation text row or thin banner. Bottom tab bar with 首页, 点餐, 预约, 活动, 我的.
Color palette: white as dominant background, black text, cool light gray dividers, very subtle gray gradient only if needed, black capsule primary button only for one key action if present
Text language: Chinese
Required visible text: "首页", "当前门店", "营业中", "点餐", "预约", "活动", "我的券", "金币", "积分", "券", "查看活动"
Required UI: white-first clean home, no sign-in/check-in, no Alipay, no large dark hero image, no dense stacked cards
Constraints: keep it simple, spacious, premium, not card-heavy, no poster wall, no colorful restaurant app style
Avoid: dark full-screen hero, busy brand slogan, big marketing banner, nested cards, heavy shadows, colorful food promotion, beige/cream palette, purple/blue gradients
```

## 15. v6 修改来源

用户指出 v5 首页仍不符合参考图：

- 必须参考 `design-demo/微信图片_20260714061140_27_188.png` 的结构。
- 顶部大面积空白不是普通留白，而是 Banner 图区域。
- 首页上方应该是品牌/小程序顶部 + 大 Banner 图位。
- 会员信息卡片从 Banner 下缘叠上来。
- 整体仍以白色、黑色和浅灰为主。

## 16. 首页 Banner + 会员卡叠层版 v6

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Input image role: Use the provided demo image only as layout and style reference. Do not copy its AXIS brand, logo, text, member number, or exact icons.
Primary request: Redesign the InwardClub mini program home page following the reference layout. The top large blank area is a banner image area, not ordinary whitespace. Show a clean top mini-program header with simple Inward Club brand mark/text on the left and WeChat menu capsule on the right. Below it, reserve a very large white/light banner area for a future brand or activity banner image. A white member card overlaps upward from the bottom edge of the banner area, creating the same layered effect as the reference.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first, black typography, light gray page background below the banner, subtle shadows only for floating member card
Composition/framing: single vertical phone screen. Top 40-45% is a mostly white banner area with brand header and empty banner image placeholder. Around the lower edge of this banner, place a large rounded white member card floating above light gray page background. Member card includes greeting, nickname, membership level black capsule, member number, and three main shortcuts: 我的入场券, 我的优惠券, 活动列表. Below member card show compact full-width rows/cards: 订单中心 with 查看所有订单, 会员积分 with points and black capsule 积分兑换 button, 近期活动 with 查看全部. Bottom tab bar may be visible but understated.
Color palette: dominant white, black, cool light gray, subtle black line, one restrained dark red accent only for member number if needed. No colorful banner content.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "订单中心", "查看所有订单", "会员积分", "积分兑换", "近期活动", "查看全部"
Required UI: top banner image area as large blank/placeholder, overlapping member card, three shortcut icons inside member card, simple white rows below
Constraints: no sign-in/check-in, no Alipay, no dark hero poster, no dense function grid, no ordinary store-info top layout
Avoid: filling the banner with store info, heavy marketing poster, colorful restaurant UI, beige/cream palette, purple/blue gradients, nested cards, excessive icons
```

## 17. v7 修改来源

用户对首页 v6 继续提出两点修改：

- 首页需要显示最近的门店，并提供切换门店和导航按钮。
- 首页信息太多，会员板块下面只放最新活动列表即可，不再展示订单中心、会员积分等额外模块。

## 18. 首页 Banner + 最近门店 + 会员卡 + 最新活动 v7

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Input image role: Use the provided demo image as layout reference: top banner area, floating member card, clean white/gray sections. Do not copy AXIS brand or exact icons.
Primary request: Redesign the InwardClub mini program home page with a simpler structure. Keep the top large banner image area and the floating member card, but add nearest/current store information with switch store and navigation actions. Under the member card, show only a latest activity list. Remove order center, member points, points exchange, and other extra modules from the home page.
Style/medium: high-fidelity mobile UI mockup, premium minimalist WeChat mini program UI, white-first, black typography, light gray page background, restrained shadows
Composition/framing: single vertical phone screen. Top header with Inward Club brand on the left and WeChat capsule on the right. Large white/light banner image placeholder area. Within or just below the lower part of the banner area, show a compact nearest store row: "最近门店", store name, distance/status, a text action "切换门店", and a circular navigation button "导航". A white member card overlaps the banner lower edge, with greeting, nickname, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Below member card, only show a section title "最新活动" and 2-3 simple activity rows with date/status and a "查看全部" text link. Bottom tab bar can be shown as 首页, 点餐, 预约, 活动, 我的.
Color palette: dominant white, black, cool light gray, subtle divider lines, restrained dark red accent only for member number if needed. No colorful banner content.
Text language: Chinese
Required visible text: "Inward Club", "最近门店", "Inward Club 三里屯店", "营业中", "距您 1.2km", "切换门店", "导航", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最新活动", "查看全部"
Required UI: top banner image area, nearest store row with switch store and navigation, overlapping member card, latest activity list only below member card
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no dense information modules, no ordinary store-info-only top layout
Avoid: filling the home page with many feature cards, dark hero poster, colorful restaurant UI, beige/cream palette, purple/blue gradients, heavy shadows, nested cards
```

## 19. v8 修改来源

用户对首页 v7 继续提出两点修改：

- 最近门店不要放在 Banner 区域内，应该放在用户/会员卡片下面。
- 最近门店不需要营业状态，样式参考 `01-home-v5.png` 中最近门店的干净横向样式。
- 最新活动需要改为竖版大图横向排列，一行四个。

## 20. 首页 Banner + 会员卡 + 门店横栏 + 横向活动海报 v8

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Input image role: Use the provided demo image as layout reference for top banner and floating member card. Use 01-home-v5 nearest-store row style as reference for the store row: clean horizontal row, store name, distance, switch store, navigation. Do not copy exact previous text mistakes or icons.
Primary request: Redesign the InwardClub mini program home page. Keep the top large banner image area and floating member card. Move nearest/current store information below the member card, not inside the banner. The store row should be clean and compact, similar to the 01-home-v5 nearest store style, with store name, distance, switch store action, and navigation button, but no business/open status. Below that, show only a latest activity section using vertical poster cards arranged horizontally, one row with four portrait cards visible or partially visible. Remove order center, member points, points exchange, and all other extra modules.
Style/medium: high-fidelity mobile UI mockup, premium minimalist WeChat mini program UI, white-first, black typography, light gray page background, subtle floating shadows
Composition/framing: single vertical phone screen. Top header with Inward Club brand and WeChat capsule. Large white/light banner image placeholder area. A white member card overlaps the banner lower edge, with greeting, nickname, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Directly below the member card place a clean nearest store row: label 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航 icon/button. Do not show 营业中. Below store row show 最新活动 with 查看全部, then a horizontal row of four vertical portrait activity poster cards with title/date/status under or over each poster. Bottom tab bar: 首页, 点餐, 预约, 活动, 我的.
Color palette: dominant white, black, cool light gray, subtle divider lines, restrained dark red accent only for member number if needed. Activity posters should remain grayscale/black-white, not colorful.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "黑桃周赛"
Required UI: top banner image area, floating member card, store row below member card with switch and navigation but no business status, latest activity as horizontal row of four vertical poster cards
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no dense information modules
Avoid: putting store row inside banner, showing 营业中, horizontal activity list rows, colorful restaurant UI, dark hero poster, nested cards, heavy shadows
```

## 21. v9 修改来源

用户确认底部菜单图标后，要求重新设计首页。

本次首页必须保留并强化以下已确认规则：

- 底部菜单固定为：首页、预约、点餐、我的。
- 底部菜单图标必须使用 `design/mini-program/tab-icons/` 中定稿的 SVG 风格与语义。
- 首页结构继续参考 demo：顶部 Banner 图位、会员卡叠层。
- 最近门店放在会员卡下面，不显示营业状态。
- 最新活动使用竖版大图横向排列，一行四个。

## 22. 首页定稿方向 v9

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Primary request: Redesign the InwardClub home page as the final clean premium version. Follow the confirmed structure: top banner image area, floating member card, nearest store row below member card, latest activity vertical poster row, and a fixed four-item bottom tab bar.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first, black and cool-gray typography, quiet luxury, clean spacing
Composition/framing: single vertical phone screen. Top header with Inward Club brand on the left and WeChat capsule on the right. Large white/light grayscale banner image area. A white member card overlaps the lower edge of the banner, with Hello, Authoz, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Directly below the member card, show a clean nearest store row similar to 01-home-v5: 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航. Do not show 营业中 or any business status. Below store row show 最新活动 with 查看全部, then one horizontal row of four vertical portrait poster cards. Bottom tab bar must contain exactly four items in this order: 首页, 预约, 点餐, 我的.
Bottom tab icon requirements: 首页 uses a house icon, 预约 uses a calendar icon, 点餐 uses a cloche/serving-dish with knife icon, 我的 uses a user icon. These must visually match the finalized black linear SVG style from design/mini-program/tab-icons. Active 首页 icon is black; inactive icons are gray. Do not add 活动 as a bottom tab.
Color palette: dominant white, black, cool light gray, subtle divider lines, restrained dark red accent only for member number if needed. Activity posters are grayscale/black-white.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "黑桃周赛", "首页", "预约", "点餐", "我的"
Required UI: top banner image area, floating member card, store row below member card with no business status, latest activity as one row of four vertical poster cards, fixed four-item bottom tab using finalized icon semantics
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no 活动 tab in bottom navigation
Avoid: adding fifth tab, using temporary AI-generated tab concepts, putting store row inside banner, showing 营业中, horizontal activity list rows, colorful restaurant style, dark hero poster, dense modules, nested cards
```

## 23. v10 修改来源

用户指出首页 v9 的问题：

- 画面太瘪，需要按 iPhone 17 分辨率重新设计。
- 产品配色以黑白为主，指的是 UI 主色，不是所有 Banner 和活动图片都必须变成灰度图。

本轮设备基准：

- iPhone 17 / iPhone 17 Pro 原生分辨率：1206 × 2622 px。
- 设计画布按竖屏 1206 × 2622 px。

## 24. 首页 iPhone 17 原生画布 v10

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Canvas: portrait iPhone 17 native resolution, 1206 × 2622 px, tall screen, not compressed or squashed
Primary request: Redesign the InwardClub home page for iPhone 17 native resolution. Make the layout taller, more breathable, and less flattened than the previous version. Keep the confirmed structure: top banner image area, floating member card, nearest store row below the member card, latest activity section with four vertical poster cards, and fixed four-item bottom tab bar.
Important color rule: The product UI palette is black/white/light gray, but content images such as the banner and activity posters may use tasteful real color. Do not convert all images to grayscale. Use restrained full-color photography or poster art inside image areas while keeping the interface chrome, text, buttons, and dividers black/white/gray.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, cool light gray surfaces, understated luxury, strong spacing
Composition/framing: single full-screen mobile UI. Top safe area and WeChat capsule. Large banner image area with a refined real-color or lightly desaturated club/lifestyle image. A floating white member card overlaps the lower edge of the banner with Hello, Authoz, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Below the member card, show a clean nearest store row: 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航. Do not show 营业中 or business status. Below store row show 最新活动 with 查看全部, then one horizontal row of four vertical portrait activity poster cards; posters may contain tasteful color images. Bottom tab bar exactly four items in this order: 首页, 预约, 点餐, 我的.
Bottom tab icon requirements: 首页 house icon, 预约 calendar icon, 点餐 cloche/serving-dish with knife icon, 我的 user icon. Match the finalized black linear SVG style from design/mini-program/tab-icons. Active 首页 icon black; inactive icons gray. Do not add 活动 as a bottom tab.
Color palette: UI uses white, black, cool gray, subtle dividers. Image content may include restrained color accents, natural warm/cool tones, and real poster photography. Avoid making image content all grayscale.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "黑桃周赛", "首页", "预约", "点餐", "我的"
Required UI: iPhone 17 tall canvas, top banner image area, floating member card, store row below member card with no business status, latest activity as one row of four vertical poster cards with tasteful color allowed, fixed four-item bottom tab using finalized icon semantics
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no 活动 tab in bottom navigation
Avoid: flattened/squashed layout, all grayscale imagery, adding fifth tab, temporary AI tab concepts, putting store row inside banner, showing 营业中, horizontal activity list rows, colorful restaurant style, dark hero poster, dense modules, nested cards
```

## 25. v11 修改来源

用户对首页 v10 提出比例调整：

- Banner 图太矮，需要更高。
- 活动图片太高，需要调矮一点。

本轮继续沿用：

- iPhone 17 原生画布：1206 × 2622 px。
- UI 以黑白灰为主，但 Banner 和活动图片允许真实色彩。
- 底部菜单固定为：首页、预约、点餐、我的，并使用定稿图标语义。

## 26. 首页 Banner 加高 + 活动海报压低 v11

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Canvas: portrait iPhone 17 native resolution, 1206 × 2622 px
Primary request: Refine the v10 InwardClub home page layout. Keep the same premium white-first UI and confirmed structure, but make the top banner image area taller and more immersive. Make the latest activity poster cards shorter than v10, so they do not dominate the page.
Important layout proportions: The banner image should be noticeably taller than v10, approximately 28-32% of the visible screen before the floating member card overlap. The latest activity portrait cards should be shorter and more compact, approximately 65-75% of the previous poster height, while still staying vertical and arranged in one horizontal row of four.
Important color rule: UI chrome, text, buttons, and dividers use black/white/light gray. Banner and activity posters may use tasteful real color; do not force them to grayscale.
Composition/framing: single iPhone 17 full-screen UI. Top safe area and WeChat capsule. Taller refined club/lifestyle banner image. Floating white member card overlaps the lower edge of the banner. Member card includes Hello, Authoz, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Below member card: compact nearest store row with 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航, no 营业中. Below store row: 最新活动 with 查看全部 and one horizontal row of four shorter vertical poster cards. Bottom tab bar exactly four items: 首页, 预约, 点餐, 我的.
Bottom tab icon requirements: 首页 house icon, 预约 calendar icon, 点餐 cloche/serving-dish with knife icon, 我的 user icon. Match finalized black linear SVG style from design/mini-program/tab-icons.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "首页", "预约", "点餐", "我的"
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no 活动 tab in bottom navigation
Avoid: short banner, oversized activity posters, flattened/squashed layout, all grayscale imagery, adding fifth tab, showing 营业中, dense modules
```

## 27. v12 修改来源

用户继续设计活动列表页，并指出上次 v4 版本太黑。

本次要求：

- 保留活动列表的相册式堆叠轮播。
- 整体背景改成白色/浅灰为主，不要大面积黑底。
- 产品 UI 仍以黑白灰为主，但活动海报图片允许真实色彩，不强制灰度。
- 历史活动仍保持轻量文字入口。
- 活动列表页没有底部 TabBar，也没有顶部导航区域。

## 28. 活动列表浅色相册轮播 v12

```text
Use case: ui-mockup
Asset type: WeChat mini program activity list screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Redesign the InwardClub activity list page to be much lighter than the previous dark version. Keep the premium photo-album style stacked carousel interaction, but use a white/light-gray page background instead of black. The active event should be a large centered portrait card with previous and next cards peeking behind it like a photo stack. Activity poster images may use tasteful real color, while the UI chrome remains black/white/light gray.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained shadows, clean spacing
Composition/framing: single vertical content screen with no top navigation bar and no bottom tab bar. Do not show back arrow, page title navigation bar, WeChat capsule, or bottom navigation. Start directly with the activity page content. Main section title "近期活动". Centered stacked album carousel with one large active activity poster and two side/back posters partially visible. Under carousel show small swipe cue "左右滑动" with dots/arrows. Show concise activity meta and a black capsule "查看详情" button. Historical activities should be only a small text link near the bottom: "历史活动 >"; do not show a large history list.
Color rule: Interface background, text, divider, buttons are black/white/light gray. Activity poster images can be real color, tasteful and premium, not all grayscale.
Text language: Chinese
Required visible text: "活动", "近期活动", "黑桃周赛", "左右滑动", "查看详情", "历史活动"
Required UI: light background, album-like stacked carousel, visible previous/next card edges, swipe affordance, active card detail CTA, small text-only history entry
Constraints: no sign-in/check-in, no Alipay, no large historical activity list, no tabbar, no top navigation area, no WeChat capsule, no back arrow, no dark full-screen background
Avoid: top nav bar, bottom tab bar, mostly black page, all grayscale imagery, flat single card with no stack, full-width history list, colorful restaurant promotion style, purple/blue gradients, heavy shadows, dense modules
```

## 29. v13 修改来源

用户对活动列表 v12 提出两点修改：

- 页面背景色更喜欢 `design-demo/微信图片_20260714061142_28_188.png` 的深灰到黑色渐变。
- 内容有重复，活动列表应该展示 InwardClub 项目真正需要的信息，而不是重复放大同一活动标题。

继续保留的规则：

- 活动列表页没有底部 TabBar。
- 活动列表页没有顶部导航区域。
- 不显示返回箭头、微信胶囊、状态栏。
- 历史活动保持轻量入口。

## 30. 活动列表深灰渐变业务信息版 v13

```text
Use case: ui-mockup
Asset type: WeChat mini program activity list content screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Redesign the InwardClub activity list page using the provided demo background style: dark gray at the top fading into black, elegant and quiet. Do not make the content repetitive. Keep an album-like stacked carousel for active events, but the page should show useful activity information: event time, store/location, ticket price or ticket tier, remaining tickets/seats, and status. Include a light entry for "我的入场券" and a small "历史活动" text entry.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, dark gray-to-black gradient background inspired by the demo image, white/gray typography, black/white restrained UI
Composition/framing: single vertical content screen with no top navigation bar and no bottom tab bar. Do not show status bar, back arrow, WeChat capsule, or bottom navigation. Start with a large centered title "近期活动". Below, use a stacked album carousel with a current event poster in front and previous/next cards peeking behind. The active poster may have tasteful real color or dark club photography. Do not repeat the same title multiple times outside necessary places.
Content under carousel: show one concise information block for the selected event: title "黑桃周赛", time "每周六 19:30", store "Inward Club 三里屯店", ticket tier "标准票 ¥128", remaining "余票 24", status "报名中". Primary black/gray capsule button "查看详情". Secondary capsule or text entry "我的入场券". Bottom small text link "历史活动 >".
Color rule: Background follows the demo: dark gray top gradient to black. UI text and controls are white, gray, and black. Poster images can use restrained real color; do not force all images to grayscale.
Text language: Chinese
Required visible text: "近期活动", "黑桃周赛", "每周六 19:30", "Inward Club 三里屯店", "标准票 ¥128", "余票 24", "报名中", "查看详情", "我的入场券", "历史活动"
Required UI: dark demo-like gradient background, album-like stacked carousel, non-repetitive business info block, visible previous/next card edges, swipe affordance, my ticket entry, small history entry
Constraints: no sign-in/check-in, no Alipay, no large historical activity list, no tabbar, no top navigation area, no WeChat capsule, no back arrow, no status bar
Avoid: white/light page background, mostly repeated activity title, duplicated content blocks, full-width history list, colorful restaurant promotion style, purple/blue gradients, dense modules
```

## 31. v14 修改来源

用户纠正 v13：

- 应该在 v12 的版本中修改。
- 只是去掉重复内容，不需要展示特别多内容。
- 活动详情页会展示详细信息，列表页只需要必要摘要。

继续保留：

- 活动列表页没有顶部导航区域。
- 活动列表页没有底部 TabBar。
- v12 的浅色背景和相册式堆叠轮播方向。
- 历史活动是轻量文字入口。

## 32. 活动列表 v12 简化内容版 v14

```text
Use case: ui-mockup
Asset type: WeChat mini program activity list content screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Refine the v12 InwardClub activity list page. Keep the v12 light white/light-gray background and premium album-like stacked carousel. Remove repetitive content and avoid showing too many details. The activity list page only needs a simple summary because the activity detail page will show full details.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained shadows, clean spacing
Composition/framing: single vertical content screen with no top navigation bar and no bottom tab bar. Do not show status bar, back arrow, WeChat capsule, or bottom navigation. Start directly with page content. Main section title "近期活动". Centered stacked album carousel with one large active event poster and previous/next posters peeking behind. Under carousel show small swipe cue "左右滑动" with dots/arrows. Below it show only a concise activity summary: title "黑桃周赛", brief meta "本周六 · Inward Club 三里屯店", status "报名中", and a black capsule "查看详情" button. Add a light "我的入场券" entry and a small text link "历史活动 >" near the bottom.
Content rule: Do not repeat the title in multiple large places outside the poster and summary. Do not show detailed ticket tier, price, remaining tickets, reward, participant count, or long description on this list page.
Color rule: Interface background, text, dividers, buttons are black/white/light gray. Activity poster images can use tasteful real color, not forced grayscale.
Text language: Chinese
Required visible text: "近期活动", "黑桃周赛", "本周六", "Inward Club 三里屯店", "报名中", "查看详情", "我的入场券", "历史活动"
Required UI: light background, album-like stacked carousel, visible previous/next card edges, swipe affordance, concise non-repetitive summary, my ticket entry, small history entry
Constraints: no sign-in/check-in, no Alipay, no large historical activity list, no tabbar, no top navigation area, no WeChat capsule, no back arrow, no status bar, no detailed ticket info on list page
Avoid: detail-page information overload, ticket tier, ticket price, remaining ticket count, reward info, participant count, repeated large title blocks, full-width history list, mostly black page, all grayscale imagery, dense modules
```

## 33. 活动详情浅色购票版 v15

```text
Use case: ui-mockup
Asset type: WeChat mini program activity detail and ticket purchase screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Redesign the InwardClub activity detail page after the finalized light activity list. The list page only shows a brief summary; this detail page should show the full event information, ticket tiers, stock, sale time, payment methods, and a clear purchase action.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained dividers, quiet luxury, clean spacing
Composition/framing: single vertical detail screen. Use a clean white/light-gray page background. At the top show a large activity hero image area with tasteful real color or lightly desaturated club/table-game lifestyle imagery, not a loud concert poster. Overlay only minimal dark gradient if needed for readable title. Below the hero, show the activity title "黑桃周赛" and status "报名中". Then use clean full-width detail rows with thin dividers for time, store/location, distance, and sale window. Add a short activity description section. Add a ticket selection section with two or three ticket tiers as horizontal rows, not nested cards: "早鸟票", "预售票", "双人票"; each row shows price, remaining stock, sale time, WeChat/coin payment tags, and selected state. Include a small quantity stepper for ticket count. Use a fixed bottom purchase bar with selected total price and a black primary button "立即购票".
Content rule: The activity detail page can show ticket price, remaining stock, sale time, purchase limit, payment methods, and concise activity content. Keep the description short and structured; do not create a long article layout.
Color rule: Interface chrome, text, dividers, ticket rows, and buttons are black/white/light gray. Hero/activity imagery can use restrained real color, not forced grayscale. Use black primary CTA and subtle gray selected states.
Text language: Chinese
Required visible text: "活动详情", "黑桃周赛", "报名中", "本周六 20:00", "Inward Club 三里屯店", "距您 1.2km", "活动介绍", "票档选择", "早鸟票", "预售票", "双人票", "剩余", "售卖至", "微信", "金币", "限购 4 张", "立即购票"
Required UI: large hero image, title and status, time/store/distance rows, activity description, ticket tier selector with stock and sale time, payment method tags, quantity stepper, fixed bottom buy bar
Constraints: no sign-in/check-in, no Alipay, no bottom tabbar, no WeChat capsule, no status bar, no colorful marketing poster, no nested cards, no heavy shadows, no repeated title blocks
Avoid: mostly black page, nightclub concert UI, red coupon style, ecommerce delivery language, noisy decorations, oversized history activity area, dense unreadable text, fake tabbar icons
```

## 34. 活动详情两状态交互 v16

用户纠正：活动详情页默认显示活动详情或票类型，点击购买后才弹出选择票档；支付方式可以在票档选择弹窗中选择。因此 v16 分为默认详情态和票档/支付弹窗态。

提示词详见：

- `design/mini-program/prompt-history/v16/GPT_IMAGE_2_PROMPTS.v16.md`

生成文件：

- 默认详情态：`design/mini-program/generated/v16/07-activity-detail-v16-default-iphone17.png`
- 票档/支付弹窗态：`design/mini-program/generated/v16/07-activity-detail-v16-purchase-sheet-iphone17.png`

核心规则：

- 默认详情态只展示活动详情和票类型概览。
- 默认详情态不显示支付方式选择。
- 默认详情态不显示数量步进器。
- 点击“立即购票”后弹出票档选择弹窗。
- 弹窗中选择票档、数量和支付方式。
- 支付方式只允许微信支付和金币支付，不出现支付宝。

## 35. 预约德州扑克桌两状态 v17

用户新增预约页要求：桌子必须是德州扑克桌面，一张桌子有 9 个座位，座位有预约状态和空闲状态，页面需要有“排队等候”按钮。

提示词详见：

- `design/mini-program/prompt-history/v17/GPT_IMAGE_2_PROMPTS.v17.md`

生成文件：

- 默认态：`design/mini-program/generated/v17/05-reservation-v17-default-iphone17.png`
- 预约确认弹窗态：`design/mini-program/generated/v17/05-reservation-v17-confirm-sheet-iphone17.png`

核心规则：

- 桌子必须是德州扑克桌面。
- 每张桌子固定 9 个座位。
- 座位状态至少包含空闲、已预约、已选择。
- 默认态必须有“排队等候”按钮。
- 点击空闲座位后弹出预约确认弹层。
- 确认弹层内提供“改为排队等候”替代动作。

## 36. 预约多桌非拉伸交付 v18

用户指出预约页需要考虑多张桌子的情况，并提醒此前若把图片强行拉伸到 iPhone 17 比例，可能误导 Claude。

提示词详见：

- `design/mini-program/prompt-history/v18/GPT_IMAGE_2_PROMPTS.v18.md`

生成文件：

- 多桌默认态最终图：`design/mini-program/final/reservation/05-reservation-final-multi-table.png`
- 确认弹窗态最终图：`design/mini-program/final/reservation/05-reservation-final-confirm-sheet.png`
- 两状态等比预览最终图：`design/mini-program/final/reservation/05-reservation-final-contact-sheet.png`

核心规则：

- 预约页必须支持多张桌子。
- 默认展开一张德州扑克桌，其他桌以紧凑摘要展示。
- 展开的桌子必须固定 9 个座位。
- 页面必须有“排队等候”按钮。
- 点击空闲座位后弹出预约确认弹层。
- v18 以源图自然比例为准，不再输出非等比拉伸的 iPhone 17 适配图作为 Claude 像素参考。

## 37. 点餐两状态 v19

继续设计点餐页面，并遵守之前确认的点餐要求：左侧分类栏，右侧商品一行两个，每个商品有数量步进器，底部购物车/结算栏，线下门店点餐语义，不出现配送/快递/买家留言，不出现支付宝。

提示词详见：

- `design/mini-program/prompt-history/v19/GPT_IMAGE_2_PROMPTS.v19.md`

生成文件：

- 默认空购物车态：`design/mini-program/generated/v19/03-ordering-v19-default-source.png`
- 已加购态：`design/mini-program/generated/v19/03-ordering-v19-active-cart-source.png`
- 两状态等比预览：`design/mini-program/generated/v19/03-ordering-v19-contact-sheet.png`

核心规则：

- 左侧固定分类栏。
- 右侧商品区一行两个商品。
- 每个商品有图片、名称、价格、微信/金币提示和数量步进器。
- 底部购物车栏展示已选数量、合计金额和“去结算”。
- 底部菜单固定为：首页、预约、点餐、我的。
- 商品图片允许有节制的真实色彩，UI chrome 仍以黑白灰为主。
- Claude 应以源图自然比例、页面结构和业务规则为准，不按非等比拉伸图逐像素还原。

## 38. 点餐紧凑商品区 v20

用户反馈：v19 右侧商品区域不太合适，需要调整。

提示词详见：

- `design/mini-program/prompt-history/v20/GPT_IMAGE_2_PROMPTS.v20.md`

生成文件：

- 紧凑商品区默认态：`design/mini-program/generated/v20/03-ordering-v20-compact-default-source.png`
- 紧凑商品区已加购态：`design/mini-program/generated/v20/03-ordering-v20-compact-active-cart-source.png`
- 两状态等比预览：`design/mini-program/generated/v20/03-ordering-v20-contact-sheet.png`

核心规则：

- 右侧商品区从大卡片调整为紧凑两列商品格/货架。
- 商品图片降低高度，避免大海报感。
- 商品名称、价格、支付提示、步进器需要统一对齐。
- 保留左侧分类栏、右侧一行两个商品、商品步进器、底部购物车栏。
- 线下门店点餐，不出现配送、快递、买家留言或支付宝。

## 39. 订单确认/付款页 v21

继续设计订单确认页，也就是点餐后的付款页面。

提示词详见：

- `design/mini-program/prompt-history/v21/GPT_IMAGE_2_PROMPTS.v21.md`

最终文件：

- 订单确认/付款页：`design/mini-program/final/order-confirmation/04-order-confirmation-final-payment.png`

核心规则：

- 当前门店、桌台/座位必须清楚。
- 展示商品明细、备注、支付方式、金币余额、金额汇总。
- 支付方式只允许微信支付和金币支付。
- 不出现支付宝。
- 不出现配送、快递、收货地址、买家留言等外卖/电商语义。
- 无底部 TabBar。

## 40. 个人中心 v22

继续设计个人中心页面。

提示词详见：

- `design/mini-program/prompt-history/v22/GPT_IMAGE_2_PROMPTS.v22.md`

最终文件：

- 个人中心白底资产版：`design/mini-program/final/member-center/08-member-center-final-profile.png`

核心规则：

- 页面以白底、黑白灰 UI chrome 为主，头像可保留自然色。
- 顶部展示头像、昵称、会员等级、邀请码和复制动作。
- 钱包摘要展示金币、积分、券。
- 订单中心展示点餐订单、活动订单、充值订单、兑换订单。
- 功能列表展示我的券、我的入场券、邀请好友、排行榜、会员权益。
- 工作人员入口仅 staff 身份可见，普通用户不展示。
- 底部菜单固定为：首页、预约、点餐、我的，且“我的”为激活态。
- 开发时底部 Tab 图标必须使用 `design/mini-program/tab-icons/` 定稿 SVG，不照抄 AI 临时图标。
- 不出现签到入口，不出现支付宝，不做红色券市场或花哨钱包卡。
- Claude 应以源图自然比例、页面结构和业务规则为准，不按非等比拉伸图逐像素还原。

## 41. 个人中心子页面套件 v23

继续完成个人中心页面中的所有子链接页面。

提示词详见：

- `design/mini-program/prompt-history/v23/GPT_IMAGE_2_PROMPTS.v23.md`

生成文件：

- 个人资料：`design/mini-program/final/member-subpages/01-profile-edit-v23.png`
- 钱包流水：`design/mini-program/final/member-subpages/02-wallet-ledger-v23.png`
- 订单中心：`design/mini-program/final/member-subpages/03-order-center-v23.png`
- 点餐订单详情：`design/mini-program/final/member-subpages/04-food-order-detail-v23.png`
- 活动订单详情：`design/mini-program/final/member-subpages/05-activity-order-detail-v23.png`
- 充值订单详情：`design/mini-program/final/member-subpages/06-recharge-order-detail-v23.png`
- 兑换订单详情：`design/mini-program/final/member-subpages/07-redemption-order-detail-v23.png`
- 我的券：`design/mini-program/final/member-subpages/08-my-coupons-v23.png`
- 我的入场券：`design/mini-program/final/member-subpages/09-my-tickets-v23.png`
- 邀请好友：`design/mini-program/final/member-subpages/10-invitations-v23.png`
- 排行榜：`design/mini-program/final/member-subpages/11-rankings-v23.png`
- 会员权益：`design/mini-program/final/member-subpages/12-member-benefits-v23.png`
- 工作人员首页：`design/mini-program/final/member-subpages/13-staff-home-v23.png`
- 合成预览：`design/mini-program/final/member-subpages/v23-member-subpages-contact-sheet.png`

核心规则：

- 这批图是个人中心 v22 的子页面最终套件，用户已确认定稿。
- 金币、积分点击进入同一个钱包流水页，通过分段控件切换资产类型。
- 券资产和“我的券”列表入口进入同一个我的券页。
- 四个订单入口进入同一个订单中心页，顶部用分段控件切换类型；每类订单再进入对应详情页。
- 我的入场券独立为票夹页。
- 工作人员入口进入工作人员首页，工作人员只能操作已绑定门店，不允许切换门店。
- AI 图中的二维码/核销码/兑换码只是视觉占位；开发必须由接口返回或本地根据接口数据生成真实码。
- AI 图中的示例昵称、金额、订单号、时间、门店、数量和文案不得照抄，业务字段以接口文档和页面架构为准。
- 不出现签到入口，不出现支付宝，不出现配送、快递、收货地址、买家留言。
- Claude 应以源图自然比例、页面结构和业务规则为准，不按非等比拉伸图逐像素还原。

## 42. 个人中心资产操作弹层 v24

补充充值金币和存取积分交互。

提示词详见：

- `design/mini-program/prompt-history/v24/GPT_IMAGE_2_PROMPTS.v24.md`

生成文件：

- 点击金币后的充值金币弹层：`design/mini-program/final/member-asset-sheets/01-coin-recharge-sheet-v24.png`
- 点击积分后的存取积分弹层：`design/mini-program/final/member-asset-sheets/02-point-saving-sheet-v24.png`
- 合成预览：`design/mini-program/final/member-asset-sheets/v24-asset-action-sheets-contact-sheet.png`

核心规则：

- 这批图是个人中心 v22 的资产点击弹层最终套件，用户已确认定稿。
- 点击个人中心的金币资产时弹出“充值金币”底部弹层。
- 点击个人中心的积分资产时弹出“存取积分”底部弹层。
- 充值金币展示当前金币、充值档位、微信支付和确认充值；充值档位来自 `GET /api/v2/mini/recharge-products`，确认后调用 `POST /api/v2/mini/recharge-orders`。
- 充值金币只允许微信支付，不出现支付宝、金币支付、提现、银行卡或现金到账。
- 存取积分展示当前积分、存入/取出切换、当前门店、积分数量、备注和提交申请；提交后调用 `POST /api/v2/mini/point-savings`。
- 存取积分不允许切换门店、门店下拉或跨店选择；提交后由工作人员审核。
- 弹层中的金额、档位、积分数量、比例和规则文案以后由后台规则配置驱动，不写死为代码常量。
- AI 图中的示例数值、门店、金额和文案不得照抄，业务字段以接口文档和页面架构为准。

## 43. 支付与金币 icon v25

补充支付方式与个人中心资产区 icon 资产。

提示词详见：

- `design/mini-program/prompt-history/v25/GPT_IMAGE_2_PROMPTS.v25.md`

最终文件：

- 金币 icon：`mini-program/miniprogram/assets/payment/coin.png`
- 微信支付官方 icon：`mini-program/miniprogram/assets/payment/wechat-pay.png`

核心规则：

- 金币 icon 由 `gpt-image-2` 生成，提示词必须保留，落地为透明 PNG。
- 微信支付 icon 不使用 AI 生成，必须使用微信官方品牌素材。
- 微信支付官方素材来源为微信设计资源库 `https://wechat.design/tool/brand/` 中的微信支付 Emblem 官方包。
- 仅允许对微信支付官方素材做裁切、透明背景和尺寸适配，不允许重绘、变色或使用近似图标。

## 44. 入场券与优惠券 icon v26

补充“我的入场券”和“我的券”相关 icon 资产。

提示词详见：

- `design/mini-program/prompt-history/v26/GPT_IMAGE_2_PROMPTS.v26.md`

最终文件：

- 入场券 icon：`mini-program/miniprogram/assets/icons/ticket-gpt.png`
- 优惠券 icon：`mini-program/miniprogram/assets/icons/coupon-gpt.png`

核心规则：

- 两枚 icon 均由 `gpt-image-2` 生成，提示词必须保留，落地为透明 PNG。
- 入场券 icon 用黑金入场票、皇冠和二维码细节，强调“票夹/入场”。
- 优惠券 icon 用白色券体和黑色吊牌，强调“券/权益”，避免与入场券混淆。
- 页面空状态、个人中心资产条、优惠券列表入口统一引用本次定稿资产。

## 45. 深色页面无数据空状态插画 v27

根据参考图重新生成活动列表无数据插画。

提示词详见：

- `design/mini-program/prompt-history/v27/GPT_IMAGE_2_PROMPTS.v27.md`

最终文件：

- 无数据插画：`mini-program/miniprogram/assets/empty/activity-empty.png`
- 验收预览：`design/mini-program/final/icon-assets/empty-state-soft-v27-preview.png`

核心规则：

- 这不是功能 icon，而是空状态插画。
- 插画 icon 必须单独提取为透明 PNG 资产，不允许只存在于整屏 mock 设计图中。
- 小程序所有无数据空状态默认使用该插画，包括入场券、优惠券、订单、预约、流水、门店、工作人员空记录等，不再按业务类型单独换图标。
- 视觉参考用户提供的深色页面空状态：低对比、轻纸片、小卷线、大面积留白。
- 空状态图形不要太亮、不要太硬、不要有厚重背景或明显卡片感。
- 文案使用中灰色，整体要安静、舒服，不抢页面标题。

---

## v29 — 会员权益等级 Banner（8 个等级）

会员权益页顶部「当前等级」区域改为等级 banner 背景图，根据会员等级 code 展示对应图。

完整提示词详见：

- `design/mini-program/generated/v29/member-banners-prompts.json`

输出路径：

- `mini-program/miniprogram/assets/benefits/`

8 个等级（code → 文件 → 名称）：

- `normal` → `v1-normal.jpg` → 普通会员
- `bronze` → `v2-bronze.jpg` → 青铜会员
- `silver` → `v3-silver.jpg` → 白银会员
- `gold` → `v4-gold.jpg` → 黄金会员
- `platinum` → `v5-platinum.jpg` → 铂金会员
- `diamond` → `v6-diamond.jpg` → 钻石会员
- `star` → `v7-star.jpg` → 星耀会员
- `master` → `v8-master.jpg` → 大师会员

核心规则：

- 横向 banner，作为 native 文字叠加的背景图，左/中上部留安全空白。
- 黑白灰基底 + 各等级金属材质点缀（青铜/白银/黄金/铂金/钻石等），克制低对比。
- 绝对无文字、无字符、无 logo、无水印。
- benefits.js 中建立 code → banner 映射，后端 code 不认识时回退 `v1-normal.jpg`。

---

## v32 — 会员权益等级 Banner 定稿（8 个等级）

v32 版会员等级 banner 已定稿并落地。会员权益页顶部由「当前等级单张 banner」改为「全部等级会员卡横向 swiper」：初始定位当前等级，可左右滑动预览更高等级卡以引导升级；卡面沿用 native 文字叠加（等级名 / 序号 / 当前标识 / 当前卡成长值进度 / 高等级卡升级引导）。

- 提示词：`design/mini-program/generated/v32/member-banners-prompts.json`
- 生成源图：`output/imagegen/member-banners-v32/final/`
- 最终资源：`mini-program/miniprogram/assets/benefits/`（文件名沿用 `v1-normal.jpg` … `v8-master.jpg`，LEVEL_BANNER 映射不变）

视觉方向承接 v29（黑白灰基底 + 各等级金属材质点缀、克制低对比、无文字/logo/水印），本轮定稿更清晰区分相邻等级；单文件均 < 200K。

---

## v42 — 会员权益等级 Banner VIP 徽章放大定稿（8 个等级）

v42 基于 v41 定稿图做局部微调：仅放大卡面左上角 VIP 等级徽章（约 1.45x–1.65x，移动端更易辨识），其余构图、卡边框、深色背景、右侧徽章、材质与等级配色保持不变。

- 提示词：`design/mini-program/generated/v42/member-banners-prompts.json`
- 生成源图：`output/imagegen/member-banners-v42/final/`
- 最终资源：`mini-program/miniprogram/assets/benefits/`（文件名沿用 `v1-normal.jpg` … `v8-master.jpg`，LEVEL_BANNER 映射不变）

核心规则：

- 仅调整左上角 VIP 徽章尺寸，不改变整体构图、卡面材质、等级配色与右侧徽章。
- 徽章文字固定为 VIP1–VIP8，不含空格、标点或多余字符。
- 保持标准 16:9 banner 比例；不出现卡号、二维码、条码、logo、水印或除 VIP 编号外的文字。
- `output/imagegen/member-banners-v42/final/` 为源图，只读不改；落地到 `assets/benefits` 的正式资源中，超过 200K 的文件（v2-bronze/v3-silver/v6-diamond/v7-star/v8-master）已用 sharp（mozjpeg，逐步降 quality）二次压缩至 <=200K，v1-normal/v4-gold/v5-platinum 原图已满足限制未改动。
