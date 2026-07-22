# gpt-image-2 小程序设计图提示词 v13

## 1. 本次修改来源

用户对活动列表 v12 提出两点修改：

- 页面背景色更喜欢 `design-demo/微信图片_20260714061142_28_188.png` 的深灰到黑色渐变。
- 内容有重复，活动列表应该展示 InwardClub 项目真正需要的信息，而不是重复放大同一活动标题。

继续保留的规则：

- 活动列表页没有底部 TabBar。
- 活动列表页没有顶部导航区域。
- 不显示返回箭头、微信胶囊、状态栏。
- 历史活动保持轻量入口。

## 2. 活动列表深灰渐变业务信息版 v13

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
