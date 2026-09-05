const assert = require('node:assert/strict');
const test = require('node:test');

const memberAccess = require('../miniprogram/utils/member-access');

let pageDefinition = null;
let componentDefinition = null;
let switchedTo = '';
let loginRequests = 0;

global.Page = (definition) => {
  pageDefinition = definition;
};
global.Component = (definition) => {
  componentDefinition = definition;
};
global.wx = {
  getStorageSync() { return null; },
  removeStorageSync() {},
  switchTab({ url }) { switchedTo = url; },
};

memberAccess.requireCompleteProfile = () => {
  loginRequests += 1;
};

function freshRequire(path) {
  const resolved = require.resolve(path);
  delete require.cache[resolved];
  require(resolved);
}

test.beforeEach(() => {
  pageDefinition = null;
  componentDefinition = null;
  switchedTo = '';
  loginRequests = 0;
});

test('guest can open the ordering tab without a login prompt', () => {
  freshRequire('../miniprogram/custom-tab-bar/index');
  componentDefinition.methods.onTap.call(
    { data: componentDefinition.data },
    { currentTarget: { dataset: { index: 2 } } }
  );

  assert.equal(switchedTo, '/pages/order/order');
  assert.equal(loginRequests, 0);
});

test('guest can add menu items but checkout still requests login', () => {
  freshRequire('../miniprogram/pages/order/order');
  const item = { id: 7, qty: 0, priceCent: 3900 };
  const page = Object.assign({}, pageDefinition, {
    data: Object.assign({}, pageDefinition.data, {
      groups: [{ id: 1, items: [item] }],
      items: [item],
    }),
    qty: {},
    setData(next) { Object.assign(this.data, next); },
  });

  page.onQtyChange({ currentTarget: { dataset: { id: 7 } }, detail: { value: 1 } });
  assert.equal(page.data.cartCount, 1);
  assert.equal(loginRequests, 0);

  page.onCheckout();
  assert.equal(loginRequests, 1);
});
