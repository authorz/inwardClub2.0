# gpt-image-2 小程序设计图提示词 v23

## 1. 本次修改来源

继续完成个人中心页面中的所有子链接页面。

入口来源于个人中心 v22：

- 个人资料入口：头像/昵称区域右侧箭头。
- 钱包资产入口：金币、积分、券。
- 订单中心入口：点餐订单、活动订单、充值订单、兑换订单。
- 功能列表入口：我的券、我的入场券、邀请好友、排行榜、会员权益。
- staff 身份入口：工作人员入口。

页面合并策略：

- 金币、积分点击进入同一个“钱包流水”页，通过分段控件切换资产类型。
- 券资产和“我的券”列表入口进入同一个“我的券”页。
- 四个订单入口进入同一个“订单中心”页，顶部用分段控件切换类型；每类订单再有详情页。
- “我的入场券”独立为票夹页。
- 工作人员入口进入工作人员首页；后续核销页继续参考已通过的工作人员核销图。

继续遵守：

- UI chrome 以黑、白、浅灰为主。
- 内容图片、头像、活动海报可保留有节制的真实色彩。
- 不出现签到入口。
- 不出现支付宝。
- 小程序支付只允许微信和金币。
- 列表用分割线和通栏区块，不做重阴影卡片堆叠。
- 有底部主 Tab 的页面必须固定为：首页、预约、点餐、我的，并使用定稿 SVG；子页面可不显示底部 Tab。
- 开发时底部 Tab 图标必须使用 `design/mini-program/tab-icons/` 定稿 SVG，不照抄 AI 临时图标。
- 交付图不做非等比拉伸，Claude 以源图自然比例、结构和业务规则为准。

## 2. 个人资料页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program profile edit screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub member profile page opened from the profile header in "我的". It should be a clean account information editor, white-first, serious and operational.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, black/white/light-gray interface, thin dividers, no heavy cards
Composition/framing: single vertical child page with native top title "个人资料" and back affordance. Show avatar row with round avatar and action "更换头像". Then full-width form rows with thin dividers: "昵称 Authoz", "手机号 138****6272", "会员等级 普卡", "邀请码 8A7B3C", "绑定邀请人 未绑定". Add a low-emphasis note row "资料仅用于会员权益和订单通知". Bottom fixed black primary button "保存". No bottom tabbar.
Color rule: UI chrome, rows, labels, buttons are black/white/light gray. Avatar may use natural color.
Text language: Chinese
Required visible text: "个人资料", "更换头像", "昵称", "Authoz", "手机号", "138****6272", "会员等级", "普卡", "邀请码", "8A7B3C", "绑定邀请人", "未绑定", "资料仅用于会员权益和订单通知", "保存"
Constraints: no sign-in/check-in, no Alipay, no bottom tabbar, no marketplace language, no decorative wallet card
Avoid: dense settings wall, nested cards, colorful gradients, red coupon style, tiny unreadable text
```

## 3. 钱包流水页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program wallet ledger screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub wallet and asset ledger page. It should explain coins, points, coupons, and transaction history clearly.
Style/medium: high-fidelity mobile UI mockup, white-first asset dashboard, black typography, light gray dividers, serious ledger interface
Composition/framing: single vertical child page with title "钱包流水" and back affordance. Top asset summary as a clean horizontal strip: "金币 2680", "积分 5680", "券 3张". Under it show segmented control "金币 / 积分 / 券". Active tab "金币". Show filter row "全部 收入 支出 调整". Ledger list uses full-width rows with thin dividers: "点餐支付", "-56", "微信补差 · 今天 20:18"; "活动购票返还", "+80", "黑桃周赛 · 昨天"; "系统调整", "+200", "规则补发 · 07-12". Bottom has subtle text "仅展示近 90 天记录". No bottom tabbar.
Color rule: UI chrome, rows, labels, amounts, dividers are black/white/light gray; positive amount may use dark charcoal, negative amount black; no bright finance colors.
Text language: Chinese
Required visible text: "钱包流水", "金币", "2680", "积分", "5680", "券", "3张", "全部", "收入", "支出", "调整", "点餐支付", "-56", "微信补差", "今天 20:18", "活动购票返还", "+80", "黑桃周赛", "昨天", "系统调整", "+200", "规则补发", "仅展示近 90 天记录"
Constraints: no sign-in/check-in, no Alipay, no balance withdrawal, no bank card, no bottom tabbar
Avoid: stock trading UI, colorful charts, flashy wallet card, nested cards, unreadable tiny text
```

