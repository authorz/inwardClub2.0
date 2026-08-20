// 工作人员首页 — 固定当前绑定门店，无切换入口
// Reference: design/mini-program/final/member-subpages/13-staff-home-v23.png
const api = require('../../services/api');

Page({
  data: { loading: true, home: null },

  onShow() {
    this.load();
  },

  load() {
    Promise.all([api.staff.home(), api.staff.getTodayOperations()])
      .then(([homeRes, operationsRes]) => {
        const d = homeRes.data || {};
        const summary = (operationsRes.data && operationsRes.data.summary) || {};
        this.setData({
          loading: false,
          home: {
            storeName: (d.store && d.store.name) || '',
            pendingReview: d.pendingReview || 0,
            todayVerifications: d.todayVerifications || 0,
            coinConsumption: summary.coinConsumptionAmount || 0,
            pointDeposit: summary.pointDepositAmount || 0,
            pointWithdrawal: summary.pointWithdrawalAmount || 0,
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
