// 核销历史 — 当前门店核销记录
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { VERIFY_RESULT_LABEL } = require('../../constants/index');

Page({
  data: { loading: true, list: [] },

  onLoad() {
    api
      .staff.getVerifications({ pageSize: 50 })
      .then((res) => {
        const list = (res.data || []).map((v) => ({
          id: v.id,
          activityTitle: v.activityTitle || v.activityName || v.title || '活动名称未记录',
          code: fmt.codeGroups(v.code),
          result: v.result,
          resultLabel: VERIFY_RESULT_LABEL[v.result] || v.result,
          success: v.result === 'success',
          timeText: fmt.dateTime(v.at),
          memberName: v.memberName || '',
        }));
        this.setData({ list, loading: false });
      })
      .catch(() => this.setData({ loading: false }));
  },
});
