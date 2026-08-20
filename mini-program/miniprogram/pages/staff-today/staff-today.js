// 今日营业明细 — 当前门店金币消费与积分存取记录
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');

const PAGE_SIZE = 20;
const TYPE_LABEL = {
  coin_consumption: '金币消费',
  point_deposit: '积分存入',
  point_withdrawal: '积分提取',
};
const STATUS_LABEL = { pending: '待审核', approved: '已通过', rejected: '已驳回', completed: '已完成' };

Page({
  data: {
    loading: true,
    refreshing: false,
    loadingMore: false,
    hasMore: true,
    errorText: '',
    date: '',
    summary: {
      coinConsumptionAmount: 0,
      coinConsumptionAmountText: '0',
      coinConsumptionCount: 0,
      pointDepositAmount: 0,
      pointDepositAmountText: '0',
      pointDepositCount: 0,
      pointWithdrawalAmount: 0,
      pointWithdrawalAmountText: '0',
      pointWithdrawalCount: 0,
    },
    tabs: [
      { key: 'all', label: '全部' },
      { key: 'coin_consumption', label: '金币消费' },
      { key: 'point_deposit', label: '积分存入' },
      { key: 'point_withdrawal', label: '积分提取' },
    ],
    activeType: 'all',
    list: [],
  },

  onShow() {
    this.page = 0;
    this.loadPage(1);
  },

  loadPage(page, options) {
    const opts = options || {};
    const requestSeq = page === 1 ? (this.requestSeq = (this.requestSeq || 0) + 1) : this.requestSeq;
    const params = { page, pageSize: PAGE_SIZE };
    if (this.data.activeType !== 'all') params.type = this.data.activeType;
    return api.staff
      .getTodayOperations(params)
      .then((res) => {
        if (requestSeq !== this.requestSeq) return;
        const data = res.data || {};
        const rows = (data.entries || []).map((item) => this.mapEntry(item));
        const patch = {
          loading: false,
          refreshing: false,
          loadingMore: false,
          hasMore: rows.length === PAGE_SIZE,
          errorText: '',
          list: page === 1 ? rows : this.data.list.concat(rows),
        };
        this.page = page;
        if (page === 1) {
          const summary = data.summary || {};
          patch.date = data.date || '';
          patch.summary = Object.assign({}, this.data.summary, summary, {
            coinConsumptionAmountText: fmt.amount(summary.coinConsumptionAmount),
            pointDepositAmountText: fmt.amount(summary.pointDepositAmount),
            pointWithdrawalAmountText: fmt.amount(summary.pointWithdrawalAmount),
          });
        }
        this.setData(patch);
      })
      .catch((err) => {
        if (requestSeq !== this.requestSeq) return;
        const message = (err && err.message) || '今日营业明细加载失败';
        this.setData({
          loading: false,
          refreshing: false,
          loadingMore: false,
          errorText: this.data.list.length ? '' : message,
        });
        if (this.data.list.length || opts.refreshing) ui.error(message);
      });
  },

  mapEntry(item) {
    const isCoin = item.type === 'coin_consumption';
    const isWithdrawal = item.type === 'point_withdrawal';
    return {
      recordKey: item.recordKey,
      type: item.type,
      typeLabel: TYPE_LABEL[item.type] || item.type,
      memberName: item.memberName || '未命名会员',
      phoneText: fmt.maskPhone(item.phone) || '未绑定手机号',
      amountText: `${isCoin || isWithdrawal ? '-' : '+'}${fmt.amount(item.amount)} ${isCoin ? '金币' : '积分'}`,
      statusLabel: STATUS_LABEL[item.status] || item.status,
      orderText: item.businessOrderNo || '',
      timeText: fmt.dateTime(item.createdAt),
    };
  },

  selectType(e) {
    const activeType = e.currentTarget.dataset.type;
    if (activeType === this.data.activeType) return;
    this.setData({ activeType, loading: true, list: [], errorText: '', hasMore: true });
    this.loadPage(1);
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
