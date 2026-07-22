// 支付结果 — 成功/失败 + 后续入口
const TYPE_ORDER = { food: 'food', activity: 'activity', recharge: 'recharge', coupon: 'coupon' };

Page({
  data: { success: true, amount: '', type: 'food', title: '支付成功', desc: '' },

  onLoad(options) {
    const success = (options.status || 'success') === 'success';
    const type = options.type || 'food';
    this.setData({
      success,
      type,
      amount: options.amount ? decodeURIComponent(options.amount) : '',
      title: success ? '支付成功' : '支付未完成',
      desc: success ? '订单已提交，可在订单中心查看' : '如已扣款请稍后在订单中心确认',
    });
  },

  viewOrder() {
    const orderType = TYPE_ORDER[this.data.type] || 'food';
    wx.redirectTo({ url: '/pages/order-center/order-center?type=' + orderType });
  },

  goHome() {
    wx.switchTab({ url: '/pages/index/index' });
  },
});
