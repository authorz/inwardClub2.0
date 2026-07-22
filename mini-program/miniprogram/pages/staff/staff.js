// 工作人员首页 — 固定当前绑定门店，无切换入口
// Reference: design/mini-program/final/member-subpages/13-staff-home-v23.png
const api = require('../../services/api');

Page({
  data: { loading: true, home: null },

  onShow() {
    this.load();
  },

  load() {
    api
      .staff.home()
      .then((res) => {
        const d = res.data || {};
        this.setData({
          loading: false,
          home: {
            storeName: (d.store && d.store.name) || '',
            pendingReview: d.pendingReview || 0,
            todayVerifications: d.todayVerifications || 0,
            todayActivity: d.todayActivity || null,
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  goVerify() {
    wx.navigateTo({ url: '/pages/staff-verify/staff-verify' });
  },
  goReview() {
    wx.navigateTo({ url: '/pages/staff-point-review/staff-point-review' });
  },
  goRecords() {
    wx.navigateTo({ url: '/pages/staff-verifications/staff-verifications' });
  },
  goToday() {
    wx.navigateTo({ url: '/pages/staff-today/staff-today' });
  },
});
