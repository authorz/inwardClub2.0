# 小程序最终设计图选型

更新时间：2026-07-15

## 1. 设计状态

当前状态：核心页面结构通过，可进入“开发前设计冻结”评审。

本文件用于固定 Codex 开发时应该参考的设计图版本，避免误用首轮图或旧版提示词。

## 2. 最终推荐图

| 页面 | 最终参考图 | 提示词版本 | 结论 |
| --- | --- | --- | --- |
| 首页 | `design/mini-program/final/home/01-home-final-iphone17.png` | v11 | 通过，iPhone 17 原生画布，Banner 加高、活动海报压低，主次更合理 |
| 门店选择 | `design/mini-program/generated/02-store-selection.png` | v1 | 可作为视觉方向参考，门店字段以接口文档为准 |
| 点餐 | `design/mini-program/generated/v20/03-ordering-v20-compact-default-source.png` + `design/mini-program/generated/v20/03-ordering-v20-compact-active-cart-source.png` | v20 | 候选通过，右侧商品区改为紧凑两列商品格，空购物车/已加购两状态完整，等待用户确认是否定稿 |
| 订单确认 | `design/mini-program/final/order-confirmation/04-order-confirmation-final-payment.png` | v21 | 通过，线下门店付款页，含商品明细、微信/金币支付、金币余额和确认支付 |
| 预约 | `design/mini-program/final/reservation/05-reservation-final-multi-table.png` + `design/mini-program/final/reservation/05-reservation-final-confirm-sheet.png` | v18 | 通过，多桌预约、展开桌 9 座、排队等候、确认弹窗完整；源图比例优先 |
| 活动列表 | `design/mini-program/final/activity-list/06-activity-list-final-iphone17.png` | v14 | 通过，基于 v12 浅色相册轮播，仅保留必要摘要，详情信息留给活动详情页 |
| 活动详情 | `design/mini-program/generated/v16/07-activity-detail-v16-default-iphone17.png` + `design/mini-program/generated/v16/07-activity-detail-v16-purchase-sheet-iphone17.png` | v16 | 候选通过，分为默认详情态和点击购买后的票档/支付弹窗态，等待用户确认是否定稿 |
| 我的首页 | `design/mini-program/final/member-center/08-member-center-final-profile.png` | v22 | 通过，白底资产/订单/功能入口结构清晰，用户已确认定稿 |
| 我的券/票夹 | `design/mini-program/generated/09-coupons-tickets.png` | v1 | 通过，状态和核销入口明确，不像促销券市场 |
| 工作人员核销 | `design/mini-program/generated/v2/10-staff-verification-v2.png` | v2 | 通过，当前门店固定，无切换门店入口 |

## 3. 必须保留的提示词记录

提示词历史不得覆盖或删除：

- `design/mini-program/prompt-history/v1/GPT_IMAGE_2_PROMPTS.v1.md`
- `design/mini-program/prompt-history/v2/GPT_IMAGE_2_PROMPTS.v2.md`
- `design/mini-program/prompt-history/v3/GPT_IMAGE_2_PROMPTS.v3.md`
- `design/mini-program/prompt-history/v4/GPT_IMAGE_2_PROMPTS.v4.md`
- `design/mini-program/prompt-history/v5/GPT_IMAGE_2_PROMPTS.v5.md`
- `design/mini-program/prompt-history/v6/GPT_IMAGE_2_PROMPTS.v6.md`
- `design/mini-program/prompt-history/v7/GPT_IMAGE_2_PROMPTS.v7.md`
- `design/mini-program/prompt-history/v8/GPT_IMAGE_2_PROMPTS.v8.md`
- `design/mini-program/prompt-history/v9/GPT_IMAGE_2_PROMPTS.v9.md`
- `design/mini-program/prompt-history/v10/GPT_IMAGE_2_PROMPTS.v10.md`
- `design/mini-program/prompt-history/v11/GPT_IMAGE_2_PROMPTS.v11.md`
- `design/mini-program/prompt-history/v12/GPT_IMAGE_2_PROMPTS.v12.md`
- `design/mini-program/prompt-history/v13/GPT_IMAGE_2_PROMPTS.v13.md`
- `design/mini-program/prompt-history/v14/GPT_IMAGE_2_PROMPTS.v14.md`
- `design/mini-program/prompt-history/v15/GPT_IMAGE_2_PROMPTS.v15.md`
- `design/mini-program/prompt-history/v16/GPT_IMAGE_2_PROMPTS.v16.md`
- `design/mini-program/prompt-history/v17/GPT_IMAGE_2_PROMPTS.v17.md`
- `design/mini-program/prompt-history/v18/GPT_IMAGE_2_PROMPTS.v18.md`
- `design/mini-program/prompt-history/v19/GPT_IMAGE_2_PROMPTS.v19.md`
- `design/mini-program/prompt-history/v20/GPT_IMAGE_2_PROMPTS.v20.md`
- `design/mini-program/prompt-history/v21/GPT_IMAGE_2_PROMPTS.v21.md`
- `design/mini-program/prompt-history/v22/GPT_IMAGE_2_PROMPTS.v22.md`
- `design/mini-program/prompt-history/v23/GPT_IMAGE_2_PROMPTS.v23.md`
- `design/mini-program/prompt-history/v24/GPT_IMAGE_2_PROMPTS.v24.md`

