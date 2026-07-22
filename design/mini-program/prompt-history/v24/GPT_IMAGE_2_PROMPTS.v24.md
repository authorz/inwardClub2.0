# gpt-image-2 小程序设计图提示词 v24

## 1. 本次修改来源

用户补充：

- 还差充值金币和存取积分页面。
- 希望在个人中心完成：点击个人中心的金币和积分后弹出界面来充值、存取积分。

页面策略：

- 不新增底部 Tab 页面。
- 不从个人中心跳转到复杂独立页。
- 点击“金币”资产时弹出“充值金币”底部弹层。
- 点击“积分”资产时弹出“存取积分”底部弹层。
- 充值金币成功后生成充值订单，后续进入已定稿的充值订单详情。
- 存取积分提交后生成积分存取/审核记录，门店工作人员在工作人员入口中审核。

接口关联：

- `GET /api/v2/mini/wallet`
- `GET /api/v2/mini/wallet/ledger`
- `GET /api/v2/mini/recharge-products`
- `POST /api/v2/mini/recharge-orders`
- `POST /api/v2/mini/point-savings`

继续遵守：

- UI chrome 以黑、白、浅灰为主。
- 不出现签到入口。
- 不出现支付宝。
- 充值金币支付方式只允许微信支付。
- 金币不能直接提现，不出现提现、银行卡、现金到账等金融语义。
- 积分存取是门店/俱乐部业务动作，需要门店、数量、备注和审核状态，不做金融提现界面。
- 弹层中的金额、档位、比例、规则文案以后由后台规则配置驱动，不写死为代码常量。
- AI 图中的示例数值不得照抄，业务字段以接口文档和页面架构为准。
- 交付图不做非等比拉伸，Claude 以源图自然比例、结构和业务规则为准。

## 2. 点击金币后的充值金币弹层 v24

```text
Use case: ui-mockup
Asset type: WeChat mini program member center coin recharge bottom sheet
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the state after a user taps the "金币" asset in the InwardClub member center. Show the member center page dimmed in the background and a clean bottom action sheet for coin recharge.
Style/medium: high-fidelity WeChat mini program UI mockup, premium white-first black/gray interface, native bottom sheet, thin dividers, serious asset workflow
Composition/framing: single vertical screen. Background should visibly be the finalized "我的" member center style: title "我的", profile header, asset strip with "金币 2680 / 积分 5680 / 券 3张", order center and list entries, bottom tabbar "首页 / 预约 / 点餐 / 我的" with 我的 active. Apply a translucent dark overlay to background. From bottom, show a white rounded-top bottom sheet occupying around 45-55% of the screen. Sheet title "充值金币" with close icon. Show current balance row "当前金币 2680". Show recharge product choices from backend as clean selectable rows or compact two-column grid: "¥100 得 1000金币", "¥300 得 3200金币", "¥500 得 5500金币". Active choice has black border or black selected indicator. Show payment method row "支付方式 微信支付"; do not show coin payment or Alipay. Show rule hint "到账金币以后台充值档位为准". Bottom fixed sheet footer: total "合计 ¥300" and black primary button "确认充值".
Color rule: UI chrome, text, dividers, selected state, and button are black/white/light gray. No gold luxury overload, no green WeChat logo color, no red promotion style.
Text language: Chinese
Required visible text: "我的", "金币", "2680", "积分", "5680", "券", "3张", "首页", "预约", "点餐", "我的", "充值金币", "当前金币", "¥100", "1000金币", "¥300", "3200金币", "¥500", "5500金币", "支付方式", "微信支付", "到账金币以后台充值档位为准", "合计 ¥300", "确认充值"
Required UI: dimmed member center background, bottom sheet, close icon, current coin balance, recharge product choices, selected recharge product, WeChat payment only, total amount, confirm recharge button
Constraints: no sign-in/check-in, no Alipay, no withdrawal, no bank card, no cash out, no coin payment for recharge, no independent full page, no non-proportional stretching
Avoid: fintech investing UI, flashy gold wallet card, red coupon/promo style, heavy nested cards, tiny unreadable text
```

## 3. 点击积分后的存取积分弹层 v24

```text
Use case: ui-mockup
Asset type: WeChat mini program member center point saving and withdrawal bottom sheet
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the state after a user taps the "积分" asset in the InwardClub member center. Show a bottom sheet for saving or withdrawing points within club/store operations.
Style/medium: high-fidelity WeChat mini program UI mockup, premium white-first black/gray interface, native bottom sheet, thin dividers, operational and non-financial
Composition/framing: single vertical screen. Background should visibly be the finalized "我的" member center style with asset strip; apply translucent dark overlay. From bottom, show a white rounded-top bottom sheet occupying around 52-62% of the screen. Sheet title "存取积分" with close icon. Show current points row "当前积分 5680". Add segmented control "存入 / 取出", active "存入". Show store row "门店 Inward Club 三里屯店" with no store switch dropdown. Show amount input area "积分数量" with example "1000" and quick chips "500 / 1000 / 2000". Show note input row "备注 选填". Show review hint "提交后由工作人员审核". Add secondary text link "查看积分流水". Bottom fixed sheet footer has black primary button "提交申请". No payment method in this points sheet.
Color rule: UI chrome, text, dividers, selected state, and button are black/white/light gray. No finance cash-out colors. No gold, red, or green promotion style.
Text language: Chinese
Required visible text: "我的", "金币", "2680", "积分", "5680", "券", "3张", "存取积分", "当前积分", "存入", "取出", "门店", "Inward Club 三里屯店", "积分数量", "1000", "500", "2000", "备注", "选填", "提交后由工作人员审核", "查看积分流水", "提交申请"
Required UI: dimmed member center background, bottom sheet, close icon, current points balance, segmented control for save/withdraw, fixed current store row, points amount input, quick amount chips, optional note row, staff review hint, submit button
Constraints: no sign-in/check-in, no Alipay, no payment method, no bank card, no cash withdrawal, no store switcher, no store dropdown, no cross-store selection, no independent full page, no non-proportional stretching
Avoid: financial withdrawal UI, cash-out language, stock trading UI, colorful gradients, dense forms, heavy nested cards, tiny unreadable text
```
