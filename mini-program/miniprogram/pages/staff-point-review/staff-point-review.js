// 积分审核 — 存取积分申请审核列表（当前门店）
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const validation = require('../../utils/validation');

const PS_STATUS = { pending: '待审核', approved: '已通过', rejected: '已驳回' };

Page({
  data: { loading: true, list: [], keyword: '' },

  onShow() {
    this.load(this.data.keyword.length >= 3 ? this.data.keyword : '');
  },

  load(phone) {
    this.setData({ loading: true });
    const params = { status: 'pending', pageSize: 50 };
    if (phone) params.phone = phone;
    api
      .staff.getPointSavings(params)
      .then((res) => {
        const list = (res.data || []).map((p) => {
          const memberName = p.memberName || '未命名会员';
          return {
            id: p.id,
            memberName,
            memberAvatarUrl: p.memberAvatarUrl || '',
            memberAvatarText: String(memberName).trim().slice(0, 1) || '会',
            phone: p.phone || '',
            phoneText: fmt.maskPhone(p.phone) || '未绑定手机号',
            pointsText: fmt.amount(p.points),
            storeName: p.storeName || '当前门店',
            status: p.status,
            statusLabel: PS_STATUS[p.status] || p.status,
            timeText: fmt.dateTime(p.createdAt),
            pending: p.status === 'pending',
          };
        });
        this.setData({ loading: false, list });
      })
      .catch(() => this.setData({ loading: false }));
  },

  onSearch(e) {
    const keyword = (e.detail.value || '').replace(/\D/g, '').slice(0, 11);
    this.setData({ keyword });
    if (!keyword) this.load();
  },

  search() {
    let phone = '';
    try {
      phone = validation.plainText(this.data.keyword, { label: '手机号', min: 3, max: 11 });
    } catch (err) {
      ui.toast(err.message);
      return;
    }
    this.load(phone);
  },

  clearSearch() {
    this.setData({ keyword: '' });
    this.load();
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
