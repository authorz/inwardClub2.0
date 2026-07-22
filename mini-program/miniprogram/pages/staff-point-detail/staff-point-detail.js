// 积分审核详情 — 单条存取积分申请 + 通过/驳回
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const { POINT_SAVING_LABEL } = require('../../constants/index');

const PS_STATUS = { pending: '待审核', approved: '已通过', rejected: '已驳回' };

Page({
  data: { loading: true, item: null, pending: false },

  onLoad(options) {
    this.id = options.id;
    this.load();
  },

  load() {
    api
      .staff.getPointSaving(this.id)
      .then((res) => {
        const p = res.data || {};
        this.setData({
          loading: false,
          pending: p.status === 'pending',
          item: {
            memberName: p.memberName,
            directionLabel: POINT_SAVING_LABEL[p.direction] || p.direction,
            points: p.points,
            storeName: p.storeName,
            note: p.note || '无',
            statusLabel: PS_STATUS[p.status] || p.status,
            timeText: fmt.dateTime(p.createdAt),
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  review(e) {
    const decision = e.currentTarget.dataset.decision;
    ui.showLoading('提交中');
    api
      .staff.reviewPointSaving(this.id, { decision }, http.uuid())
      .then(() => {
        ui.hideLoading();
        ui.success(decision === 'reject' ? '已驳回' : '已通过');
        setTimeout(() => wx.navigateBack(), 500);
      })
      .catch((err) => {
        ui.hideLoading();
        ui.error((err && err.message) || '操作失败');
      });
  },
});
