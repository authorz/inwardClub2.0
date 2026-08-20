// 积分审核 — 当前门店待审核积分存入列表
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');

const PAGE_SIZE = 20;
const PS_STATUS = { pending: '待审核', approved: '已通过', rejected: '已驳回' };

Page({
  data: {
    loading: true,
    refreshing: false,
    loadingMore: false,
    hasMore: true,
    errorText: '',
    list: [],
    keyword: '',
    submittingId: '',
  },

  onShow() {
    this.page = 0;
    this.loadPage(1, this.data.keyword);
  },

  mapItem(p) {
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
  },

  loadPage(page, phone, options) {
    const opts = options || {};
    const requestSeq = page === 1 ? (this.requestSeq = (this.requestSeq || 0) + 1) : this.requestSeq;
    const params = { status: 'pending', page, pageSize: PAGE_SIZE };
    if (phone) params.phone = phone;
    return api.staff
      .getPointSavings(params)
      .then((res) => {
        if (requestSeq !== this.requestSeq) return;
        const rows = (res.data || []).map((p) => this.mapItem(p));
        const total = res.meta && Number.isFinite(Number(res.meta.total)) ? Number(res.meta.total) : 0;
        this.page = page;
        this.setData({
          loading: false,
          refreshing: false,
          loadingMore: false,
          errorText: '',
          list: page === 1 ? rows : this.data.list.concat(rows),
          hasMore: total ? page * PAGE_SIZE < total : rows.length === PAGE_SIZE,
        });
      })
      .catch((err) => {
        if (requestSeq !== this.requestSeq) return;
        const message = (err && err.message) || '积分审核列表加载失败';
        this.setData({
          loading: false,
          refreshing: false,
          loadingMore: false,
          errorText: this.data.list.length ? '' : message,
        });
        if (this.data.list.length || opts.refreshing) ui.error(message);
      });
  },

  onPullDownRefresh() {
    if (this.data.refreshing) return;
    this.setData({ refreshing: true, errorText: '' });
    this.loadPage(1, this.data.keyword, { refreshing: true });
  },

  loadMore() {
    if (this.data.loading || this.data.refreshing || this.data.loadingMore || !this.data.hasMore) return;
    this.setData({ loadingMore: true });
    this.loadPage((this.page || 0) + 1, this.data.keyword);
  },

  retry() {
    this.setData({ loading: true, errorText: '' });
    this.loadPage(1, this.data.keyword);
  },

  onSearch(e) {
    const keyword = (e.detail.value || '').replace(/\D/g, '').slice(0, 11);
    this.setData({ keyword });
    if (!keyword) {
      this.setData({ loading: true, list: [] });
      this.loadPage(1, '');
    }
  },

  search() {
    this.setData({ loading: true, list: [], errorText: '' });
    this.loadPage(1, this.data.keyword);
  },

  clearSearch() {
    this.setData({ keyword: '', loading: true, list: [], errorText: '' });
    this.loadPage(1, '');
  },

  review(e) {
    if (this.data.submittingId) return;
    const { id, decision } = e.currentTarget.dataset;
    const item = this.data.list.find((row) => String(row.id) === String(id));
    const approving = decision === 'approve';
    wx.showModal({
      title: approving ? '确认通过申请？' : '确认驳回申请？',
      content: item ? `${item.memberName}，存入 ${item.pointsText} 积分` : '确认提交本次审核结果',
      confirmText: approving ? '确认通过' : '确认驳回',
      confirmColor: approving ? '#111111' : '#7a2f2f',
      success: (result) => {
        if (!result.confirm || this.data.submittingId) return;
        this.submitReview(id, decision);
      },
    });
  },

  submitReview(id, decision) {
    this.setData({ submittingId: id });
    ui.showLoading('提交中');
    api.staff
      .reviewPointSaving(id, { decision }, http.uuid())
      .then(() => {
        ui.hideLoading();
        this.setData({ submittingId: '' });
        ui.success(decision === 'reject' ? '已驳回' : '已通过');
        this.loadPage(1, this.data.keyword);
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submittingId: '' });
        ui.error((err && err.message) || '操作失败');
      });
  },

  goDetail(e) {
    if (this.data.submittingId) return;
    wx.navigateTo({ url: '/pages/staff-point-detail/staff-point-detail?id=' + e.currentTarget.dataset.id });
  },
});
