# gpt-image-2 小程序设计图提示词 v19

## 1. 本次修改来源

继续设计点餐页面，并遵守之前已确认的点餐要求：

- 左侧为分类栏。
- 右侧商品一行展示两个商品。
- 商品可以选择数量，每个商品有步进器。
- 底部有购物车/结算动作。
- 小程序底部菜单固定为：首页、预约、点餐、我的。
- 开发时底部菜单图标必须使用 `design/mini-program/tab-icons/` 定稿 SVG。
- UI chrome 以黑白灰为主，商品图片可以使用有节制的真实色彩，不强制灰度。
- 点餐是线下门店点餐，不是外卖，不出现配送/快递/买家留言。
- 不出现支付宝，只允许微信和金币。
- 交付图不做非等比拉伸，Claude 以源图自然比例、结构和业务规则为准。

## 2. 点餐商品列表默认态 v19

```text
Use case: ui-mockup
Asset type: WeChat mini program in-store ordering product list default state
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Redesign the InwardClub ordering page for in-store ordering. The page must use a left category rail and a right two-column product grid. Each product has image, name, price, stock/payment hint, and a quantity stepper. Default state can show all quantities as 0 and an empty cart bar.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained dividers, no food-delivery promotion style
Composition/framing: single vertical ordering screen. Top title "点餐". Under title show a compact store/table context row: "Inward Club 三里屯店" and "1号桌 · 5号座位". Main content is split: left fixed vertical category rail with categories "酒水", "小吃", "饮品", "套餐"; active category has black vertical indicator and stronger text. Right side product area is a two-column grid, exactly two products per row. Product cells are clean white/light-gray blocks with subtle dividers, not heavy shadow cards. Each product includes a tasteful real-color but restrained product image, product name, price, payment hints "微信 / 金币", stock/sold-out hint when needed, and a stepper with minus, quantity, plus. Bottom fixed cart bar shows cart icon, selected count 0, total "¥0.00", black primary button "去结算". Bottom tabbar has exactly four items: 首页, 预约, 点餐, 我的, with 点餐 active.
Product examples: "经典啤酒", "苏打水", "花生米", "香脆鸡块", "柠檬茶", "美式咖啡".
Color rule: UI chrome, text, dividers, cart bar, buttons are black/white/light gray. Product images may use restrained real color, not forced grayscale. Avoid red/yellow food delivery palette.
Text language: Chinese
Required visible text: "点餐", "Inward Club 三里屯店", "1号桌 · 5号座位", "酒水", "小吃", "饮品", "套餐", "经典啤酒", "苏打水", "花生米", "香脆鸡块", "微信", "金币", "购物车", "¥0.00", "去结算", "首页", "预约", "点餐", "我的"
Required UI: left category rail, right two-column product grid, product image, product price, payment hint, quantity stepper on every product, fixed bottom cart bar, bottom tabbar
Constraints: exactly two products per row, no Alipay, no sign-in/check-in, no delivery/express/buyer-message language, no fake extra tab item, no activity tab, no non-proportional stretching
Avoid: one-column product list, colorful food delivery app style, red discount tags, heavy nested cards, oversized product cards, tiny unreadable text, all-grayscale product images
```

## 3. 点餐已加购状态 v19

```text
Use case: ui-mockup
Asset type: WeChat mini program in-store ordering product list with cart items state
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Design the ordering page after the user has added products. Keep the same left category rail and right two-column product grid. Show selected quantities in product steppers and an active bottom cart bar with total amount and checkout action.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, restrained dividers, no food-delivery promotion style
Composition/framing: single vertical ordering screen. Same structure as default state: top title "点餐", store/table context row, left category rail, right two-column product grid, bottom tabbar. In the product grid, "经典啤酒" quantity is 2, "花生米" quantity is 1, "苏打水" quantity is 1. The stepper must clearly show minus, current quantity, plus. Bottom fixed cart bar is active: cart icon, "已选 4 件", total "¥56.00", black primary button "去结算". Include a subtle hint "微信 / 金币可用". Do not show full cart drawer yet; this is only the product browsing state with active cart summary.
Color rule: UI chrome, text, dividers, cart bar, buttons are black/white/light gray. Product images may use restrained real color, not forced grayscale. Active quantities use black text/border, not bright colored badges.
Text language: Chinese
Required visible text: "点餐", "Inward Club 三里屯店", "1号桌 · 5号座位", "经典啤酒", "苏打水", "花生米", "香脆鸡块", "微信", "金币", "已选 4 件", "¥56.00", "去结算", "首页", "预约", "点餐", "我的"
Required UI: selected product quantities in steppers, active cart bar, two-column grid, left category rail, bottom tabbar
Constraints: exactly two products per row, no Alipay, no sign-in/check-in, no delivery/express/buyer-message language, no full cart drawer, no fake extra tab item, no activity tab, no non-proportional stretching
Avoid: promotional delivery homepage, coupon market style, colorful badges, dense cart overlay, one-column list, non-proportional stretched UI
```
