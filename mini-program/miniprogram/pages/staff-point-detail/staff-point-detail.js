// 积分审核详情 — 单条存取积分申请 + 通过/驳回
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const { POINT_SAVING_LABEL } = require('../../constants/index');

const PS_STATUS = { pending: '待审核', approved: '已通过', rejected: '已驳回' };

Page({
  data: { loading: true, item: null, pending: false, submitting: false, errorText: '' },

  onLoad(options) {
    this.id = options.id;
    this.load();
  },

  load() {
    this.setData({ loading: true, errorText: '' });
    api
      .staff.getPointSaving(this.id)
      .then((res) => {
        const p = res.data || {};
        this.setData({
          loading: false,
          errorText: '',
          pending: p.status === 'pending',
          item: {
            status: p.status,
            memberName: p.memberName,
            phoneText: fmt.maskPhone(p.phone) || '未绑定手机号',
            directionLabel: POINT_SAVING_LABEL[p.direction] || p.direction,
            points: p.points,
            basePoints: p.basePoints || 0,
            excessPoints: p.excessPoints || 0,
            awardedPoints: p.awardedPoints || 0,
            awardedCoins: p.awardedCoins || 0,
            calculationDescription: p.calculationDescription || '',
            storeName: p.storeName,
            note: p.note || '无',
            statusLabel: PS_STATUS[p.status] || p.status,
            timeText: fmt.dateTime(p.createdAt),
          },
        });
      })
      .catch((err) => this.setData({
        loading: false,
        errorText: (err && err.message) || '审核详情加载失败',
      }));
  },

  review(e) {
    if (this.data.submitting || !this.data.pending) return;
    const decision = e.currentTarget.dataset.decision;
    const approving = decision === 'approve';
    wx.showModal({
      title: approving ? '确认通过申请？' : '确认驳回申请？',
      content: this.data.item ? `${this.data.item.memberName}，存入 ${this.data.item.points} 积分` : '确认提交本次审核结果',
      confirmText: approving ? '确认通过' : '确认驳回',
      confirmColor: approving ? '#111111' : '#7a2f2f',
      success: (result) => {
        if (result.confirm && !this.data.submitting) this.submitReview(decision);
      },
    });
  },

  submitReview(decision) {
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    api
      .staff.reviewPointSaving(this.id, { decision }, http.uuid())
      .then(() => {
        ui.hideLoading();
        this.setData({ submitting: false, pending: false });
        ui.success(decision === 'reject' ? '已驳回' : '已通过');
        setTimeout(() => wx.navigateBack(), 500);
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '操作失败');
      });
  },
});
