# gpt-image-2 小程序设计图提示词 v12

## 1. 本次修改来源

用户继续设计活动列表页，并指出上次 v4 版本太黑。

本次要求：

- 保留活动列表的相册式堆叠轮播。
- 整体背景改成白色/浅灰为主，不要大面积黑底。
- 产品 UI 仍以黑白灰为主，但活动海报图片允许真实色彩，不强制灰度。
- 历史活动仍保持轻量文字入口。
- 活动列表页没有底部 TabBar，也没有顶部导航区域。

## 2. 活动列表浅色相册轮播 v12

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
