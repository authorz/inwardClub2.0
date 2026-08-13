// 首页 — 活动轮播 + 会员卡 + 最近门店
// Reference: design/mini-program/final/home/01-home-final-iphone17.png
const api = require('../../services/api');
const auth = require('../../utils/auth');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const invitation = require('../../utils/invitation');
const { mergeCachedProfile } = require('../../utils/member-profile');
const { distance } = require('../../utils/format');

function storeDistanceLabel(store) {
  if (!store) return '';
  if (typeof store.distanceMeters === 'number') {
    return store.distanceMeters <= 0 ? '0m' : distance(store.distanceMeters);
  }
  const lat = store.lat != null ? store.lat : store.latitude;
  const lng = store.lng != null ? store.lng : store.longitude;
  return lat == null || lng == null ? '位置未配置' : '距离未知';
}

Page({
  data: {
    loggedIn: false,
    me: {},
    store: null,
    storeDistanceText: '',
    activities: [],
    signInStatus: {
      signedToday: false,
      streakDays: 0,
      rewardPoints: 0,
      nextRewardPoints: 0,
      dailyRewards: [],
    },
    signInSubmitting: false,
    showSignInSheet: false,
    showSignInReward: false,
    signInReward: 0,
    signInStreak: 0,
    // custom navigation metrics (px)
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
  },

  onLoad(options) {
    if (auth.isLoggedIn()) invitation.clear();
    else invitation.capture(options || {});
    this.measureNav();
    this.loadAll();
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

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 0 });
    }
    // Login state can change while the page is cached (e.g. after visiting a
    // gated page). Pull fresh member data when it flips to logged-in.
    if (auth.isLoggedIn() && !this.data.loggedIn) {
      this.refreshMe();
    } else if (auth.isLoggedIn()) {
      this.loadSignInStatus();
    } else if (!auth.isLoggedIn() && this.data.loggedIn) {
      this.setData({
        loggedIn: false,
        me: {},
        showSignInSheet: false,
        signInStatus: { signedToday: false, streakDays: 0, rewardPoints: 0, nextRewardPoints: 0, dailyRewards: [] },
      });
    }
    // Reflect a store change made on the store-select page.
    const s = storeCtx.get();
    if (s && (!this.data.store || s.id !== this.data.store.id)) {
      this.setData({
        store: s,
        storeDistanceText: storeDistanceLabel(s),
      });
    }
  },

  // Store + activities are open to guests; member data is fetched only when
  // authenticated so the card never shows fake member info before login.
  loadAll() {
    const loggedIn = auth.isLoggedIn();
    Promise.all([
      loggedIn ? api.getMe().catch(() => ({ data: {} })) : Promise.resolve({ data: {} }),
      this.resolveStore(),
      api.getActivities().catch(() => ({ data: [] })),
    ]).then(([meRes, store, actRes]) => {
      // getMe (services/api.js) already normalizes the VIP tier shape
      // (tierCode/tierName/tierShort); mergeCachedProfile only backfills
      // avatar/nickname/gender, so the tier fields pass through untouched.
      const me = loggedIn ? mergeCachedProfile(meRes.data || {}) : {};
      this.setData({
        loggedIn,
        me,
        store,
        storeDistanceText: storeDistanceLabel(store),
        activities: (actRes.data || []).slice(0, 4),
      });
      if (loggedIn) this.loadSignInStatus();
      // 广告 Banner 接口暂时停用：首页顶部统一展示活动列表图片。
      // if (store) api.getBanners(store.id).then(...);
    });
  },

  // Refresh just the member card (after login, or when returning logged-in).
  refreshMe() {
    return api
      .getMe()
      .then((res) => {
        let me = res.data || {};
        // Don't let an empty server avatar clobber a locally-chosen one
        // (e.g. the avatar just picked in the login sheet).
        if (!me.avatarUrl && this.data.me && this.data.me.avatarUrl) {
          me.avatarUrl = this.data.me.avatarUrl;
        }
        me = mergeCachedProfile(me);
        this.setData({ loggedIn: true, me });
        return this.loadSignInStatus();
      })
      .catch(() => this.setData({ loggedIn: auth.isLoggedIn() }));
  },

  // Triggered by the 登录 button in the member card.
  onLogin() {
    this.openLogin();
  },

  // Gate member-only entries through the dedicated login/registration page.
  requireLogin(next) {
    if (auth.isLoggedIn()) return next();
    this.openLogin(next);
  },

  openLogin(next) {
    wx.navigateTo({
      url: '/pages/login/login',
      success: (res) => {
        if (typeof next !== 'function' || !res.eventChannel) return;
        res.eventChannel.on('loginSuccess', () => {
          this.refreshMe();
          next();
        });
      },
    });
  },

  // app.onSessionExpired broadcasts here when a 401 invalidated the session: drop
  // the member card at once so the guest state (with the 登录 button) shows.
  onSessionExpired() {
    this.setData({
      loggedIn: false,
      me: {},
      showSignInSheet: false,
      signInStatus: { signedToday: false, streakDays: 0, rewardPoints: 0, nextRewardPoints: 0, dailyRewards: [] },
    });
  },

  loadSignInStatus() {
    if (!auth.isLoggedIn()) return Promise.resolve();
    return api
      .getSignInStatus()
      .then((res) => this.setData({ signInStatus: res.data || {} }))
      .catch(() => {});
  },

  onSignIn() {
    if (this.data.signInSubmitting) return;
    this.requireLogin(() => {
      this.setData({ showSignInSheet: true });
      if (!(this.data.signInStatus.dailyRewards || []).length) this.loadSignInStatus();
    });
  },

  closeSignInSheet() {
    if (!this.data.signInSubmitting) this.setData({ showSignInSheet: false });
  },

  confirmSignIn() {
    if (this.data.signInSubmitting || this.data.signInStatus.signedToday) return;
    this.setData({ signInSubmitting: true });
    api
      .signIn()
      .then((res) => {
        const result = res.data || {};
        const status = {
          signedToday: true,
          streakDays: result.streakDays || 1,
          rewardPoints: result.pointsEarned || 0,
          nextRewardPoints: this.data.signInStatus.nextRewardPoints || 0,
          dailyRewards: this.data.signInStatus.dailyRewards || [],
        };
        this.setData({ signInSubmitting: false, showSignInSheet: false, signInStatus: status });
        if (result.alreadySigned) {
          ui.toast('今天已经签到过了');
          return;
        }
        clearTimeout(this._signInRewardTimer);
        this.setData({
          showSignInReward: true,
          signInReward: result.pointsEarned || 0,
          signInStreak: result.streakDays || 1,
        });
        this._signInRewardTimer = setTimeout(() => {
          this.setData({ showSignInReward: false });
        }, 1600);
      })
      .catch((err) => {
        this.setData({ signInSubmitting: false });
        if (err && err.code === 'CONFLICT') {
          this.setData({ showSignInSheet: false });
          this.loadSignInStatus();
          ui.toast('今天已经签到过了');
          return;
        }
        ui.error((err && err.message) || '签到失败，请重试');
      });
  },

  // Current store = the member's persisted pick, else the nearest by location
  // (resolved centrally in store-context). Home is usually the first tab opened,
  // so this bootstraps the store for the whole app.
  resolveStore() {
    return storeCtx.ensureStore().then(() => storeCtx.refreshCurrent()).then((store) => {
      // After the store is set, check whether the member has traveled away from
      // their manually-picked store. Non-blocking — runs after the page renders.
      storeCtx.checkNearbyChange().then((nearest) => {
        if (!nearest) return;
        ui.confirm({
          title: '附近门店已更新',
          content: '最近的门店是"' + nearest.name + '"，要切换吗？',
          confirmText: '切换',
          cancelText: '保留',
        }).then((yes) => {
          if (!yes) return;
          storeCtx.switchStore(nearest);
          this.setData({
            store: nearest,
            storeDistanceText: storeDistanceLabel(nearest),
          });
        });
      });
      return store;
    });
  },

  inviteShareData() {
    const me = this.data.me || {};
    const inviteCode = auth.isLoggedIn() ? me.inviteCode || '' : '';
    return {
      inviteCode,
      title: '欢迎加入INWARD',
    };
  },

  onShareAppMessage() {
    const share = this.inviteShareData();
    const inviteCode = share.inviteCode;
    const path = inviteCode
      ? '/pages/index/index?invite=' + encodeURIComponent(inviteCode)
      : '/pages/index/index';
    return {
      title: share.title,
      path,
      imageUrl: 'https://assets.inwardclub.com/logo/logo-2.jpg',
    };
  },

  onShareTimeline() {
    const share = this.inviteShareData();
    return {
      title: share.title,
      query: share.inviteCode ? 'invite=' + encodeURIComponent(share.inviteCode) : '',
      imageUrl: 'https://assets.inwardclub.com/logo/logo-2.jpg',
    };
  },

  goTickets() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/tickets/tickets' }));
  },
  goCoupons() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/coupons/coupons' }));
  },
  goFranchiseInquiry() {
    wx.navigateTo({ url: '/pages/franchise-inquiry/franchise-inquiry' });
  },
  onBannerImageLoad(e) {
    const index = Number(e.currentTarget.dataset.index);
    if (!Number.isInteger(index) || index < 0 || index >= this.data.activities.length) return;
    this.setData({ [`activities[${index}].imageLoaded`]: true });
  },
  goActivity(e) {
    wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + e.currentTarget.dataset.id });
  },
  goActivityList() {
    wx.navigateTo({ url: '/pages/activity-list/activity-list' });
  },
  switchStore() {
    wx.navigateTo({ url: '/pages/store-select/store-select' });
  },
  navigateStore() {
    const store = this.data.store;
    const latitude = Number(store && (store.lat != null ? store.lat : store.latitude));
    const longitude = Number(store && (store.lng != null ? store.lng : store.longitude));
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
      ui.toast('当前门店暂未配置导航位置');
      return;
    }
    wx.openLocation({
      latitude,
      longitude,
      name: store.name || 'InwardClub',
      address: store.address || '',
      scale: 16,
      fail: () => ui.toast('地图打开失败，请稍后重试'),
    });
  },
});
