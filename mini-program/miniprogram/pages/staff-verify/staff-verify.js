// 活动核销 — 扫码核销 + 手输核销码，固定当前门店（无切换）
// Reference: design/mini-program/generated/v2/10-staff-verification-v2.png
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const { VERIFY_RESULT_LABEL } = require('../../constants/index');
const validation = require('../../utils/validation');

const PAGE_SIZE = 20;

Page({
  data: {
    loading: true,
    refreshing: false,
    loadingMore: false,
    hasMore: true,
    errorText: '',
    verifying: false,
    storeName: '',
    todayActivity: null,
    verifications: [],
    lastResult: null,
    stats: { total: 0, today: 0 },
  },

  onLoad() {
    this.page = 0;
    this.loadPage(1, { withOverview: true });
  },

  loadPage(page, options) {
    const opts = options || {};
    const requestSeq = page === 1 ? (this.requestSeq = (this.requestSeq || 0) + 1) : this.requestSeq;
    const verifications = api.staff.getVerifications({ page, pageSize: PAGE_SIZE });
    const overview = opts.withOverview
      ? Promise.all([api.staff.getTodayActivities(), api.staff.home()])
      : Promise.resolve([]);

    return Promise.all([verifications, overview])
      .then(([verRes, overviewRes]) => {
        if (requestSeq !== this.requestSeq) return;
        const rows = (verRes.data || []).map((v) => ({
          id: v.id,
          code: fmt.codeGroups(v.code),
          result: v.result,
          resultLabel: VERIFY_RESULT_LABEL[v.result] || v.result,
          success: v.result === 'success',
          timeText: fmt.dateTime(v.at),
          memberName: v.memberName || '',
        }));
        const total = verRes.meta && Number.isFinite(Number(verRes.meta.total))
          ? Number(verRes.meta.total)
          : (page - 1) * PAGE_SIZE + rows.length;
        const patch = {
          loading: false,
          refreshing: false,
          loadingMore: false,
          errorText: '',
          hasMore: page * PAGE_SIZE < total,
          verifications: page === 1 ? rows : this.data.verifications.concat(rows),
        };
        this.page = page;
        if (opts.withOverview) {
          const todayRes = overviewRes[0] || {};
          const homeRes = overviewRes[1] || {};
          const home = homeRes.data || {};
          patch.todayActivity = (todayRes.data || [])[0] || null;
          patch.storeName = (home.store && home.store.name) || '';
          patch.stats = { total, today: Number(home.todayVerifications) || 0 };
        } else {
          patch.stats = Object.assign({}, this.data.stats, { total });
        }
        this.setData(patch);
      })
      .catch((err) => {
        if (requestSeq !== this.requestSeq) return;
        const message = (err && err.message) || '核销记录加载失败';
        this.setData({
          loading: false,
          refreshing: false,
          loadingMore: false,
          errorText: this.data.verifications.length ? '' : message,
        });
        if (this.data.verifications.length || opts.refreshing) ui.error(message);
      });
  },

  onPullDownRefresh() {
    if (this.data.refreshing) return;
    this.setData({ refreshing: true, errorText: '' });
    this.loadPage(1, { withOverview: true, refreshing: true });
  },

  loadMore() {
    if (this.data.loading || this.data.refreshing || this.data.loadingMore || !this.data.hasMore) return;
    this.setData({ loadingMore: true });
    this.loadPage((this.page || 0) + 1);
  },

  retry() {
    this.setData({ loading: true, errorText: '' });
    this.loadPage(1, { withOverview: true });
  },

  scan() {
    wx.scanCode({
      success: (res) => this.doVerify(res.result),
      fail: () => {},
    });
  },

  manual() {
    wx.showModal({
      title: '输入核销码',
      editable: true,
      placeholderText: '请输入6位数字核销码',
      confirmColor: '#111111',
      success: (r) => {
        if (r.confirm && r.content) this.doVerify(r.content);
      },
    });
  },

  doVerify(code) {
    if (this.data.verifying) return;
    try {
      code = validation.verificationCode(code);
    } catch (err) {
      ui.toast(err.message);
      return;
    }
    this.setData({ verifying: true });
    ui.showLoading('核销中');
    api.staff
      .verifyTicket({ code }, http.uuid())
      .then((res) => {
        ui.hideLoading();
        this.setData({ verifying: false });
        const result = res.data || {};
        const success = result.status === 'used' || result.result === 'success';
        this.setData({
          lastResult: {
            success,
            resultLabel: success ? '核销成功' : (VERIFY_RESULT_LABEL[result.result] || '核销失败'),
            message: result.message || '',
            memberName: result.memberName || '',
            codeText: fmt.codeGroups(result.ticketNo || result.code || code),
          },
        });
        if (success) ui.success('核销成功');
        else ui.error(result.message || '核销失败');
        this.loadPage(1, { withOverview: true });
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ verifying: false });
        ui.error((err && err.message) || '核销失败');
      });
  },
});
