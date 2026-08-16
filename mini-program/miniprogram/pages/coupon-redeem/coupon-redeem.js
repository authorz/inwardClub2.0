// 券兑换 — 复用点餐页的商品网格、选择栏和门店切换，不展示商品分类。
const api = require('../../services/api');
const http = require('../../utils/request');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');

const COUPON_TYPE_LABEL = {
  exchange: '兑换券',
  discount: '折扣券',
  cash: '代金券',
};

function formatMenuPrice(cent) {
  const value = Number(cent || 0);
  if (value % 100 === 0) return String(Math.floor(value / 100));
  return (value / 100).toFixed(2);
}

Page({
  data: {
    loading: true,
    submitting: false,
    store: null,
    stores: [],
    entitlementId: '',
    templateId: '',
    couponStoreId: '',
    couponType: '',
    couponTypeLabel: '优惠券',
    couponName: '',
    couponAmountCent: 0,
    couponAmountText: '0.00',
    validUntil: '',
    items: [],
    selectedItems: [],
    selectedCount: 0,
    totalCent: 0,
    totalText: '0.00',
    remainText: '0.00',
    canConfirm: false,
    showStoreSheet: false,
    showSelectionSheet: false,
    showUnderuseConfirm: false,
  },

  onLoad(query) {
    this._requestId = 0;
    const entitlementId = query.entitlementId || query.id || '';
    const couponType = query.couponType || query.type || '';
    const amountCent = Number(query.valueCent || query.amount || 0);
    this.setData({
      entitlementId,
      templateId: query.templateId || '',
      couponStoreId: query.storeId || '',
      couponType,
      couponTypeLabel: COUPON_TYPE_LABEL[couponType] || '优惠券',
      couponName: decodeURIComponent(query.name || ''),
      couponAmountCent: amountCent,
      couponAmountText: fmt.centToYuan(amountCent),
      remainText: fmt.centToYuan(amountCent),
      validUntil: decodeURIComponent(query.validUntil || ''),
    });
    if (!entitlementId) {
      this.setData({ loading: false });
      ui.toast('优惠券参数不正确');
      return;
    }
    this.load();
  },

  load() {
    Promise.all([this.loadStores().catch(() => []), storeCtx.ensureStore()])
      .then(([stores, currentStore]) => {
        const requiredStore = this.data.couponStoreId
          ? stores.find((store) => String(store.id) === String(this.data.couponStoreId))
          : null;
        const store = requiredStore || currentStore;
        if (store) {
          if (requiredStore) storeCtx.set(requiredStore);
          this.useStore(store, true);
        }
        else this.setData({ loading: false });
      })
      .catch(() => this.setData({ loading: false }));
  },

  loadStores() {
    return storeCtx.listNearby().then((stores) => {
      this.setData({ stores });
      return stores;
    });
  },

  useStore(store, force) {
    const changed = !this.data.store || String(this.data.store.id) !== String(store.id);
    this.setData({ store });
    if (changed || force) {
      this.clearSelection();
      this.loadEligibleItems(store.id);
    }
  },

  loadEligibleItems(storeId) {
    const requestId = ++this._requestId;
    this.setData({ loading: true, items: [] });
    api
      .getCouponRedeemableItems({
        entitlementId: this.data.entitlementId,
        storeId,
      })
      .then((res) => {
        if (requestId !== this._requestId) return;
        const data = res.data || {};
        const coupon = data.coupon || {};
        const amountCent = Number(coupon.valueCent || this.data.couponAmountCent || 0);
        const couponType = coupon.couponType || this.data.couponType;
        const items = (data.items || []).map((item) => this.decorateItem(item, amountCent));
        this.setData({
          loading: false,
          items,
          couponType,
          couponTypeLabel: COUPON_TYPE_LABEL[couponType] || '优惠券',
          couponName: coupon.name || this.data.couponName,
          couponAmountCent: amountCent,
          couponAmountText: fmt.centToYuan(amountCent),
          remainText: fmt.centToYuan(amountCent),
          validUntil: coupon.expiresAt || this.data.validUntil,
        });
      })
      .catch((err) => {
        if (requestId !== this._requestId) return;
        this.setData({ loading: false, items: [] });
        ui.toast((err && err.message) || '可兑换商品加载失败');
      });
  },

  decorateItem(item, amountCent) {
    const priceCent = Number(item.unitPriceCent || 0);
    const stock = Number(item.stockQuantity || 0);
    const amountMax = priceCent > 0 ? Math.floor(amountCent / priceCent) : 0;
    // stockQuantity=0 表示不限量；步进器单次最多选择 99 件。
    const maxQty = stock > 0 ? Math.min(stock, amountMax) : Math.min(99, amountMax);
    return {
      id: item.itemId,
      name: item.name,
      description: item.description || '',
      initial: (item.name || '兑').slice(0, 1),
      imageUrl: item.imageUrl || '',
      priceCent,
      priceText: formatMenuPrice(priceCent),
      priceCompact: String(Math.floor(priceCent / 100)).length >= 3,
      stock,
      maxQty,
      qty: 0,
      soldOut: maxQty <= 0,
    };
  },

  onQtyChange(e) {
    const id = e.currentTarget.dataset.id;
    const quantity = Number(e.detail.value || 0);
    const current = this.data.items.find((item) => String(item.id) === String(id));
    if (!current) return;
    const nextTotal = this.data.totalCent - current.qty * current.priceCent + quantity * current.priceCent;
    if (nextTotal > this.data.couponAmountCent) {
      ui.toast('所选商品金额不能超过券面额');
      return;
    }
    const items = this.data.items.map((item) =>
      String(item.id) === String(id) ? Object.assign({}, item, { qty: quantity }) : item
    );
    this.recalculate(items);
  },

  recalculate(items) {
    const selectedItems = [];
    let selectedCount = 0;
    let totalCent = 0;
    items.forEach((item) => {
      if (item.qty <= 0) return;
      selectedCount += item.qty;
      totalCent += item.qty * item.priceCent;
      selectedItems.push(
        Object.assign({}, item, {
          lineTotalText: fmt.centToYuan(item.qty * item.priceCent),
        })
      );
    });
    const remainCent = Math.max(this.data.couponAmountCent - totalCent, 0);
    this.setData({
      items,
      selectedItems,
      selectedCount,
      totalCent,
      totalText: fmt.centToYuan(totalCent),
      remainText: fmt.centToYuan(remainCent),
      canConfirm: totalCent > 0 && totalCent <= this.data.couponAmountCent,
    });
    if (!selectedCount && this.data.showSelectionSheet) {
      this.setData({ showSelectionSheet: false });
    }
  },

  clearSelection() {
    const items = this.data.items.map((item) => Object.assign({}, item, { qty: 0 }));
    this.setData({
      items,
      selectedItems: [],
      selectedCount: 0,
      totalCent: 0,
      totalText: '0.00',
      remainText: this.data.couponAmountText,
      canConfirm: false,
      showSelectionSheet: false,
      showUnderuseConfirm: false,
    });
  },

  openSelectionSheet() {
    if (!this.data.selectedCount) {
      ui.toast('请先选择商品');
      return;
    }
    this.setData({ showSelectionSheet: true });
  },

  closeSelectionSheet() {
    this.setData({ showSelectionSheet: false });
  },

  onClearSelection() {
    this.clearSelection();
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
    const store = this.data.stores.find(
      (item) => String(item.id) === String(e.currentTarget.dataset.id)
    );
    if (!store) return;
    storeCtx.set(store);
    this.setData({ showStoreSheet: false });
    this.useStore(store, true);
  },

  confirmRedeem() {
    if (!this.data.canConfirm || this.data.submitting) {
      if (!this.data.selectedCount) ui.toast('请先选择商品');
      return;
    }
    if (this.data.showSelectionSheet) this.setData({ showSelectionSheet: false });
    if (this.data.totalCent < this.data.couponAmountCent) {
      this.setData({ showUnderuseConfirm: true });
      return;
    }
    this.submitRedeem();
  },

  cancelUnderuseConfirm() {
    this.setData({ showUnderuseConfirm: false });
  },

  confirmUnderuseRedeem() {
    this.setData({ showUnderuseConfirm: false });
    this.submitRedeem();
  },

  submitRedeem() {
    if (this.data.submitting || !this.data.store) return;
    this.setData({ submitting: true });
    const items = this.data.selectedItems.map((item) => ({
      itemId: item.id,
      quantity: item.qty,
    }));
    api
      .createCouponRedemption(
        {
          entitlementId: this.data.entitlementId,
          storeId: this.data.store.id,
          items,
        },
        http.uuid()
      )
      .then((res) => {
        const redemptionId = res.data && res.data.id;
        if (!redemptionId) throw new Error('兑换记录创建失败');
        wx.redirectTo({
          url: `/pages/redemption-order-detail/redemption-order-detail?id=${redemptionId}`,
        });
      })
      .catch((err) => {
        this.setData({ submitting: false });
        ui.toast((err && err.message) || '兑换失败，请稍后重试');
      });
  },

  noop() {},
});
