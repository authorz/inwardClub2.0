// 首页 — banner + 会员卡 + 最近门店 + 最新活动
// Reference: design/mini-program/final/home/01-home-final-iphone17.png
const api = require('../../services/api');
const auth = require('../../utils/auth');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const { saveProfilePersistently, mergeCachedProfile } = require('../../utils/member-profile');

Page({
  data: {
    loading: true,
    loggedIn: false,
    showLoginSheet: false,
    loginSubmitting: false,
    loginMode: 'register', // 'register' (new user, all fields) | 'bindPhone' (returning, phone only)
    loginForm: {
      avatarUrl: '',
      nickName: '',
      gender: '',
      phoneAuthed: false,
      phoneCode: '',
      phoneEncryptedData: '',
      phoneIv: '',
    },
    me: {},
    store: null,
    banner: null,
    activities: [],
    // custom navigation metrics (px)
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
  },

  onLoad() {
    this.measureNav();
    this.loadAll();
  },

  // Size the custom nav bar to the status bar + WeChat capsule button.
  measureNav() {
    try {
      const win = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const cap = wx.getMenuButtonBoundingClientRect();
      const statusBar = win.statusBarHeight || 20;
      const gap = Math.max(cap.top - statusBar, 4);
      this.setData({
        navStatusBar: statusBar,
        navContentHeight: cap.height + gap * 2,
        navRightGap: Math.max(win.windowWidth - cap.left + 8, 96),
      });
    } catch (e) {
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
    } else if (!auth.isLoggedIn() && this.data.loggedIn) {
      this.setData({ loggedIn: false, me: {} });
    }
    // Reflect a store change made on the store-select page.
    const s = storeCtx.get();
    if (s && (!this.data.store || s.id !== this.data.store.id)) {
      this.setData({ store: s });
      this.loadBanner(s.id);
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
        activities: (actRes.data || []).slice(0, 4),
        loading: false,
      });
      if (store) this.loadBanner(store.id);
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
      })
      .catch(() => this.setData({ loggedIn: auth.isLoggedIn() }));
  },

  // Triggered by the 登录 button in the member card.
  onLogin() {
    this.silentLogin();
  },

  // Gate for member-only entries: log in silently, then run `next`. Returning
  // members go straight through; new members complete the profile sheet first.
  requireLogin(next) {
    if (auth.isLoggedIn()) return next();
    this.silentLogin(next);
  },

  // Silent login: exchange a wx.login code for a session WITHOUT any profile
  // input, then branch on the backend's identity signals:
  //   - returning member WITH a phone (isNew=false, profile.phone) -> straight in
  //   - returning member WITHOUT a phone -> force the phone-binding sheet (law)
  //   - first-time member (isNew=true) -> full registration form (all required)
  silentLogin(next) {
    if (this.data.loginSubmitting) return;
    this.setData({ loginSubmitting: true });
    wx.login({
      success: (loginRes) => {
        api
          .wechatLogin({ code: loginRes.code })
          .then((r) => {
            const d = r.data || {};
            const tk = d.token || d; // { token: TokenPair, profile, isNew }
            const profile = d.profile || {};
            auth.save({
              accessToken: tk.accessToken,
              refreshToken: tk.refreshToken,
              subjectType: d.subjectType,
              storeId: d.storeId,
            });
            if (d.isNew) {
              // Registration: full profile required.
              this._loginNextAction = next || null;
              this.setData({ loginSubmitting: false, loginMode: 'register', showLoginSheet: true });
            } else if (!profile.phone) {
              // Returning member missing a phone: must bind one before continuing.
              this._loginNextAction = next || null;
              this.setData({ loginSubmitting: false, loginMode: 'bindPhone', showLoginSheet: true });
            } else {
              // Returning member with a phone: straight in.
              this.setData({ loginSubmitting: false, loggedIn: true });
              this.refreshMe();
              if (typeof next === 'function') next();
            }
          })
          .catch((err) => {
            this.setData({ loginSubmitting: false });
            if (err && err.isApiError) ui.error(err.message || '登录失败');
          });
      },
      fail: () => {
        this.setData({ loginSubmitting: false });
        ui.error('登录失败');
      },
    });
  },

  closeLoginSheet() {
    this._loginNextAction = null;
    this.setData({ showLoginSheet: false });
  },

  // app.onSessionExpired broadcasts here when a 401 invalidated the session: drop
  // the member card at once so the guest state (with the 登录 button) shows.
  onSessionExpired() {
    this.setData({ loggedIn: false, me: {} });
  },

  onChooseLoginAvatar(e) {
    this.setData({ 'loginForm.avatarUrl': e.detail.avatarUrl });
  },

  onLoginNickname(e) {
    this.setData({ 'loginForm.nickName': e.detail.value });
  },

  onGetLoginPhoneNumber(e) {
    const d = e.detail || {};
    this.setData({
      'loginForm.phoneAuthed': !!(d.code || d.encryptedData),
      'loginForm.phoneCode': d.code || '',
      'loginForm.phoneEncryptedData': d.encryptedData || '',
      'loginForm.phoneIv': d.iv || '',
    });
  },

  onLoginGender(e) {
    this.setData({ 'loginForm.gender': e.detail.value });
  },

  // Sheet submit. The session already exists (silentLogin ran). Two modes:
  //   register  — new member: avatar + nickname + phone + gender ALL required.
  //   bindPhone — returning member missing a phone: phone only (legal mandate).
  confirmLogin() {
    if (this.data.loginSubmitting) return;
    const form = this.data.loginForm;
    const nickName = (form.nickName || '').trim();
    const register = this.data.loginMode === 'register';
    const phoneAuthed = form.phoneAuthed && (form.phoneCode || form.phoneEncryptedData);

    if (register) {
      if (!form.avatarUrl) return ui.toast('请选择头像');
      if (!nickName) return ui.toast('请填写昵称');
      if (!phoneAuthed) return ui.toast('请获取手机号');
      if (!form.gender) return ui.toast('请选择性别');
    } else {
      if (!phoneAuthed) return ui.toast('请获取手机号');
    }
    this.setData({ loginSubmitting: true });

    // Phone binding is required in both modes. In register mode also push the
    // rest of the profile; in bindPhone mode submit nothing else.
    const tasks = [
      api.bindPhone({ code: form.phoneCode, encryptedData: form.phoneEncryptedData, iv: form.phoneIv }),
    ];
    if (register) {
      const mePatch = {};
      if (nickName) mePatch.nickname = nickName;
      if (form.gender) mePatch.gender = form.gender;
      if (Object.keys(mePatch).length) tasks.push(api.updateMe(mePatch).catch(() => {}));
    }

    Promise.all(tasks)
      .then(() => {
        // Persist the picked profile locally so a refresh restores the avatar
        // even when getMe returns an empty avatarUrl.
        const profile = {};
        if (form.avatarUrl) profile.avatarUrl = form.avatarUrl;
        if (nickName) profile.nickname = nickName;
        if (form.gender) profile.gender = form.gender;
        const next = this._loginNextAction;
        this._loginNextAction = null;
        const patch = { showLoginSheet: false, loginSubmitting: false, loggedIn: true };
        if (form.avatarUrl) patch['me.avatarUrl'] = form.avatarUrl;
        this.setData(patch);
        if (Object.keys(profile).length) {
          saveProfilePersistently(profile).then((saved) => {
            if (saved && saved.avatarUrl && saved.avatarUrl !== form.avatarUrl) {
              this.setData({ 'me.avatarUrl': saved.avatarUrl });
            }
          });
        }
        this.refreshMe();
        if (typeof next === 'function') next();
      })
      .catch((err) => {
        this.setData({ loginSubmitting: false });
        if (err && err.isApiError) ui.error(err.message || '提交失败');
        else ui.error('提交失败');
      });
  },

  // Current store = the member's persisted pick, else the nearest by location
  // (resolved centrally in store-context). Home is usually the first tab opened,
  // so this bootstraps the store for the whole app.
  resolveStore() {
    return storeCtx.ensureStore().then((store) => {
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
          this.setData({ store: nearest });
          this.loadBanner(nearest.id);
        });
      });
      return store;
    });
  },

  loadBanner(storeId) {
    if (!storeId) return;
    api
      .getBanners(storeId)
      .then((res) => this.setData({ banner: (res.data || [])[0] || null }))
      .catch(() => {});
  },

  goTickets() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/tickets/tickets' }));
  },
  goCoupons() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/coupons/coupons' }));
  },
  goActivities() {
    wx.navigateTo({ url: '/pages/activity-list/activity-list' });
  },
  goActivity(e) {
    wx.navigateTo({ url: '/pages/activity-detail/activity-detail?id=' + e.currentTarget.dataset.id });
  },
  switchStore() {
    wx.navigateTo({ url: '/pages/store-select/store-select' });
  },
  navStore() {
    const s = this.data.store;
    if (s && s.lat && s.lng) {
      wx.openLocation({ latitude: s.lat, longitude: s.lng, name: s.name, address: s.address || '' });
    } else {
      ui.toast('暂无门店位置');
    }
  },
});
