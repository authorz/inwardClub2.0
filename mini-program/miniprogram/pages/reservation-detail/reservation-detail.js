// 预约详情 — 座位/时间/门店 + 取消预约
const api = require('../../services/api');
const ui = require('../../utils/ui');
const http = require('../../utils/request');
const fmt = require('../../utils/format');
const { RESERVATION_STATUS_LABEL } = require('../../constants/index');

Page({
  data: { loading: true, rsv: null, canCancel: false },

  onLoad(options) {
    this.id = options.id;
    this.load();
  },

  load() {
    api
      .getReservation(this.id)
      .then((res) => {
        const r = res.data || {};
        this.setData({
          loading: false,
          // server reservation status enum is booked|arrived|cancelled|expired.
          canCancel: r.status === 'booked',
          rsv: {
            id: r.id,
            tableName: r.tableName,
            seatNo: r.seatNo,
            storeName: r.storeName,
            timeText: fmt.dateTime(r.createdAt),
            status: r.status,
            statusLabel: RESERVATION_STATUS_LABEL[r.status] || r.status,
            note: '预约没有到店时间限制；如不再需要该座位，请及时取消预约',
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  cancel() {
    ui.confirm({ content: '确认取消该预约？', confirmText: '取消预约' }).then((ok) => {
      if (!ok) return;
      ui.showLoading('取消中');
      api
        .cancelReservation(this.id, http.uuid())
        .then(() => {
          ui.hideLoading();
          ui.success('已取消');
          setTimeout(() => wx.navigateBack(), 500);
        })
        .catch((err) => {
          ui.hideLoading();
          ui.error((err && err.message) || '取消失败');
        });
    });
  },
});
