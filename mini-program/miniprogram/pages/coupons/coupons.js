// 我的券 — 可用/已使用/已过期 分段 + 带券进入点餐
// Reference: design/mini-program/final/member-subpages/08-my-coupons-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { COUPON_STATUS_LABEL } = require('../../constants/index');

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
          amountCent: c.amountCent,
          amountText: fmt.centToYuan(c.amountCent),
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
        this.applyFilter();
      })
      .catch(() => this.setData({ loading: false }));
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
    const params = [
      `entitlementId=${c.id}`,
      `templateId=${c.templateId || ''}`,
      `storeId=${c.storeId || ''}`,
      `couponType=${c.type}`,
      `valueCent=${c.amountCent || 0}`,
      `name=${encodeURIComponent(c.name)}`,
      `validUntil=${encodeURIComponent(c.validUntil || '')}`,
    ].join('&');
    wx.navigateTo({ url: `/pages/coupon-redeem/coupon-redeem?${params}` });
  },

  goRecords() {
    wx.navigateTo({ url: '/pages/wallet-ledger/wallet-ledger?asset=coupons' });
  },
});