## 4. 订单中心页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program order center list screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub order center opened from any order entry. It must support food, activity, recharge, and redemption orders through a segmented control.
Style/medium: high-fidelity mobile UI mockup, clean operational order list, white-first, thin dividers, no heavy card stack
Composition/framing: single vertical child page with title "订单中心" and back affordance. Top segmented control has four tabs: "点餐", "活动", "充值", "兑换", active "点餐". Below show compact filter row "全部 / 待支付 / 已完成 / 已取消". Order list uses full-width rows separated by thin dividers, not cards. Row 1: "点餐订单 #F2026071501", status "已完成", store "Inward Club 三里屯店", context "1号桌 · 5号座位", time "今天 20:18", amount "¥56.00", payment "微信 / 金币". Row 2: "点餐订单 #F2026071408", status "备餐中", amount "¥88.00". Each row has right arrow. No bottom tabbar.
Color rule: black/white/light gray UI; status uses black text or gray capsule, no bright colors.
Text language: Chinese
Required visible text: "订单中心", "点餐", "活动", "充值", "兑换", "全部", "待支付", "已完成", "已取消", "点餐订单", "#F2026071501", "Inward Club 三里屯店", "1号桌 · 5号座位", "今天 20:18", "¥56.00", "微信 / 金币", "#F2026071408", "备餐中", "¥88.00"
Constraints: no sign-in/check-in, no Alipay, no delivery, no courier, no shipping address, no buyer message, no bottom tabbar
Avoid: ecommerce marketplace order list, delivery app color palette, card-heavy layout, tiny text
```

## 5. 点餐订单详情页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program food order detail screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub food order detail page for offline club dining.
Style/medium: high-fidelity mobile UI mockup, white-first operational detail page, thin dividers, black primary text
Composition/framing: single vertical child page with title "点餐订单详情" and back affordance. Top status block with "已完成" and order number "#F2026071501". Show rows: "门店 Inward Club 三里屯店", "桌台 1号桌 · 5号座位", "下单时间 今天 20:18", "支付方式 微信支付 + 金币抵扣". Product detail list with thin dividers: "经典啤酒 x2 ¥36.00", "苏打水 x1 ¥8.00", "花生米 x1 ¥12.00". Amount section: "商品金额 ¥56.00", "金币抵扣 -¥8.00", "实付 ¥48.00". Bottom actions: text button "再来一单", black button "查看小票". No bottom tabbar.
Color rule: black/white/light gray UI; no colorful food delivery styling.
Text language: Chinese
Required visible text: "点餐订单详情", "已完成", "#F2026071501", "门店", "Inward Club 三里屯店", "桌台", "1号桌 · 5号座位", "下单时间", "今天 20:18", "支付方式", "微信支付 + 金币抵扣", "经典啤酒", "苏打水", "花生米", "商品金额", "金币抵扣", "实付", "再来一单", "查看小票"
Constraints: no sign-in/check-in, no Alipay, no delivery, no shipping address, no courier, no buyer message, no bottom tabbar
Avoid: food delivery UI, red/yellow promo tags, large product cards, nested cards
```

