# gpt-image-2 小程序设计图提示词 v10

## 1. 本次修改来源

用户指出首页 v9 的问题：

- 画面太瘪，需要按 iPhone 17 分辨率重新设计。
- 产品配色以黑白为主，指的是 UI 主色，不是所有 Banner 和活动图片都必须变成灰度图。

本轮设备基准：

- iPhone 17 / iPhone 17 Pro 原生分辨率：1206 × 2622 px。
- 设计画布按竖屏 1206 × 2622 px。

## 2. 首页 iPhone 17 原生画布 v10

```text
Use case: ui-mockup
Asset type: WeChat mini program home screen
Canvas: portrait iPhone 17 native resolution, 1206 × 2622 px, tall screen, not compressed or squashed
Primary request: Redesign the InwardClub home page for iPhone 17 native resolution. Make the layout taller, more breathable, and less flattened than the previous version. Keep the confirmed structure: top banner image area, floating member card, nearest store row below the member card, latest activity section with four vertical poster cards, and fixed four-item bottom tab bar.
Important color rule: The product UI palette is black/white/light gray, but content images such as the banner and activity posters may use tasteful real color. Do not convert all images to grayscale. Use restrained full-color photography or poster art inside image areas while keeping the interface chrome, text, buttons, and dividers black/white/gray.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, cool light gray surfaces, understated luxury, strong spacing
Composition/framing: single full-screen mobile UI. Top safe area and WeChat capsule. Large banner image area with a refined real-color or lightly desaturated club/lifestyle image. A floating white member card overlaps the lower edge of the banner with Hello, Authoz, membership level black capsule, member number, and three shortcuts: 我的入场券, 我的优惠券, 活动列表. Below the member card, show a clean nearest store row: 最近门店, Inward Club 三里屯店, 距您 1.2km, 切换门店, 导航. Do not show 营业中 or business status. Below store row show 最新活动 with 查看全部, then one horizontal row of four vertical portrait activity poster cards; posters may contain tasteful color images. Bottom tab bar exactly four items in this order: 首页, 预约, 点餐, 我的.
Bottom tab icon requirements: 首页 house icon, 预约 calendar icon, 点餐 cloche/serving-dish with knife icon, 我的 user icon. Match the finalized black linear SVG style from design/mini-program/tab-icons. Active 首页 icon black; inactive icons gray. Do not add 活动 as a bottom tab.
Color palette: UI uses white, black, cool gray, subtle dividers. Image content may include restrained color accents, natural warm/cool tones, and real poster photography. Avoid making image content all grayscale.
Text language: Chinese
Required visible text: "Inward Club", "Hello", "Authoz", "普卡", "NO.6272 0020 035", "我的入场券", "我的优惠券", "活动列表", "最近门店", "Inward Club 三里屯店", "距您 1.2km", "切换门店", "导航", "最新活动", "查看全部", "黑桃周赛", "首页", "预约", "点餐", "我的"
Required UI: iPhone 17 tall canvas, top banner image area, floating member card, store row below member card with no business status, latest activity as one row of four vertical poster cards with tasteful color allowed, fixed four-item bottom tab using finalized icon semantics
Constraints: no sign-in/check-in, no Alipay, no order center block, no member points block, no points exchange button, no business status in store row, no 活动 tab in bottom navigation
Avoid: flattened/squashed layout, all grayscale imagery, adding fifth tab, temporary AI tab concepts, putting store row inside banner, showing 营业中, horizontal activity list rows, colorful restaurant style, dark hero poster, dense modules, nested cards
```
