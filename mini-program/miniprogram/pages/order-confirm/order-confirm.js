// 订单确认 / 付款 — 线下门店点餐付款（微信 / 金币，单选，不支持组合支付）
// Reference: design/mini-program/final/order-confirmation/04-order-confirmation-final-payment.png
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const pay = require('../../utils/pay');
const draft = require('../../utils/order-draft');

// 商品缺失 payChannels 时的兼容默认（老数据视为两种都支持）
const DEFAULT_CHANNELS = ['wechat', 'coin'];

Page({
  data: {
    ready: false,
    store: null,
    tableText: '',
    items: [],
    note: '',
    supportsWechat: true,
    supportsCoin: true,
    payMethod: '', // '' | 'wechat' | 'coin'，单选
    coinDisabled: false,
    coinBalance: 0,
    goodsText: '0.00',
    payText: '0.00',
    submitting: false,
  },

  onLoad() {
    const d = draft.get();
    if (!d || d.type !== 'food') {
      this.setData({ ready: true });
      return;
    }
    const items = d.items.map((it) => ({
      id: it.id, // kept so createFoodOrder can send the server's itemId
      name: it.name,
      qty: it.qty,
      imageUrl: it.imageUrl || '',
      priceCent: it.priceCent,
      payChannels: it.payChannels,
      lineText: fmt.centToYuan(it.priceCent * it.qty),
    }));
    // 所有商品都支持某渠道，才算整单支持；缺失按兼容默认
    const channelsOf = (it) =>
      Array.isArray(it.payChannels) && it.payChannels.length ? it.payChannels : DEFAULT_CHANNELS;
    const supportsWechat = items.every((it) => channelsOf(it).indexOf('wechat') !== -1);
    const supportsCoin = items.every((it) => channelsOf(it).indexOf('coin') !== -1);
    this.totalCent = d.totalCent;
    this.setData({
      ready: true,
      store: d.store,
      tableText: d.tableText || '',
      items,
      supportsWechat,
      supportsCoin,
      goodsText: fmt.centToYuan(d.totalCent),
      payText: fmt.centToYuan(d.totalCent),
    });
    api.getWallet().then((res) => {
      const coins = (res.data && res.data.coins) || 0; // 1 金币 = 1 分
      // 余额不足以支付整单，或余额为 0，则金币不可选
      this.setData({
        coinBalance: coins,
        coinDisabled: coins <= 0 || coins < this.totalCent,
      });
      this.applyDefaults();
    });
    this.applyDefaults();
  },

  // 单选默认：优先微信；不支持微信但金币可用时默认金币
  applyDefaults() {
    const { supportsWechat, supportsCoin, coinDisabled } = this.data;
    let payMethod = '';
    if (supportsWechat) payMethod = 'wechat';
    else if (supportsCoin && !coinDisabled) payMethod = 'coin';
    this.setData({ payMethod });
  },

  selectWechat() {
    if (!this.data.supportsWechat) return;
    this.setData({ payMethod: 'wechat' });
  },

  selectCoin() {
    if (!this.data.supportsCoin || this.data.coinDisabled) return;
    this.setData({ payMethod: 'coin' });
  },

  onNote(e) {
    this.setData({ note: e.detail.value });
  },

  confirmPay() {
    const store = this.data.store;
    if (!store || this.data.submitting) return;
    const payChannel = this.data.payMethod;
    if (payChannel !== 'wechat' && payChannel !== 'coin') {
      ui.toast('请选择支付方式');
      return;
    }
    this.setData({ submitting: true });
    ui.showLoading('支付中');
    api
      .createFoodOrder(
        {
          storeId: store.id,
          tableText: this.data.tableText,
          note: this.data.note,
          items: this.data.items,
          totalCent: this.totalCent,
          payChannel,
        },
        http.uuid()
      )
      .then((res) => {
        const poid = (res.data && res.data.paymentOrderId) || 'po_food';
        return payChannel === 'coin'
          ? api.payByCoin(poid, http.uuid())
          : api.payWechatJsapi(poid, http.uuid()).then((r) => pay.settle(r));
      })
      .then(() => {
        ui.hideLoading();
        this.setData({ submitting: false });
        draft.clear();
        wx.redirectTo({ url: '/pages/pay-result/pay-result?type=food&status=success&amount=' + this.data.payText });
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '支付失败');
      });
  },

  editStore() {
    wx.navigateTo({ url: '/pages/store-select/store-select' });
  },
});
