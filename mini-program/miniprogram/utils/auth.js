/**
 * Auth/session storage. Holds access + refresh tokens and the decoded identity
 * (member vs staff). The request client reads the access token from here and
 * calls refresh() on 401. No token logic lives in pages.
 */
const STORAGE_KEY = 'ic_session_v2';

let session = null; // { accessToken, refreshToken, subjectType, storeId, expiresAt }

function load() {
  if (session) return session;
  try {
    session = wx.getStorageSync(STORAGE_KEY) || null;
  } catch (e) {
    session = null;
  }
  return session;
}

function save(next) {
  session = next || null;
  try {
    if (session) wx.setStorageSync(STORAGE_KEY, session);
    else wx.removeStorageSync(STORAGE_KEY);
  } catch (e) {}
}

function getAccessToken() {
  const s = load();
  return s && s.accessToken;
}

function getRefreshToken() {
  const s = load();
  return s && s.refreshToken;
}

function setTokens(tokens) {
  const s = load() || {};
  save(Object.assign({}, s, tokens));
}

function isStaff() {
  const s = load();
  return !!(s && s.subjectType === 'staff');
}

function getStoreId() {
  const s = load();
  return s && s.storeId;
}

function clear() {
  save(null);
}

function isLoggedIn() {
	const s = load();
	return !!(s && s.accessToken && s.subjectType !== 'pre_member');
}

function isPreRegistered() {
	const s = load();
	return !!(s && s.accessToken && s.subjectType === 'pre_member');
}

function hasReservationIdentity() {
	return !!getAccessToken();
}

module.exports = {
  load,
  save,
  getAccessToken,
  getRefreshToken,
  setTokens,
  isStaff,
  getStoreId,
	isLoggedIn,
	isPreRegistered,
	hasReservationIdentity,
	clear,
};
