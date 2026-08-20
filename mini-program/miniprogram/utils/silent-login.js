/**
 * OpenID-only silent identity bootstrap. It creates/reuses the member row from
 * wx.login without asking for profile data, and shares one in-flight request
 * across app launch, reservations and activity purchases.
 */
const api = require('../services/api');
const auth = require('./auth');
const { saveProfile } = require('./member-profile');

let pending = null;

function wechatCode() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: (res) => {
        if (res && res.code) resolve(res.code);
        else reject(new Error('微信身份获取失败，请重试'));
      },
      fail: () => reject(new Error('微信身份获取失败，请重试')),
    });
  });
}

function requestIdentity() {
  return wechatCode()
    .then((code) => api.preRegister({ code }))
    .then((res) => {
      const result = (res && res.data) || {};
      const token = result.token || {};
      if (!token.accessToken || !token.refreshToken) {
        throw new Error('微信身份获取失败，请重试');
      }
      const subjectType = result.subjectType || 'pre_member';
      auth.save({
        accessToken: token.accessToken,
        refreshToken: token.refreshToken,
        subjectType,
        storeId: result.storeId,
      });
      if (subjectType !== 'pre_member') {
        const profile = result.profile || {};
        saveProfile({
          avatarUrl: profile.avatarUrl || '',
          nickname: profile.nickname || '',
          gender: profile.gender || '',
        });
      }
      return result;
    });
}

function ensure() {
  if (auth.getAccessToken()) return Promise.resolve(auth.load());
  if (pending) return pending;
  pending = requestIdentity().finally(() => {
    pending = null;
  });
  return pending;
}

module.exports = { ensure };
