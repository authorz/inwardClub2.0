# gpt-image-2 小程序设计图提示词 v4

## 1. 本次修改来源

用户对活动列表提出新的视觉和交互要求：

- 活动列表的大图轮播需要有相册式堆叠感觉。
- 主活动图像居中突出，前后活动像相册一样在左右或后方露出层叠边缘。
- 可以左滑右滑切换上一个/下一个活动。
- 历史活动入口不需要大面积列表，只保留简单文字入口。

## 2. 活动列表相册式堆叠轮播 v4

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
