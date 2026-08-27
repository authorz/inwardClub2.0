// 我的 — 会员资产 / 订单入口 / 会员功能 · 工作人员功能 grid / 资产操作弹层
// Reference: design/mini-program/final/member-center/08-member-center-final-profile.png
//            design/mini-program/final/member-asset-sheets/*
const api = require('../../services/api');
const auth = require('../../utils/auth');
const ui = require('../../utils/ui');
const http = require('../../utils/request');
const pay = require('../../utils/pay');
const storeCtx = require('../../utils/store-context');
const silentLogin = require('../../utils/silent-login');
const { amount } = require('../../utils/format');
const { mergeCachedProfile } = require('../../utils/member-profile');
const { POINT_SAVING } = require('../../constants/index');
const validation = require('../../utils/validation');

const MINE_MENU_ASSET_BASE = 'https://assets.inwardclub.com/public/mine-menu/';

// 会员等级短标：优先按 tierCode 映射 VIP1-8，其次从 tierName 取 VIPn，兜底 VIP1
const TIER_SHORT = { normal: 'VIP1', bronze: 'VIP2', silver: 'VIP3', gold: 'VIP4', platinum: 'VIP5', diamond: 'VIP6', star: 'VIP7', master: 'VIP8' };
function tierShortOf(me) {
  if (me && me.tierCode && TIER_SHORT[me.tierCode]) return TIER_SHORT[me.tierCode];
  const m = me && me.tierName && me.tierName.match(/VIP\s*(\d+)/i);
  if (m) return 'VIP' + m[1];
  return 'VIP1';
}

