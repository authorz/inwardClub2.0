// 选择门店 — 按距离组织，含营业状态、导航、电话；点选即切换当前门店
// Reference: design/mini-program/generated/02-store-selection.png
const storeCtx = require('../../utils/store-context');
const fmt = require('../../utils/format');
const ui = require('../../utils/ui');

Page({
  data: {
    loading: true,
    stores: [],
    currentId: '',
  },

  onLoad() {
    const cur = storeCtx.get();
    this.setData({ currentId: cur ? cur.id : '' });
    // Location-aware, distance-sorted list (shared with the default-store resolve).
    storeCtx
      .listNearby()
      .then((list) => {
        const stores = (list || []).map((s) => ({
          id: s.id,
          name: s.name,
          open: s.open,
          hours: s.hours,
          address: s.address,
          addressDetail: s.addressDetail,
          phone: s.phone,
          lat: s.lat,
          lng: s.lng,
          distanceText: fmt.distance(s.distanceMeters),
          raw: s,
        }));
        this.setData({ stores, loading: false });
      })
      .catch(() => this.setData({ loading: false }));
  },

  onSelect(e) {
    const s = this.data.stores.find((x) => x.id === e.currentTarget.dataset.id);
    if (!s) return;
    storeCtx.switchStore(s.raw);
    ui.toast('已切换门店');
    setTimeout(() => wx.navigateBack(), 350);
  },

  onNav(e) {
    const s = this.data.stores.find((x) => x.id === e.currentTarget.dataset.id);
    if (s && s.lat && s.lng) {
      wx.openLocation({ latitude: s.lat, longitude: s.lng, name: s.name, address: (s.address || '') + (s.addressDetail || '') });
    }
  },

  onCall(e) {
    const s = this.data.stores.find((x) => x.id === e.currentTarget.dataset.id);
    if (s && s.phone) wx.makePhoneCall({ phoneNumber: s.phone });
  },

  goDetail(e) {
    wx.navigateTo({ url: '/pages/store-detail/store-detail?id=' + e.currentTarget.dataset.id });
  },
});
