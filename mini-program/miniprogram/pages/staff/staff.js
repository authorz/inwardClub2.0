// 工作人员首页 — 员工可在后台授予的门店范围内切换当前操作门店
// Reference: design/mini-program/final/member-subpages/13-staff-home-v23.png
const api = require('../../services/api');
const auth = require('../../utils/auth');

Page({
  data: { loading: true, switching: false, home: null, stores: [] },

  onShow() {
    this.load();
  },

  load() {
    Promise.all([
      api.staff.home(),
      api.staff.getTodayOperations(),
      api.staff.getStores().catch(() => ({ data: [] })),
    ])
      .then(([homeRes, operationsRes, storesRes]) => {
        const d = homeRes.data || {};
        const summary = (operationsRes.data && operationsRes.data.summary) || {};
        this.setData({
          loading: false,
          stores: storesRes.data || [],
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

  chooseStore() {
    const stores = this.data.stores || [];
    if (stores.length < 2 || this.data.switching) return;
    wx.showActionSheet({
      itemList: stores.map((store) => store.name),
      success: ({ tapIndex }) => {
        const store = stores[tapIndex];
        if (!store || Number(store.id) === Number(auth.getStoreId())) return;
        this.setData({ switching: true });
        api.staff.switchStore(store.id)
          .then((res) => {
            const result = res.data || {};
            const token = result.token || {};
            auth.save({
              accessToken: token.accessToken,
              refreshToken: token.refreshToken,
              subjectType: result.subjectType || 'staff',
              storeId: result.storeId,
            });
            this.setData({ loading: true });
            return this.load();
          })
          .catch((err) => {
            wx.showToast({ title: (err && err.message) || '切换失败，请重试', icon: 'none' });
          })
          .finally(() => this.setData({ switching: false }));
      },
    });
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
