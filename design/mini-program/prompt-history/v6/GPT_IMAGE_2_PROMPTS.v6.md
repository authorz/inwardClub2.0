# gpt-image-2 小程序设计图提示词 v6

## 1. 本次修改来源

用户指出 v5 首页仍不符合参考图：

- 必须参考 `design-demo/微信图片_20260714061140_27_188.png` 的结构。
- 顶部大面积空白不是普通留白，而是 Banner 图区域。
- 首页上方应该是品牌/小程序顶部 + 大 Banner 图位。
- 会员信息卡片从 Banner 下缘叠上来。
- 整体仍以白色、黑色和浅灰为主。

## 2. 首页 Banner + 会员卡叠层版 v6

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
