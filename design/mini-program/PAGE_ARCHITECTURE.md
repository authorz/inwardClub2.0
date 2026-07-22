# 小程序页面架构设计（排除签到）

## 1. 设计方向

整体视觉：参考 `design-demo/` 中的黑白/灰阶视觉：大面积留白、黑色胶囊按钮、浅灰分隔、深灰到黑的渐变空状态。大气、简洁、克制，可以使用黑白/灰阶渐变。

设计关键词：

- 高级会员俱乐部
- 清晰交易工具
- 黑白秩序感
- 留白和强排版
- 少装饰，重状态

禁止：

- 不做签到相关页面、弹窗或入口。
- 不做彩色餐饮促销风。
- 不做无意义卡片堆叠首页。活动列表允许使用相册式堆叠轮播，因为业务明确要求左右滑动展示上一个/下一个活动。
- 不做大量圆角卡片、阴影卡片、嵌套卡片。
- 不在小程序出现支付宝入口。

## 2. 推荐主导航

最终建议 4 个 Tab：

```text
首页
预约
点餐
我的
```

工作人员能力不放入 Tab。按 staff 身份在“我的”页或扫码入口显示“工作人员入口”。

底部菜单图标必须使用 `design/mini-program/tab-icons/` 中的定稿 SVG：`home.svg`、`reservation.svg`、`order.svg`、`me.svg`。后续页面设计和开发验收不得使用 AI 图中临时生成的 Tab 图标替代。

## 3. 从接口反推的页面清单

### 3.1 首页

接口来源：

- `GET /api/v2/mini/stores?lat=&lng=`
- `GET /api/v2/mini/stores/{storeID}`
- `GET /api/v2/mini/stores/{storeID}/banners`
- `GET /api/v2/mini/recharge-products`
- `GET /api/v2/mini/membership-tiers`

页面模块：

- 画布基准：iPhone 17 原生 1206×2622 px；首页纵向节奏不能压扁。
- 配色规则：产品 UI 以黑白灰为主，但 Banner、门店图和活动海报属于内容图片，允许有节制的真实色彩，不强制灰度化。
- 最近门店摘要：门店名、距离、切换门店、导航按钮；首页此处不显示营业状态。
- 顶部 Banner 图位：首页上方大面积空白/浅色区域是 Banner 图区域，不是普通信息留白；不要把门店信息塞满顶部。
- 最近门店：放在会员卡片下面，使用简洁横向门店栏；包含门店名、距离、切换门店和导航按钮；不显示营业状态。
- 会员卡叠层：会员信息卡片从 Banner 下缘叠上来，形成参考图中的浮层关系。
- 会员卡快捷入口：我的入场券、我的优惠券、活动列表。
- 会员卡和门店栏下方只展示最新活动列表，不展示订单中心、会员积分、积分兑换等额外模块。
- 最新活动：使用竖版大图横向排列，一行四个活动海报。
- 整体白色为主，浅灰页面底，黑色文字和胶囊按钮，不使用大面积深色 Hero。

注意：不包含签到。

### 3.2 门店选择与门店详情

接口来源：

- `GET /api/v2/mini/stores?lat=&lng=`
- `GET /api/v2/mini/stores/{storeID}`

页面模块：

- 门店列表：按距离排序。
- 门店详情：地址、电话、营业状态、导航按钮、门店介绍。
- 选择门店后回到首页刷新上下文。

布局建议：

- 列表用分割线和大字号门店名，不使用卡片。
- 距离和状态用细标签，不使用彩色块。

### 3.3 点餐

接口来源：

- `GET /api/v2/mini/stores/{storeID}/catalog/categories`
- `GET /api/v2/mini/stores/{storeID}/catalog/items`
- `POST /api/v2/mini/food-orders`
- `POST /api/v2/mini/payment-orders/{paymentOrderID}/wechat-jsapi`
- `POST /api/v2/mini/payment-orders/{paymentOrderID}/pay-by-coin`

页面：

```text
点餐首页
商品详情
购物车/订单确认
支付方式选择
支付结果
点餐订单详情
```

点餐首页模块：

- 左侧固定分类栏。
- 右侧商品区一行展示两个商品。
- 商品单元包含图片、名称、价格、库存/售罄、支付方式提示。
- 每个商品有数量步进器：减号、数量、加号。
- 底部购物车栏：总价、已选数量、去结算。

订单确认模块：

- 商品明细。
- 支付方式：微信、金币。
- 金币余额/不足提示。
- 备注、桌号或门店上下文。

设计注意：

- 微信和金币是唯一支付方式。
- 购物车支付成功后必须清空。
- 商品区允许两列商品单元，但避免重阴影卡片；使用浅灰分隔、白底、极轻边界。

### 3.4 预约

接口来源：

