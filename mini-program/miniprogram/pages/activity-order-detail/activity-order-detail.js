// 活动订单详情 — 活动信息 + 核销码 + 支付金额
// Reference: design/mini-program/final/member-subpages/05-activity-order-detail-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');
const codeart = require('../../utils/codeart');
const { ORDER_STATUS_LABEL } = require('../../constants/index');

Page({
  data: { loading: true, order: null, qr: [], showCode: false },

  onLoad(options) {
    api
      .getActivityOrder(options.id)
      .then((res) => {
        const d = res.data || {};
        // The verify code is available when the order contains an active ticket.
        const hasUsableTicket = Array.isArray(d.tickets) && d.tickets.some((t) => t.status === 'active');
        const showCode = hasUsableTicket;
        this.setData({
          loading: false,
          showCode,
          qr: showCode ? codeart.grid(d.verifyCode || d.orderNo) : [],
          order: {
            orderNo: d.orderNo,
            statusText: ORDER_STATUS_LABEL[d.status] || d.status,
            activityId: d.activityId,
            activityTitle: d.activityTitle,
            timeText: d.timeText,
            storeName: d.storeName,
            ticketName: d.ticketName,
            qty: d.qty,
            verifyCode: fmt.codeGroups(d.verifyCode),
            payChannelText: d.payChannel === 'coupon'
              ? '优惠券兑换'
              : (d.payChannel === 'coin' ? '金币支付' : '微信支付'),
            amountText: fmt.centToYuan(d.amountCent),
            createdText: fmt.dateTime(d.createdAt),
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  copyCode() {
    if (this.data.order.verifyCode) ui.copy(this.data.order.verifyCode, '核销码已复制');
  },

  goActivity() {
    const id = this.data.order && this.data.order.activityId;
    if (id) wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + id });
  },
});