## 6. 活动订单详情页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program activity ticket order detail screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub activity order detail page with ticket status and verification code area.
Style/medium: high-fidelity mobile UI mockup, premium light ticket detail, black/white/light-gray interface, restrained event poster color
Composition/framing: single vertical child page with title "活动订单详情" and back affordance. Top status "待核销" and order number "#A2026071506". Show a compact horizontal activity info area with small poster thumbnail and text: "黑桃周赛", "本周六 19:30", "Inward Club 三里屯店", "标准票 x1". Below show ticket code block: large QR placeholder style square, text "核销码 8392 6174", note "入场时向工作人员出示". Then rows: "支付方式 微信支付", "订单金额 ¥128.00", "购买时间 今天 18:42". Bottom black button "查看活动详情". No bottom tabbar.
Color rule: UI chrome black/white/light gray. Event thumbnail can use restrained real color.
Text language: Chinese
Required visible text: "活动订单详情", "待核销", "#A2026071506", "黑桃周赛", "本周六 19:30", "Inward Club 三里屯店", "标准票 x1", "核销码", "8392 6174", "入场时向工作人员出示", "支付方式", "微信支付", "订单金额", "¥128.00", "购买时间", "今天 18:42", "查看活动详情"
Constraints: no sign-in/check-in, no Alipay, no bottom tabbar, no delivery language
Avoid: nightclub concert UI, loud colors, dense article layout, unreadable QR details
```

## 7. 充值订单详情页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program recharge order detail screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub recharge order detail page for coin recharge records.
Style/medium: high-fidelity mobile UI mockup, white-first payment record page, serious ledger style, black/gray UI
Composition/framing: single vertical child page with title "充值订单详情" and back affordance. Top status "充值成功" and order number "#R2026071502". Main amount area: "充值金额 ¥500.00", "到账金币 5200", "赠送金币 200". Rows: "支付方式 微信支付", "支付时间 今天 17:12", "到账时间 今天 17:12", "交易说明 充值到账后不可直接提现". Bottom black button "查看钱包流水". No bottom tabbar.
Color rule: black/white/light-gray UI; no gold luxury overload.
Text language: Chinese
Required visible text: "充值订单详情", "充值成功", "#R2026071502", "充值金额", "¥500.00", "到账金币", "5200", "赠送金币", "200", "支付方式", "微信支付", "支付时间", "今天 17:12", "到账时间", "交易说明", "充值到账后不可直接提现", "查看钱包流水"
Constraints: no sign-in/check-in, no Alipay, no bank card, no withdrawal, no bottom tabbar
Avoid: flashy wallet card, stock/finance trading UI, bright colored gradients
```

## 8. 兑换订单详情页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program coupon redemption order detail screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub redemption order detail page for coupon/product exchange.
Style/medium: high-fidelity mobile UI mockup, clean operational redemption detail, black/white/light-gray UI
Composition/framing: single vertical child page with title "兑换订单详情" and back affordance. Top status "待取用" and order number "#E2026071503". Show redemption item rows with small monochrome icon/thumbnail area: "香槟券兑换", "消耗券 1张", "有效期至 2026-07-22". Show code area "兑换码 5268 9031" and note "请在门店向工作人员出示". Rows: "门店 Inward Club 三里屯店", "兑换时间 今天 19:05", "状态 待工作人员确认". Bottom actions: secondary "取消兑换", black button "出示兑换码". No bottom tabbar.
Color rule: black/white/light gray; avoid red coupon marketplace style.
Text language: Chinese
Required visible text: "兑换订单详情", "待取用", "#E2026071503", "香槟券兑换", "消耗券", "1张", "有效期至", "2026-07-22", "兑换码", "5268 9031", "请在门店向工作人员出示", "门店", "Inward Club 三里屯店", "兑换时间", "今天 19:05", "状态", "待工作人员确认", "取消兑换", "出示兑换码"
Constraints: no sign-in/check-in, no Alipay, no delivery, no shipping address, no bottom tabbar
Avoid: ecommerce coupon market, red envelopes, big promo tags, nested cards
```

## 9. 我的券页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program my coupons screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Redesign the InwardClub my coupons page in the same white-first style as member center v22.
Style/medium: high-fidelity mobile UI mockup, premium minimalist coupon entitlement list, not a coupon marketplace
Composition/framing: single vertical child page with title "我的券" and back affordance. Top segmented control: "可用 / 已使用 / 已过期", active "可用". Add concise summary row "可用 3 张". Coupon list uses restrained full-width entitlement rows with thin dividers and small ticket icon, not red coupon cards. Rows: "酒水兑换券", "可兑换指定酒水", "有效期至 2026-07-22", action "去兑换"; "活动抵扣券", "购票可抵 ¥30", "有效期至 2026-08-01", action "查看适用活动". Bottom small text link "兑换记录". No bottom tabbar.
Color rule: black/white/light gray UI; no red coupon marketplace palette.
Text language: Chinese
Required visible text: "我的券", "可用", "已使用", "已过期", "可用 3 张", "酒水兑换券", "可兑换指定酒水", "有效期至 2026-07-22", "去兑换", "活动抵扣券", "购票可抵 ¥30", "有效期至 2026-08-01", "查看适用活动", "兑换记录"
Constraints: no sign-in/check-in, no Alipay, no marketplace coupon feed, no bottom tabbar
Avoid: red packets, huge discount badges, nested cards, colorful ecommerce coupon style
```

