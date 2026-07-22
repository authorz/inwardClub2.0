# gpt-image-2 小程序设计图提示词 v7

## 1. 本次修改来源

用户对首页 v6 继续提出两点修改：

- 首页需要显示最近的门店，并提供切换门店和导航按钮。
- 首页信息太多，会员板块下面只放最新活动列表即可，不再展示订单中心、会员积分等额外模块。

## 2. 首页 Banner + 最近门店 + 会员卡 + 最新活动 v7

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Input image role: Use the provided demo image as layout reference: top banner area, floating member card, clean white/gray sections. Do not copy AXIS brand or exact icons.
Primary request: Redesign the InwardClub mini program home page with a simpler structure. Keep the top large banner image area and the floating member card, but add nearest/current store information with switch store and navigation actions. Under the member card, show only a latest activity list. Remove order center, member points, points exchange, and other extra modules from the home page.
Style/medium: high-fidelity mobile UI mockup, premium minimalist WeChat mini program UI, white-first, black typography, light gray page background, restrained shadows
Composition/framing: single vertical phone screen. Top header with Inward Club brand on the left and WeChat capsule on the right. Large white/light banner image placeholder area. Within or just below the lower part of the banner area, show a compact nearest store row: "最近门店", store name, distance/status, a text action "切换门店", and a circular navigation button "导航". A white member card overlaps the banner lower edge, with greeting, nickname, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Below member card, only show a section title "最新活动" and 2-3 simple activity rows with date/status and a "查看全部" text link. Bottom tab bar can be shown as 首页, 点餐, 预约, 活动, 我的.
Color palette: dominant white, black, cool light gray, subtle divider lines, restrained dark red accent only for member number if needed. No colorful banner content.
Text language: Chinese
Required visible text: "Inward Club", "最近门店", "Inward Club 三里屯店", "营业中", "距您 1.2km", "切换门店", "导航", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最新活动", "查看全部"
Required UI: top banner image area, nearest store row with switch store and navigation, overlapping member card, latest activity list only below member card
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no dense information modules, no ordinary store-info-only top layout
Avoid: filling the home page with many feature cards, dark hero poster, colorful restaurant UI, beige/cream palette, purple/blue gradients, heavy shadows, nested cards
```
