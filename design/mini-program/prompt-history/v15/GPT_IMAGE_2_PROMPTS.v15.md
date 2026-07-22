# gpt-image-2 小程序设计图提示词 v15

## 1. 本次修改来源

活动列表 v14 已定稿，接下来设计活动详情页。

活动详情页需要承接活动列表页故意省略的信息：

- 活动头图。
- 活动时间、门店/地点、距离。
- 活动描述。
- 场次/票档、价格、库存、售卖时间。
- 微信/金币支付方式。
- 底部购票动作。

继续遵守：

- 小程序产品 UI 以黑白灰为主。
- 活动头图允许有节制的真实色彩，不强制灰度。
- 不出现签到。
- 不出现支付宝。
- 活动详情页不能像彩色营销海报，也不要嵌套卡片堆叠。

## 2. 活动详情浅色购票版 v15

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