Page({
  data: {
    me: {},
    wallet: { coins: 0, points: 0, couponCount: 0 },
    coinsText: '0',
    pointsText: '0',
    genderIcon: '',
    isStaff: false,
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
    orderEntry: { label: '订单中心', icon: '/assets/mine-menu/order-center.svg' },
    // 菜单图标优先使用七牛公共资源，避免本地包资源被误删或增大主包体积。
    memberMenuEntries: [
      { label: '充值', icon: MINE_MENU_ASSET_BASE + 'recharge-coins.png', action: 'recharge' },
      { label: '存积分', icon: '/assets/mine-menu/save-points.png', action: 'point', dir: POINT_SAVING.DEPOSIT },
      // { label: '取积分', icon: MINE_MENU_ASSET_BASE + 'withdraw-points.png', action: 'point', dir: POINT_SAVING.WITHDRAW },
      { label: '排行榜', icon: MINE_MENU_ASSET_BASE + 'rankings.png', action: 'nav', url: '/pages/rankings/rankings' },
      { label: '邀请有礼', icon: MINE_MENU_ASSET_BASE + 'invite-gift.png', action: 'nav', url: '/pages/invitations/invitations' },
      { label: '交易记录', icon: MINE_MENU_ASSET_BASE + 'transactions.png', action: 'nav', url: '/pages/wallet-ledger/wallet-ledger' },
      // { label: '会员权益', icon: MINE_MENU_ASSET_BASE + 'member-benefits.png', action: 'nav', url: '/pages/benefits/benefits' },
      { label: '咨询客服', icon: '/assets/mine-menu/customer-service.png', action: 'nav', url: '/pages/customer-service/customer-service' },
      // { label: '加入社群', icon: MINE_MENU_ASSET_BASE + 'community.png', action: 'toast' },
    ],
    // 工作人员功能入口 — 仅员工选中自己的绑定门店时显示
    staffMenuEntries: [
      { label: '门票核销', icon: MINE_MENU_ASSET_BASE + 'staff-verify.png', url: '/pages/staff-verify/staff-verify' },
      { label: '积分核销', icon: MINE_MENU_ASSET_BASE + 'staff-point-review.png', url: '/pages/staff-point-review/staff-point-review' },
      { label: '今日明细', icon: MINE_MENU_ASSET_BASE + 'transactions.png', url: '/pages/staff-today/staff-today' },
      { label: '核销记录', icon: MINE_MENU_ASSET_BASE + 'staff-records.png', url: '/pages/staff-verifications/staff-verifications' },
    ],

    // recharge sheet
    showRecharge: false,
    products: [],
    productId: '',
    rechargeNotice: '',
    customAmount: '', // 自定义充值金额（元，正整数）
    rechargeTotal: '0',

    // point-saving sheet
    showPoint: false,
    pointDir: POINT_SAVING.DEPOSIT,
    pointSheetTitle: '存积分',
    pointSubmitText: '提交存积分申请',
    pointAmount: '',
    pointStoreName: '',
    submitting: false,
  },

  onLoad() {
    this.measureNav();
  },

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
      this.getTabBar().setData({ selected: 3 });
    }
    this.loadProfile();
  },

  loadProfile() {
    if (!auth.isLoggedIn()) {
      this.onSessionExpired();
      return;
    }
    Promise.all([
      api.getMe().catch(() => ({ data: {} })),
      api.getWallet().catch(() => ({ data: {} })),
      // wallet has no couponCount field on the backend — derive it from the
      // real coupons list instead.
      api.getCoupons().catch(() => ({ data: [] })),
      storeCtx.ensureStore().catch(() => null),
    ]).then(([meRes, walletRes, couponsRes, currentStore]) => {
      // Fall back to the cached avatar/nickname/gender when getMe returns them
      // empty, so a refresh still shows the WeChat avatar (not a black placeholder).
      const me = mergeCachedProfile(meRes.data || {});
      // Accept both nickname (our API) and nickName (WeChat) shapes.
      me.nickname = me.nickname || me.nickName || '';
      me.tierShort = tierShortOf(me);
      const couponCount = (couponsRes.data || []).filter((c) => c.status === 'unused').length;
      const wallet = Object.assign({ coins: 0, points: 0 }, walletRes.data, { couponCount });
      const staffStoreId = auth.getStoreId();
      const isStaffAtCurrentStore =
        auth.isStaff() &&
        currentStore &&
        String(currentStore.id) === String(staffStoreId);
      this.setData({
        me,
        wallet,
        coinsText: amount(wallet.coins),
        pointsText: amount(wallet.points),
        genderIcon: this.genderIconOf(me.gender),
        isStaff: !!isStaffAtCurrentStore,
      });
    });
  },

  // Gender: 1/'male'=male, 2/'female'=female, 其它/未登录 不显示性别贴标
  genderIconOf(gender) {
    if (gender === 1 || gender === 'male') return '/assets/icons/gender-male.svg';
    if (gender === 2 || gender === 'female') return '/assets/icons/gender-female.svg';
    return '';
  },

  // app.onSessionExpired broadcasts here when a 401 invalidated the session: drop
  // member data at once instead of leaving stale coins/tier on screen.
  onSessionExpired() {
    this.setData({
      me: {},
      wallet: { coins: 0, points: 0, couponCount: 0 },
      coinsText: '0',
      pointsText: '0',
      genderIcon: '',
      isStaff: false,
    });
    if (this.identityRecovery) return;
    this.identityRecovery = silentLogin
      .ensure()
      .then(() => {
        if (auth.isLoggedIn()) this.loadProfile();
      })
      .catch(() => {})
      .finally(() => {
        this.identityRecovery = null;
      });
  },

  // Gate member-only actions through the dedicated login/registration page.
  requireLogin(next) {
    if (auth.isLoggedIn()) return next();
    wx.navigateTo({
      url: '/pages/login/login',
      success: (res) => {
        if (!res.eventChannel) return;
        res.eventChannel.on('loginSuccess', () => {
          this.loadProfile();
          next();
        });
      },
    });
  },

  /* ---------- navigation ---------- */
  goProfile() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/profile/profile' }));
  },
  goBenefits() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/benefits/benefits' }));
  },
  copyInvite() {
    if (this.data.me.inviteCode) ui.copy(this.data.me.inviteCode, '邀请码已复制');
  },
  goOrder() {
    this.requireLogin(() => wx.navigateTo({ url: '/pages/order-center/order-center' }));
  },

  /* ---------- asset taps ---------- */
  onAsset(e) {
    const asset = e.currentTarget.dataset.asset;
    this.requireLogin(() => {
      // 金币查看流水；积分直接发起取积分；优惠券进入券列表。
      if (asset === 'coins') wx.navigateTo({ url: '/pages/wallet-ledger/wallet-ledger?asset=coins' });
      else if (asset === 'points') this.openPoint(POINT_SAVING.WITHDRAW);
      else wx.navigateTo({ url: '/pages/coupons/coupons' });
    });
  },

  /* ---------- 会员功能 grid ---------- */
  goMember(e) {
    const item = this.data.memberMenuEntries[e.currentTarget.dataset.index];
    if (!item) return;
    this.requireLogin(() => {
      if (item.action === 'point') this.openPoint(item.dir);
      else if (item.action === 'recharge') this.openRecharge();
      else if (item.action === 'toast') ui.toast('社群功能即将上线');
      else wx.navigateTo({ url: item.url });
    });
  },

  /* ---------- 工作人员功能入口 ---------- */
  goStaffMenu(e) {
    if (!this.data.isStaff) return;
    const item = this.data.staffMenuEntries[e.currentTarget.dataset.index];
    if (item) wx.navigateTo({ url: item.url });
  },

  /* ---------- coin recharge sheet (wechat only) ---------- */
  openRecharge() {
    this.setData({ showRecharge: true });
    if (!this.data.products.length) {
      api.getRechargeProducts().then((res) => {
        const products = (res.data || []).map((p) => ({
          id: p.id,
          priceYuan: String(Math.round(p.priceCent / 100)),
          amountCent: p.priceCent,
          totalCoins: p.coins || 0,
          pointsAmount: p.points || 0,
        }));
        const first = products[0];
        this.setData({
          products,
          productId: first ? first.id : '',
          rechargeNotice: (res.meta && res.meta.rechargeNotice) || '',
          rechargeTotal: first ? first.priceYuan : '0',
        });
      });
    }
  },
  closeRecharge() {
    this.setData({ showRecharge: false });
  },
  pickProduct(e) {
    const id = e.currentTarget.dataset.id;
    if (id === 'custom') {
      this.setData({ productId: 'custom', rechargeTotal: this.data.customAmount || '0' });
      return;
    }
    const p = this.data.products.find((x) => x.id === id);
    this.setData({ productId: id, rechargeTotal: p ? p.priceYuan : '0' });
  },
  // 自定义金额输入：只保留正整数，并自动选中 custom 档位
  onCustomAmount(e) {
    const val = (e.detail.value || '').replace(/\D/g, '').replace(/^0+/, '');
    this.setData({ customAmount: val, productId: 'custom', rechargeTotal: val || '0' });
  },
  confirmRecharge() {
    if (this.data.submitting) return;
    let payload;
    if (this.data.productId === 'custom') {
	  let yuan;
	  try {
	    yuan = validation.integer(this.data.customAmount, { label: '充值金额', min: 1, max: 1000000 });
	  } catch (err) {
	    return ui.toast(err.message);
	  }
      payload = { productId: '', amountCent: yuan * 100, payChannel: 'wechat' };
    } else {
      const p = this.data.products.find((x) => x.id === this.data.productId);
      if (!p) return;
      payload = { productId: p.id, amountCent: p.amountCent, payChannel: 'wechat' };
    }
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    api
      .createRechargeOrder(payload, http.uuid())
      .then((res) => api.payWechatJsapi((res.data && res.data.paymentOrderId) || 'po_recharge', http.uuid()))
      .then((r) => pay.settle(r))
      .then(() => {
        ui.hideLoading();
        this.setData({ submitting: false, showRecharge: false });
        ui.success('充值已提交');
        this.loadProfile();
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '充值失败');
      });
  },

  /* ---------- point saving / immediate withdrawal sheet ---------- */
  openPoint(dir) {
    const isWithdraw = dir === POINT_SAVING.WITHDRAW;
    this.setData({
      showPoint: true,
      pointAmount: '',
      pointDir: isWithdraw ? POINT_SAVING.WITHDRAW : POINT_SAVING.DEPOSIT,
      pointSheetTitle: isWithdraw ? '取积分' : '存积分',
      pointSubmitText: isWithdraw ? '确认取积分' : '提交存积分申请',
      pointStoreName: (storeCtx.get() || {}).name || '当前门店',
    });
    // Ensure a store is resolved (nearest by default) so confirmPoint has a storeId
    // even when 我的 is the first page opened.
    storeCtx.ensureStore().then((store) => {
      if (store) this.setData({ pointStoreName: store.name });
    });
  },
  closePoint() {
    this.setData({ showPoint: false });
  },
  onPointAmount(e) {
    this.setData({ pointAmount: e.detail.value });
  },
  goLedger() {
    this.setData({ showPoint: false });
    wx.navigateTo({ url: '/pages/wallet-ledger/wallet-ledger?asset=points' });
  },
  confirmPoint() {
    let points;
    try {
      points = validation.integer(this.data.pointAmount, { label: '积分数量', min: 1, max: 1000000000 });
    } catch (err) {
      ui.toast(err.message);
      return;
    }
    if (this.data.submitting) return;
    const store = storeCtx.get();
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    api
      .createPointSaving(
        { storeId: store && store.id, direction: this.data.pointDir, points, note: '' },
        http.uuid()
      )
      .then((res) => {
        ui.hideLoading();
        const isWithdraw = this.data.pointDir === POINT_SAVING.WITHDRAW;
        const nextData = { submitting: false, showPoint: false };
        if (isWithdraw && res && res.data) {
          nextData['wallet.points'] = res.data.balanceAfter;
          nextData.pointsText = amount(res.data.balanceAfter);
        }
        this.setData(nextData);
        ui.success(isWithdraw ? '提取积分成功，请前去前台领取' : '已提交，待工作人员审核');
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '提交失败');
      });
  },
});