## 10. 我的入场券页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program my tickets wallet screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub my event tickets page for active and historical activity tickets.
Style/medium: high-fidelity mobile UI mockup, premium light ticket wallet, black/white/light-gray interface, restrained event poster thumbnails
Composition/framing: single vertical child page with title "我的入场券" and back affordance. Top segmented control: "待使用 / 已使用 / 已过期", active "待使用". Main list uses full-width ticket rows with thin dividers and small portrait poster thumbnail. Row 1: "黑桃周赛", "本周六 19:30", "Inward Club 三里屯店", "标准票 x1", status "待核销", action "出示票码". Row 2: "会员私享局", "07-28 20:00", "双人票 x1", action "查看详情". No bottom tabbar.
Color rule: UI chrome black/white/light gray; event thumbnails can use restrained real color.
Text language: Chinese
Required visible text: "我的入场券", "待使用", "已使用", "已过期", "黑桃周赛", "本周六 19:30", "Inward Club 三里屯店", "标准票 x1", "待核销", "出示票码", "会员私享局", "07-28 20:00", "双人票 x1", "查看详情"
Constraints: no sign-in/check-in, no Alipay, no bottom tabbar, no large activity list carousel
Avoid: concert app style, heavy cards, bright neon, tiny text
```

## 11. 邀请好友页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program invitation and referral screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub invitation page. It should show invite code, binding status, invite records, and rule result without hardcoding sensitive business promises.
Style/medium: high-fidelity mobile UI mockup, white-first member referral page, clean and calm, black/gray UI
Composition/framing: single vertical child page with title "邀请好友" and back affordance. Top code area with "我的邀请码 8A7B3C", copy action "复制", and black button "分享给好友". Show rule summary as plain rows: "邀请奖励", "按后台规则发放", "仅在有效消费后结算". Show invite stats strip: "已邀请 12", "已生效 8", "待结算 2". Invite record list with dividers: "Luna", "已生效", "2026-07-12"; "Ken", "待结算", "2026-07-10". No bottom tabbar.
Color rule: black/white/light gray UI; no gold commission style, no red envelope style.
Text language: Chinese
Required visible text: "邀请好友", "我的邀请码", "8A7B3C", "复制", "分享给好友", "邀请奖励", "按后台规则发放", "仅在有效消费后结算", "已邀请", "12", "已生效", "8", "待结算", "2", "邀请记录", "Luna", "Ken"
Constraints: no sign-in/check-in, no Alipay, no fixed commission percentage, no cash withdrawal promise, no bottom tabbar
Avoid: MLM style, red envelope UI, flashy gold reward card, dense legal text
```

