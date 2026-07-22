# gpt-image-2 小程序设计图提示词 v14

## 1. 本次修改来源

用户纠正 v13：

- 应该在 v12 的版本中修改。
- 只是去掉重复内容，不需要展示特别多内容。
- 活动详情页会展示详细信息，列表页只需要必要摘要。

继续保留：

- 活动列表页没有顶部导航区域。
- 活动列表页没有底部 TabBar。
- v12 的浅色背景和相册式堆叠轮播方向。
- 历史活动是轻量文字入口。

## 2. 活动列表 v12 简化内容版 v14

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
