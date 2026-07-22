# 小程序设计提示词日志

用于记录用户对小程序界面的具体要求与实现提示词，方便后续 Claude 高保真还原。

---

## 2026-07-14 — 首页最小返工（home）

### 用户要求
1. 首页启用自定义导航（`navigationStyle: custom`），顶部加入 brand nav：左侧 image 使用裁剪 wordmark logo，右侧为微信胶囊留出安全距离；白底、高级简洁。
2. banner 下移到自定义 nav 之后，保持彩色图片，不做 grayscale filter。
3. 会员卡三个入口（入场券 / 优惠券 / 活动列表）的 CSS 占位图标（`<view class="ic-ic ...">`）替换为公共图片资源，改用 `<image>` 引用黑白线性图标。
4. 删除“最近门店”标题 section header，仅保留门店条。
5. 调紧 `home__store` 与 `components/store-bar` 内边距、图标尺寸、按钮尺寸，让门店名称/距离/切换/导航更舒展。

### 实现提示词（高保真还原要点）
- **自定义导航**：`home.js` 中 `measureNav()` 用 `wx.getWindowInfo()` + `wx.getMenuButtonBoundingClientRect()` 计算 `navStatusBar`（状态栏高）、`navContentHeight`（胶囊高 + 上下间距×2）、`navRightGap`（窗口宽 − 胶囊 left + 8，为胶囊留白）。
  - WXML：`.home__nav`（白底，`padding-top = navStatusBar px`）> `.home__nav-bar`（`height = navContentHeight px`，`padding-right = navRightGap px`，左内边距 32rpx）> `.home__nav-logo` image。
  - logo 资源：`/assets/brand/logo-wordmark.png`（800×353 裁剪 wordmark），`mode="heightFix"`，`height: 52rpx`。
- **banner**：保持原彩色 `aspectFill` 图片，无 filter；位于自定义 nav 之后正常流式排布。
- **入口图标**：改为 `<image class="home__entry-ic" src="/assets/icons/home-*.png" mode="aspectFit" />`，尺寸 48rpx。
  - 资源：`assets/icons/home-ticket.png` / `home-coupon.png` / `home-activity.png`，黑白线性风格（#111 描边、透明背景），用 PIL 生成（避免微信对 SVG 的兼容风险）。
- **门店条**：删除 `.home__sec`（最近门店标题）；`.home__store` 提供唯一卡片盒子（`margin: 40rpx 24rpx 0; padding: 20rpx 24rpx; 背景 fill-2; 圆角 20rpx`）；`ic-store-bar` 传 `flat="{{true}}"` 使其透明、无内边距，避免双层内边距。
  - store-bar 紧凑化：thumb 72rpx、main margin 18rpx、name 30rpx、sub 22rpx、actions gap 6rpx、vline 高 44rpx、act-ico 36rpx。

### 涉及文件
- `mini-program/miniprogram/pages/home/home.{json,wxml,wxss,js}`
- `mini-program/miniprogram/components/store-bar/store-bar.wxss`
- `mini-program/miniprogram/assets/icons/home-{ticket,coupon,activity}.png`（新增）
- `mini-program/miniprogram/assets/brand/logo-wordmark.png`（复用）

---

## 2026-07-14 · 首页 logo 居中 + gpt-image-2 图标板重画（v2 图标）

用户截图反馈：自定义导航 logo 靠左不好看，改为居中（仍避让胶囊）；会员卡三个入口图标 + 门店条切换/导航图标太丑（CSS 占位 / 字符箭头），要求用 gpt-image-2 重画后裁切替换。

### gpt-image-2 图标板
- 图标板路径：`output/imagegen/home-icons-gpt-image-2-v1.png`（1024×1024，一行五个：TICKET / COUPON / EVENTS / SWITCH / NAV）
- 提示词源文件：`tmp/imagegen/home-icon-board-prompt.txt`
- 提示词全文：

