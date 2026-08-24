// 活动列表 — 深色自定义导航 + tower-swiper 塔式堆叠轮播；展示当前可报名活动
// Reference: design-demo 深色黑灰渐变风格
const api = require('../../services/api');
const fmt = require('../../utils/format');

const REVIEW_PAGE_SIZE = 10;
const REVIEW_FALLBACK_LOGO = 'https://assets.inwardclub.com/public/images/inward-logo-optimized.gif?imageMogr2/format/png';
const CURRENT_ACTIVITY_STATUSES = new Set(['enrolling', 'upcoming']);

function reviewItem(a) {
  return {
    id: a.id,
    title: a.title,
    timeText: a.timeText,
    storeName: a.storeName,
    imageUrl: a.imageUrl || '',
  };
}

Page({
  data: {
    loading: true,
    activities: [],
    current: 0,
    currentItem: null,
    // mode: 'current' 报名中（塔式轮播） | 'review' 精彩回顾（已结束列表）
    mode: 'current',
    reviews: [],
    reviewLoaded: false,
    reviewLoading: false,
    reviewLoadError: false,
    reviewPage: 0,
    reviewHasMore: true,
    reviewFallbackLogo: REVIEW_FALLBACK_LOGO,
    // custom navigation metrics (px)
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
  },

  onLoad() {
    this.measureNav();
    // 已发布且尚未开始的活动同样可以开放报名，不能因活动日期在未来而隐藏。
    api
      .getActivities()
      .catch(() => ({ data: [] }))
      .then((listRes) => {
        const activities = (listRes.data || [])
          .filter((a) => CURRENT_ACTIVITY_STATUSES.has(a.status))
          .map((a) => ({
            id: a.id,
            title: a.title,
            tone: a.tone,
            imageUrl: a.imageUrl || '',
            timeText: a.timeText,
            storeName: a.storeName,
            dateRangeText: fmt.monthDayRange(a.startAt, a.endAt),
          }));
        this.setData({
          activities,
          current: 0,
          currentItem: activities[0] || null,
          loading: false,
        });
      });
  },

  // Size the custom nav bar to the status bar + WeChat capsule button.
  measureNav() {
    try {
      const win = wx.getWindowInfo();
      const cap = wx.getMenuButtonBoundingClientRect();
      const statusBar = win.statusBarHeight || 20;
      const gap = Math.max(cap.top - statusBar, 4);
      this.setData({
        navStatusBar: statusBar,
        navContentHeight: cap.height + gap * 2,
        navRightGap: Math.max(win.windowWidth - cap.left + 8, 96),
      });
    } catch {
      /* keep defaults */
    }
  },

  goBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) wx.navigateBack();
    else wx.switchTab({ url: '/pages/index/index' });
  },

  // Active card changed inside the tower-swiper.
  onCardChange(e) {
    const current = e.detail.current;
    this.setData({ current, currentItem: this.data.activities[current] || null });
  },

  // Front card tapped inside the tower-swiper → open its detail.
  onCardTap(e) {
    const item = e.detail.item;
    if (item) wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + item.id });
  },

  goDetail() {
    const item = this.data.currentItem;
    if (item) wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + item.id });
  },

  // 我的入场券入口 → tickets 页（未登录由 http 层 401 兜底）
  goTickets() {
    wx.navigateTo({ url: '/pages/tickets/tickets' });
  },

  // 切到「近期活动」（塔式轮播）
  onCurrent() {
    if (this.data.mode !== 'current') this.setData({ mode: 'current' });
  },

  // 切到「精彩回顾」（已结束活动列表），首次进入时懒加载
  onReview() {
    if (this.data.mode === 'review') return;
    this.setData({ mode: 'review' });
    if (this.data.reviewLoaded || this.data.reviewLoading) return;
    this.loadReviews({ refresh: true });
  },

  loadReviews({ refresh = false } = {}) {
    if (this._reviewRequesting) {
      if (refresh) wx.stopPullDownRefresh();
      return;
    }
    if (!refresh && !this.data.reviewHasMore) return;

    const page = refresh ? 1 : this.data.reviewPage + 1;
    this._reviewRequesting = true;
    this.setData({ reviewLoading: true, reviewLoadError: false });

    return api
      .getActivities({ scope: 'history', page, pageSize: REVIEW_PAGE_SIZE })
      .then((listRes) => {
        const rows = (listRes.data || []).map(reviewItem);
        const previous = refresh ? [] : this.data.reviews;
        const seen = new Set(previous.map((item) => String(item.id)));
        const appended = rows.filter((item) => !seen.has(String(item.id)));
        const reviews = previous.concat(appended);
        const meta = listRes.meta || {};
        const total = Number(meta.total);
        const hasTotal = meta.total != null && !Number.isNaN(total);
        const hasMore = hasTotal ? page * REVIEW_PAGE_SIZE < total : rows.length === REVIEW_PAGE_SIZE;

        this.setData({
          reviews,
          reviewLoaded: true,
          reviewLoading: false,
          reviewLoadError: false,
          reviewPage: page,
          reviewHasMore: hasMore,
        });
      })
      .catch(() => {
        this.setData({ reviewLoaded: true, reviewLoading: false, reviewLoadError: true });
      })
      .finally(() => {
        this._reviewRequesting = false;
        wx.stopPullDownRefresh();
      });
  },

  retryReviews() {
    this.loadReviews({ refresh: !this.data.reviews.length });
  },

  onReachBottom() {
    if (this.data.mode === 'review') this.loadReviews();
  },

  onPullDownRefresh() {
    if (this.data.mode !== 'review') {
      wx.stopPullDownRefresh();
      return;
    }
    this.loadReviews({ refresh: true });
  },

  // 精彩回顾列表项点击 → 进入活动详情
  onReviewTap(e) {
    const id = e.currentTarget.dataset.id;
    if (id) wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + id });
  },
});
