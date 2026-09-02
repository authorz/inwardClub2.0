/**
 * Current store selection shared across tabs (home / reservation / ordering).
 *
 * Almost every member data read/write is store-scoped, so pages call
 * `ensureStore()` before fetching. Semantics:
 *   - Every resolved store is persisted so store-scoped requests can immediately
 *     reuse its id. Manual selections remain authoritative until changed again.
 *   - An automatically selected store is re-resolved from location on each app
 *     launch, then the cached store and id are refreshed.
 *   - When location is unavailable/denied, the default falls back to the first
 *     store.
 *
 * This context only controls the consumer-facing store used by ordering,
 * reservations and browsing. A staff token's operational store scope is kept
 * separately in auth and is enforced by the server on `/mini/staff/*` routes.
 */
const api = require('../services/api');

const STORAGE_KEY = 'ic_current_store';
const SOURCE_KEY = 'ic_current_store_source';
const SOURCE_AUTO = 'auto';
const SOURCE_MANUAL = 'manual';

let current = null;
let resolvedThisSession = false;
let ensuring = null; // single-flight for the initial default resolve

function readCached() {
  try {
    return wx.getStorageSync(STORAGE_KEY) || null;
  } catch {
    return null;
  }
}

function readSource(cached) {
  try {
    const source = wx.getStorageSync(SOURCE_KEY);
    if (source === SOURCE_AUTO || source === SOURCE_MANUAL) return source;
  } catch {}
  // Existing installations only persisted manual selections under STORAGE_KEY.
  return cached && cached.id ? SOURCE_MANUAL : '';
}

function get() {
  if (current) return current;
  current = readCached();
  return current;
}

function persist(store, source) {
  current = store || null;
  resolvedThisSession = Boolean(current);
  try {
    if (current) {
      wx.setStorageSync(STORAGE_KEY, current);
      wx.setStorageSync(SOURCE_KEY, source);
    } else {
      wx.removeStorageSync(STORAGE_KEY);
      wx.removeStorageSync(SOURCE_KEY);
    }
  } catch {}
  return current;
}

// set persists an explicit store switch. Existing callers invoke it only when a
// user flow has deliberately changed the active store.
function set(store) {
  return persist(store, SOURCE_MANUAL);
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

// ensureStore returns the current store, resolving a default once per session.
// Manual selections win; cached automatic selections are fallbacks while the
// nearest store is re-resolved and persisted for the current launch.
function ensureStore() {
  if (current && resolvedThisSession) return Promise.resolve(current);
  const cached = current || readCached();
  if (cached && cached.id && readSource(cached) === SOURCE_MANUAL) {
    current = cached;
    resolvedThisSession = true;
    return Promise.resolve(cached);
  }
  if (ensuring) return ensuring;
  ensuring = listNearby()
    .then((stores) => {
      const nearest = stores[0] || cached || null;
      return nearest ? persist(nearest, SOURCE_AUTO) : null;
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
  const source = readSource(selected) || SOURCE_AUTO;
  return listNearby().then((stores) => {
    const refreshed = stores.find((store) => Number(store.id) === Number(selected.id));
    if (!refreshed) return selected;
    return persist(refreshed, source);
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
  const manual = get();
  if (!manual || !manual.id || readSource(manual) !== SOURCE_MANUAL) return Promise.resolve(null);
  return listNearby().then((stores) => {
    const nearest = stores[0];
    if (!nearest || nearest.id === manual.id) return null;
    return nearest;
  });
}

module.exports = { get, set, getId, switchStore, ensureStore, refreshCurrent, listNearby, checkNearbyChange };
