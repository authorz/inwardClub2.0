// 排队等候 — 加入队列并展示当前排位
const api = require('../../services/api');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const http = require('../../utils/request');

Page({
  data: { loading: true, entry: null, storeName: '' },

  onLoad() {
    // One idempotency key for this waitlist session: 刷新排位 re-issues the same
    // POST, so a stable key lets the backend return the existing entry instead
    // of enqueuing a duplicate on every refresh.
    this._joinKey = http.uuid();
    // Ensure a store (nearest by default) before joining — waitlist is store-scoped.
    storeCtx.ensureStore().then((store) => {
      this.setData({ storeName: store ? store.name : '' });
      this.join(store);
    });
  },

  join(store) {
    this.setData({ loading: true });
    api
      .joinWaitlist({ storeId: store && store.id }, this._joinKey)
      .then((res) => this.setData({ entry: res.data || {}, loading: false }))
      .catch(() => this.setData({ loading: false }));
  },

  refresh() {
    this.join(storeCtx.get());
    ui.toast('已刷新');
  },

  leave() {
    ui.confirm({ content: '确认离开排队队列？', confirmText: '离开' }).then((ok) => {
      if (ok) {
        ui.success('已离开队列');
        setTimeout(() => wx.navigateBack(), 500);
      }
    });
  },
});
