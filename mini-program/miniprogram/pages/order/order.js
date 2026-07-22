// 点餐 — 左侧分类栏 + 右侧两列商品格 + 底部购物车（线下门店点餐）
// Reference: design/mini-program/generated/v20/03-ordering-v20-compact-default-source.png
//            design/mini-program/generated/v20/03-ordering-v20-compact-active-cart-source.png
const api = require('../../services/api');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const draft = require('../../utils/order-draft');

const SELECTED_COUPON_KEY = 'ic_selected_coupon_v1';

function formatMenuPrice(cent) {
  const n = Number(cent || 0);
  if (n % 100 === 0) return String(Math.floor(n / 100));
  return (n / 100).toFixed(2);
}

Page({
  data: {
    loading: true,
    store: null,
    tableText: '',
    groups: [], // [{ id, name, items: [{...item, qty, priceText, soldOut }] }]
    activeCat: '',
    intoView: '',
    cartCount: 0,
    totalText: '0.00',
    selectedCoupon: null,
    stores: [],
    showStoreSheet: false,
    // custom navigation metrics (px)
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
  },

  onLoad() {
    this.qty = {}; // itemId -> qty
    this.measureNav();
    this.load();
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 2 });
    }
    this.syncSelectedCoupon();
    const s = storeCtx.get();
    if (s && this.data.store && s.id !== this.data.store.id) {
      this.setData({ store: s });
      this.loadCatalog(s.id);
    }
  },

  syncSelectedCoupon() {
    try {
      this.setData({ selectedCoupon: wx.getStorageSync(SELECTED_COUPON_KEY) || null });
    } catch {
      this.setData({ selectedCoupon: null });
    }
  },

  clearSelectedCoupon() {
    try {
      wx.removeStorageSync(SELECTED_COUPON_KEY);
    } catch {}
    this.setData({ selectedCoupon: null });
  },

  // Size the custom nav bar to the status bar + WeChat capsule button.
  measureNav() {
    try {
      const win = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const cap = wx.getMenuButtonBoundingClientRect();
      const statusBar = win.statusBarHeight || 20;
      const gap = Math.max(cap.top - statusBar, 4);
      this.setData({
        navStatusBar: statusBar,
        navContentHeight: cap.height + gap * 2,
        navRightGap: Math.max(win.windowWidth - cap.left + 8, 96),
      });
    } catch {
      /* keep defaults */
    }
  },

  load() {
    // Populate the in-page store-switch sheet, and resolve the current store
    // (persisted pick, else nearest) before loading its catalog.
    this.loadStores();
    storeCtx.ensureStore().then((store) => {
      this.setData({ store });
      if (store) this.loadCatalog(store.id);
      else this.setData({ loading: false });
    });
  },

  loadStores() {
    return storeCtx.listNearby().then((stores) => {
      this.setData({ stores });
      return stores;
    });
  },

  loadCatalog(storeId) {
    this.setData({ loading: true });
    Promise.all([api.getCategories(storeId), api.getItems(storeId)])
      .then(([catRes, itemRes]) => {
        const cats = catRes.data || [];
        const items = itemRes.data || [];
        const groups = cats
          .map((c) => ({
            id: c.id,
            name: c.name,
            items: items
              .filter((it) => it.categoryId === c.id)
              .map((it) => this.decorateItem(it)),
          }))
          .filter((g) => g.items.length);
        this.setData({
          groups,
          activeCat: groups.length ? groups[0].id : '',
          loading: false,
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  decorateItem(it) {
    const priceText = formatMenuPrice(it.priceCent);
    const yuanDigits = String(Math.floor((it.priceCent || 0) / 100)).length;
    return {
      id: it.id,
      categoryId: it.categoryId,
      name: it.name,
      priceCent: it.priceCent,
      priceText,
      priceCompact: yuanDigits >= 3,
      imageUrl: it.imageUrl || '',
      payChannels: it.payChannels,
      stock: it.stock,
      soldOut: it.stock <= 0,
      qty: this.qty[it.id] || 0,
    };
  },

  onPickCat(e) {
    const id = e.currentTarget.dataset.id;
    this.setData({ activeCat: id, intoView: 'cat-' + id });
  },

  onQtyChange(e) {
    const id = e.currentTarget.dataset.id;
    const qty = e.detail.value;
    if (qty > 0) this.qty[id] = qty;
    else delete this.qty[id];
    // update the single item's qty in place
    const groups = this.data.groups;
    for (const g of groups) {
      const it = g.items.find((x) => x.id === id);
      if (it) {
        it.qty = qty;
        break;
      }
    }
    this.setData({ groups });
    this.recalc();
  },

  recalc() {
    let count = 0;
    let totalCent = 0;
    this.data.groups.forEach((g) =>
      g.items.forEach((it) => {
        if (it.qty > 0) {
          count += it.qty;
          totalCent += it.qty * it.priceCent;
        }
      })
    );
    this.totalCent = totalCent;
    this.setData({ cartCount: count, totalText: fmt.centToYuan(totalCent) });
  },

  openStoreSheet() {
    if (this.data.stores.length) {
      this.setData({ showStoreSheet: true });
      return;
    }
    this.loadStores().then(() => this.setData({ showStoreSheet: true }));
  },

  closeStoreSheet() {
    this.setData({ showStoreSheet: false });
  },

  onSelectStore(e) {
    const store = this.data.stores.find((s) => s.id === e.currentTarget.dataset.id);
    if (!store) return;
    storeCtx.set(store);
    this.qty = {};
    this.totalCent = 0;
    this.setData({
      store,
      showStoreSheet: false,
      cartCount: 0,
      totalText: '0.00',
    });
    this.loadCatalog(store.id);
  },

  onCheckout() {
    if (!this.data.cartCount) {
      ui.toast('请先选择商品');
      return;
    }
    const lines = [];
    this.data.groups.forEach((g) =>
      g.items.forEach((it) => {
        if (it.qty > 0) {
          lines.push({ id: it.id, name: it.name, qty: it.qty, priceCent: it.priceCent, imageUrl: it.imageUrl, payChannels: it.payChannels });
        }
      })
    );
    draft.set({
      type: 'food',
      store: this.data.store,
      tableText: this.data.tableText,
      items: lines,
      totalCent: this.totalCent,
      coupon: this.data.selectedCoupon,
    });
    wx.navigateTo({ url: '/pages/order-confirm/order-confirm' });
  },
});
