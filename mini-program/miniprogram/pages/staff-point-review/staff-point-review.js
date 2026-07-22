// 积分审核 — 存取积分申请审核列表（当前门店）
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const { POINT_SAVING_LABEL } = require('../../constants/index');

const PS_STATUS = { pending: '待审核', approved: '已通过', rejected: '已驳回' };

Page({
  data: { loading: true, list: [], keyword: '' },

  onShow() {
    this.load();
  },

  load() {
    this.setData({ loading: true });
    api
      .staff.getPointSavings({ pageSize: 50 })
      .then((res) => {
        this.all = (res.data || []).map((p) => ({
          id: p.id,
          memberName: p.memberName,
          phone: p.phone || '',
          directionLabel: POINT_SAVING_LABEL[p.direction] || p.direction,
          points: p.points,
          storeName: p.storeName,
          status: p.status,
          statusLabel: PS_STATUS[p.status] || p.status,
          timeText: fmt.dateTime(p.createdAt),
          pending: p.status === 'pending',
        }));
        this.setData({ loading: false });
        this.filter();
      })
      .catch(() => this.setData({ loading: false }));
  },

  onSearch(e) {
    this.setData({ keyword: e.detail.value });
    this.filter();
  },

  filter() {
    const kw = (this.data.keyword || '').trim().toLowerCase();
    const all = this.all || [];
    const list = kw
      ? all.filter((p) => (p.memberName || '').toLowerCase().includes(kw) || (p.phone || '').includes(kw))
      : all;
    this.setData({ list });
  },

  review(e) {
    const { id, decision } = e.currentTarget.dataset;
    ui.showLoading('提交中');
    api
      .staff.reviewPointSaving(id, { decision }, http.uuid())
      .then(() => {
        ui.hideLoading();
        ui.success(decision === 'reject' ? '已驳回' : '已通过');
        this.load();
      })
      .catch((err) => {
        ui.hideLoading();
        ui.error((err && err.message) || '操作失败');
      });
  },

  goDetail(e) {
    wx.navigateTo({ url: '/pages/staff-point-detail/staff-point-detail?id=' + e.currentTarget.dataset.id });
  },
});
