// 核销历史 — 当前门店核销记录
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const { VERIFY_RESULT_LABEL } = require('../../constants/index');

const PAGE_SIZE = 20;

Page({
  data: {
    loading: true,
    refreshing: false,
    loadingMore: false,
    hasMore: true,
    errorText: '',
    list: [],
  },

  onLoad() {
    this.page = 0;
    this.loadPage(1);
  },

  loadPage(page, options) {
    const opts = options || {};
    const requestSeq = page === 1 ? (this.requestSeq = (this.requestSeq || 0) + 1) : this.requestSeq;
    return api.staff
      .getVerifications({ page, pageSize: PAGE_SIZE })
      .then((res) => {
        if (requestSeq !== this.requestSeq) return;
        const rows = (res.data || []).map((v) => ({
          id: v.id,
          activityTitle: v.activityTitle || v.activityName || v.title || '活动名称未记录',
          code: fmt.codeGroups(v.code),
          result: v.result,
          resultLabel: VERIFY_RESULT_LABEL[v.result] || v.result,
          success: v.result === 'success',
          timeText: fmt.dateTime(v.at),
          memberName: v.memberName || '',
        }));
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
        const message = (err && err.message) || '核销记录加载失败';
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
    this.loadPage(1, { refreshing: true });
  },

  loadMore() {
    if (this.data.loading || this.data.refreshing || this.data.loadingMore || !this.data.hasMore) return;
    this.setData({ loadingMore: true });
    this.loadPage((this.page || 0) + 1);
  },

  retry() {
    this.setData({ loading: true, errorText: '' });
    this.loadPage(1);
  },
});
