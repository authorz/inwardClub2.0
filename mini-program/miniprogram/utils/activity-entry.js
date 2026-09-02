const SHARED_ACTIVITY_VALUE = '1';

function allowsGuestPurchase(options) {
  return String((options && options.shared) || '') === SHARED_ACTIVITY_VALUE;
}

function sharePath(activityId) {
  return `/pages/activity-detail/activity-detail?id=${encodeURIComponent(activityId || '')}&shared=${SHARED_ACTIVITY_VALUE}`;
}

function shareQuery(activityId) {
  return `id=${encodeURIComponent(activityId || '')}&shared=${SHARED_ACTIVITY_VALUE}`;
}

function purchasePath(activityId, guestPurchaseAllowed) {
  const shared = guestPurchaseAllowed ? `&shared=${SHARED_ACTIVITY_VALUE}` : '';
  return `/pages/activity-purchase/activity-purchase?id=${encodeURIComponent(activityId || '')}${shared}`;
}

module.exports = { allowsGuestPurchase, sharePath, shareQuery, purchasePath };
