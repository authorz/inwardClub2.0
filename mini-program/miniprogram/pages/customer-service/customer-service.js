// 咨询客服 — 展示小程序当前门店的客服联系方式。
const api = require('../../services/api');
const storeCtx = require('../../utils/store-context');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');

Page({
  data: {
    loading: true,
    store: null,
    distanceText: '',
  },

  onLoad() {
    this.loadContactStore();
  },

  loadContactStore() {
    this.setData({ loading: true });
    storeCtx
      .ensureStore()
      .then((currentStore) => {
        if (!currentStore || !currentStore.id) return null;
        return api
          .getStore(currentStore.id)
          .then((res) => Object.assign({}, currentStore, res.data || {}))
          .catch(() => currentStore);
      })
      .then((store) => {
        this.setData({
          store,
          distanceText: store ? fmt.distance(store.distanceMeters) : '',
          loading: false,
        });
      })
      .catch(() => this.setData({ store: storeCtx.get() || null, loading: false }));
  },

  onPreviewQR() {
    const store = this.data.store;
    if (!store || !store.customerServiceQrUrl) return;
    wx.previewImage({
      current: store.customerServiceQrUrl,
      urls: [store.customerServiceQrUrl],
    });
  },

  onCall() {
    const store = this.data.store;
    if (!store || !store.phone) return ui.toast('该门店暂未设置联系电话');
    wx.makePhoneCall({ phoneNumber: store.phone });
  },

  onCopyPhone() {
    const store = this.data.store;
    if (store && store.phone) ui.copy(store.phone, '联系电话已复制');
  },
});
