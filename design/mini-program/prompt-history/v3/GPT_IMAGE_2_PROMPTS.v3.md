# gpt-image-2 小程序设计图提示词 v3

## 1. 本次修改来源

基于 v2 初验：

- `05-reservation-v2.png` 有预约确认弹层，但只生成了 8 个座位，不满足每桌 9 座硬要求。
- `03-ordering-v2.png` 结构正确，但商品图和支付图标偏彩色；为了贴近参考图黑白配色，补一张更严格灰阶版本。

## 2. 点餐商品选择 v3：严格灰阶

```text
Mobile WeChat mini program ordering screen, strict black-white grayscale UI. Left vertical category rail. Right side two-column product grid, exactly two products per row. Use monochrome grayscale product photos or grayscale placeholders, no colored packaging, no colored WeChat or coin icons. Each product cell has name, price, payment hints as black/gray text, and a quantity stepper with minus, number, plus. Fixed bottom cart bar with black capsule checkout button. Chinese text: 点餐, 酒水, 小吃, 饮品, 微信, 金币, 去结算. No Alipay. Avoid colorful food delivery style.
```

## 3. 预约桌子九座 v3：强制 9 个座位

```text
Mobile WeChat mini program reservation page, closely matching the provided reservation reference. White and light gray UI with black text. Top title 预约, current store row, 预约规则与牌桌礼仪. Show one table module: 1号桌, 更新时间 06:16:10, 基础积分 50/100, 座位 0/9, status 预约中. Inside the table module, draw one central poker table graphic and EXACTLY NINE separate seat buttons labeled 可预定. Seat placement must be: four seats along the top edge, one seat on the left side, one seat on the right side, and three seats along the bottom edge. After one seat is selected, show a bottom confirmation sheet with 已选择 1号桌 · 3号座位, 取消, 确认预约. Do not draw eight seats. Do not draw more than nine seats. Chinese UI, no Alipay, no sign-in.
```
