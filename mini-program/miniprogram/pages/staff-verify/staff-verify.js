// 活动核销 — 扫码核销 + 手输核销码，固定当前门店（无切换）
// Reference: design/mini-program/generated/v2/10-staff-verification-v2.png
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const { VERIFY_RESULT_LABEL } = require('../../constants/index');

Page({
  data: {
    loading: true,
    storeName: '',
    todayActivity: null,
    verifications: [],
    lastResult: null,
    stats: { verified: 0, success: 0, failed: 0, void: 0 },
  },

  onLoad() {
    this.refresh(true);
  },

  refresh(withStore) {
    const tasks = [api.staff.getTodayActivities(), api.staff.getVerifications()];
    if (withStore) tasks.push(api.staff.home());
    Promise.all(tasks)
      .then(([todayRes, verRes, homeRes]) => {
        const patch = {
          loading: false,
          todayActivity: (todayRes.data || [])[0] || null,
          verifications: (verRes.data || []).map((v) => ({
            id: v.id,
            code: fmt.codeGroups(v.code),
            result: v.result,
            resultLabel: VERIFY_RESULT_LABEL[v.result] || v.result,
            success: v.result === 'success',
            timeText: fmt.dateTime(v.at),
            memberName: v.memberName || '',
          })),
        };
        patch.stats = this.computeStats(patch.verifications);
        if (homeRes) patch.storeName = (homeRes.data && homeRes.data.store && homeRes.data.store.name) || '';
        this.setData(patch);
      })
      .catch(() => this.setData({ loading: false }));
  },

  computeStats(list) {
    return {
      verified: list.length,
      success: list.filter((x) => x.result === 'success').length,
      failed: list.filter((x) => x.result === 'failed').length,
      void: list.filter((x) => x.result === 'void').length,
    };
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
      placeholderText: '请输入核销码',
      confirmColor: '#111111',
      success: (r) => {
        if (r.confirm && r.content) this.doVerify(r.content);
      },
    });
  },

  doVerify(code) {
    ui.showLoading('核销中');
    api
      .staff.verifyTicket({ code }, http.uuid())
      .then((res) => {
        ui.hideLoading();
        const d = res.data || {};
        this.setData({
          lastResult: {
            success: d.result === 'success',
            resultLabel: VERIFY_RESULT_LABEL[d.result] || d.result,
            message: d.message || '',
            memberName: d.memberName || '',
            codeText: fmt.codeGroups(d.code || code),
          },
        });
        if (d.result === 'success') ui.success('核销成功');
        else ui.error(d.message || '核销失败');
        this.refresh(false);
      })
      .catch((err) => {
        ui.hideLoading();
        ui.error((err && err.message) || '核销失败');
      });
  },
});
