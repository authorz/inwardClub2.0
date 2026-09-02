const assert = require('node:assert/strict');
const test = require('node:test');

const activityEntry = require('../miniprogram/utils/activity-entry');

test('only an explicit share marker allows guest purchase', () => {
  assert.equal(activityEntry.allowsGuestPurchase({ shared: '1' }), true);
  assert.equal(activityEntry.allowsGuestPurchase({ shared: '0' }), false);
  assert.equal(activityEntry.allowsGuestPurchase({}), false);
  assert.equal(activityEntry.allowsGuestPurchase(), false);
});

test('share routes preserve the guest purchase marker', () => {
  assert.equal(
    activityEntry.sharePath('activity/1'),
    '/pages/activity-detail/activity-detail?id=activity%2F1&shared=1'
  );
  assert.equal(activityEntry.shareQuery('activity/1'), 'id=activity%2F1&shared=1');
  assert.equal(
    activityEntry.purchasePath('activity/1', true),
    '/pages/activity-purchase/activity-purchase?id=activity%2F1&shared=1'
  );
  assert.equal(
    activityEntry.purchasePath('activity/1', false),
    '/pages/activity-purchase/activity-purchase?id=activity%2F1'
  );
});
