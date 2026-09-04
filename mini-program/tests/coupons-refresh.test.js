const assert = require('node:assert/strict');
const test = require('node:test');

const api = require('../miniprogram/services/api');

let pageDefinition = null;
global.Page = (definition) => {
  pageDefinition = definition;
};
global.wx = {
  navigateTo() {},
  redirectTo() {},
};

function createCouponsPage() {
  const pagePath = require.resolve('../miniprogram/pages/coupons/coupons');
  delete require.cache[pagePath];
  pageDefinition = null;
  require(pagePath);

  const page = Object.assign({}, pageDefinition, {
    data: JSON.parse(JSON.stringify(pageDefinition.data)),
    setData(next, callback) {
      Object.assign(this.data, next);
      if (callback) callback();
    },
  });
  page.startCountdown = () => {};
  return page;
}

test('coupon list refreshes whenever the page becomes visible again', async () => {
  let requests = 0;
  api.getCouponCategories = () => Promise.resolve({ data: [{ id: 3, name: '酒水券' }] });
  api.getCoupons = () => {
    requests += 1;
    return Promise.resolve({
      data: requests === 1 ? [] : [{
        id: 42,
        templateId: 7,
        storeId: 1,
        name: '畅饮券',
        type: 'alcohol',
        categoryId: 3,
        categoryName: '酒水券',
        status: 'unused',
        validUntil: '2099-01-01 00:00:00',
      }],
    });
  };

  const page = createCouponsPage();
  await page.onShow();
  assert.equal(page.data.couponCount, 0);

  await page.onShow();
  assert.equal(requests, 2);
  assert.equal(page.data.couponCount, 1);
  assert.equal(page.data.list[0].id, 42);
});
