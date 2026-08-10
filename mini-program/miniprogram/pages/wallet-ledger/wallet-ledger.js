// 钱包流水 — 金币/积分/券分段 + 方向分段 + 流水列表
// Reference: design/mini-program/final/member-subpages/02-wallet-ledger-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const { ASSET_TYPE } = require('../../constants/index');

const DIRS = [
  { label: '全部', value: 'all' },
  { label: '收入', value: 'income' },
  { label: '支出', value: 'expense' },
];

Page({
  data: {
    loading: true,
    asset: ASSET_TYPE.COIN,
    assetOptions: [
      { label: '金币', value: ASSET_TYPE.COIN },
      { label: '积分', value: ASSET_TYPE.POINT },
      { label: '券', value: ASSET_TYPE.COUPON },
      { label: '充值', value: ASSET_TYPE.RECHARGE },
    ],
    dirOptions: DIRS,
    dir: 'all',
    all: [],
    list: [],
  },

  onLoad(options) {
    if (options.asset) this.setData({ asset: options.asset });
    this.loadLedger();
  },

  onAssetChange(e) {
    this.setData({ asset: e.detail.value, dir: 'all' });
    this.loadLedger();
  },

  onDirChange(e) {
    this.setData({ dir: e.detail.value });
    this.applyFilter();
  },

  loadLedger() {
    this.setData({ loading: true });
    api
      .getWalletLedger({ asset: this.data.asset })
      .then((res) => {
        const all = (res.data || []).map((x) => ({
          id: x.id,
          title: x.title,
          note: x.note,
          direction: x.direction,
          timeText: fmt.dateTime(x.createdAt),
          deltaText: (x.delta > 0 ? '+' : '') + fmt.amount(x.delta),
        }));
        this.setData({ all, loading: false });
        this.applyFilter();
      })
      .catch(() => this.setData({ loading: false }));
  },

  applyFilter() {
    const dir = this.data.dir;
    const list = dir === 'all' ? this.data.all : this.data.all.filter((x) => x.direction === dir);
    this.setData({ list });
  },
});
