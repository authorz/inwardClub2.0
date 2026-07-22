// 充值订单详情 — 充值金额 / 到账金币 / 赠送金币 / 支付时间
// Reference: design/mini-program/final/member-subpages/06-recharge-order-detail-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { ORDER_STATUS_LABEL } = require('../../constants/index');

Page({
  data: { loading: true, done: false, order: null },

  onLoad(options) {
    api
      .getRechargeOrder(options.id)
      .then((res) => {
        const d = res.data || {};
        this.setData({
          loading: false,
          done: d.status === 'completed',
          order: {
            orderNo: d.orderNo,
            statusText: d.status === 'completed' ? '充值成功' : ORDER_STATUS_LABEL[d.status] || d.status,
            amountText: fmt.centToYuan(d.amountCent),
            coins: d.coins,
            bonusCoins: d.bonusCoins || 0,
            payChannelText: '微信支付',
            paidText: fmt.dateTime(d.paidAt),
            creditedText: fmt.dateTime(d.creditedAt),
            note: d.note || '',
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  goLedger() {
    wx.navigateTo({ url: '/pages/wallet-ledger/wallet-ledger?asset=coins' });
  },
});
