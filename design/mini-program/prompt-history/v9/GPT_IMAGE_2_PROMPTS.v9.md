# gpt-image-2 小程序设计图提示词 v9

## 1. 本次修改来源

用户确认底部菜单图标后，要求重新设计首页。

本次首页必须保留并强化以下已确认规则：

- 底部菜单固定为：首页、预约、点餐、我的。
- 底部菜单图标必须使用 `design/mini-program/tab-icons/` 中定稿的 SVG 风格与语义。
- 首页结构继续参考 demo：顶部 Banner 图位、会员卡叠层。
- 最近门店放在会员卡下面，不显示营业状态。
- 最新活动使用竖版大图横向排列，一行四个。

## 2. 首页定稿方向 v9

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
