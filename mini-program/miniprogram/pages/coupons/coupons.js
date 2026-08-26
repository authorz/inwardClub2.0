// 我的券 — 仅展示可用券，按券分类筛选 + 带券进入点餐
// Reference: design/mini-program/final/member-subpages/08-my-coupons-v23.png
const api = require('../../services/api');
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
  },

  onLoad() {
    Promise.all([
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
        const all = (couponsRes.data || []).map((c) => ({
          id: c.id,
          templateId: c.templateId,
          storeId: c.storeId,
          name: c.name,
          desc: c.desc,
          type: c.type,
          categoryId: String(c.categoryId || ''),
          categoryLabel: c.categoryName || typeLabels[String(c.categoryId)] || '福利券',
          typeLabel: c.categoryName || typeLabels[String(c.categoryId)] || '福利券',
          validUntil: c.validUntil,
          status: c.status,
          action: c.action || 'none',
          actionText: '去使用',
          dateText: c.validUntil || '-',
        })).filter((coupon) => isUsableCoupon(coupon, now));
        this.setData({ categories, loading: false });
        this.setCouponState(all, categories, 'all');
        this.refreshCountdowns();
        if (all.length) this.startCountdown();
      })
      .catch(() => this.setData({ loading: false }));
  },

  onShow() {
    if (this.data.all.length) this.startCountdown();
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
    if (!c || c.status !== 'unused') return;
    if (c.type === 'event_ticket') {
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

  goRecords() {
    wx.navigateTo({ url: '/pages/wallet-ledger/wallet-ledger?asset=coupons' });
  },
});