- `GET /api/v2/mini/stores/{storeID}/tables`
- `GET /api/v2/mini/stores/{storeID}/seats`
- `GET /api/v2/mini/reservations`
- `POST /api/v2/mini/reservations`
- `GET /api/v2/mini/reservations/{reservationID}`
- `POST /api/v2/mini/reservations/{reservationID}/cancel`
- `POST /api/v2/mini/waitlist-entries`

页面：

```text
桌台列表
座位列表
预约确认
排队等候
我的预约
预约详情
```

桌台列表模块：

- 当前门店。
- 以桌子为单位展示。
- 桌子视觉必须明确为德州扑克桌面，不使用普通餐桌或抽象平面图。
- 页面必须支持多张桌子；默认可以展开一张桌子，其他桌以紧凑摘要展示。
- 每张桌子固定 9 个座位。
- 桌子显示基础积分、更新时间、已预约座位数、预约状态。
- 每个座位可点击。
- 座位必须展示状态，至少包含空闲、已预约、已选择。
- 点击座位后弹出预约选项框，确认是否预约。
- 参考图中的桌面布局：一个桌子模块内围绕牌桌摆放 9 个座位。
- 页面需要提供“排队等候”按钮；点击后进入排队等候流程。

座位列表模块：

- 座位状态：可用、已预约、游戏中、维护中。
- 选择座位后进入确认。

我的预约：

- 当前预约。
- 历史预约。
- 取消预约。

设计注意：

- 状态不能只靠颜色，必须有文字/图标。
- 预约页可以使用轻量白底桌子模块承载复杂座位布局，但不要做重阴影卡片堆叠。
- 德州扑克桌面可以使用深色桌面材质作为功能图形，但页面 UI chrome、文字、按钮和分割线仍以黑白灰为主。
- 设计图如存在非等比适配版本，开发时不得逐像素照抄拉伸比例；应以源图自然比例、页面结构和业务规则为准。

### 3.5 活动

接口来源：

- `GET /api/v2/mini/stores/{storeID}/activities`
- `GET /api/v2/mini/activities`
- `GET /api/v2/mini/activities/{activityID}`
- `POST /api/v2/mini/activity-orders`
- `GET /api/v2/mini/activity-orders`
- `GET /api/v2/mini/activity-orders/{orderID}`

页面：

```text
活动列表
活动详情
票档选择
活动下单确认
活动支付结果
活动票夹
活动订单详情
```

活动列表模块：

- 未结束活动：相册式堆叠轮播，主活动大图居中，前后活动卡片在左右或后方露出，支持左右滑动展示上一个/下一个活动。
- 点击活动卡片后展开/进入活动详情。
- 历史活动：只保留轻量文字入口，进入后再展示历史活动列表。
- 活动列表页本身不显示顶部导航区域、返回箭头、微信胶囊或底部 TabBar。
- 活动列表页最终参考 v14 的浅色相册轮播方向。
- 活动列表页只展示必要摘要：活动名、简单时间/门店、报名状态、查看详情、我的入场券入口；票种/价格、余票、奖励、人数、长描述等详情放到活动详情页。
- 活动列表页避免重复堆叠同一活动标题。

活动详情模块：

- 活动头图。
- 门店距离/时间/地点。
- 活动描述。
- 默认详情态只展示活动详情和票类型/票档概览，不展开完整票档选择。
- 默认详情态不显示支付方式选择，不显示数量步进器。
- 底部购买按钮：点击后弹出票档选择弹窗。
- 票档选择弹窗：选择票档、数量和支付方式。
- 票档选择弹窗展示库存、售卖时间、限购和合计金额。
- 支付方式可在票档选择弹窗中选择，只允许微信和金币。

票夹模块：

- 可使用票。
- 已使用票。
- 已过期/已退款票。
- 核销码/二维码入口。

设计注意：

- 活动可以一次买多张，但服务端生成多张票。
- 票状态必须醒目。
- 活动卡片可以使用黑白渐变、活动头图、票档摘要和状态，但不要变成彩色营销海报。

### 3.6 我的

接口来源：

- `GET /api/v2/mini/me`
- `PATCH /api/v2/mini/me`
- `POST /api/v2/mini/me/phone-bindings`
- `GET /api/v2/mini/wallet`
- `GET /api/v2/mini/wallet/ledger`
- `GET /api/v2/mini/recharge-products`
- `POST /api/v2/mini/recharge-orders`
- `POST /api/v2/mini/point-savings`
- `GET /api/v2/mini/coupons`
- `GET /api/v2/mini/invitations`
- `POST /api/v2/mini/invitations/bind`
- `GET /api/v2/mini/rankings`
- `GET /api/v2/mini/membership-tiers`

页面：

```text
我的首页
个人资料
钱包
钱包流水
充值金币弹层
存取积分弹层
我的券
邀请
排行榜
会员权益
订单中心
```

