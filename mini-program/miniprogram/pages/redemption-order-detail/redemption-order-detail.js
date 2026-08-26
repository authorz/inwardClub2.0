// 兑换订单详情 — 兑换物 / 兑换码 / 门店确认状态
// Reference: design/mini-program/final/member-subpages/07-redemption-order-detail-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');
const { ORDER_STATUS_LABEL } = require('../../constants/index');

Page({
  data: { loading: true, order: null, qr: [], directUsed: false },

  onLoad(options) {
    api
      .getRedemptionOrder(options.id)
      .then((res) => {
        const d = res.data || {};
        const directUsed = d.couponType === 'event_ticket';
        this.setData({
          loading: false,
          directUsed,
          qr: directUsed ? [] : codeart.grid(d.code || d.orderNo),
          order: {
            id: d.id,
            orderNo: d.orderNo,
            statusText: directUsed ? '已使用' : (d.status === 'pending_verify' ? '待取用' : ORDER_STATUS_LABEL[d.status] || d.status),
            title: d.title,
            couponName: d.couponName,
            qty: d.qty || 1,
            validUntil: d.validUntil,
            code: fmt.codeGroups(d.code),
            storeName: d.storeName,
            createdText: fmt.dateTime(d.createdAt),
            statusHint: directUsed ? '赛事券已核销' : (d.status === 'pending_verify' ? '待工作人员确认' : ''),
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  copyCode() {
    if (this.data.order.code) ui.copy(this.data.order.code, '兑换码已复制');
  },

  cancel() {
    ui.confirm({ content: '确认取消该兑换？', confirmText: '取消兑换' }).then((ok) => {
      if (ok) {
        ui.success('已取消');
        setTimeout(() => wx.navigateBack(), 500);
      }
    });
  },
});
