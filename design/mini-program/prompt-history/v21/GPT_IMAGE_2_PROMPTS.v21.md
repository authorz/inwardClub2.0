# gpt-image-2 小程序设计图提示词 v21

## 1. 本次修改来源

继续设计订单确认页，也就是点餐后的付款页面。

继续遵守：

- 这是线下门店点餐付款页，不是外卖。
- 不出现配送、快递、收货地址、买家留言。
- 不出现支付宝。
- 小程序支付方式只允许微信支付和金币支付。
- 需要展示当前门店、桌号/座位、商品明细、备注、支付方式、金币余额、合计和确认支付。
- UI chrome 以黑白灰为主，商品小图可以使用有节制真实色彩。
- 交付图不做非等比拉伸，Claude 以源图自然比例、结构和业务规则为准。

## 2. 订单确认/付款页 v21

```text
Use case: ui-mockup
Asset type: WeChat mini program in-store food order confirmation and payment screen
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Design the InwardClub in-store order confirmation/payment page after the user taps checkout from the ordering page. This is not delivery. Show store/table context, selected product details, optional remark, payment method selection for WeChat and coins, coin balance hint, total amount, and fixed bottom confirm payment action.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, thin dividers, quiet checkout/payment flow
Composition/framing: single vertical checkout screen. Top navigation title "确认订单" with back arrow. No bottom tabbar on this checkout page. First section is store/table context with clean rows: "当前门店 Inward Club 三里屯店" and "桌台 1号桌 · 5号座位". Next section title "商品明细", with compact item rows using small product thumbnails, item name, quantity, unit/subtotal price: "经典啤酒 x1 ¥28.00", "苏打水 x1 ¥12.00", "花生米 x1 ¥16.00". Add a "备注" row with placeholder "口味、冰块等需求". Add "支付方式" section with two selectable rows: "微信支付" selected by default, "金币支付" optional. Show "金币余额 860 金币，可抵 ¥8.60" as a secondary info row. Add price summary rows: "商品金额 ¥56.00", "金币抵扣 ¥0.00", "合计 ¥56.00". Fixed bottom payment bar: total amount on left and black primary button "确认支付".
Payment rule: Only WeChat Pay and coin pay are available. No Alipay. Coin pay can be selected only when allowed by order rules and balance is enough; show balance hint.
Color rule: UI chrome, text, dividers, payment rows, and buttons are black/white/light gray. Product thumbnails may use restrained real color. Payment selected state uses black check or black outline, not bright color.
Text language: Chinese
Required visible text: "确认订单", "当前门店", "Inward Club 三里屯店", "桌台", "1号桌 · 5号座位", "商品明细", "经典啤酒", "苏打水", "花生米", "备注", "口味、冰块等需求", "支付方式", "微信支付", "金币支付", "金币余额", "860 金币", "商品金额", "金币抵扣", "合计", "¥56.00", "确认支付"
Required UI: store/table context, item detail rows, remark row, WeChat/coin payment selector, coin balance hint, price summary, fixed bottom confirm payment bar
Constraints: no Alipay, no delivery, no express shipping, no recipient address, no buyer message, no sign-in/check-in, no bottom tabbar, no fake extra tab item, no non-proportional stretching
Avoid: ecommerce delivery checkout, address card, shipping fee, red coupon marketplace style, colorful payment badges, oversized cards, heavy shadows, dense unreadable text
```
