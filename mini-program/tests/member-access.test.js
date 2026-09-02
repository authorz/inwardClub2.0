const assert = require('node:assert/strict');
const test = require('node:test');

const storage = new Map();
let navigateOptions = null;

global.wx = {
  getStorageSync: (key) => storage.get(key) || null,
  setStorageSync: (key, value) => storage.set(key, value),
  removeStorageSync: (key) => storage.delete(key),
  navigateTo: (options) => {
    navigateOptions = options;
  },
};

const auth = require('../miniprogram/utils/auth');
const api = require('../miniprogram/services/api');
const memberAccess = require('../miniprogram/utils/member-access');

test.beforeEach(() => {
  storage.clear();
  auth.clear();
  navigateOptions = null;
  api.getMe = () => Promise.resolve({ data: {} });
});

test('completed members continue after the profile endpoint confirms required fields', async () => {
  let continued = false;
  let profileRequests = 0;
  auth.save({ accessToken: 'member-token-complete', subjectType: 'member' });
  api.getMe = () => {
    profileRequests += 1;
    return Promise.resolve({
      data: { avatarUrl: 'https://cdn.test/avatar.jpg', nickname: '会员', gender: 'male', phone: '13800000000' },
    });
  };

  assert.equal(await memberAccess.requireCompleteProfile(() => { continued = true; }), true);
  assert.equal(profileRequests, 1);
  assert.equal(continued, true);
  assert.equal(navigateOptions, null);
});

test('members missing required profile fields are sent to login', async () => {
  let continued = false;
  auth.save({ accessToken: 'member-token-incomplete', subjectType: 'member' });
  api.getMe = () => Promise.resolve({
    data: { avatarUrl: 'https://cdn.test/avatar.jpg', nickname: '会员', gender: 'male', phone: '' },
  });

  assert.equal(await memberAccess.requireCompleteProfile(() => { continued = true; }), false);
  assert.equal(continued, false);
  assert.equal(navigateOptions.url, '/pages/login/login');
});

test('pre-members must finish login before continuing', async () => {
  let continued = false;
  let loginSuccess = null;
  auth.save({ accessToken: 'pre-member-token', subjectType: 'pre_member' });

  assert.equal(await memberAccess.requireCompleteProfile(() => { continued = true; }), false);
  assert.equal(navigateOptions.url, '/pages/login/login');
  navigateOptions.success({
    eventChannel: {
      on: (eventName, handler) => {
        assert.equal(eventName, 'loginSuccess');
        loginSuccess = handler;
      },
    },
  });
  assert.equal(continued, false);

  loginSuccess();
  assert.equal(continued, true);
});
