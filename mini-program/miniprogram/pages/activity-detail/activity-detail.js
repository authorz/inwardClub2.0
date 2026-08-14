// 活动详情 — 海报长页，预约操作跳转独立购票页面
const api = require('../../services/api');
const fmt = require('../../utils/format');
const storeCtx = require('../../utils/store-context');

const STATUS_LABEL = { enrolling: '报名中', upcoming: '即将开始', ended: '已结束', soldout: '已售罄' };

function isActivityExpired(activity) {
  if (!activity) return false;
  if (activity.status === 'ended') return true;
  const endAt = activity.endAt ? new Date(activity.endAt).getTime() : NaN;
  return Number.isFinite(endAt) && endAt <= Date.now();
}

function displayStoreForActivity(activity, stores) {
  const list = stores || [];
  if (activity.storeId != null) {
    return list.find((store) => Number(store.id) === Number(activity.storeId)) || null;
  }
  const current = storeCtx.get();
  if (current && current.id != null) {
    return list.find((store) => Number(store.id) === Number(current.id)) || current;
  }
  return list[0] || null;
}

Page({
  data: {
    loading: true,
    loadError: '',
    activity: null,
    minPriceText: '',

    // Custom navigation metrics (px).
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
    navSolid: false,

    // 地址信息（后端暂无经纬度时仅展示文字，不开启导航）
    mapPoint: null,
  },

  onLoad(options) {
    this.measureNav();
    const id = options.id;
    Promise.all([api.getActivity(id), storeCtx.listNearby()])
      .then(([res, stores]) => {
        const a = res.data || {};
        const displayStore = displayStoreForActivity(a, stores);
        const activityPayChannels = a.payChannels && a.payChannels.length ? a.payChannels : [];
        const purchaseLimit = Math.max(0, Number(a.purchaseLimit) || 0);
        const tickets = (a.ticketTypes || []).map((t) => ({
          id: t.id,
          name: t.name,
          priceCent: t.priceCent,
          priceText: fmt.centToYuan(t.priceCent),
          unlimitedStock: t.unlimitedStock,
          remainingStock: t.remainingStock,
          stockText: t.unlimitedStock ? '不限量' : `剩余 ${t.remainingStock} 张`,
          payChannels: t.payChannels && t.payChannels.length ? t.payChannels : activityPayChannels,
          maxTicketsPerOrder: Math.max(0, Number(t.maxTicketsPerOrder) || 0),
        }));
        tickets.forEach((ticket) => {
          const limits = [purchaseLimit, ticket.maxTicketsPerOrder, Number(ticket.remainingStock)]
            .filter((value) => value > 0);
          ticket.maxQuantity = limits.length ? Math.min.apply(null, limits) : 99;
          ticket.limitText = ticket.maxTicketsPerOrder
            ? `单次限购 ${ticket.maxTicketsPerOrder} 张`
            : purchaseLimit
              ? `每人限购 ${purchaseLimit} 张`
              : '不限购';
        });
        const min = tickets.reduce((acc, ticket) => (acc === null || ticket.priceCent < acc.priceCent ? ticket : acc), null);
        const detailNodes = a.content || a.detailHtml || a.introHtml || a.detail || a.intro || '';
        // 单店活动使用绑定门店；全门店活动使用用户当前选择或最近的门店。
        const lat = displayStore && displayStore.latitude != null
          ? displayStore.latitude
          : a.latitude != null ? a.latitude : a.lat;
        const lng = displayStore && displayStore.longitude != null
          ? displayStore.longitude
          : a.longitude != null ? a.longitude : a.lng;
        const storeName = (displayStore && displayStore.name) || a.storeName || 'Inward Club';
        const address = (displayStore && displayStore.address) || a.address || a.addressDetail || '';
        const distanceMeters = displayStore && displayStore.distanceMeters != null
          ? displayStore.distanceMeters
          : a.distanceMeters;
        const distanceText = distanceMeters != null
          ? Number(distanceMeters) === 0 ? '0m' : fmt.distance(distanceMeters)
          : lat != null && lng != null ? '未定位' : '';
        const startDate = fmt.dotMonthDay(a.startAt);
        const endDate = fmt.dotMonthDay(a.endAt);
        const startTime = fmt.dateTime(a.startAt, { timeOnly: true });
        const endTime = fmt.dateTime(a.endAt, { timeOnly: true });
        const mapPoint =
          lat != null && lng != null
            ? { latitude: Number(lat), longitude: Number(lng), name: storeName, address }
            : null;
        const isExpired = isActivityExpired(a);
        this.setData({
          loading: false,
          activity: {
            id: a.id,
            title: a.title,
            tone: a.tone,
            imageUrl: a.imageUrl || '',
            statusText: STATUS_LABEL[a.status] || '报名中',
            dateText: startDate && endDate ? `${startDate} - ${endDate}` : startDate || endDate,
            clockText: startTime && endTime ? `${startTime}-${endTime}` : startTime || endTime,
            storeName,
            address,
            locationText: [storeName, address].filter(Boolean).join(' · '),
            storeScopeText: a.scopeType === 'global' || a.storeId == null
              ? displayStore ? '当前门店' : '全部门店'
              : '共1家',
            description: a.description || '',
            distanceText,
            detailNodes,
            hasDetail: !!detailNodes,
            purchaseLimit,
            purchaseLimitText: purchaseLimit ? `每人限购 ${purchaseLimit} 张` : '不限购',
            tickets,
            hasTickets: tickets.length > 0,
            isExpired,
          },
          mapPoint,
          minPriceText: min ? min.priceText : '',
        });
      })
      .catch(() => this.setData({ loading: false, loadError: '活动加载失败，请返回后重试' }));
  },

  measureNav() {
    try {
      const win = wx.getWindowInfo();
      const cap = wx.getMenuButtonBoundingClientRect();
      const statusBar = win.statusBarHeight || 20;
      const gap = Math.max(cap.top - statusBar, 4);
      this._navRevealAt = (win.windowWidth * 680) / 750;
      this.setData({
        navStatusBar: statusBar,
        navContentHeight: cap.height + gap * 2,
        navRightGap: Math.max(win.windowWidth - cap.left + 8, 96),
      });
    } catch {
      this._navRevealAt = 340;
    }
  },

  onScroll(e) {
    const navSolid = e.detail.scrollTop >= (this._navRevealAt || 340);
    if (navSolid !== this.data.navSolid) this.setData({ navSolid });
  },

  goHome() {
    wx.switchTab({ url: '/pages/index/index' });
  },

  onShareAppMessage() {
    const activity = this.data.activity || {};
    return {
      title: activity.title || 'InwardClub 活动',
      path: `/pages/activity-detail/activity-detail?id=${activity.id || ''}`,
      imageUrl: activity.imageUrl || 'https://assets.inwardclub.com/logo/logo-2.jpg',
    };
  },

  onShareTimeline() {
    const activity = this.data.activity || {};
    return {
      title: activity.title || 'InwardClub 活动',
      query: `id=${activity.id || ''}`,
      imageUrl: activity.imageUrl || 'https://assets.inwardclub.com/logo/logo-2.jpg',
    };
  },

  // 地址行 / 地图卡片点击 → 拉起导航
  openLocation() {
    const p = this.data.mapPoint;
    if (!p) return;
    wx.openLocation({ latitude: p.latitude, longitude: p.longitude, name: p.name, address: p.address, scale: 16 });
  },

  openPurchase() {
    const a = this.data.activity;
    if (!a || !a.id || !a.hasTickets || a.isExpired) return;
    wx.navigateTo({ url: `/pages/activity-purchase/activity-purchase?id=${a.id}` });
  },
});
