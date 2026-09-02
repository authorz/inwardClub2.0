const auth = require('./auth');
const api = require('../services/api');

let verifiedToken = '';

function hasRequiredProfile(profile) {
  const member = profile || {};
  return Boolean(member.avatarUrl && member.nickname && member.gender && member.phone);
}

function requireCompleteProfile(next) {
  const accessToken = auth.getAccessToken();
  if (!auth.isLoggedIn() || !accessToken) {
    openLogin(next);
    return Promise.resolve(false);
  }
  if (verifiedToken === accessToken) {
    if (typeof next === 'function') next();
    return Promise.resolve(true);
  }
  return api.getMe()
    .then((res) => {
      if (!hasRequiredProfile(res.data)) {
        openLogin(next);
        return false;
      }
      verifiedToken = accessToken;
      if (typeof next === 'function') next();
      return true;
    })
    .catch(() => {
      openLogin(next);
      return false;
    });
}

function openLogin(next) {
  wx.navigateTo({
    url: '/pages/login/login',
    success: (res) => {
      if (typeof next !== 'function' || !res.eventChannel) return;
      res.eventChannel.on('loginSuccess', next);
    },
  });
}

module.exports = { hasRequiredProfile, requireCompleteProfile };
