const auth = require('./utils/auth');
const invitation = require('./utils/invitation');
const silentLogin = require('./utils/silent-login');
const { clearProfile } = require('./utils/member-profile');

App({
  globalData: {
    systemInfo: null,
  },

  onLaunch(options) {
    this.syncPendingInvitation(options);
    try {
      this.globalData.systemInfo = wx.getWindowInfo();
    } catch {}
    // Establish an OpenID-only identity without interrupting the entry page.
    // Profile completion remains an explicit action from 首页/我的.
    silentLogin
      .ensure()
      .then(() => this.syncPendingInvitation(options))
      .catch(() => {});
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
   * Called by the request layer (utils/request) when a 401 can't be recovered
   * (refresh failed / no refresh token). Silently drops the dead session and
   * lets visible pages immediately fall back to their logged-out state.
   */
  onSessionExpired() {
    auth.clear();
    clearProfile();
    // Immediately drop logged-in UI on any page currently in the stack.
    getCurrentPages().forEach((p) => {
      if (p && typeof p.onSessionExpired === 'function') {
        try {
          p.onSessionExpired();
        } catch {}
      }
    });
  },
});
