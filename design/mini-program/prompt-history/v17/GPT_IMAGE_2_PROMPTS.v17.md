# gpt-image-2 小程序设计图提示词 v17

## 1. 本次修改来源

用户继续设计预约界面，并新增要求：

- 桌子必须是德州扑克桌面。
- 一张桌子有 9 个座位。
- 座位有预约状态和空闲状态。
- 页面需要有一个“排队等候”按钮。

继续遵守：

- 底部菜单固定为：首页、预约、点餐、我的。
- 底部菜单图标开发时必须使用 `design/mini-program/tab-icons/` 定稿 SVG。
- 预约页可以用轻量桌子模块承载复杂座位布局，但不做重阴影卡片堆叠。
- 点击空闲座位后弹出预约确认弹层。

## 2. 预约德州扑克桌默认态 v17

```text
Use case: ui-mockup
Asset type: WeChat mini program reservation table selection default state
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Redesign the InwardClub reservation page. The main table must visually be a Texas Hold'em poker table, not a generic board-game table. Each table has exactly 9 seats arranged around the poker table. Seats must clearly show two states: 空闲 and 已预约. The page must include a clear "排队等候" button for waitlist when seats are unavailable or the user does not want a specific seat.
Style/medium: high-fidelity mobile UI mockup, premium mobile club app, white-first interface, black typography, restrained gray dividers, realistic but clean poker-table layout, app-native iOS-like hierarchy
Composition/framing: single vertical reservation screen with clean top navigation title "预约". Show current store row "Inward Club 三里屯店" with distance. Show a compact date/time selector row. Show status legend: 空闲, 已预约, 已选择. Show one main full-width table section for "1号桌" with metadata: "德州扑克桌", "座位 5/9", "更新时间 06:16". The central table graphic must be a Texas Hold'em oval poker felt table: oval/rounded rectangle, dark charcoal or very deep green felt, subtle spade mark, chip/card details very restrained. Place EXACTLY NINE separate seat controls around it, numbered 1 through 9. Seat states: seats 1, 3, 6, 8 are "已预约" disabled gray; seats 2, 4, 5, 7, 9 are "空闲" white/black outline. Do not draw more or fewer than nine seats. Below the table section, show two actions: black primary "排队等候" and secondary text/link "我的预约". Bottom tabbar must show exactly four items: 首页, 预约, 点餐, 我的, with 预约 active.
Interaction rule: Default state only browses table/seats. Tapping an 空闲 seat opens the reservation confirmation bottom sheet in the next state.
Color rule: UI chrome, text, dividers, buttons are black/white/light gray. Poker table surface may use restrained dark charcoal or deep green felt as functional table imagery. Avoid colorful casino styling.
Text language: Chinese
Required visible text: "预约", "Inward Club 三里屯店", "今天 20:00", "空闲", "已预约", "已选择", "1号桌", "德州扑克桌", "座位 5/9", "更新时间 06:16", "1号座位", "2号座位", "3号座位", "4号座位", "5号座位", "6号座位", "7号座位", "8号座位", "9号座位", "排队等候", "我的预约", "首页", "预约", "点餐", "我的"
Required UI: Texas Hold'em poker table graphic, exactly 9 numbered seat controls, clear available/reserved state, waitlist button, bottom tabbar with reservation active
Constraints: exactly 9 seats, no 8 seats, no 10 seats, no sign-in/check-in, no Alipay, no colorful casino poster, no nested cards, no heavy shadows, no fake extra tab item, no activity tab
Avoid: generic restaurant table, abstract floor map, dense multiple-table map, red casino style, neon gambling UI, tiny unreadable seat labels, confusing state colors
```

## 3. 预约德州扑克桌确认弹窗态 v17

```text
Use case: ui-mockup
Asset type: WeChat mini program reservation seat confirmation bottom sheet state
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Design the second state of the InwardClub reservation page after the user taps an available seat on the Texas Hold'em table. The background should be the same reservation table page dimmed. A bottom sheet asks the user to confirm the reservation.
Style/medium: high-fidelity mobile UI mockup, premium mobile club app, white bottom sheet, black typography, restrained gray dividers, app-native iOS-like hierarchy
Composition/framing: single vertical reservation screen. The background remains the same "1号桌" Texas Hold'em poker table with exactly 9 seats, dimmed by a subtle overlay. Seat 5 or 7 should be selected with clear selected state. From the bottom, show a white rounded-top confirmation sheet. Sheet title "确认预约". Show selected info: "1号桌 · 5号座位", "今天 20:00", "Inward Club 三里屯店". Show a short rule hint "请按预约时间到店，超时将释放座位". Include two actions: secondary "取消" and black primary "确认预约". Include an understated "改为排队等候" text action inside the sheet.
Interaction rule: This state only appears after tapping an 空闲 seat. Reserved seats cannot be selected. Waitlist is available as an alternative action.
Color rule: UI chrome, text, dividers, buttons are black/white/light gray. Selected seat uses black outline/fill or dark accent; reserved seats are gray disabled. Poker table surface remains restrained dark charcoal/deep green.
Text language: Chinese
Required visible text: "确认预约", "1号桌 · 5号座位", "今天 20:00", "Inward Club 三里屯店", "请按预约时间到店", "改为排队等候", "取消", "确认预约"
Required UI: dimmed reservation page background, exactly 9 seats still visible, one selected available seat, bottom confirmation sheet, cancel and confirm buttons, waitlist alternative
Constraints: exactly 9 seats, no Alipay, no sign-in/check-in, no extra tab item, no payment method, no delivery/order language
Avoid: generic restaurant reservation, full-screen modal, colorful casino style, dense legal text, hidden selected seat
```
