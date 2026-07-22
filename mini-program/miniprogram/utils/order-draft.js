/**
 * Order draft — a tiny in-memory hand-off used to carry a checkout intent
 * (food cart / activity ticket / coin recharge) from the source page to the
 * order-confirm / pay-result pages without stuffing objects into the URL.
 * Lives for the app session; cleared once the flow finishes.
 */
let draft = null;

function set(next) {
  draft = next || null;
}

function get() {
  return draft;
}

function clear() {
  draft = null;
}

module.exports = { set, get, clear };
