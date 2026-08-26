// 支付结果 — 成功/失败 + 后续入口
const TYPE_ORDER = { food: 'food', activity: 'activity', recharge: 'recharge', coupon: 'coupon' };

Page({
  data: { success: true, amount: '', type: 'food', orderId: '', title: '支付成功', desc: '' },

  onLoad(options) {
    const success = (options.status || 'success') === 'success';
    const type = options.type || 'food';
    const eventCouponUsed = success && type === 'coupon';
    this.setData({
      success,
      type,
      orderId: options.id || '',
      amount: options.amount ? decodeURIComponent(options.amount) : '',
      title: eventCouponUsed ? '使用成功' : (success ? '支付成功' : '支付未完成'),
      desc: eventCouponUsed
        ? '赛事券已核销，小票正在打印'
        : (success ? '订单已提交，可在订单中心查看' : '如已扣款请稍后在订单中心确认'),
    });
  },

  viewOrder() {
    if (this.data.type === 'coupon' && this.data.orderId) {
      wx.redirectTo({ url: `/pages/redemption-order-detail/redemption-order-detail?id=${this.data.orderId}` });
      return;
    }
    const orderType = TYPE_ORDER[this.data.type] || 'food';
    wx.redirectTo({ url: '/pages/order-center/order-center?type=' + orderType });
  },

  goHome() {
    wx.switchTab({ url: '/pages/index/index' });
  },
});
