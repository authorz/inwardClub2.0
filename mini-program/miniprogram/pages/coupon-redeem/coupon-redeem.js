// 券兑换 — 券面额抵扣，选择允许分类内的商品，总价须 <= 券面额
// 复用点餐页结构：自定义导航 + 左侧分类栏 + 右侧两列商品 + 底部兑换栏（class 前缀 cr__）
const api = require('../../services/api');
const http = require('../../utils/request');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');

// 券类型 -> 允许的分类
const TYPE_CATS = {
  drink: ['cat_1', 'cat_2'],
  food: ['cat_4', 'cat_5'],
  snack: ['cat_3'],
};

function formatMenuPrice(cent) {
  const n = Number(cent || 0);
  if (n % 100 === 0) return String(Math.floor(n / 100));
  return (n / 100).toFixed(2);
}

Page({
  data: {
    loading: true,
    store: null,
    couponId: '',
    couponType: '',
    couponName: '',
    couponAmountCent: 0,
    couponAmountText: '0.00',
    groups: [],
    activeCat: '',
    intoView: '',
    selectedCount: 0,
    totalCent: 0,
    totalText: '0.00',
    remainText: '0.00',
    overCent: 0,
    overText: '0.00',
    canConfirm: false,
    // custom navigation metrics (px)
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
  },

  onLoad(query) {
    this.qty = {}; // itemId -> qty
    const amountCent = Number(query.amount || 0);
    this.setData({
      couponId: query.id || '',
      couponType: query.type || '',
      couponName: decodeURIComponent(query.name || ''),
      couponAmountCent: amountCent,
      couponAmountText: fmt.centToYuan(amountCent),
      remainText: fmt.centToYuan(amountCent),
    });
    this.allowCats = TYPE_CATS[query.type] || [];
    this.measureNav();
    this.load();
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
    // Resolve the current store (persisted pick, else nearest) before loading its
    // catalog — coupon redemption is store-scoped.
    storeCtx.ensureStore().then((store) => {
      this.setData({ store });
      if (store) this.loadCatalog(store.id);
      else this.setData({ loading: false });
    });
  },

  loadCatalog(storeId) {
    this.setData({ loading: true });
    Promise.all([api.getCategories(storeId), api.getItems(storeId)])
      .then(([catRes, itemRes]) => {
        const cats = (catRes.data || []).filter((c) => this.allowCats.includes(c.id));
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
    const amount = this.data.couponAmountCent;
    const overCent = Math.max(totalCent - amount, 0);
    const remainCent = Math.max(amount - totalCent, 0);
    this.setData({
      selectedCount: count,
      totalCent,
      totalText: fmt.centToYuan(totalCent),
      remainText: fmt.centToYuan(remainCent),
      overCent,
      overText: fmt.centToYuan(overCent),
      canConfirm: totalCent > 0 && totalCent <= amount,
    });
  },

  confirmRedeem() {
    const { totalCent, couponAmountCent } = this.data;
    if (totalCent <= 0) {
      ui.toast('请先选择商品');
      return;
    }
    if (totalCent > couponAmountCent) {
      ui.toast('已超出券面额');
      return;
    }
    const items = [];
    this.data.groups.forEach((g) =>
      g.items.forEach((it) => {
        if (it.qty > 0) {
          items.push({ id: it.id, name: it.name, qty: it.qty, priceCent: it.priceCent });
        }
      })
    );
    api
      .createCouponRedemption(
        {
          couponId: this.data.couponId,
          couponType: this.data.couponType,
          storeId: this.data.store ? this.data.store.id : '',
          items,
          totalCent,
          couponAmountCent,
        },
        http.uuid()
      )
      .then((r) => {
        const id = r.data && r.data.id;
        wx.redirectTo({ url: `/pages/redemption-order-detail/redemption-order-detail?id=${id}` });
      });
  },
});