```
Use case: logo-brand
Asset type: WeChat mini program home page icon design board for implementation handoff
Primary request: Create a clean black-and-white icon set redesign for Inward Club home page. Icons needed: 1) admission ticket / entry pass, 2) coupon / voucher, 3) activity list / event list, 4) switch store, 5) navigation / route arrow.
Scene/backdrop: flat pure white background, icon grid only, no phone mockup, no UI screen.
Subject: five separate icons arranged in one row with generous spacing; each icon centered in its own invisible 160x160 square cell.
Style/medium: premium minimalist vector-like line icons, monochrome black strokes, slightly rounded corners, refined luxury club feeling, not childish, not emoji, not generic app-store clipart.
Composition/framing: square 1024x1024 canvas, icons large and crisp, one row across the middle, consistent baseline and visual weight. Add tiny labels below in English only: TICKET, COUPON, EVENTS, SWITCH, NAV.
Color palette: #111111 strokes on #ffffff background only.
Materials/textures: no textures, no gradients, no shadows, no 3D.
Constraints: consistent 2.5px equivalent stroke, transparent-feeling white background, icon silhouettes easy to crop into separate app assets, preserve ample whitespace between icons, no Chinese text, no brand logo, no watermark.
Avoid: filled chunky icons, gray low-contrast strokes, decorative circles around every icon, hand-drawn wobble, pixelated edges, overly complex details, stock icon look, emojis, phone screenshot.
```

### 图标裁切/处理
- 用 PIL 从图标板按列自动检测五个图标包围盒（y 380–560 波段扫描），逐个裁切。
- 透明化：`alpha = 255 - L`、RGB 全黑，天然抠掉白底并保留抗锯齿边缘（不是硬阈值），确保浅色背景下无白底块。
- 归一化视觉重量：各图标按最长边缩放到 144×144 画布的 80% 居中，统一视觉大小；线性 #111 风格一致。
- 产物（144×144 透明 PNG）：
  - `assets/icons/home-ticket.png` / `home-coupon.png` / `home-activity.png`（替换旧入口图标）
  - `assets/icons/home-switch.png` / `home-nav.png`（新增，门店条切换/导航）

### 实现要点
- **logo 居中**：`home__nav-bar` 改 `justify-content: center`，左右内边距对称设为 `navRightGap`（=胶囊留白），logo 在整栏正中且不与右侧胶囊重叠。
- **门店条**：`store-bar.wxml` 切换/导航由 CSS 边框圆环 + 字符 `➤` 改为 `<image src="/assets/icons/home-{switch,nav}.png" mode="aspectFit">`；删除 `.sbar__ico-switch` / `.sbar__ico-nav` 伪元素样式。

### 涉及文件
- `mini-program/miniprogram/pages/home/home.{wxml,wxss}`
- `mini-program/miniprogram/components/store-bar/store-bar.{wxml,wxss}`
- `mini-program/miniprogram/assets/icons/home-{ticket,coupon,activity,switch,nav}.png`

## 2026-07-14 首页会员卡入口图标放大改实心

用户反馈：首页会员卡里的三个入口图标（我的入场券、我的优惠券、活动列表）太小、线条太细，看起来小气。

要求：三个入口图标放大并改成实心/半实心黑色，视觉更稳重大气，去掉细线风格；保持透明背景 PNG、黑白主色、无白底块；不动门店条与 tabbar 及其他页面。

处理：
- 重绘 `home-ticket.png`（实心票券+边缘缺口+虚线撕口）、`home-coupon.png`（实心标签牌+镂空孔）、`home-activity.png`（实心日历），均为 192×192 RGBA 透明实心黑图。
- `home.wxss` 中 `.home__entry-ic` 显示尺寸由 48rpx 放大到 60rpx。
- `home.wxml` 三枚入口仍引用上述三个资源，未改动。

## 2026-07-14 首页会员卡入口图标：实心改回稳重线性版

背景：用户反馈上一版实心图标不满意，决定改回线性风格；但最早那版线性太细、不大气，需要重新设计更粗、更稳重的线条。

