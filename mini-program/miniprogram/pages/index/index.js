// 首页 — banner + 会员卡 + 最近门店 + 最新活动
// Reference: design/mini-program/final/home/01-home-final-iphone17.png
const api = require('../../services/api');
const auth = require('../../utils/auth');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const invitation = require('../../utils/invitation');
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
      phoneMasked: '', // 后端解密回传的打码号码（授权即绑后显示）
      phoneBound: false, // 手机号是否已在授权时绑定成功
      phoneBinding: false, // 换号请求进行中，防重复点击
    },
    me: {},
    store: null,
    banner: null,
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
      if (loggedIn) this.loadSignInStatus();
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
        return this.loadSignInStatus();
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
            const tk = d.token || {};
            const profile = d.profile || {};
            if (d.isNew) {
              // First-time user: the backend created NO member and issued NO
              // session — only a register ticket. Nothing is in the DB yet. Hold
              // the ticket; the member is inserted only when the form is submitted
              // (confirmLogin -> api.register). Do NOT save a session here.
              this._registerTicket = d.registerTicket || '';
              this._loginNextAction = next || null;
              this.setData({ loginSubmitting: false, loginMode: 'register', loginForm: this.freshLoginForm(), showLoginSheet: true });
              return;
            }
            // Returning member: a real session exists — persist it now.
            auth.save({
              accessToken: tk.accessToken,
              refreshToken: tk.refreshToken,
              subjectType: d.subjectType,
              storeId: d.storeId,
            });
            invitation.clear();
            if (!profile.phone) {
              // Returning member missing a phone: must bind one before continuing.
              this._loginNextAction = next || null;
              this.setData({ loginSubmitting: false, loginMode: 'bindPhone', loginForm: this.freshLoginForm(), showLoginSheet: true });
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

  // 打开登录/注册弹窗前的干净表单，避免上次残留（尤其 phoneBound）。
  freshLoginForm() {
    return {
      avatarUrl: '', nickName: '', gender: '',
      phoneAuthed: false, phoneCode: '', phoneEncryptedData: '', phoneIv: '',
      phoneMasked: '', phoneBound: false, phoneBinding: false,
    };
  },

  onChooseLoginAvatar(e) {
    this.setData({ 'loginForm.avatarUrl': e.detail.avatarUrl });
  },

  // type="nickname" 的微信昵称：选择后有时只在 bindblur（失焦）而非 bindinput
  // 才回传值，故 input 与 blur 都绑到这里。blur 传空时不覆盖已填内容。
  onLoginNickname(e) {
    console.log(e);
    const v = (e.detail && e.detail.value) || '';
    if (e.type === 'blur' && !v) return;
    this.setData({ 'loginForm.nickName': v });
  },

  // 微信授权仅返回一次性 code；前端拿不到明文号码。
  //   注册模式：调用公开接口解密 code 获取脱敏号码显示在表单中。
  //   绑手机模式（老会员）：已有会话，立即调 bindPhone 解密绑定并回填打码号码。
  onGetLoginPhoneNumber(e) {
    const d = e.detail || {};
    const code = d.code || '';
    // 用户拒绝授权或未拿到 code
    if (!code && !d.encryptedData) return;

    if (this.data.loginMode === 'register') {
      // 注册模式：调用公开接口解密一次性 code，回填脱敏号码。后端把手机号写进
      // 新的 register ticket 返回，注册时直接从 ticket 读，不再二次解密 code。
      if (this.data.loginForm.phoneBinding) return;
      this.setData({ 'loginForm.phoneBinding': true });
      api
        .getPhoneMask({ registerTicket: this._registerTicket, phoneCode: code })
        .then((res) => {
          const d = (res && res.data) || {};
          // 换成携带手机号的新 ticket，供"完成注册"使用。
          if (d.registerTicket) this._registerTicket = d.registerTicket;
          this.setData({
            'loginForm.phoneBound': true,
            'loginForm.phoneAuthed': true,
            'loginForm.phoneMasked': d.phoneMasked || '',
            'loginForm.phoneBinding': false,
          });
        })
        .catch((err) => {
          this.setData({ 'loginForm.phoneBinding': false });
          ui.error((err && err.message) || '手机号获取失败，请重试');
        });
      return;
    }

    if (this.data.loginForm.phoneBinding) return;
    this.setData({ 'loginForm.phoneBinding': true });
    api
      .bindPhone({ code, encryptedData: d.encryptedData, iv: d.iv })
      .then((res) => {
        const masked = (res && res.data && res.data.phoneMasked) || '';
        this.setData({
          'loginForm.phoneBound': true,
          'loginForm.phoneAuthed': true,
          'loginForm.phoneMasked': masked,
          'loginForm.phoneBinding': false,
        });
      })
      .catch((err) => {
        this.setData({ 'loginForm.phoneBinding': false });
        ui.error((err && err.message) || '手机号获取失败，请重试');
      });
  },

  onLoginGender(e) {
    this.setData({ 'loginForm.gender': e.detail.value });
  },

  // Sheet submit. Two modes:
  //   register  — FIRST-TIME member: this is the ONLY action that creates the
  //               member. Validates the full form, then POSTs the register ticket
  //               + nickname + held phone code; the backend resolves the phone and
  //               inserts the row, returning a real session. Nothing was persisted
  //               before this tap.
  //   bindPhone — returning member: session already exists and the phone was bound
  //               at authorize time, so this only closes the sheet.
  confirmLogin() {
    if (this.data.loginSubmitting) return;
    const form = this.data.loginForm;
    const nickName = (form.nickName || '').trim();
    const register = this.data.loginMode === 'register';

    if (register) {
      if (!form.avatarUrl) return ui.toast('请选择头像');
      if (!nickName) return ui.toast('请填写昵称');
      if (!form.phoneBound) return ui.toast('请获取手机号');
      if (!form.gender) return ui.toast('请选择性别');
    } else {
      if (!form.phoneBound) return ui.toast('请获取手机号');
    }
    this.setData({ loginSubmitting: true });

    // finish persists the locally-picked profile (avatar is never uploaded;
    // nickname is set server-side at register; gender is local-only) and flips the
    // card to logged-in.
    const finish = () => {
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
    };

    if (!register) {
      // bindPhone mode: session + phone already established during the form.
      finish();
      return;
    }

    // Register mode: upload the chosen avatar to get a public https URL, then
    // create the member. The phone was already authorized at 获取手机号 time and
    // rides inside this._registerTicket (no phoneCode is re-sent here — the
    // one-time code was already consumed).
    this.ensureUploadedAvatar(form.avatarUrl)
      .then((avatarUrl) =>
        api.register({
          registerTicket: this._registerTicket,
          avatarUrl,
          nickname: nickName,
          gender: form.gender,
          inviterCode: invitation.get(),
        })
      )
      .then((r) => {
        const d = r.data || {};
        const tk = d.token || {};
        this._registerTicket = '';
        auth.save({
          accessToken: tk.accessToken,
          refreshToken: tk.refreshToken,
          subjectType: d.subjectType,
          storeId: d.storeId,
        });
        invitation.clear();
        finish();
      })
      .catch((err) => {
        this.setData({ loginSubmitting: false });
        if (err && err.isApiError) ui.error(err.message || '注册失败');
        else ui.error((err && err.message) || '注册失败，请重试');
      });
  },

  // Upload the locally-chosen avatar (chooseAvatar returns a device-local temp
  // file, NOT an https URL) to object storage and resolve its public https URL.
  // An already-uploaded https URL passes through untouched.
  ensureUploadedAvatar(path) {
    if (/^https:\/\//.test(path)) return Promise.resolve(path);
    return api.uploadRegisterAvatar(path, this._registerTicket).then((res) => {
      const url = (res && res.data && res.data.avatarUrl) || '';
      if (!url) throw new Error('头像上传失败，请重试');
      return url;
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

  onBannerTap() {
    const linkUrl = this.data.banner && this.data.banner.linkUrl;
    if (!linkUrl || !linkUrl.startsWith('/pages/activity-detail/activity-detail?id=')) return;
    wx.navigateTo({ url: linkUrl });
  },

  inviteShareData() {
    const me = this.data.me || {};
    const inviteCode = auth.isLoggedIn() ? me.inviteCode || '' : '';
    return {
      inviteCode,
      title: inviteCode
        ? (me.nickname || me.nickName || '会员') + '邀请你加入inwardClub会员'
        : '邀请你加入inwardClub会员',
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
