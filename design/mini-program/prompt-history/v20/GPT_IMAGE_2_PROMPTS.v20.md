# gpt-image-2 小程序设计图提示词 v20

## 1. 本次修改来源

用户反馈：

- v19 点餐页右侧商品区域不太合适，需要调整。

本轮调整重点：

- 保留左侧分类栏、右侧一行两个商品、商品步进器、底部购物车栏。
- 调整右侧商品区比例：商品图不要过高，商品单元更紧凑。
- 商品区不要像大卡片堆叠，改成轻量货架/商品格。
- 商品信息、价格、支付提示和步进器需要统一对齐，方便扫描和连续加购。
- 商品图片允许有节制真实色彩，但 UI chrome 仍以黑白灰为主。
- 交付图不做非等比拉伸，Claude 以源图自然比例、结构和业务规则为准。

## 2. 点餐紧凑商品区默认态 v20

```text
Use case: ui-mockup
Asset type: WeChat mini program in-store ordering compact product grid default state
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Refine the InwardClub ordering page from v19. The right product area was too card-like and oversized. Keep the left category rail and right two-column product grid, but make the right product area more compact, structured, and shelf-like. Product images should be shorter, product cells should have light dividers instead of heavy card feeling, and all product info/steppers should align consistently.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained dividers, efficient in-store ordering UI
Composition/framing: single vertical ordering screen. Top title "点餐". Under title show compact store/table context row "Inward Club 三里屯店" and "1号桌 · 5号座位". Main content split: left fixed category rail with "酒水", "小吃", "饮品", "套餐"; active category has black vertical indicator and stronger text. Right side is a compact two-column product shelf. Exactly two products per row. Each product cell uses a short fixed-height product image at top, not a tall poster; below image show name, price, small "微信 / 金币" payment hint, stock hint if needed, and a compact stepper aligned to the lower-right. Use thin dividers and very light borders, no large shadow cards. Show 8 products within the scrollable right area if possible. Bottom fixed cart bar shows empty cart: "购物车", count 0, "¥0.00", black button "去结算". Bottom tabbar exactly four items: 首页, 预约, 点餐, 我的, with 点餐 active.
Product examples: "经典啤酒", "苏打水", "花生米", "香脆鸡块", "柠檬茶", "美式咖啡", "薯条", "果盘".
Color rule: UI chrome, text, dividers, cart bar, and buttons are black/white/light gray. Product photos may use restrained real color, not forced grayscale. Avoid food delivery red/yellow promotion palette.
Text language: Chinese
Required visible text: "点餐", "Inward Club 三里屯店", "1号桌 · 5号座位", "酒水", "小吃", "饮品", "套餐", "经典啤酒", "苏打水", "花生米", "香脆鸡块", "微信", "金币", "购物车", "¥0.00", "去结算", "首页", "预约", "点餐", "我的"
Required UI: left category rail, compact right two-column product shelf, exactly two products per row, shorter product images, aligned product names/prices/payment hints/steppers, fixed bottom cart bar, bottom tabbar
Constraints: right product area must not look like oversized cards; exactly two products per row; no Alipay; no sign-in/check-in; no delivery/express/buyer-message language; no fake extra tab item; no activity tab; no non-proportional stretching
Avoid: huge product cards, tall poster images, heavy shadows, nested cards, one-column list, colorful food delivery app style, red discount tags, tiny unreadable labels, all-grayscale product images
```

## 3. 点餐紧凑商品区已加购态 v20

```text
Use case: ui-mockup
Asset type: WeChat mini program in-store ordering compact product grid active cart state
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Refine the active-cart ordering page from v19. Keep the right product area compact and shelf-like, not oversized cards. Show selected quantities in aligned steppers and an active bottom cart bar.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained dividers, efficient in-store ordering UI
Composition/framing: same layout as default state. Top title "点餐", compact store/table context row, left category rail, right compact two-column product shelf. Product images are short fixed-height thumbnails. Selected quantities: "经典啤酒" quantity 2, "苏打水" quantity 1, "花生米" quantity 1. Stepper is compact and aligned consistently at the bottom-right of each product cell. Bottom fixed cart bar active: cart icon, "已选 4 件", "微信 / 金币可用", total "¥56.00", black button "去结算". Do not show a full cart drawer.
Color rule: UI chrome, text, dividers, cart bar, and buttons are black/white/light gray. Product photos may use restrained real color. Active quantities use black text/border, not bright badges.
Text language: Chinese
Required visible text: "点餐", "Inward Club 三里屯店", "1号桌 · 5号座位", "经典啤酒", "苏打水", "花生米", "香脆鸡块", "微信", "金币", "已选 4 件", "¥56.00", "去结算", "首页", "预约", "点餐", "我的"
Required UI: compact two-column product shelf, selected quantities in steppers, active cart bar, left category rail, bottom tabbar
Constraints: right product area must not look like oversized cards; exactly two products per row; no Alipay; no sign-in/check-in; no delivery/express/buyer-message language; no full cart drawer; no fake extra tab item; no activity tab; no non-proportional stretching
Avoid: huge product cards, tall product posters, dense cart overlay, colorful badges, promotional delivery homepage, one-column list, non-proportional stretched UI
```