图标板（gpt-image-2 v2）：`output/imagegen/home-entry-line-icons-gpt-image-2-v2.png`
提示词源文件：`tmp/imagegen/home-entry-line-icons-v2-prompt.txt`
（提示词方向：#111111 monoline，5-6px 等效粗中等描边、圆角端点、无填充实心；TICKET=带缺口+虚线撕口的入场券，COUPON=带百分号的折叠优惠券，EVENTS=日历/活动列表混合。）

处理：
- 从图标板裁切 TICKET/COUPON/EVENTS 三枚线性图标，按亮度反相生成 alpha（保留抗锯齿、无白底块），描边填 #111111，输出 256×256 RGBA 透明 PNG。
- 覆盖 `home-ticket.png`、`home-coupon.png`、`home-activity.png`，替换掉上一版实心图标。
- `home.wxss` 中 `.home__entry-ic` 显示尺寸保持 60rpx（线性图在 60rpx 下清晰协调，无需调整）。
- `home.wxml` 三枚入口仍引用上述三个资源，未改动文字/布局/门店条/tabbar。

---

## 2026-07-14 · 活动列表（近期活动）深色返工 + gpt-image-2 空状态图标

### 用户要求
1. 自定义导航（`navigationStyle: custom`），去掉系统白色导航；页面背景改用 demo 深色黑灰渐变（顶部 #303030 → 底部近黑），标题白色。
2. 空状态图标用 gpt-image-2 生成的纸片/邀请函图形，裁切透明化为 `assets/empty/activity-empty.png`，去掉白底块；文案「还没有新的内容哦，请期待一下吧」。
3. 去掉「我的入场券」入口。
4. 去掉「报名中」状态展示（本页仅展示报名中数据）。
5. 活动卡片更紧凑，做扇形/相册堆叠感，保留左右滑动，卡片不再白底系统感。

### gpt-image-2 空状态图标
- 图标板：`output/imagegen/activity-empty-icon-gpt-image-2-v1.png`（1024×1024，白底居中的深色折叠纸片 + 上方小卷角）
- 提示词源文件：`tmp/imagegen/activity-empty-icon-prompt.txt`（use case: logo-brand；abstract folded ticket/invitation paper，charcoal #202020 + highlight #8a8a8a，纯白底、无文字、易抠图）

### 图标裁切/处理（一次性 PIL + numpy 处理，脚本未保留）
- 按 darkness `d = 255 − L` 检测内容包围盒（阈值 18），加 24px padding 裁切。
- 透明化：`alpha = clip(d*3)` —— 纸片主体与切线保持不透明，仅真实边缘羽化，白底自然抠掉、无白块。
- 重上色：`gray = clip(95 + L*0.55)`（灰度随源亮度提升），把深炭色纸片映射为在深色页面上清晰可读的浅灰纸片、切线更亮，贴合 demo 质感。
- 居中到方形画布，输出 `mini-program/miniprogram/assets/empty/activity-empty.png`（420×420 RGBA 透明，alpha extrema 0–255）。
- 深色预览：`tmp/imagegen/activity-empty-preview.png`。

### 页面设计调整
- `activity-list.json`：`navigationStyle: custom`、`backgroundColor: #171717`。
- 自定义 nav：`measureNav()`（复用 home 方案，`wx.getWindowInfo` + `getMenuButtonBoundingClientRect`）算状态栏高/胶囊高/右侧避让；左侧 CSS 箭头返回、标题「近期活动」居中避让胶囊。
- 背景：`.al` 用 `linear-gradient(180deg,#303030,#1c1c1c 42%,#0c0c0c)`；主标题 + 分隔条改白色/半透明白融入深色。
- 扇形堆叠：swiper `previous/next-margin 120rpx`，JS `decorate()` 给每张卡标 `active/left/right/back`，CSS 对应 `scale + rotate(±7deg) + translateY + 递减 opacity + z-index`，卡高 560rpx（较原 720rpx 更紧凑）。
- summary 只保留标题 / 时间 / 门店 / 「查看详情」白色 pill；删除报名中 pill、我的入场券卡片、历史活动 section（本页仅报名中数据）。
- 空状态：深色页居中显示上述透明 PNG + 文案。

