const auth = require('./utils/auth');
const api = require('./services/api');
const ui = require('./utils/ui');
const invitation = require('./utils/invitation');

App({
  globalData: {
    systemInfo: null,
  },

  onLaunch(options) {
    this.syncPendingInvitation(options);
    try {
      this.globalData.systemInfo = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
    } catch (e) {}
    // No silent/auto login here on purpose: WeChat requires wx.getUserProfile to
    // be triggered by an explicit user tap, so login is started from a page
    // (see loginWithWechat) — never on launch.
  },

  onShow(options) {
    this.syncPendingInvitation(options);
  },

  syncPendingInvitation(options) {
    if (auth.isLoggedIn()) {
      invitation.clear();
      return;
    }
    invitation.capture((options && options.query) || {});
  },

  /**
   * Interactive WeChat login. MUST be called synchronously from a user tap so
   * the wx.getUserProfile gesture check passes.
   * Flow: wx.getUserProfile({ desc }) -> wx.login() -> api.wechatLogin({ code,
   * profile }) -> persist session. Resolves the session payload on success and
   * rejects on user refusal / network error so the caller can toast.
   * Single-flighted so a double tap can't open two auth popups.
   */
  loginWithWechat() {
    if (auth.isLoggedIn()) return Promise.resolve(auth.load());
    if (this._loginFlow) return this._loginFlow;

    this._loginFlow = new Promise((resolve, reject) => {
      wx.getUserProfile({
        desc: '用于完善会员资料',
        success: (profileRes) => {
          wx.login({
            success: (loginRes) => {
              api
                .wechatLogin({
                  code: loginRes.code,
                  userInfo: profileRes.userInfo,
                  rawData: profileRes.rawData,
                  signature: profileRes.signature,
                  encryptedData: profileRes.encryptedData,
                  iv: profileRes.iv,
                })
                .then((r) => {
                  const d = r.data || {};
                  // Mini login returns the token plus its member/staff identity.
                  const tk = d.token || {};
                  auth.save({
                    accessToken: tk.accessToken,
                    refreshToken: tk.refreshToken,
                    subjectType: d.subjectType,
                    storeId: d.storeId,
                  });
                  resolve(r.data);
                })
                .catch(reject);
            },
            fail: reject,
          });
        },
        // user tapped 取消 in the authorization popup, or gesture check failed
        fail: reject,
      });
    }).finally(() => {
      this._loginFlow = null;
    });

    return this._loginFlow;
  },

  /**
   * Login with a profile the page has already collected via WeChat's current
   * open-ability set (chooseAvatar / type=nickname / getPhoneNumber) plus a
   * manually picked gender. No wx.getUserProfile here, so it does NOT need to
   * run inside the tap gesture.
   * payload: { userInfo: { nickName, avatarUrl, gender }, phoneCode,
   *            phoneEncryptedData, phoneIv }
   * Flow: wx.login() -> api.wechatLogin({ code, ...payload }) -> persist session.
   * Shares the single-flight lock with loginWithWechat.
   */
  loginWithProfile(payload) {
    if (auth.isLoggedIn()) return Promise.resolve(auth.load());
    if (this._loginFlow) return this._loginFlow;
    const p = payload || {};

    this._loginFlow = new Promise((resolve, reject) => {
      wx.login({
        success: (loginRes) => {
          api
            .wechatLogin({
              code: loginRes.code,
              userInfo: p.userInfo || {},
              phoneCode: p.phoneCode,
              phoneEncryptedData: p.phoneEncryptedData,
              phoneIv: p.phoneIv,
            })
            .then((r) => {
              const d = r.data || {};
              const tk = d.token || d; // nested { token } shape, tolerant of flat
              auth.save({
                accessToken: tk.accessToken,
                refreshToken: tk.refreshToken,
                subjectType: d.subjectType,
                storeId: d.storeId,
              });
              resolve(r.data);
            })
            .catch(reject);
        },
        fail: reject,
      });
    }).finally(() => {
      this._loginFlow = null;
    });

    return this._loginFlow;
  },

  /**
   * Called by the request layer (utils/request) when a 401 can't be recovered
   * (refresh failed / no refresh token). Drops the dead session, lets any visible
   * page clear its logged-in UI at once, and prompts the user to re-login.
   * Single-flighted so a burst of parallel 401s only prompts once.
   */
  onSessionExpired() {
    if (this._sessionExpiredFlow) return;
    this._sessionExpiredFlow = true;
    auth.clear();
    // Immediately drop logged-in UI on any page currently in the stack.
    getCurrentPages().forEach((p) => {
      if (p && typeof p.onSessionExpired === 'function') {
        try {
          p.onSessionExpired();
        } catch (e) {}
      }
    });
    ui.confirm({
      title: '登录已过期',
      content: '登录状态已失效，请重新登录',
      confirmText: '去登录',
      showCancel: false,
    })
      .then(() => wx.switchTab({ url: '/pages/index/index' }))
      .finally(() => {
        this._sessionExpiredFlow = false;
      });
  },
});
