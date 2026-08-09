/**
 * Order draft — a tiny in-memory hand-off used to carry a checkout intent
 * (food cart / activity ticket / coin recharge) from the source page to the
 * order-confirm / pay-result pages without stuffing objects into the URL.
 * Lives for the app session; cleared once the flow finishes.
 */
let draft = null;
let completion = null;

function set(next) {
  draft = next || null;
}

function get() {
  return draft;
}

function clear() {
  draft = null;
}

// Mark a completed flow while clearing its checkout draft. Source pages consume
// this one-shot signal onShow so tab-page instances can reset stale local state.
function complete(type, payload) {
  draft = null;
  completion = Object.assign({ type }, payload || {});
}

function consumeCompletion(type) {
  if (!completion || completion.type !== type) return null;
  const value = completion;
  completion = null;
  return value;
}

module.exports = { set, get, clear, complete, consumeCompletion };
