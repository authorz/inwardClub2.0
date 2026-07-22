# gpt-image-2 小程序设计图提示词 v11

## 1. 本次修改来源

用户对首页 v10 提出比例调整：

- Banner 图太矮，需要更高。
- 活动图片太高，需要调矮一点。

本轮继续沿用：

- iPhone 17 原生画布：1206 × 2622 px。
- UI 以黑白灰为主，但 Banner 和活动图片允许真实色彩。
- 底部菜单固定为：首页、预约、点餐、我的，并使用定稿图标语义。

## 2. 首页 Banner 加高 + 活动海报压低 v11

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Canvas: portrait iPhone 17 native resolution, 1206 × 2622 px
Primary request: Refine the v10 InwardClub home page layout. Keep the same premium white-first UI and confirmed structure, but make the top banner image area taller and more immersive. Make the latest activity poster cards shorter than v10, so they do not dominate the page.
Important layout proportions: The banner image should be noticeably taller than v10, approximately 28-32% of the visible screen before the floating member card overlap. The latest activity portrait cards should be shorter and more compact, approximately 65-75% of the previous poster height, while still staying vertical and arranged in one horizontal row of four.
Important color rule: UI chrome, text, buttons, and dividers use black/white/light gray. Banner and activity posters may use tasteful real color; do not force them to grayscale.
Composition/framing: single iPhone 17 full-screen UI. Top safe area and WeChat capsule. Taller refined club/lifestyle banner image. Floating white member card overlaps the lower edge of the banner. Member card includes Hello, Authoz, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Below member card: compact nearest store row with 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航, no 营业中. Below store row: 最新活动 with 查看全部 and one horizontal row of four shorter vertical poster cards. Bottom tab bar exactly four items: 首页, 预约, 点餐, 我的.
Bottom tab icon requirements: 首页 house icon, 预约 calendar icon, 点餐 cloche/serving-dish with knife icon, 我的 user icon. Match finalized black linear SVG style from design/mini-program/tab-icons.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "首页", "预约", "点餐", "我的"
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no 活动 tab in bottom navigation
Avoid: short banner, oversized activity posters, flattened/squashed layout, all grayscale imagery, adding fifth tab, showing 营业中, dense modules
```
