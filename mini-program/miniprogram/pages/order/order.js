// 点餐 — 左侧分类栏 + 右侧两列商品格 + 底部购物车（线下门店点餐）
// Reference: design/mini-program/generated/v20/03-ordering-v20-compact-default-source.png
//            design/mini-program/generated/v20/03-ordering-v20-compact-active-cart-source.png
const api = require('../../services/api');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const draft = require('../../utils/order-draft');
const memberAccess = require('../../utils/member-access');

const SELECTED_COUPON_KEY = 'ic_selected_coupon_v1';

const CATEGORY_ICONS = {
  1: '/assets/menu-category/voucher-redemption.svg',
  2: '/assets/menu-category/whisky-neat.svg',
  3: '/assets/menu-category/soft-drinks.svg',
  4: '/assets/menu-category/classic-beer.svg',
  6: '/assets/menu-category/voucher-sets.svg',
  24: '/assets/menu-category/classic-cocktails.svg',
  26: '/assets/menu-category/signature-drinks.svg',
  27: '/assets/menu-category/signature-drinks.svg',
};

function categoryIcon(category) {
  return CATEGORY_ICONS[String(category.id)] || category.imageUrl || '';
}

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
    groups: [], // [{ id, name, iconUrl, items: [{...item, qty, priceText, soldOut }] }]
    items: [],
    itemsLoading: false,
    activeCat: '',
    activeCatName: '',
    activeCatType: 'product',
    cartCount: 0,
    cartItems: [],
    totalText: '0.00',
    selectedCoupon: null,
    stores: [],
    showStoreSheet: false,
    showCartSheet: false,
  },

  onLoad() {
    this.qty = {}; // itemId -> qty
    this._catalogRequestId = 0;
    this._itemRequestId = 0;
    this._catalogStoreId = '';
    this.load();
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 2, hidden: this.data.showStoreSheet });
    }
    if (draft.consumeCompletion('food')) {
      this.clearCart();
    }
    this.syncSelectedCoupon();
    const s = storeCtx.get();
    if (s) this.useStore(s);
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

  load() {
    // Populate the in-page store-switch sheet, and resolve the current store
    // (persisted pick, else nearest) before loading its catalog.
    this.loadStores();
    storeCtx.ensureStore().then((store) => {
      if (store) this.useStore(store);
      else this.setData({ loading: false });
    });
  },

  useStore(store, force) {
    const storeId = String(store.id);
    const changed = !this.data.store || String(this.data.store.id) !== storeId;
    if (changed || force) {
      this.qty = {};
      this.totalCent = 0;
      this.setData({
        store,
        cartCount: 0,
        cartItems: [],
        totalText: '0.00',
      });
    } else {
      this.setData({ store });
    }
    if (force) this._catalogStoreId = '';
    if (this._catalogStoreId !== storeId) this.loadCatalog(store.id);
  },

  loadStores() {
    return storeCtx.listNearby().then((stores) => {
      this.setData({ stores });
      return stores;
    });
  },

  loadCatalog(storeId) {
    const requestId = ++this._catalogRequestId;
    this._catalogStoreId = String(storeId);
    this.setData({
      loading: true,
      itemsLoading: false,
      groups: [],
      items: [],
      activeCat: '',
      activeCatName: '',
    });
    api
      .getCategories(storeId)
      .then((catRes) => {
        if (requestId !== this._catalogRequestId) return;
        const groups = (catRes.data || []).map((category) => ({
          id: category.id,
          name: category.name,
          iconUrl: categoryIcon(category),
          categoryType: category.categoryType || 'product',
          items: [],
        }));
        const first = groups[0] || null;
        this.setData({
          groups,
          activeCat: first ? first.id : '',
          activeCatName: first ? first.name : '',
          activeCatType: first ? first.categoryType : 'product',
          loading: false,
        });
        if (first) return this.loadCategoryItems(storeId, first.id, requestId);
      })
      .catch(() => {
        if (requestId === this._catalogRequestId) this.setData({ loading: false });
      });
  },

  loadCategoryItems(storeId, categoryId, catalogRequestId) {
    const requestId = ++this._itemRequestId;
    this.setData({ itemsLoading: true, items: [] });
    return api
      .getItems(storeId, { categoryId, pageSize: 100 })
      .then((itemRes) => {
        if (
          catalogRequestId !== this._catalogRequestId ||
          requestId !== this._itemRequestId ||
          String(this.data.activeCat) !== String(categoryId)
        ) {
          return;
        }
        const items = (itemRes.data || []).map((item) => this.decorateItem(item));
        const groups = this.data.groups.map((group) =>
          String(group.id) === String(categoryId) ? Object.assign({}, group, { items }) : group
        );
        this.setData({ groups, items, itemsLoading: false });
      })
      .catch(() => {
        if (requestId === this._itemRequestId) this.setData({ itemsLoading: false, items: [] });
      });
  },

  decorateItem(it) {
    const priceText = formatMenuPrice(it.priceCent);
    const yuanDigits = String(Math.floor((it.priceCent || 0) / 100)).length;
    return {
      id: it.id,
      categoryId: it.categoryId,
      name: it.name,
      description: (it.description || '').trim(),
      initial: (it.name || '餐').slice(0, 1),
      priceCent: it.priceCent,
      priceText,
      priceCompact: yuanDigits >= 3,
      imageUrl: it.imageUrl || '',
      payChannels: it.payChannels,
      payText: (it.payChannels || []).map((channel) => channel === 'coin' ? '金币' : '微信').join(' / ') + '可用',
      itemType: it.itemType || 'food',
      isCoupon: it.itemType === 'coupon',
      pointsReward: Number(it.pointsReward || 0),
      stock: it.stock,
      soldOut: it.stock <= 0,
      qty: this.qty[it.id] || 0,
    };
  },

  onPickCat(e) {
    const id = e.currentTarget.dataset.id;
    if (String(id) === String(this.data.activeCat)) return;
    const category = this.data.groups.find((group) => String(group.id) === String(id));
    if (!category || !this.data.store) return;
    this.setData({ activeCat: category.id, activeCatName: category.name, activeCatType: category.categoryType });
    this.loadCategoryItems(this.data.store.id, category.id, this._catalogRequestId);
  },

  onQtyChange(e) {
    memberAccess.requireCompleteProfile(() => this.applyQtyChange(e));
  },

  applyQtyChange(e) {
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
    const items = this.data.items;
    const activeItem = items.find((item) => item.id === id);
    if (activeItem) activeItem.qty = qty;
    this.setData({ groups, items });
    this.recalc();
  },

  recalc() {
    let count = 0;
    let totalCent = 0;
    const cartItems = [];
    this.data.groups.forEach((g) =>
      g.items.forEach((it) => {
        if (it.qty > 0) {
          count += it.qty;
          totalCent += it.qty * it.priceCent;
          cartItems.push(Object.assign({}, it, { lineTotalText: fmt.centToYuan(it.qty * it.priceCent) }));
        }
      })
    );
    this.totalCent = totalCent;
    this.setData({ cartCount: count, cartItems, totalText: fmt.centToYuan(totalCent) });
    if (!count && this.data.showCartSheet) this.setCartSheetVisible(false);
  },

  clearCart() {
    this.qty = {};
    this.totalCent = 0;
    const resetItem = (item) => Object.assign({}, item, { qty: 0 });
    const groups = this.data.groups.map((group) =>
      Object.assign({}, group, { items: group.items.map(resetItem) })
    );
    const items = this.data.items.map(resetItem);
    this.setData({ groups, items, cartCount: 0, cartItems: [], totalText: '0.00' });
  },

  openCartSheet() {
    if (!this.data.cartCount) {
      ui.toast('购物车还是空的');
      return;
    }
    this.setCartSheetVisible(true);
  },

  closeCartSheet() {
    this.setCartSheetVisible(false);
  },

  setCartSheetVisible(show) {
    this.setData({ showCartSheet: show });
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ hidden: show });
      return;
    }
    const toggleTabBar = show ? wx.hideTabBar : wx.showTabBar;
    if (typeof toggleTabBar === 'function') toggleTabBar({ animation: false });
  },

  onClearCart() {
    this.clearCart();
    this.setCartSheetVisible(false);
  },

  openStoreSheet() {
    if (this.data.stores.length) {
      this.setStoreSheetVisible(true);
      return;
    }
    this.loadStores().then(() => this.setStoreSheetVisible(true));
  },

  closeStoreSheet() {
    this.setStoreSheetVisible(false);
  },

  setStoreSheetVisible(show) {
    this.setData({ showStoreSheet: show });
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ hidden: show });
      return;
    }
    const toggleTabBar = show ? wx.hideTabBar : wx.showTabBar;
    if (typeof toggleTabBar === 'function') toggleTabBar({ animation: false });
  },

  onSelectStore(e) {
    const store = this.data.stores.find((s) => s.id === e.currentTarget.dataset.id);
    if (!store) return;
    storeCtx.set(store);
    this.setStoreSheetVisible(false);
    this.useStore(store, true);
  },

  onCheckout() {
    memberAccess.requireCompleteProfile(() => this.checkout());
  },

  checkout() {
    if (!this.data.cartCount) {
      ui.toast('请先选择商品');
      return;
    }
    const hasCouponProduct = this.data.cartItems.some((item) => item.isCoupon);
    if (hasCouponProduct && this.data.selectedCoupon) {
      ui.toast('券商品不能使用券兑换，请先移除已选券');
      return;
    }
    if (this.data.showCartSheet) this.setCartSheetVisible(false);
    const lines = [];
    this.data.groups.forEach((g) =>
      g.items.forEach((it) => {
        if (it.qty > 0) {
          lines.push({
            id: it.id,
            name: it.name,
            qty: it.qty,
            priceCent: it.priceCent,
            imageUrl: it.imageUrl,
            payChannels: it.payChannels,
            itemType: it.itemType,
            pointsReward: it.pointsReward,
          });
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
