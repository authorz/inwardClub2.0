// 门店详情 — 单店信息 + 导航/电话 + 设为当前门店
const api = require('../../services/api');
const storeCtx = require('../../utils/store-context');
const fmt = require('../../utils/format');

Page({
  data: {
    loading: true,
    store: null,
    distanceText: '',
  },

  onLoad(options) {
    api
      .getStore(options.id)
      .then((res) => {
        const s = res.data || {};
        this.setData({ store: s, distanceText: fmt.distance(s.distanceMeters), loading: false });
      })
      .catch(() => this.setData({ loading: false }));
  },

  onNav() {
    const s = this.data.store;
    if (s && s.lat && s.lng) {
      wx.openLocation({ latitude: s.lat, longitude: s.lng, name: s.name, address: (s.address || '') + (s.addressDetail || '') });
    }
  },

  onCall() {
    const s = this.data.store;
    if (s && s.phone) wx.makePhoneCall({ phoneNumber: s.phone });
  },

  setCurrent() {
    if (this.data.store) storeCtx.switchStore(this.data.store);
    wx.navigateBack();
  },
});
