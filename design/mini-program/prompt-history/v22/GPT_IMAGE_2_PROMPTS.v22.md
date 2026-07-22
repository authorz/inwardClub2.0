# gpt-image-2 小程序设计图提示词 v22

## 1. 本次修改来源

继续设计个人中心页面。

需要承接页面架构：

- 会员头像、昵称、等级、邀请码。
- 钱包摘要：金币、积分、券。
- 订单入口：点餐、活动、充值、兑换。
- 我的券、邀请、排行榜、会员权益。
- staff 身份时显示工作人员入口。

继续遵守：

- 底部菜单固定为：首页、预约、点餐、我的。
- 开发时底部菜单图标必须使用 `design/mini-program/tab-icons/` 定稿 SVG。
- UI chrome 以黑白灰为主。
- 个人中心需要严肃、清晰，不做花哨钱包卡。
- 不出现签到入口。
- 不出现支付宝。
- 交付图不做非等比拉伸，Claude 以源图自然比例、结构和业务规则为准。

## 2. 个人中心白底资产版 v22

```text
Use case: ui-mockup
Asset type: WeChat mini program member center profile and wallet overview screen
Canvas: portrait mobile UI screen, keep natural app proportions, do not distort or stretch UI elements
Primary request: Redesign the InwardClub member center page. Use a white-first premium black/gray interface, not a mostly black page. The page should show member profile, member tier, invite code, wallet summary, order entries, coupon/ticket entries, invitation, ranking, membership benefits, and a conditional staff entry. Keep it clear and operational, not a decorative wallet card.
Style/medium: high-fidelity mobile UI mockup, premium minimalist club app, white-first interface, black typography, light gray surfaces, thin dividers, serious asset/account dashboard
Composition/framing: single vertical "我的" screen. Top has title "我的". Show profile header with round avatar, nickname "Authoz", member tier black capsule "普卡", invite code "邀请码 8A7B3C" with copy icon, and a subtle profile arrow. Below show wallet summary as a clean horizontal asset strip, not a flashy card: "金币 2680", "积分 5680", "券 3张". Then show order center section "订单中心" with four compact icon entries: "点餐订单", "活动订单", "充值订单", "兑换订单". Then show a clean list section with thin dividers: "我的券", "我的入场券", "邀请好友", "排行榜", "会员权益". Add a low-emphasis row "工作人员入口" with small "仅员工可见" tag. Bottom tabbar exactly four items: 首页, 预约, 点餐, 我的, with 我的 active.
Color rule: UI chrome, text, dividers, rows, icons, and buttons are black/white/light gray. Avatar may use natural color. Avoid large dark background, purple/blue gradients, red coupon style, and gold luxury overload.
Text language: Chinese
Required visible text: "我的", "Authoz", "普卡", "邀请码 8A7B3C", "金币", "2680", "积分", "5680", "券", "3张", "订单中心", "点餐订单", "活动订单", "充值订单", "兑换订单", "我的券", "我的入场券", "邀请好友", "排行榜", "会员权益", "工作人员入口", "仅员工可见", "首页", "预约", "点餐", "我的"
Required UI: profile header, member tier, invite code copy action, wallet asset summary, order center entries, list entries, staff-only entry, bottom tabbar with my active
Constraints: no sign-in/check-in, no Alipay, no fake extra tab item, no activity tab, no marketplace/coupon promotion page, no non-proportional stretching
Avoid: mostly black page, flashy wallet card, red coupon market style, dense entrance wall, oversized cards, nested cards, decorative gradients, unreadable tiny text
```
