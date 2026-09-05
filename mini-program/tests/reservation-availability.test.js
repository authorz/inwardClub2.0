const assert = require('node:assert/strict');
const test = require('node:test');

let pageDefinition = null;

global.Page = (definition) => {
  pageDefinition = definition;
};
global.wx = {
  getStorageSync() { return null; },
  removeStorageSync() {},
};

const pagePath = require.resolve('../miniprogram/pages/reservation/reservation');
delete require.cache[pagePath];
require(pagePath);

function createPage() {
  return Object.assign({}, pageDefinition, {
    data: JSON.parse(JSON.stringify(pageDefinition.data)),
    setData(next) { Object.assign(this.data, next); },
  });
}

test('reservation buttons are disabled outside the configured window', () => {
  const page = createPage();
  const table = page.decorate(
    { id: 1, name: '一号桌', capacity: 1 },
    [{ id: 7, name: '1', status: 'available' }],
    new Map(),
    new Map(),
    false,
    false
  );

  assert.equal(table.canReserve, true);
  assert.equal(table.actionDisabled, true);
  assert.equal(
    page.normalizeReservationAvailability({
      reservable: false,
      reservationCutoff: '20:00',
      unavailableReason: '今日预约已截止',
    }).availabilityHint,
    '今日预约已于 20:00 截止'
  );
});

test('cutoff keeps cancellation available for an existing booking', () => {
  const page = createPage();
  page.data.tables = [{
    id: 1,
    isMineTable: true,
    mineReservationStatus: 'booked',
    actionDisabled: false,
  }];

  page.closeReservationWindow({ reservationCutoff: '20:00' });

  assert.equal(page.data.reservationAvailability.reservable, false);
  assert.equal(page.data.tables[0].actionDisabled, false);
});