## 12. 排行榜页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program ranking screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub ranking page. It should support weekly, monthly, and total ranking snapshots and feel premium but restrained.
Style/medium: high-fidelity mobile UI mockup, white-first ranking list, black/gray interface, clean competitive hierarchy
Composition/framing: single vertical child page with title "排行榜" and back affordance. Top segmented control: "周榜 / 月榜 / 总榜", active "月榜". Show current user row "我的排名 12", "积分 5680". Top three ranking area uses simple podium-like typography without colorful medals: "01 Mason 12800", "02 Authoz 5680", "03 Luna 5300". Below list rows 04-10 with avatar circles, nickname, points, rank number, thin dividers. Add note "榜单每日 02:00 更新". No bottom tabbar.
Color rule: black/white/light gray UI; no bright medal colors, no gaming neon.
Text language: Chinese
Required visible text: "排行榜", "周榜", "月榜", "总榜", "我的排名", "12", "积分", "5680", "01", "Mason", "12800", "02", "Authoz", "03", "Luna", "榜单每日 02:00 更新"
Constraints: no sign-in/check-in, no Alipay, no bottom tabbar
Avoid: casino/gaming neon, gold trophy overload, dense chart, tiny text
```

## 13. 会员权益页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program membership benefits screen
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the InwardClub membership benefits page. It should show current tier, progress, tier list, and benefit results driven by backend rules.
Style/medium: high-fidelity mobile UI mockup, premium minimalist membership page, white-first, black/gray UI, restrained hierarchy
Composition/framing: single vertical child page with title "会员权益" and back affordance. Top current tier area: "当前等级 普卡", "成长值 320 / 1000", progress bar in gray. Section "本级权益" with rows: "活动优先报名", "专属券包", "生日权益", each showing "按后台规则发放". Section "等级说明" as vertical tier list: "普卡", "银卡", "黑卡", with short rule-result placeholders. Bottom text "权益内容以门店和后台规则为准". No bottom tabbar.
Color rule: black/white/light gray UI; no gold luxury overload, no purple gradients.
Text language: Chinese
Required visible text: "会员权益", "当前等级", "普卡", "成长值", "320 / 1000", "本级权益", "活动优先报名", "专属券包", "生日权益", "按后台规则发放", "等级说明", "银卡", "黑卡", "权益内容以门店和后台规则为准"
Constraints: no sign-in/check-in, no Alipay, no hardcoded benefit amounts, no bottom tabbar
Avoid: luxury gold card, dense rule article, colorful gradients, nested cards
```

## 14. 工作人员首页 v23

```text
Use case: ui-mockup
Asset type: WeChat mini program staff home screen opened from member center staff-only entry
Canvas: portrait mobile UI screen, natural app proportions, no non-proportional stretching
Primary request: Design the staff home page for staff users. Staff can bind only one store and cannot switch stores.
Style/medium: high-fidelity mobile UI mockup, operational staff dashboard, white-first, black/gray, clear task entries
Composition/framing: single vertical child page with title "工作人员" and back affordance. Top fixed store row: "当前门店 Inward Club 三里屯店", small text "已绑定，无法切换". Task entries as clean full-width rows with icons and arrows: "活动核销", "扫码核销入场券", "积分审核", "待审核 6", "核销记录", "今日 18 条", "今日活动", "黑桃周赛 19:30". Bottom low-emphasis warning "工作人员仅可操作已绑定门店数据". No bottom tabbar.
Color rule: black/white/light gray UI; no admin-console dark dashboard style.
Text language: Chinese
Required visible text: "工作人员", "当前门店", "Inward Club 三里屯店", "已绑定，无法切换", "活动核销", "扫码核销入场券", "积分审核", "待审核 6", "核销记录", "今日 18 条", "今日活动", "黑桃周赛 19:30", "工作人员仅可操作已绑定门店数据"
Constraints: staff can bind only one store, no store switcher, no store dropdown, no cross-store selection, no sign-in/check-in, no Alipay, no bottom tabbar
Avoid: total admin dashboard, dark dense console, switch store button, multi-store selector
```
