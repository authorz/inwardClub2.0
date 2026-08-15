// 订单中心 — 点餐/活动/充值/兑换 四类订单分段切换 + 状态筛选
// Reference: design/mini-program/final/member-subpages/03-order-center-v23.png
const api = require('../../services/api');
const fmt = require('../../utils/format');
const {
  ORDER_TYPE,
  ORDER_TYPE_LABEL,
  ORDER_STATUS_LABEL,
  ORDER_STATUS_TONE,
} = require('../../constants/index');

// 商品摘要：取前 2 个商品「名称 xN」，用「、」连接；超过 2 个追加「等 N 件」
function itemsSummary(items) {
  if (!items || !items.length) return '';
  const head = items.slice(0, 2).map((it) => (it.name || '') + ' x' + (it.qty || 1));
  let text = head.join('、');
  if (items.length > 2) text += ' 等' + items.length + '件';
  return text;
}

function payChannelText(ch) {
  if (!ch) return '';
  if (ch.indexOf('wechat') >= 0 && ch.indexOf('coin') >= 0) return '微信+金币';
  if (ch === 'coin') return '金币';
  if (ch === 'wechat') return '微信';
  return ch;
}

Page({
  data: {
    type: ORDER_TYPE.FOOD,
    typeOptions: Object.keys(ORDER_TYPE_LABEL).map((v) => ({ label: ORDER_TYPE_LABEL[v], value: v })),
    statusOptions: [
      { label: '全部', value: 'all' },
      { label: '进行中', value: 'active' },
      { label: '已完成', value: 'done' },
      { label: '已取消', value: 'cancelled' },
    ],
    status: 'all',
    loading: true,
    all: [],
    list: [],
    scrollEnabled: false,
  },

  onLoad(options) {
    if (options.type) this.setData({ type: options.type });
    this.loadOrders();
  },

  onReady() {
    this.updateScrollState();
  },

  onResize() {
    this.updateScrollState();
  },

  onTypeChange(e) {
    this.setData({ type: e.detail.value, status: 'all' });
    this.loadOrders();
  },

  onStatusChange(e) {
    this.setData({ status: e.detail.value });
    this.applyFilter();
  },

  loadOrders() {
    this.setData({ loading: true, scrollEnabled: false });
    const type = this.data.type;
    const fetch =
      type === ORDER_TYPE.FOOD
        ? api.getFoodOrders()
        : type === ORDER_TYPE.ACTIVITY
        ? api.getActivityOrders()
        : type === ORDER_TYPE.RECHARGE
        ? api.getRechargeOrders()
        : api.getRedemptionOrders();
    fetch
      .then((res) => {
        const all = (res.data || []).map((o) => this.normalize(type, o));
        this.setData({ all, loading: false });
        this.applyFilter();
      })
      .catch(() => this.setData({ loading: false, all: [], list: [] }, () => this.updateScrollState()));
  },

  normalize(type, o) {
    const base = {
      id: o.id,
      type,
      code: o.orderNo ? '#' + o.orderNo : '',
      status: o.status,
      statusText: ORDER_STATUS_LABEL[o.status] || o.status || '',
      tone: ORDER_STATUS_TONE[o.status] || 'neutral',
      timeText: fmt.dateTime(o.createdAt || o.paidAt),
      storeName: o.storeName || '',
      payText: payChannelText(o.payChannel),
      amountText: '',
      desc: '',
      title: '',
    };
    if (type === ORDER_TYPE.FOOD) {
      base.title = '点餐订单';
      base.desc = itemsSummary(o.items);
      base.amountText = fmt.yuan(o.payCent != null ? o.payCent : o.totalCent);
    } else if (type === ORDER_TYPE.ACTIVITY) {
      base.title = '活动订单';
      base.desc = (o.activityTitle || '') + (o.ticketName ? ' · ' + o.ticketName : '') + (o.qty ? ' x' + o.qty : '');
      base.amountText = fmt.yuan(o.amountCent);
    } else if (type === ORDER_TYPE.RECHARGE) {
      base.title = '充值订单';
      base.desc = '到账 ' + ((o.coins || 0) + (o.bonusCoins || 0)) + ' 金币';
      base.amountText = fmt.yuan(o.amountCent);
      base.payText = base.payText || '微信';
    } else {
      base.title = '兑换订单';
      base.desc = o.title || o.couponName || '';
    }
    return base;
  },

  applyFilter() {
    const s = this.data.status;
    let list = this.data.all;
    if (s === 'active') list = list.filter((o) => o.tone === 'active');
    else if (s === 'done') list = list.filter((o) => o.tone === 'done');
    else if (s === 'cancelled') list = list.filter((o) => o.tone === 'neutral' || o.tone === 'danger');
    this.setData({ list }, () => this.updateScrollState());
  },

  updateScrollState() {
    wx.nextTick(() => {
      const query = this.createSelectorQuery();
      query.select('.oce__scroll').boundingClientRect();
      query.select('.oce').boundingClientRect();
      query.exec((rects) => {
        const viewport = rects && rects[0];
        const content = rects && rects[1];
        if (!viewport || !content) return;
        const scrollEnabled = content.height > viewport.height + 2;
        if (scrollEnabled !== this.data.scrollEnabled) this.setData({ scrollEnabled });
      });
    });
  },

  onTapOrder(e) {
    const { id, type } = e.currentTarget.dataset;
    const map = {
      food: 'food-order-detail',
      activity: 'activity-order-detail',
      recharge: 'recharge-order-detail',
      coupon: 'redemption-order-detail',
    };
    const page = map[type];
    if (page) wx.navigateTo({ url: '/pages/' + page + '/' + page + '?id=' + id });
  },
});
