# gpt-image-2 小程序设计图提示词 v8

## 1. 本次修改来源

用户对首页 v7 继续提出两点修改：

- 最近门店不要放在 Banner 区域内，应该放在用户/会员卡片下面。
- 最近门店不需要营业状态，样式参考 `01-home-v5.png` 中最近门店的干净横向样式。
- 最新活动需要改为竖版大图横向排列，一行四个。

## 2. 首页 Banner + 会员卡 + 门店横栏 + 横向活动海报 v8

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Input image role: Use the provided demo image as layout reference for top banner and floating member card. Use 01-home-v5 nearest-store row style as reference for the store row: clean horizontal row, store name, distance, switch store, navigation. Do not copy exact previous text mistakes or icons.
Primary request: Redesign the InwardClub mini program home page. Keep the top large banner image area and floating member card. Move nearest/current store information below the member card, not inside the banner. The store row should be clean and compact, similar to the 01-home-v5 nearest store style, with store name, distance, switch store action, and navigation button, but no business/open status. Below that, show only a latest activity section using vertical poster cards arranged horizontally, one row with four portrait cards visible or partially visible. Remove order center, member points, points exchange, and all other extra modules.
Style/medium: high-fidelity mobile UI mockup, premium minimalist WeChat mini program UI, white-first, black typography, light gray page background, subtle floating shadows
Composition/framing: single vertical phone screen. Top header with Inward Club brand and WeChat capsule. Large white/light banner image placeholder area. A white member card overlaps the banner lower edge, with greeting, nickname, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Directly below the member card place a clean nearest store row: label 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航 icon/button. Do not show 营业中. Below store row show 最新活动 with 查看全部, then a horizontal row of four vertical portrait activity poster cards with title/date/status under or over each poster. Bottom tab bar: 首页, 点餐, 预约, 活动, 我的.
Color palette: dominant white, black, cool light gray, subtle divider lines, restrained dark red accent only for member number if needed. Activity posters should remain grayscale/black-white, not colorful.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "黑桃周赛"
Required UI: top banner image area, floating member card, store row below member card with switch and navigation but no business status, latest activity as horizontal row of four vertical poster cards
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no dense information modules
Avoid: putting store row inside banner, showing 营业中, horizontal activity list rows, colorful restaurant UI, dark hero poster, nested cards, heavy shadows
```
