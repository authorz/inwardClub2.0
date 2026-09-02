const assert = require('node:assert/strict');
const test = require('node:test');

const storage = new Map();
let stores = [];
let storeRequests = 0;

global.wx = {
  getStorageSync: (key) => storage.get(key) || null,
  setStorageSync: (key, value) => storage.set(key, value),
  removeStorageSync: (key) => storage.delete(key),
  getLocation: ({ success }) => success({ latitude: 31.2, longitude: 121.5 }),
};

const api = require('../miniprogram/services/api');

function freshStoreContext() {
  const path = require.resolve('../miniprogram/utils/store-context');
  delete require.cache[path];
  return require(path);
}

test.beforeEach(() => {
  storage.clear();
  stores = [];
  storeRequests = 0;
  api.getStores = () => {
    storeRequests += 1;
    return Promise.resolve({ data: stores });
  };
});

test('nearest automatic store is cached with its id', async () => {
  stores = [
    { id: 2, name: '远店', distanceMeters: 800 },
    { id: 1, name: '近店', distanceMeters: 100 },
  ];
  const storeCtx = freshStoreContext();

  const selected = await storeCtx.ensureStore();

  assert.equal(selected.id, 1);
  assert.equal(storage.get('ic_current_store').id, 1);
  assert.equal(storage.get('ic_current_store_source'), 'auto');
});

test('manual store switch replaces the cached store id', () => {
  const storeCtx = freshStoreContext();

  storeCtx.switchStore({ id: 9, name: '手选门店' });

  assert.equal(storage.get('ic_current_store').id, 9);
  assert.equal(storage.get('ic_current_store_source'), 'manual');
});

test('legacy cached store remains a manual selection', async () => {
  storage.set('ic_current_store', { id: 5, name: '旧缓存门店' });
  stores = [{ id: 1, name: '最近门店', distanceMeters: 10 }];
  const storeCtx = freshStoreContext();

  const selected = await storeCtx.ensureStore();

  assert.equal(selected.id, 5);
  assert.equal(storeRequests, 0);
});

test('cached automatic store is refreshed from location on a new launch', async () => {
  storage.set('ic_current_store', { id: 5, name: '上次最近门店' });
  storage.set('ic_current_store_source', 'auto');
  stores = [{ id: 3, name: '本次最近门店', distanceMeters: 20 }];
  const storeCtx = freshStoreContext();

  const selected = await storeCtx.ensureStore();

  assert.equal(selected.id, 3);
  assert.equal(storage.get('ic_current_store').id, 3);
  assert.equal(storage.get('ic_current_store_source'), 'auto');
});
