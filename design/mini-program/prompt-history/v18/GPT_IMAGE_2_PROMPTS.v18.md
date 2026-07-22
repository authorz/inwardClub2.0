# gpt-image-2 小程序设计图提示词 v18

## 1. 本次修改来源

用户指出预约页还需要考虑多张桌子的情况，并提醒此前若把生成图强行拉伸到 iPhone 17 比例，会让 Claude 误解真实 UI 比例。

本轮调整：

- 预约页默认状态必须体现多张桌子，而不是永远只有一张桌。
- 第一张/当前桌可以展开展示完整德州扑克桌和 9 个座位。
- 其他桌以紧凑桌台摘要展示，不必每张都完整展开。
- 仍保留“排队等候”按钮。
- 交付图不得做非等比拉伸；Claude 还原时以源图比例、页面结构和文档规则为准，不按被拉伸图逐像素照抄。

## 2. 预约多桌默认态 v18

```text
Use case: ui-mockup
Asset type: WeChat mini program reservation multi-table default state
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Redesign the InwardClub reservation page to support multiple tables. The page should show a realistic multi-table reservation workflow: one expanded Texas Hold'em table with exactly 9 seats, plus other tables shown as compact table summaries below. The user can choose a seat on the expanded table or switch to another table. Include a clear "排队等候" button.
Style/medium: high-fidelity mobile UI mockup, premium club app, white-first interface, black typography, restrained gray dividers, realistic but clean Texas Hold'em table graphic, app-native iOS-like hierarchy
Composition/framing: single vertical reservation screen. Top title "预约". Show current store row "Inward Club 三里屯店" and date/time row "今天 20:00". Show status legend "空闲 / 已预约 / 已选择". Main section "可预约桌台". The first table section is expanded: "1号桌", "德州扑克桌", "座位 5/9", "更新时间 06:16". Inside it, draw a Texas Hold'em oval poker table with EXACTLY NINE separate numbered seat controls around it. States: 1, 3, 6, 8 已预约; 2, 4, 5, 7, 9 空闲. Below the expanded table, show compact rows for "2号桌" and "3号桌", each with mini 9-dot seat state preview, seat count such as "座位 2/9" and "座位 8/9", and a right-side action "查看座位". At the bottom show a prominent black "排队等候" button and a light "我的预约" entry. Bottom tabbar has exactly four items: 首页, 预约, 点餐, 我的, with 预约 active.
Interaction rule: Multiple tables are shown in one scrollable reservation page. Only one table is expanded at a time; tapping compact table summary expands that table. Tapping an 空闲 seat opens reservation confirmation.
Color rule: UI chrome, text, dividers, buttons are black/white/light gray. The poker table surface may use restrained dark charcoal or deep green felt as functional imagery. Avoid colorful casino styling.
Text language: Chinese
Required visible text: "预约", "Inward Club 三里屯店", "今天 20:00", "空闲", "已预约", "已选择", "可预约桌台", "1号桌", "德州扑克桌", "座位 5/9", "更新时间 06:16", "1号座位", "2号座位", "3号座位", "4号座位", "5号座位", "6号座位", "7号座位", "8号座位", "9号座位", "2号桌", "3号桌", "查看座位", "排队等候", "我的预约", "首页", "预约", "点餐", "我的"
Required UI: multiple table list, one expanded Texas Hold'em poker table, exactly 9 numbered seats on the expanded table, compact summaries for other tables, state legend, waitlist button, bottom tabbar
Constraints: do not show only one table; exactly 9 seats on expanded table; no 8 seats; no 10 seats; no sign-in/check-in; no Alipay; no colorful casino poster; no fake extra tab item; no activity tab
Avoid: non-proportional stretched UI, generic restaurant table, abstract floor map, dense casino floor map, red casino style, neon gambling UI, tiny unreadable labels
```

## 3. 预约多桌确认弹窗态 v18

```text
Use case: ui-mockup
Asset type: WeChat mini program reservation multi-table confirmation sheet state
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Design the confirmation state for the multi-table InwardClub reservation page after the user taps an available seat on the expanded Texas Hold'em table. The background should be the multi-table reservation page dimmed. A bottom sheet confirms the selected table and seat.
Style/medium: high-fidelity mobile UI mockup, premium club app, white bottom sheet, black typography, restrained gray dividers, app-native iOS-like hierarchy
Composition/framing: single vertical screen. Background is the multi-table reservation page with one expanded 1号桌 Texas Hold'em table and compact 2号桌/3号桌 rows, covered by a subtle dark overlay. Seat 5 is selected. Bottom sheet title "确认预约". Show selected info: "1号桌 · 5号座位", "今天 20:00", "Inward Club 三里屯店". Show short rule hint "请按预约时间到店，超时将释放座位". Include secondary action "取消", black primary "确认预约", and text action "改为排队等候".
Interaction rule: Confirmation appears only after tapping an 空闲 seat. Reserved seats cannot be selected. Waitlist remains available as an alternative action.
Color rule: UI chrome, text, dividers, buttons are black/white/light gray. Selected seat uses black outline/fill or dark accent; reserved seats are gray disabled. Poker table surface remains restrained dark charcoal/deep green.
Text language: Chinese
Required visible text: "确认预约", "1号桌 · 5号座位", "今天 20:00", "Inward Club 三里屯店", "请按预约时间到店", "改为排队等候", "取消", "确认预约", "2号桌", "3号桌"
Required UI: dimmed multi-table reservation page background, exactly 9 seats visible on expanded table, one selected available seat, bottom confirmation sheet, cancel and confirm buttons, waitlist alternative
Constraints: no Alipay, no sign-in/check-in, no extra tab item, no payment method, no delivery/order language, no non-proportional stretching
Avoid: single-table-only page, generic restaurant reservation, full-screen modal, colorful casino style, dense legal text, hidden selected seat
```
