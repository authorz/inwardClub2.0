# gpt-image-2 小程序设计图提示词 v16

## 1. 本次修改来源

用户纠正活动详情页交互：

- 活动详情页默认显示活动详情或票类型，不应默认展开完整票档选择和支付。
- 点击购买后才弹出选择票档的弹窗。
- 支付方式可以在票档选择弹窗中选择。
- 因此活动详情页需要两个状态：默认详情态、点击购买后的票档/支付弹窗态。

v15 处理为候选被替代，不作为最终活动详情交互。

## 2. 活动详情默认详情态 v16

```text
Use case: ui-mockup
Asset type: WeChat mini program activity detail default state screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Redesign the InwardClub activity detail default state. It should primarily show event details and a simple ticket type summary. It must NOT show an expanded ticket selector, quantity stepper, or payment method selection by default. The user taps the bottom "立即购票" button to open the ticket selection popup in the next state.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, thin dividers, clean spacing
Composition/framing: single vertical detail screen. Use a clean white/light-gray page background. Top has a large activity hero image with tasteful real-color or lightly desaturated club/table-game lifestyle imagery. Below hero show "黑桃周赛" and status "报名中". Then show full-width detail rows with thin dividers: event time, store/location, distance, sale window. Show a concise "活动介绍" section with short text. Show a "票类型" or "票档概览" section with compact rows only: early bird, presale, double ticket, each row has price range or starting price and remaining summary, but no radio selectors, no quantity steppers, no payment picker. Bottom fixed bar shows starting price or selected default summary and a black primary button "立即购票".
Interaction rule: Default state is for reading details. Ticket selection and payment selection are hidden until user taps "立即购票".
Color rule: UI chrome, text, dividers, rows, and buttons are black/white/light gray. Hero/activity imagery can use restrained real color, not forced grayscale. Avoid colorful marketing poster feeling.
Text language: Chinese
Required visible text: "活动详情", "黑桃周赛", "报名中", "本周六 20:00", "Inward Club 三里屯店", "距您 1.2km", "售卖至 06-14 18:00", "活动介绍", "票类型", "早鸟票 ¥128 起", "预售票 ¥168", "双人票 ¥308", "剩余票量", "限购 4 张", "立即购票"
Required UI: large hero image, title and status, time/store/distance rows, activity description, compact ticket type summary without selection controls, fixed bottom purchase bar
Constraints: no expanded ticket selector by default, no quantity stepper by default, no payment method picker by default, no sign-in/check-in, no Alipay, no bottom tabbar, no WeChat capsule, no status bar, no nested cards, no heavy shadows
Avoid: full ticket purchase form on the default page, visible WeChat/coin payment selection on default page, ecommerce delivery language, mostly black page, noisy decorations, dense unreadable text
```

## 3. 活动详情票档/支付弹窗态 v16

```text
Use case: ui-mockup
Asset type: WeChat mini program activity detail purchase popup state screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Design the second state of the InwardClub activity detail page after the user taps "立即购票". The background should be the activity detail page dimmed or subtly visible, with a bottom sheet popup for choosing ticket tier, quantity, and payment method.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white bottom sheet over softly dimmed detail page, black typography, thin dividers, quiet luxury
Composition/framing: single vertical screen showing the activity detail page in the background with a translucent dark overlay. From the bottom, show a clean white rounded-top bottom sheet occupying the lower 55-65% of the screen. Bottom sheet title: "选择票档". Include close icon. Ticket tier rows: "早鸟票", "预售票", "双人票"; each row shows price, remaining stock, sale time, and selected state. Include one quantity stepper for the selected ticket. Include "支付方式" with two selectable rows or segmented controls: "微信支付" and "金币支付"; no Alipay. Include purchase limit hint "限购 4 张". At bottom of the sheet show total price and a black primary button "确认购买".
Interaction rule: This state appears only after tapping the default page purchase button. Ticket selection, quantity selection, and payment method selection happen inside this popup.
Color rule: UI chrome, text, dividers, rows, and buttons are black/white/light gray. Payment method selected state can use black outline/check mark; avoid bright colors. Background overlay can be black with 35-45% opacity.
Text language: Chinese
Required visible text: "选择票档", "早鸟票", "预售票", "双人票", "剩余", "售卖至", "数量", "支付方式", "微信支付", "金币支付", "限购 4 张", "合计 ¥128", "确认购买"
Required UI: dimmed detail background, bottom sheet, close icon, ticket tier selector, quantity stepper, payment method selector, purchase limit hint, total price, confirm button
Constraints: no Alipay, no sign-in/check-in, no bottom tabbar, no full-page navigation transition, no nested cards, no heavy shadows, no ecommerce delivery language
Avoid: payment selection on the default page, large historical activity section, red coupon style, colorful payment badges, noisy decorations, unreadable dense rows
```

