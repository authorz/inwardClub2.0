// 我的预约 — 预约记录列表
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { RESERVATION_STATUS_LABEL } = require('../../constants/index');

Page({
  data: { loading: true, list: [] },

  onShow() {
    this.load();
  },

  load() {
    this.setData({ loading: true });
    api
      .getReservations({ pageSize: 50 })
      .then((res) => {
        const list = (res.data || []).map((r) => ({
          id: r.id,
          tableName: r.tableName,
          seatNo: r.seatNo,
          storeName: r.storeName,
          timeText: r.timeText || fmt.dateTime(r.createdAt),
          status: r.status,
          statusLabel: RESERVATION_STATUS_LABEL[r.status] || r.status,
        }));
        this.setData({ list, loading: false });
      })
      .catch(() => this.setData({ loading: false }));
  },

  goDetail(e) {
    wx.navigateTo({ url: '/pages/reservation-detail/reservation-detail?id=' + e.currentTarget.dataset.id });
  },
});
