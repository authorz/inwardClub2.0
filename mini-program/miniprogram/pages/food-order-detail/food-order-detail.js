// 点餐订单详情 — 门店/桌台/明细/支付/金额
// Reference: design/mini-program/final/member-subpages/04-food-order-detail-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { ORDER_STATUS_LABEL, ORDER_STATUS_TONE } = require('../../constants/index');

function payChannelText(ch) {
  if (!ch) return '';
  if (ch.indexOf('wechat') >= 0 && ch.indexOf('coin') >= 0) return '微信支付 + 金币抵扣';
  if (ch === 'coin') return '金币支付';
  if (ch === 'coupon') return '优惠券兑换';
  return '微信支付';
}

Page({
  data: { loading: true, order: null, done: false },

  onLoad(options) {
    api
      .getFoodOrder(options.id)
      .then((res) => {
        const d = res.data || {};
        const isCoupon = d.payChannel === 'coupon';
        const discountCent = isCoupon ? d.totalCent : (d.discountCent || 0);
        const items = (d.items || []).map((it) => ({
          itemId: it.itemId,
          name: it.name,
          qty: it.qty,
          imageUrl: it.imageUrl || '',
          amountText: fmt.centToYuan((it.priceCent || 0) * (it.qty || 1)),
        }));
        this.setData({
          loading: false,
          done: (ORDER_STATUS_TONE[d.status] || '') === 'done',
          order: {
            orderNo: d.orderNo,
            statusText: ORDER_STATUS_LABEL[d.status] || d.status,
            storeName: d.storeName,
            tableText: d.tableText,
            timeText: fmt.dateTime(d.createdAt),
            payChannelText: payChannelText(d.payChannel),
            goodsText: fmt.centToYuan(d.totalCent),
            showDiscount: discountCent > 0,
            discountLabel: isCoupon ? '优惠券抵扣' : '优惠抵扣',
            discountText: fmt.centToYuan(discountCent),
            payText: fmt.centToYuan(d.payCent != null ? d.payCent : d.totalCent),
            refundText: d.refundCent ? fmt.centToYuan(d.refundCent) : '',
            items,
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  reorder() {
    wx.switchTab({ url: '/pages/order/order' });
  },
});