我的首页模块：

- 会员头像、昵称、等级、邀请码。
- 钱包摘要：金币、积分、券。
- 订单入口：点餐、活动、充值、兑换。
- 我的券、邀请、排行榜、会员权益。
- staff 身份时显示工作人员入口。

钱包流水：

- 积分、金币、余额/成长值等资产切换。
- 收入/支出/调整/退款状态。

充值金币弹层：

- 从个人中心点击金币资产触发。
- 展示当前金币、充值档位、微信支付和确认充值。
- 充值档位来自 `GET /api/v2/mini/recharge-products`。
- 确认后调用 `POST /api/v2/mini/recharge-orders` 创建充值订单。
- 不展示支付宝、金币支付、提现、银行卡或现金到账。

存取积分弹层：

- 从个人中心点击积分资产触发。
- 展示当前积分、存入/取出切换、当前门店、积分数量、备注和提交申请。
- 调用 `POST /api/v2/mini/point-savings` 创建积分存取申请。
- 当前门店由后端上下文或用户当前门店确定，不允许跨店选择。
- 提交后由工作人员审核。

我的券：

- 可用券。
- 已使用。
- 已过期。
- 可兑换商品入口。

设计注意：

- 资产页面要严肃、清晰，不做花哨钱包卡。
- 邀请奖励、VIP 福利只展示规则结果，具体数值以后由规则配置驱动。

### 3.7 订单中心

接口来源：

- `GET /api/v2/mini/food-orders`
- `GET /api/v2/mini/food-orders/{orderID}`
- `GET /api/v2/mini/activity-orders`
- `GET /api/v2/mini/activity-orders/{orderID}`
- `GET /api/v2/mini/recharge-orders`
- `GET /api/v2/mini/recharge-orders/{orderID}`
- `POST /api/v2/mini/coupon-redemptions`

页面：

```text
订单中心
点餐订单详情
活动订单详情
充值订单详情
兑换订单详情
```

设计注意：

- 用顶部分段控件切换订单类型。
- 列表用分割线，不用卡片。
- 状态、金额、时间、门店、支付方式必须清楚。

### 3.8 工作人员入口

接口来源：

- `GET /api/v2/store/point-savings`
- `POST /api/v2/store/point-savings/{requestID}/review`
- `GET /api/v2/store/activities/today`
- `POST /api/v2/store/tickets/verify`
- `GET /api/v2/store/verifications`

页面：

```text
工作人员首页
积分审核列表
积分审核详情
活动核销
今日活动
核销历史
```

设计注意：

- 只对 staff 展示。
- 必须显示当前门店。
- 工作人员只能绑定一个门店；当前门店由服务端 staff 身份返回，小程序端不提供门店选择、门店切换或绑定其他门店入口。
- 核销成功/失败反馈必须明确。
- 可以使用扫码或手输核销码。

## 4. 第一批需要生成的设计图

本轮先出 10 张核心高保真页面图：

1. 首页
2. 门店选择
3. 点餐首页
4. 订单确认
5. 预约桌台/座位
6. 活动列表相册式堆叠轮播
7. 活动详情/票档
8. 我的首页
9. 我的券/票夹
10. 工作人员核销

不生成签到页面。

## 5. 通用组件方向

- 顶部门店上下文条。
- 黑白底部 TabBar。
- 全宽海报式 Banner。
- 分割线列表。
- 分段控件。
- 底部操作栏。
- 底部弹层。
- 状态文本 + 图标。
- 资产数字摘要。
- 核销码展示区。

## 6. GPT Image 2 出图规格

统一画幅：

```text
手机小程序高保真 UI mockup
竖图
建议 9:16
单张只展示一个页面
不要拼多屏长图
```

统一风格：

```text
black and white premium mini program UI
clean spacious typography
editorial grayscale hierarchy
subtle black-to-white gradients allowed
avoid meaningless card-heavy layout
activity list uses swipeable event cards by requirement
ordering page uses left category rail and two-column product grid with steppers
reservation page uses table modules with 9 seats and a reservation confirmation bottom sheet/modal
no colorful restaurant style
no beige/cream palette
no purple/blue gradients
no marketing hero website layout
```

## 7. 提示词留档规则

每一次设计稿修改都必须保留当次提示词。

目录规则：

```text
design/mini-program/prompt-history/v1/
design/mini-program/prompt-history/v2/
design/mini-program/prompt-history/v3/
```

当前正在使用的提示词可以放在 `design/mini-program/GPT_IMAGE_2_PROMPTS.md`，但每次修改前必须复制归档到对应版本目录。归档文件不得覆盖。

文字要求：

- 中文界面。
- 文案短而真实。
- 不要求图片中的文字完全可直接用于开发，但主要标题和按钮要清楚。