## 4. 活动详情票档/支付弹窗态 v16 重跑修正

第一次弹窗态生成时，背景被模型换成了英文活动和不同城市信息。为避免 Claude 误用，重跑弹窗态并强化以下要求：

- 背景必须仍然是“黑桃周赛”详情页。
- 不能出现英文活动名。
- 不能出现无关城市/活动。
- 不显示状态栏、微信胶囊或底部 TabBar。
- 只有选中的票档显示数量步进器。

```text
Use case: ui-mockup
Asset type: WeChat mini program activity detail purchase popup state screen
Canvas: portrait iPhone 17 style tall mobile screen, 1206 × 2622 px target
Primary request: Design the second state of the InwardClub activity detail page after the user taps "立即购票". It must be the SAME activity as the default page: "黑桃周赛" at "Inward Club 三里屯店". The background should be the black peach weekly event detail page dimmed and subtly visible, with a bottom sheet popup for choosing ticket tier, quantity, and payment method.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white bottom sheet over softly dimmed detail page, black typography, thin dividers, quiet luxury
Composition/framing: single vertical screen, no status bar, no WeChat capsule, no bottom tabbar. Background is the activity detail page for "黑桃周赛" with the same club/table-game hero image, title, time, store, and activity detail content, covered by a translucent black overlay. From the bottom, show a clean white rounded-top bottom sheet occupying the lower 58-65% of the screen. Bottom sheet title: "选择票档". Include a close icon. Ticket tier rows: "早鸟票", "预售票", "双人票"; each row shows price, remaining stock, sale time, and selected state. Only the currently selected ticket row shows the quantity stepper; unselected rows do not show quantity steppers. Include "支付方式" with two selectable rows or segmented controls: "微信支付" and "金币支付"; no Alipay. Include purchase limit hint "限购 4 张". At bottom of the sheet show total price and a black primary button "确认购买".
Interaction rule: This state appears only after tapping the default page purchase button. Ticket selection, selected ticket quantity, and payment method selection happen inside this popup.
Color rule: UI chrome, text, dividers, rows, and buttons are black/white/light gray. Payment method selected state uses black outline/check mark; avoid bright colors. Background overlay is black with 35-45% opacity.
Text language: Chinese
Required visible text: "黑桃周赛", "选择票档", "早鸟票", "预售票", "双人票", "剩余", "售卖至", "数量", "支付方式", "微信支付", "金币支付", "限购 4 张", "合计 ¥128", "确认购买"
Required UI: dimmed black peach weekly event detail background, bottom sheet, close icon, ticket tier selector, one quantity stepper only on selected ticket, payment method selector, purchase limit hint, total price, confirm button
Constraints: no Alipay, no sign-in/check-in, no status bar, no WeChat capsule, no bottom tabbar, no English event title, no different event background, no full-page navigation transition, no nested cards, no heavy shadows, no ecommerce delivery language
Avoid: payment selection on the default page, unrelated event names, Mindscape, Shanghai event, large historical activity section, red coupon style, colorful payment badges, noisy decorations, unreadable dense rows
```
