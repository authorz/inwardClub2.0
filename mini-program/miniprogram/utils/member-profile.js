// Local cache for the profile the user picks in the login sheet. The token is
// persisted by auth, but the chosen avatar/nickname/gender are not — and
// getMe may return an empty avatarUrl — so we keep a local copy to
// restore them after a refresh / re-entry. Shared by home 与 我的 页.
const PROFILE_KEY = 'ic_member_profile_v1';

function loadProfile() {
  try {
    return wx.getStorageSync(PROFILE_KEY) || null;
  } catch {
    return null;
  }
}

function saveProfile(profile) {
  try {
    wx.setStorageSync(PROFILE_KEY, profile);
  } catch {}
}

// chooseAvatar returns a temporary path (wxfile://tmp_… / http://tmp/…) that is
// wiped on refresh/restart, so a cached avatarUrl pointing at it fails to load.
// Detect those so we can copy them into a persistent local file before caching.
function isTempAvatar(url) {
  if (!url || typeof url !== 'string') return false;
  if (/^https?:\/\//i.test(url)) return false;
  // Already-persistent files live under the user store; temp files don't.
  if (url.indexOf('/store/') !== -1) return false;
  return true;
}

// Like saveProfile, but if avatarUrl is a temporary local path it is copied to a
// persistent local file first so the cached avatar survives a refresh/restart.
// Never rejects: on any failure it falls back to saving the original profile.
function saveProfilePersistently(profile) {
  if (!profile || !isTempAvatar(profile.avatarUrl)) {
    saveProfile(profile);
    return Promise.resolve(profile);
  }
  return new Promise((resolve) => {
    wx.getFileSystemManager().saveFile({
      tempFilePath: profile.avatarUrl,
      success: (res) => {
        const persisted = Object.assign({}, profile, { avatarUrl: res.savedFilePath });
        saveProfile(persisted);
        resolve(persisted);
      },
      fail: () => {
        saveProfile(profile);
        resolve(profile);
      },
    });
  });
}

// Fill only the fields the server left empty from the cached profile, so a real
// backend avatar/nickname/gender always wins over the local copy.
function mergeCachedProfile(me) {
  const cached = loadProfile();
  if (!cached) return me;
  const merged = Object.assign({}, me);
  if (!merged.avatarUrl && cached.avatarUrl) merged.avatarUrl = cached.avatarUrl;
  if (!merged.nickname && cached.nickname) merged.nickname = cached.nickname;
  if (!merged.gender && cached.gender) merged.gender = cached.gender;
  return merged;
}

module.exports = { PROFILE_KEY, loadProfile, saveProfile, saveProfilePersistently, mergeCachedProfile };
