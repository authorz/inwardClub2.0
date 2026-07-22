// Order list row (order center / order lists). One layout for food / activity /
// recharge / redemption orders — differences are just field values.
Component({
  properties: {
    title: { type: String, value: '' }, // 点餐订单 / 活动订单 ...
    code: { type: String, value: '' }, // #F2026071501
    status: { type: String, value: '' },
    statusMap: { type: String, value: 'order' },
    storeName: { type: String, value: '' },
    desc: { type: String, value: '' }, // 1号桌 · 5号座位 / 票名
    timeText: { type: String, value: '' },
    amountText: { type: String, value: '' }, // ¥56.00
    payText: { type: String, value: '' }, // 微信 / 金币
    last: { type: Boolean, value: false }, // 列表最后一项去掉底部分割线
  },
  methods: {
    onTap() { this.triggerEvent('tap'); },
  },
});
