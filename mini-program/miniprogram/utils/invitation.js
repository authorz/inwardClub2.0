const STORAGE_KEY = 'ic_pending_inviter_code';

function normalize(code) {
  return String(code || '').trim();
}

function get() {
  return normalize(wx.getStorageSync(STORAGE_KEY));
}

// A newly opened share link always wins. Empty/non-invitation entries leave the
// current pending code untouched so it survives until registration succeeds.
function capture(query) {
  const code = normalize(query && query.invite);
  if (!code) return get();
  wx.setStorageSync(STORAGE_KEY, code);
  return code;
}

function clear() {
  wx.removeStorageSync(STORAGE_KEY);
}

module.exports = { capture, get, clear };