### 涉及文件
- `mini-program/miniprogram/pages/activity-list/activity-list.{json,wxml,wxss,js}`
- `mini-program/miniprogram/assets/empty/activity-empty.png`（新增）
- 空状态 PNG 由一次性 PIL + numpy 处理生成（裁切/透明化/重上色/居中，方法见上），临时脚本未保留

### 验证
- JSON 合法；`eslint` 0 error（1 个既有风格 warning：catch(e) 未用，与 home.js 一致）；`tsc --noEmit` 通过；`node scripts/build.js` OK（32 pages / 13 components）。
- WXML 已无「我的入场券」「报名中」及状态 pill；JS 以 `status==='enrolling'` 过滤，无报名中活动时进入空状态。
- 空状态 PNG 为 RGBA 透明（alpha 0–255），白底块已去除。

## 活动列表 — tower-swiper 塔式堆叠（升级）

原生 `swiper` + `decorate()` 手写 CSS 堆叠替换为本地化组件 `components/tower-swiper`（思路参考 zenochan/tower-swiper，未装 npm）。

### 组件 `ic-tower-swiper`
- 属性 `list`（`{id,imageUrl,title,tone}` 数组）；事件 `change{current}`、`tap{item,index}`。海报复用 `ic-poster`（全局注册），边框/阴影在组件 wxss。
- 卡片绝对定位叠放，每张按「与主卡的有符号偏移 `o=i-pos`」算 `translateX(118rpx·e)+translateY(26rpx·|e|)+rotate(8deg·e)+scale(1-0.12|e|)+opacity+z-index`（`e` 在 ±RANGE=2 内截断），`transform-origin: center bottom` 使旋转呈开扇效果；主卡 `is-active` 边框最亮、阴影最深。
- 手势：`touchstart/move/end` 记录位移，拖动时连续 `pos=current-dx/unit`（越界橡皮筋 0.35），`is-dragging` 关掉 transition 让卡跟手；抬手按阈值 0.22·unit 进位并 `triggerEvent('change')` 带 transition 归位。点侧卡置前、点主卡触发 `tap`（`_moved` 区分滑动/点击）。
- 入场动画：`list` observer 先以 `entered=false` 渲染（全部叠在中心、按深度缩放变暗），90ms 后置 `entered=true` 重渲染，靠 wxss transition 扇形展开。
- 竖向长方形海报 440×660rpx；黑底白色半透明粗边框（主卡 0.55 / 邻卡 0.24，6rpx）。

### 页面改动
- `activity-list.json` 增 `usingComponents.ic-tower-swiper`。
- wxml 用 `<ic-tower-swiper>` 取代 swiper；`onCardChange` 更新 `current/currentItem`，`onCardTap` 进详情。
- js 删除 `decorate/fanReady/onSwiperChange/goDetailBy`；wxss 删除 `.al__swiper/.al__card*` 全套。保留 nav/背景/空状态/summary/仅 enrolling 约束不变。

### 验证
- `eslint` 0 error（仅既有 catch(e) warning）；`tsc --noEmit` 通过；`node scripts/build.js` OK（32 pages / 14 component references）。

### 堆叠收紧（微调）
- 反馈：入场动画有了但展开态左右卡分得太开、跑出视口。只调 `render()` 里 `entered=true` 分支的展开参数，不动入场动画与环形手势、不动 wxss 几何与页面。
- 相邻卡 `tx 250→160rpx`、`rot 16→10deg`、`ty 90→55rpx`、`scale 0.94`、`opacity 0.58`；第二层轻微外扩 `tx +45 / ty +25 / rot +5`。主卡压住邻卡、只露侧+底边，回到紧凑相册堆叠。
- 保留：collapsed→entered 入场展开、`circularOffset` 环形手势（0↔n-1 双向绕环）。
