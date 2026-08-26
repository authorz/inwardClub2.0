// 我的券 — 仅展示可用券，按券分类筛选 + 带券进入点餐
// Reference: design/mini-program/final/member-subpages/08-my-coupons-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');

const COUPON_TYPE_LABEL = {
  event_ticket: '赛事门票券', snack: '小吃券', alcohol: '酒水券',
  beverage: '饮料券', drink: '饮品或啤酒券', meal: '餐食券', gift: '礼品券',
};

function typeOptionsOf(coupons) {
  const availableTypes = new Set(coupons.map((coupon) => coupon.type));
  const options = [{ label: '全部', value: 'all' }];
  Object.keys(COUPON_TYPE_LABEL).forEach((type) => {
    if (availableTypes.has(type)) {
      options.push({ label: COUPON_TYPE_LABEL[type], value: type });
    }
  });
  return options;
}

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

Page({
  data: {
    loading: true,
    typeOptions: [{ label: '全部', value: 'all' }],
    selectedType: 'all',
    all: [],
    list: [],
    couponCount: 0,
  },

  onLoad() {
    api
      .getCoupons({ status: 'active', pageSize: 100 })
      .then((res) => {
        const all = (res.data || []).filter((c) => c.status === 'unused').map((c) => ({
          id: c.id,
          templateId: c.templateId,
          storeId: c.storeId,
          name: c.name,
          desc: c.desc,
          type: c.type,
          typeLabel: COUPON_TYPE_LABEL[c.type] || '福利券',
          validUntil: c.validUntil,
          status: c.status,
          action: c.action || 'none',
          actionText: '去使用',
          dateText: c.validUntil || '-',
        }));
        this.setData({ all, typeOptions: typeOptionsOf(all), loading: false });
        this.refreshCountdowns();
        this.startCountdown();
        this.applyFilter();
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
    const all = this.data.all.map((item) => Object.assign({}, item, {
      countdownText: countdown(item.validUntil, now),
    }));
    this.setData({ all });
    this.applyFilter();
  },

  onTypeChange(e) {
    this.setData({ selectedType: e.detail.value });
    this.applyFilter();
  },

  applyFilter() {
    const list = this.data.selectedType === 'all'
      ? this.data.all
      : this.data.all.filter((coupon) => coupon.type === this.data.selectedType);
    this.setData({ list, couponCount: list.length });
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
