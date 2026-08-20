// 今日营业明细 — 当前门店金币消费与积分存取记录
const api = require('../../services/api');
const fmt = require('../../utils/format');

const TYPE_LABEL = {
  coin_consumption: '金币消费',
  point_deposit: '积分存入',
  point_withdrawal: '积分提取',
};
const STATUS_LABEL = { pending: '待审核', approved: '已通过', rejected: '已驳回', completed: '已完成' };

Page({
  data: {
    loading: true,
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
    this.load();
  },

  load() {
    this.setData({ loading: true });
    api.staff
      .getTodayOperations()
      .then((res) => {
        const data = res.data || {};
        const summary = data.summary || {};
        this.all = (data.entries || []).map((item) => this.mapEntry(item));
        this.setData({
          loading: false,
          date: data.date || '',
          summary: Object.assign({}, this.data.summary, summary, {
            coinConsumptionAmountText: fmt.amount(summary.coinConsumptionAmount),
            pointDepositAmountText: fmt.amount(summary.pointDepositAmount),
            pointWithdrawalAmountText: fmt.amount(summary.pointWithdrawalAmount),
          }),
        });
        this.applyFilter();
      })
      .catch(() => this.setData({ loading: false, list: [] }));
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
    this.setData({ activeType: e.currentTarget.dataset.type });
    this.applyFilter();
  },

  applyFilter() {
    const active = this.data.activeType;
    const list = active === 'all' ? (this.all || []) : (this.all || []).filter((item) => item.type === active);
    this.setData({ list });
  },
});
