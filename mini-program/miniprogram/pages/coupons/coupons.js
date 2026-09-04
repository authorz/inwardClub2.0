// 我的券 — 仅展示可用券，按券分类筛选 + 带券进入点餐
// Reference: design/mini-program/final/member-subpages/08-my-coupons-v23.png
const api = require('../../services/api');
const http = require('../../utils/request');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');

function countdown(validUntil, now) {
  const end = fmt.timestamp(validUntil);
  if (!Number.isFinite(end)) return '';
  const seconds = Math.max(0, Math.floor((end - now) / 1000));
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  if (seconds === 0) return '已到期';
  return `${days}天 ${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
}

function isUsableCoupon(coupon, now) {
  if (coupon.status !== 'unused') return false;
  if (!coupon.validUntil) return true;
  const end = fmt.timestamp(coupon.validUntil);
  return Number.isFinite(end) && end > now;
}

function typeOptionsOf(categories, coupons) {
  const categoryLabels = coupons.reduce((labels, coupon) => {
    if (coupon.categoryId) labels[coupon.categoryId] = coupon.categoryLabel;
    return labels;
  }, {});
  const options = [{ label: '全部', value: 'all' }];
  categories.forEach((category) => {
    if (!categoryLabels[category.value]) return;
    options.push(category);
    delete categoryLabels[category.value];
  });
  Object.keys(categoryLabels).forEach((value) => {
    options.push({ label: categoryLabels[value], value });
  });
  return options;
}

Page({
  data: {
    loading: true,
    typeOptions: [{ label: '全部', value: 'all' }],
    selectedType: 'all',
    categories: [],
    all: [],
    list: [],
    couponCount: 0,
    submittingId: '',
  },

  loadCoupons() {
    this.setData({ loading: true });
    return Promise.all([
      api.getCouponCategories(),
      api.getCoupons({ status: 'active', pageSize: 100 }),
    ])
      .then(([typesRes, couponsRes]) => {
        const categories = (typesRes.data || [])
          .filter((item) => item.id && item.name)
          .map((item) => ({ label: item.name, value: String(item.id) }));
        const typeLabels = categories.reduce((labels, item) => {
          labels[item.value] = item.label;
          return labels;
        }, {});
        const now = Date.now();
        const all = (couponsRes.data || []).map((c) => {
          const rawCategoryId = String(c.categoryId || '');
          const categoryLabel = c.categoryName || typeLabels[rawCategoryId] || '福利券';
          const categoryId = rawCategoryId && typeLabels[rawCategoryId] && typeLabels[rawCategoryId] !== categoryLabel
            ? `display:${c.type || 'coupon'}:${categoryLabel}`
            : rawCategoryId;
          return {
            id: c.id,
            templateId: c.templateId,
            storeId: c.storeId,
            name: c.name,
            desc: c.desc,
            type: c.type,
            categoryId,
            categoryLabel,
            typeLabel: categoryLabel,
            ruleText: c.type === 'event_ticket'
              ? '确认后直接使用'
              : (c.type === 'admission_ticket' ? '一券兑一张门票' : '一券兑一份'),
            validUntil: c.validUntil,
            status: c.status,
            action: c.action || 'none',
            actionText: '去使用',
            dateText: c.validUntil || '-',
          };
        }).filter((coupon) => isUsableCoupon(coupon, now));
        this.setData({ categories, loading: false });
        this.setCouponState(all, categories, 'all');
        this.refreshCountdowns();
        if (all.length) this.startCountdown();
      })
      .catch(() => this.setData({ loading: false }));
  },

  onShow() {
    return this.loadCoupons();
  },

  onHide() {
    this.stopCountdown();
  },

  onUnload() {
    this.stopCountdown();
  },

  startCountdown() {
    this.stopCountdown();
    this._countdownTimer = setInterval(() => this.refreshCountdowns(), 1000);
  },

  stopCountdown() {
    if (this._countdownTimer) clearInterval(this._countdownTimer);
    this._countdownTimer = null;
  },

  refreshCountdowns() {
    const now = Date.now();
    const all = this.data.all
      .filter((item) => isUsableCoupon(item, now))
      .map((item) => Object.assign({}, item, {
        countdownText: countdown(item.validUntil, now),
      }));
    this.setCouponState(all, this.data.categories, this.data.selectedType);
    if (!all.length) this.stopCountdown();
  },

  onTypeChange(e) {
    this.applyFilter(e.currentTarget.dataset.value);
  },

  setCouponState(all, categories, selectedType) {
    const typeOptions = typeOptionsOf(categories, all);
    const nextType = typeOptions.some((option) => option.value === selectedType) ? selectedType : 'all';
    const list = nextType === 'all'
      ? all
      : all.filter((coupon) => coupon.categoryId === nextType);
    this.setData({ all, typeOptions, selectedType: nextType, list, couponCount: list.length });
  },

  applyFilter(selectedType) {
    const list = selectedType === 'all'
      ? this.data.all
      : this.data.all.filter((coupon) => coupon.categoryId === selectedType);
    this.setData({ selectedType, list, couponCount: list.length });
  },

  onAction(e) {
    const c = this.data.all.find((x) => x.id === e.currentTarget.dataset.id);
    if (!c || c.status !== 'unused' || this.data.submittingId) return;
    if (c.type === 'event_ticket') {
      this.confirmEventCouponUse(c);
      return;
    }
    if (c.type === 'admission_ticket') {
      wx.navigateTo({ url: '/pages/activity-list/activity-list' });
      return;
    }
    const params = [
      `entitlementId=${c.id}`,
      `templateId=${c.templateId || ''}`,
      `storeId=${c.storeId || ''}`,
      `couponType=${c.type}`,
      `name=${encodeURIComponent(c.name)}`,
      `validUntil=${encodeURIComponent(c.validUntil || '')}`,
    ].join('&');
    wx.navigateTo({ url: `/pages/coupon-redeem/coupon-redeem?${params}` });
  },

  resolveEventCouponStore(coupon) {
    if (!coupon.storeId) return storeCtx.ensureStore();
    return storeCtx.listNearby().then((stores) =>
      stores.find((store) => String(store.id) === String(coupon.storeId)) || null
    );
  },

  confirmEventCouponUse(coupon) {
    if (this._confirmingCouponId) return;
    this._confirmingCouponId = coupon.id;
    this.resolveEventCouponStore(coupon)
      .then((store) => {
        if (!store) throw new Error('当前没有可使用赛事券的门店');
        return ui.confirm({
          title: '确认使用赛事券',
          content: `将在“${store.name}”使用“${coupon.name}”。确认后将立即核销并打印小票，无法撤销。`,
          confirmText: '确认使用',
        }).then((confirmed) => ({ confirmed, store }));
      })
      .then(({ confirmed, store }) => {
        this._confirmingCouponId = null;
        if (!confirmed) return;
        this.submitEventCouponUse(coupon, store);
      })
      .catch((err) => {
        this._confirmingCouponId = null;
        ui.error((err && err.message) || '赛事券使用失败');
      });
  },

  submitEventCouponUse(coupon, store) {
    this.setData({ submittingId: coupon.id }, () => {
      api.useEventCoupon({ entitlementId: coupon.id, storeId: store.id }, http.uuid())
        .then((res) => {
          const redemptionId = res.data && res.data.id;
          if (!redemptionId) throw new Error('赛事券订单创建失败');
          wx.redirectTo({
            url: `/pages/pay-result/pay-result?type=coupon&status=success&id=${redemptionId}`,
          });
        })
        .catch((err) => {
          this.setData({ submittingId: '' });
          ui.error((err && err.message) || '赛事券使用失败');
        });
    });
  },

  goRecords() {
    wx.navigateTo({ url: '/pages/wallet-ledger/wallet-ledger?asset=coupons' });
  },
});