当前汇总提示词：

- `design/mini-program/GPT_IMAGE_2_PROMPTS.md`

## 4. Codex 高保真还原要求

Codex 只负责开发实现，不负责重新生成设计图。

开发时必须以本文件的“最终参考图”为准，同时遵守：

- 不实现签到页面、签到入口、签到弹窗。
- 小程序支付只允许微信和金币，不出现支付宝。
- 首页必须参考 `design/mini-program/final/home/01-home-final-iphone17.png`：画布为 iPhone 17 原生 1206×2622 px，不能压扁。顶部 Banner 图位要比 v10 更高，会员卡从 Banner 下缘叠出。最近门店必须放在会员卡下面，样式参考 v5 的干净横向门店栏，只展示门店名、距离、切换门店和导航，不展示营业状态。最新活动必须是竖版大图横向排列，一行四个，但海报高度要比 v10 更矮。底部菜单固定为 `首页 / 预约 / 点餐 / 我的`，并使用 `design/mini-program/tab-icons/` 中定稿 SVG。
- 产品配色以黑白为主指 UI chrome、文字、按钮和分割线；Banner、门店图和活动海报是内容图片，允许有节制的真实色彩，不应强制灰度化。
- AI 图中的右上角微信胶囊和底部 Tab 图标只作为布局参考；开发必须使用小程序原生胶囊区域和已定稿 SVG 图标。
- 活动列表必须是可左右滑动的相册式堆叠轮播，主活动图居中突出，前后活动卡片在左右或后方露出；点击进入或展开活动详情。历史活动只保留轻量文字入口。
- 活动列表页不显示顶部导航区域、返回箭头、微信胶囊或底部 TabBar。最终参考 v14：基于 v12 的浅色相册轮播，只展示活动名、简单时间/门店摘要、报名状态、查看详情、我的入场券和历史活动入口；票种/价格、余票、奖励、人数、长描述等详情留给活动详情页。
- 活动详情页当前候选参考 v16 两状态：默认详情态 `design/mini-program/generated/v16/07-activity-detail-v16-default-iphone17.png` 只展示活动详情和票类型概览，点击底部“立即购票”后进入票档/支付弹窗态 `design/mini-program/generated/v16/07-activity-detail-v16-purchase-sheet-iphone17.png`。支付方式选择只能出现在弹窗中，可选微信支付和金币支付，不出现支付宝。
- 点餐页当前候选参考 v20 两状态：未加购/空购物车状态以 `design/mini-program/generated/v20/03-ordering-v20-compact-default-source.png` 为准，已加购态以 `design/mini-program/generated/v20/03-ordering-v20-compact-active-cart-source.png` 为准。必须是左侧分类栏、右侧两列商品网格，每个商品有数量步进器；右侧商品区应是紧凑商品格/货架，不做 oversized 商品卡片；底部购物车栏展示已选数量、总价和“去结算”。这是线下门店点餐，不出现配送、快递、买家留言或支付宝。
- 订单确认/付款页最终参考 v21：`design/mini-program/final/order-confirmation/04-order-confirmation-final-payment.png`。必须展示当前门店、桌台/座位、商品明细、备注、微信支付、金币支付、金币余额、金额汇总和“确认支付”。这是线下门店付款页，不出现配送、快递、收货地址、买家留言或支付宝。
- 预约页最终参考 v18 两状态：多桌默认态 `design/mini-program/final/reservation/05-reservation-final-multi-table.png`，确认弹窗态 `design/mini-program/final/reservation/05-reservation-final-confirm-sheet.png`。页面必须支持多张桌子；默认展开一张德州扑克桌，其他桌以紧凑摘要展示。每张桌子固定 9 个座位，座位展示空闲/已预约/已选择状态，页面必须有“排队等候”按钮；点击空闲座位后出现预约确认弹层。
- 我的首页最终参考 v22：`design/mini-program/final/member-center/08-member-center-final-profile.png`。页面必须展示头像、昵称、会员等级、邀请码复制、金币/积分/券资产摘要、点餐/活动/充值/兑换订单入口、我的券、我的入场券、邀请好友、排行榜、会员权益；工作人员入口仅 staff 身份可见。图中的 Tab 图标只作布局参考，开发必须使用 `design/mini-program/tab-icons/` 定稿 SVG。
- 工作人员只能绑定一个门店；工作人员核销页必须固定显示服务端返回的当前门店，不允许切换门店、门店下拉、跨店选择或重新绑定门店。
- AI 图中的错字、示例商品名、示例门店名、示例时间不得照抄，业务字段以接口文档和页面架构为准。
- 活动相册式堆叠轮播是允许的业务例外；其他页面避免无意义卡片堆叠和重阴影。
- 预约 v18 以源图自然比例为准，不使用此前非等比拉伸的 iPhone 17 适配图作为像素参考。确认弹窗图的背景是遮罩态，开发时应复用默认态页面作为背景，不照抄 AI 生成的深色页面细节。AI 图中的微信胶囊、系统状态栏和临时 Tab 图标只作生成噪声，开发必须使用小程序原生区域和定稿 SVG 图标。

