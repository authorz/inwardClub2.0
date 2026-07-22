// 今日活动 — 当前门店今日活动核销概览
const api = require('../../services/api');

Page({
  data: { loading: true, list: [] },

  onLoad() {
    api
      .staff.getTodayActivities()
      .then((res) => {
        const list = (res.data || []).map((a) => ({
          id: a.id,
          title: a.title,
          timeText: a.timeText,
          storeName: a.storeName,
          pendingVerify: a.pendingVerify || 0,
          verified: a.verified || 0,
        }));
        this.setData({ list, loading: false });
      })
      .catch(() => this.setData({ loading: false }));
  },

  goVerify() {
    wx.navigateTo({ url: '/pages/staff-verify/staff-verify' });
  },
});
