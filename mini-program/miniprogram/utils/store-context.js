/**
 * Current store selection shared across tabs (home / reservation / ordering).
 *
 * Almost every member data read/write is store-scoped, so pages call
 * `ensureStore()` before fetching. Semantics:
 *   - A store the member explicitly picked (switchStore) is PERSISTED and always
 *     wins — it survives app restarts until they switch again.
 *   - Otherwise the default is the store NEAREST the user's location, re-resolved
 *     on every app launch (auto-picks live in memory only, never persisted), so a
 *     member who moves gets their new nearest store next time they open the app.
 *   - When location is unavailable/denied, the default falls back to the first
 *     store.
 *
 * This context only controls the consumer-facing store used by ordering,
 * reservations and browsing. A staff token's operational store scope is kept
 * separately in auth and is enforced by the server on `/mini/staff/*` routes.
 */
const api = require('../services/api');

const STORAGE_KEY = 'ic_current_store'; // holds ONLY manual picks

let current = null; // this-session resolved store (manual from storage, or auto nearest)
let ensuring = null; // single-flight for the initial default resolve

// readManual returns the member's persisted manual pick, or null.
function readManual() {
  try {
    return wx.getStorageSync(STORAGE_KEY) || null;
  } catch {
    return null;
  }
}

function get() {
  if (current) return current;
  current = readManual();
  return current;
}

// set persists a store as the member's manual pick (or clears it with null).
function set(store) {
  current = store || null;
  try {
    if (current) wx.setStorageSync(STORAGE_KEY, current);
    else wx.removeStorageSync(STORAGE_KEY);
  } catch {}
  return current;
}

function getId() {
  const s = get();
  return s && s.id;
}

// switchStore records a store the member explicitly picked — persisted, so it
// wins over the nearest default until they switch again.
function switchStore(store) {
  return set(store);
}

// ensureStore returns the current store, resolving a default once per session
// when none is chosen. Preference: this-session store (manual or already-resolved
// nearest) > persisted manual pick > nearest to the user's location > first store.
// The nearest auto-pick is kept in memory only, so it is re-resolved on the next
// app launch. Single-flighted so concurrent callers share one resolve.
function ensureStore() {
  if (current) return Promise.resolve(current);
  const manual = readManual();
  if (manual && manual.id) {
    current = manual;
    return Promise.resolve(manual);
  }
  if (ensuring) return ensuring;
  ensuring = listNearby()
    .then((stores) => {
      current = stores[0] || null; // memory only — not persisted, re-resolved next launch
      return current;
    })
    .finally(() => {
      ensuring = null;
    });
  return ensuring;
}

// Refreshes the selected store from the distance-annotated public list. This
// keeps a persisted manual selection current after administrators update its
// address/coordinates, instead of rendering the stale object from local cache.
function refreshCurrent() {
  const selected = get();
  if (!selected || !selected.id) return ensureStore();
  return listNearby().then((stores) => {
    const refreshed = stores.find((store) => Number(store.id) === Number(selected.id));
    if (!refreshed) return selected;
    current = refreshed;
    const manual = readManual();
    if (manual && Number(manual.id) === Number(refreshed.id)) {
      try {
        wx.setStorageSync(STORAGE_KEY, refreshed);
      } catch {}
    }
    return current;
  });
}

// listNearby returns all active stores, distance-annotated and sorted
// nearest-first when a user location is available (for both the default resolve
// and the store-select list). Never rejects — returns [] on failure.
function listNearby() {
  return currentLocation().then((loc) => {
    const params = loc ? { lat: loc.latitude, lng: loc.longitude, pageSize: 50 } : { pageSize: 50 };
    return api
      .getStores(params)
      .then((res) => {
        const stores = (res.data || []).slice();
        if (loc) stores.sort((a, b) => distanceOf(a) - distanceOf(b));
        return stores;
      })
      .catch(() => []);
  });
}

function distanceOf(s) {
  return typeof s.distanceMeters === 'number' ? s.distanceMeters : Number.MAX_SAFE_INTEGER;
}

// currentLocation resolves the user's coordinates, or null when location is
// unavailable/denied — it never rejects, so store resolution always proceeds
// (the server just can't annotate distance and we fall back to the first store).
function currentLocation() {
  return new Promise((resolve) => {
    wx.getLocation({
      type: 'gcj02',
      success: (r) => resolve({ latitude: r.latitude, longitude: r.longitude }),
      fail: () => resolve(null),
    });
  });
}

// checkNearbyChange resolves the nearest store and returns it only when the
// member has a persisted manual pick that differs from their current nearest —
// signalling that they may have traveled. Returns null in all other cases
// (no manual pick, location unavailable, or already at the nearest store).
function checkNearbyChange() {
  const manual = readManual();
  if (!manual || !manual.id) return Promise.resolve(null);
  return listNearby().then((stores) => {
    const nearest = stores[0];
    if (!nearest || nearest.id === manual.id) return null;
    return nearest;
  });
}

module.exports = { get, set, getId, switchStore, ensureStore, refreshCurrent, listNearby, checkNearbyChange };