## 5. 个人中心子页面 v23

当前状态：通过，用户已确认定稿。

| 页面 | 最终参考图 | 结论 |
| --- | --- | --- |
| 个人资料 | `design/mini-program/final/member-subpages/01-profile-edit-v23.png` | 通过，资料编辑结构清楚 |
| 钱包流水 | `design/mini-program/final/member-subpages/02-wallet-ledger-v23.png` | 通过，资产摘要、分段切换、流水列表完整 |
| 订单中心 | `design/mini-program/final/member-subpages/03-order-center-v23.png` | 通过，四类订单分段切换，列表用分割线 |
| 点餐订单详情 | `design/mini-program/final/member-subpages/04-food-order-detail-v23.png` | 通过，门店/桌台/明细/支付/金额完整 |
| 活动订单详情 | `design/mini-program/final/member-subpages/05-activity-order-detail-v23.png` | 通过，活动信息、票码、支付金额完整 |
| 充值订单详情 | `design/mini-program/final/member-subpages/06-recharge-order-detail-v23.png` | 通过，充值金额、到账金币、赠送金币、支付时间完整 |
| 兑换订单详情 | `design/mini-program/final/member-subpages/07-redemption-order-detail-v23.png` | 通过，兑换物、兑换码、门店确认状态完整 |
| 我的券 | `design/mini-program/final/member-subpages/08-my-coupons-v23.png` | 通过，券状态切换和兑换入口清楚 |
| 我的入场券 | `design/mini-program/final/member-subpages/09-my-tickets-v23.png` | 通过，待使用/已使用/已过期和出示票码入口清楚 |
| 邀请好友 | `design/mini-program/final/member-subpages/10-invitations-v23.png` | 通过，邀请码、分享、规则结果、邀请记录完整 |
| 排行榜 | `design/mini-program/final/member-subpages/11-rankings-v23.png` | 通过，周/月/总榜与我的排名清楚 |
| 会员权益 | `design/mini-program/final/member-subpages/12-member-benefits-v23.png` | 通过，当前等级、成长值、权益和等级说明完整 |
| 工作人员首页 | `design/mini-program/final/member-subpages/13-staff-home-v23.png` | 通过，当前门店固定且无切换入口 |

开发注意：

- v23 子页面均为源图自然比例 1024×1792 px，不做非等比拉伸。
- 子页面没有底部 TabBar；从个人中心进入后使用小程序原生返回。
- AI 图中的二维码、核销码、兑换码只是占位，开发必须用真实接口数据生成。
- 工作人员首页必须固定服务端返回的当前绑定门店，不得新增切换门店、门店下拉或跨店筛选。
- AI 图中的示例订单号、金额、时间、门店、昵称和权益说明不得照抄，业务字段以接口文档和页面架构为准。

## 6. 个人中心资产操作弹层 v24

当前状态：通过，用户已确认定稿。

| 触发入口 | 候选参考图 | 结论 |
| --- | --- | --- |
| 点击金币 | `design/mini-program/final/member-asset-sheets/01-coin-recharge-sheet-v24.png` | 通过，充值档位、微信支付、确认充值清楚 |
| 点击积分 | `design/mini-program/final/member-asset-sheets/02-point-saving-sheet-v24.png` | 通过，存入/取出、门店、数量、审核提示清楚 |

开发注意：

- v24 是个人中心资产区的底部弹层交互，不是独立页面。
- 点击金币弹出充值金币弹层；点击积分弹出存取积分弹层。
- 充值金币档位来自 `GET /api/v2/mini/recharge-products`，确认后调用 `POST /api/v2/mini/recharge-orders`。
- 充值金币只允许微信支付，不出现支付宝、金币支付、提现、银行卡或现金到账。
- 存取积分调用 `POST /api/v2/mini/point-savings`，提交后由工作人员审核。
- 存取积分弹层固定当前门店，不允许切换门店、门店下拉或跨店选择。
- 弹层中的金额、档位、积分数量、比例和规则文案以后由后台规则配置驱动，不写死为代码常量。
- AI 图中的示例数值、门店、金额和文案不得照抄，业务字段以接口文档和页面架构为准。

## 7. 开发输入索引

Codex 开发小程序页面时读取：

- `PRODUCT.md`
- `design/mini-program/PAGE_ARCHITECTURE.md`
- `design/mini-program/FINAL_DESIGN_SELECTION.md`
- `design/mini-program/GPT_IMAGE_2_PROMPTS.md`
- `tasks/acceptance/mini-program-design-acceptance.md`
- `docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md`
- `docs/V1_API_INVENTORY_AND_V2_MAPPING.md`
