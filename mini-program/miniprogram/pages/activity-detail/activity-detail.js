// 活动详情 — 默认详情态 + 点击“立即购票”后的票档/支付弹层
// Reference: design/mini-program/generated/v16/07-activity-detail-v16-default-source.png
//            design/mini-program/generated/v16/07-activity-detail-v16-purchase-sheet-source.png
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const http = require('../../utils/request');
const pay = require('../../utils/pay');
const { PAY_METHOD, PAY_METHOD_LABEL } = require('../../constants/index');

const STATUS_LABEL = { enrolling: '报名中', upcoming: '即将开始', ended: '已结束', soldout: '已售罄' };

// 顶部锚点 tab —— 与页面三个区块 id（sec-info / sec-detail / sec-notice）一一对应
const TABS = [
  { label: '活动信息', value: 'info' },
  { label: '图文详情', value: 'detail' },
  { label: '购票须知', value: 'notice' },
];

Page({
  data: {
    loading: true,
    activity: null,
    minPriceText: '',
    minTicketName: '',

    // 顶部锚点 tab + 滚动联动
    tabs: TABS,
    activeTab: 'info',
    intoView: '',
    tabFixed: false,

    // 地址地图（后端暂无经纬度时为 null，退回文字地址行）
    mapPoint: null,
    mapMarkers: [],
    // 购票须知（用已有真实字段组装，不造条款文案）
    noticeItems: [],

    showPurchase: false,
    ticketId: '',
    ticket: null,
    qty: 1,
    payMethod: PAY_METHOD.WECHAT,
    payMethods: [PAY_METHOD.WECHAT, PAY_METHOD.COIN],
    totalText: '0.00',
    submitting: false,
  },

  onLoad(options) {
    const id = options.id;
    api
      .getActivity(id)
      .then((res) => {
        const a = res.data || {};
        const tickets = (a.ticketTypes || []).map((t) => ({
          id: t.id,
          name: t.name,
          priceCent: t.priceCent,
          priceText: fmt.centToYuan(t.priceCent),
          stock: t.stock,
          saleEndText: fmt.dateTime(t.saleEndAt),
          payChannels: t.payChannels && t.payChannels.length ? t.payChannels : [PAY_METHOD.WECHAT, PAY_METHOD.COIN],
        }));
        const min = tickets.reduce((acc, t) => (acc === null || t.priceCent < acc.priceCent ? t : acc), null);
        const purchaseLimit = a.purchaseLimit || 4;
        const detailNodes = a.detailHtml || a.introHtml || a.detail || a.intro || '';
        // 地址：仅当后端给出经纬度时才做地图卡片，否则退回文字地址行
        const lat = a.latitude != null ? a.latitude : a.lat;
        const lng = a.longitude != null ? a.longitude : a.lng;
        const mapPoint =
          lat != null && lng != null
            ? { latitude: Number(lat), longitude: Number(lng), name: a.storeName || '', address: a.address || a.addressDetail || '' }
            : null;
        this.setData({
          loading: false,
          activity: {
            id: a.id,
            title: a.title,
            tone: a.tone,
            imageUrl: a.imageUrl || '',
            statusText: STATUS_LABEL[a.status] || '报名中',
            timeText: a.timeText,
            storeName: a.storeName,
            distanceText: fmt.distance(a.distanceMeters),
            saleEndText: fmt.dateTime(a.saleEndAt),
            detailNodes,
            hasDetail: !!detailNodes,
            purchaseLimit,
            tickets,
          },
          mapPoint,
          mapMarkers: mapPoint ? [{ id: 0, latitude: mapPoint.latitude, longitude: mapPoint.longitude }] : [],
          noticeItems: this.buildNoticeItems({ tickets, purchaseLimit, saleEndText: min ? min.saleEndText : fmt.dateTime(a.saleEndAt) }),
          minPriceText: min ? min.priceText : '',
          minTicketName: min ? min.name : '',
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  // 用已有真实字段组装购票须知，不编造条款文案
  buildNoticeItems({ tickets, purchaseLimit, saleEndText }) {
    const channels = [];
    (tickets || []).forEach((t) => (t.payChannels || []).forEach((c) => channels.indexOf(c) < 0 && channels.push(c)));
    const payText = channels.map((c) => PAY_METHOD_LABEL[c] || c).join(' / ');
    const items = [];
    if (purchaseLimit) items.push({ label: '限购', text: `每人限购 ${purchaseLimit} 张` });
    if (saleEndText) items.push({ label: '售卖', text: `售卖至 ${saleEndText}` });
    if (payText) items.push({ label: '支付', text: `支持 ${payText}` });
    return items;
  },

  // ---- 顶部锚点 tab 与滚动双向联动 ----
  onReady() {
    this.observeSections();
  },
  onUnload() {
    if (this._io) this._io.disconnect();
  },

  // 点 tab → 平滑滚到区块；期间短暂锁住反向高亮，避免滚动过程 tab 抖动
  onPickTab(e) {
    const value = e.detail.value;
    this._lockUntil = Date.now() + 600;
    this.setData({ activeTab: value, intoView: 'sec-' + value });
  },

  // 滚动过 hero 后 tab 吸顶（用 scroll-view 的 scrollTop 阈值近似）
  onScroll(e) {
    const top = e.detail.scrollTop;
    const fixed = top > 200;
    if (fixed !== this.data.tabFixed) this.setData({ tabFixed: fixed });
  },

  // 反向联动：区块进入视口顶部区域 → 高亮对应 tab
  observeSections() {
    if (this._io) this._io.disconnect();
    const io = this.createIntersectionObserver({ observeAll: true });
    // 视口顶部 20% 处作为判定线
    io.relativeToViewport({ top: 0, bottom: -Math.round(this._viewH() * 0.8) }).observe('.ad__anchor', (res) => {
      if (Date.now() < (this._lockUntil || 0)) return;
      if (!res || !res.dataset) return;
      const value = res.dataset.tab;
      if (value && value !== this.data.activeTab) this.setData({ activeTab: value });
    });
    this._io = io;
  },
  _viewH() {
    try {
      return wx.getSystemInfoSync().windowHeight || 700;
    } catch (e) {
      return 700;
    }
  },

  // 地址行 / 地图卡片点击 → 拉起导航
  openLocation() {
    const p = this.data.mapPoint;
    if (!p) return;
    wx.openLocation({ latitude: p.latitude, longitude: p.longitude, name: p.name, address: p.address, scale: 16 });
  },

  openPurchase() {
    const a = this.data.activity;
    if (!a || !a.tickets.length) return;
    const first = a.tickets[0];
    this.setData({ showPurchase: true });
    this.selectTicket(first);
  },
  closePurchase() {
    this.setData({ showPurchase: false });
  },
  onPickTicket(e) {
    const t = this.data.activity.tickets.find((x) => x.id === e.currentTarget.dataset.id);
    if (t) this.selectTicket(t);
  },
  selectTicket(t) {
    const payMethods = t.payChannels;
    const payMethod = payMethods.indexOf(this.data.payMethod) >= 0 ? this.data.payMethod : payMethods[0];
    this.setData({ ticketId: t.id, ticket: t, qty: 1, payMethods, payMethod });
    this.recalc();
  },
  onQty(e) {
    this.setData({ qty: e.detail.value });
    this.recalc();
  },
  onPay(e) {
    this.setData({ payMethod: e.detail.value });
  },
  recalc() {
    const t = this.data.ticket;
    const total = t ? t.priceCent * this.data.qty : 0;
    this.setData({ totalText: fmt.centToYuan(total) });
  },

  confirmPurchase() {
    const t = this.data.ticket;
    const a = this.data.activity;
    if (!t || this.data.submitting) return;
    const amountCent = t.priceCent * this.data.qty;
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    api
      .createActivityOrder(
        { activityId: a.id, ticketTypeId: t.id, qty: this.data.qty, amountCent, payChannel: this.data.payMethod },
        http.uuid()
      )
      .then((res) => {
        const poid = (res.data && res.data.paymentOrderId) || 'po_activity';
        return this.data.payMethod === PAY_METHOD.COIN
          ? api.payByCoin(poid, http.uuid())
          : api.payWechatJsapi(poid, http.uuid()).then((r) => pay.settle(r));
      })
      .then(() => {
        ui.hideLoading();
        this.setData({ submitting: false, showPurchase: false });
        ui.success('购票成功');
        setTimeout(() => wx.navigateTo({ url: '/pages/tickets/tickets' }), 600);
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '购票失败');
      });
  },

  noop() {},
});
