const auth = require('./utils/auth');
const invitation = require('./utils/invitation');
const { clearProfile } = require('./utils/member-profile');

App({
  globalData: {
    systemInfo: null,
  },

  onLaunch(options) {
    this.syncPendingInvitation(options);
    try {
      this.globalData.systemInfo = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
    } catch {}
    // Login starts from the dedicated login page after an explicit user action,
    // never automatically during app launch.
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
