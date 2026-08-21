// 我的券 — 可用/已使用/已过期 分段 + 带券进入点餐
// Reference: design/mini-program/final/member-subpages/08-my-coupons-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { COUPON_STATUS_LABEL } = require('../../constants/index');

const COUPON_TYPE_LABEL = {
  event_ticket: '赛事门票券', snack: '小吃券', alcohol: '酒水券',
  beverage: '饮料券', meal: '餐食券',
};

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
    statusOptions: [
      { label: '可用', value: 'unused' },
      { label: '已使用', value: 'used' },
      { label: '已过期', value: 'expired' },
    ],
    status: 'unused',
    all: [],
    list: [],
    statusCount: 0,
    statusCountLabel: '可用',
  },

  onLoad() {
    api
      .getCoupons()
      .then((res) => {
        const all = (res.data || []).map((c) => ({
          id: c.id,
          templateId: c.templateId,
          storeId: c.storeId,
          name: c.name,
          desc: c.desc,
          type: c.type,
          typeLabel: COUPON_TYPE_LABEL[c.type] || '福利券',
          validUntil: c.validUntil,
          usedAt: c.usedAt,
          status: c.status,
          statusLabel: COUPON_STATUS_LABEL[c.status] || c.status,
          action: c.action || 'none',
          actionText: '去使用',
          dateLabel: c.status === 'used' ? '使用时间' : '有效期至',
          dateText: c.status === 'used' ? c.usedAt || '-' : c.validUntil || '-',
        }));
        this.setData({ all, loading: false });
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
      countdownText: item.status === 'unused' ? countdown(item.validUntil, now) : '',
    }));
    this.setData({ all });
    this.applyFilter();
  },

  onStatusChange(e) {
    this.setData({ status: e.detail.value });
    this.applyFilter();
  },

  applyFilter() {
    const list = this.data.all.filter((c) => c.status === this.data.status);
    const active = this.data.statusOptions.find((x) => x.value === this.data.status);
    this.setData({ list, statusCount: list.length, statusCountLabel: active ? active.label : '可用' });
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
