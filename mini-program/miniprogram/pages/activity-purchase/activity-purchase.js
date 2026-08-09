const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const pay = require('../../utils/pay');
const { PAY_METHOD } = require('../../constants/index');
const validation = require('../../utils/validation');

Page({
  data: {
    loading: true,
    loadError: '',
    activity: null,
    tickets: [],
    ticketId: '',
    ticket: null,
    qty: 1,
    payMethod: PAY_METHOD.WECHAT,
    payMethods: [],
    totalText: '0.00',
    submitting: false,
  },

  onLoad(options) {
    const id = options.id;
    if (!id) {
      this.setData({ loading: false, loadError: '缺少活动信息，请返回后重试' });
      return;
    }
    api
      .getActivity(id)
      .then((res) => {
        const a = res.data || {};
        const activityPayChannels = a.payChannels && a.payChannels.length ? a.payChannels : [];
        const purchaseLimit = Math.max(0, Number(a.purchaseLimit) || 0);
        const now = Date.now();
        const tickets = (a.ticketTypes || []).map((t) => {
          const maxTicketsPerOrder = Math.max(0, Number(t.maxTicketsPerOrder) || 0);
          const stock = Number(t.stock);
          const saleStartAt = t.saleStartAt ? new Date(t.saleStartAt).getTime() : NaN;
          const saleEndAt = t.saleEndAt ? new Date(t.saleEndAt).getTime() : NaN;
          let disabledReason = '';
          if (stock === 0) disabledReason = '已售罄';
          else if (!isNaN(saleStartAt) && now < saleStartAt) {
            disabledReason = `${fmt.dateTime(t.saleStartAt, { relative: false })} 开售`;
          } else if (!isNaN(saleEndAt) && now > saleEndAt) disabledReason = '已停止售卖';
          const limits = [purchaseLimit, maxTicketsPerOrder, stock].filter((value) => value > 0);
          const stockText = stock < 0 ? '不限量' : `剩余 ${stock} 张`;
          const limitText = maxTicketsPerOrder
            ? `单次限购 ${maxTicketsPerOrder} 张`
            : purchaseLimit
              ? `每人限购 ${purchaseLimit} 张`
              : '不限购';
          return {
            id: t.id,
            name: t.name,
            priceCent: t.priceCent,
            priceText: fmt.centToYuan(t.priceCent),
            stockText,
            payChannels: t.payChannels && t.payChannels.length ? t.payChannels : activityPayChannels,
            maxQuantity: limits.length ? Math.min.apply(null, limits) : 99,
            limitText,
            disabled: Boolean(disabledReason),
            disabledReason,
            metaText: disabledReason || `${stockText} · ${limitText}`,
          };
        });
        const ticket = tickets.find((item) => !item.disabled) || null;
        const payMethods = ticket ? ticket.payChannels : [];
        const payMethod = payMethods.indexOf(PAY_METHOD.WECHAT) >= 0 ? PAY_METHOD.WECHAT : payMethods[0] || '';
        this.setData({
          loading: false,
          activity: {
            id: a.id,
            title: a.title,
            imageUrl: a.imageUrl || '',
            dateText: fmt.dateRange(a.startAt, a.endAt),
            storeName: a.storeName || '',
          },
          tickets,
          ticketId: ticket ? ticket.id : '',
          ticket,
          payMethods,
          payMethod,
          totalText: ticket ? ticket.priceText : '0.00',
        });
      })
      .catch(() => this.setData({ loading: false, loadError: '活动加载失败，请返回后重试' }));
  },

  onPickTicket(e) {
    const ticket = this.data.tickets.find((item) => item.id === e.currentTarget.dataset.id);
    if (!ticket) return;
    if (ticket.disabled) {
      ui.toast(ticket.disabledReason || '该票档当前不可购买');
      return;
    }
    this.selectTicket(ticket);
  },

  selectTicket(ticket) {
    const payMethods = ticket.payChannels;
    const payMethod = payMethods.indexOf(this.data.payMethod) >= 0 ? this.data.payMethod : payMethods[0] || '';
    this.setData({
      ticketId: ticket.id,
      ticket,
      qty: 1,
      payMethods,
      payMethod,
      totalText: ticket.priceText,
    });
  },

  onQty(e) {
    const qty = e.detail.value;
    const ticket = this.data.ticket;
    this.setData({
      qty,
      totalText: ticket ? fmt.centToYuan(ticket.priceCent * qty) : '0.00',
    });
  },

  onPay(e) {
    this.setData({ payMethod: e.detail.value });
  },

  noop() {},

  confirmPurchase() {
    const ticket = this.data.ticket;
    const activity = this.data.activity;
    const payMethod = this.data.payMethod;
    if (!ticket || !activity || this.data.submitting) return;
    if (this.data.payMethods.indexOf(payMethod) < 0) {
      ui.toast('请选择支付方式');
      return;
    }
	let quantity;
	try {
	  quantity = validation.integer(this.data.qty, { label: '购票数量', min: 1, max: Math.min(ticket.maxQuantity || 99, 99) });
	} catch (err) {
	  ui.toast(err.message);
	  return;
	}
	const amountCent = ticket.priceCent * quantity;
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    api
      .createActivityOrder(
		{ activityId: activity.id, ticketTypeId: ticket.id, qty: quantity, amountCent, payChannel: payMethod },
        http.uuid()
      )
      .then((res) => {
        const paymentOrderId = (res.data && res.data.paymentOrderId) || 'po_activity';
        return payMethod === PAY_METHOD.COIN
          ? api.payByCoin(paymentOrderId, http.uuid())
          : api.payWechatJsapi(paymentOrderId, http.uuid()).then((result) => pay.settle(result));
      })
      .then(() => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.success('购票成功');
        setTimeout(() => wx.redirectTo({ url: '/pages/tickets/tickets' }), 600);
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '购票失败');
      });
  },
});
