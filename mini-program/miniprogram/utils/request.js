/**
 * Unified request client — the ONLY place that talks to the network.
 * (GLOBAL_DESIGN_RULES §7: no scattered wx.request across pages.)
 *
 * Responsibilities:
 *  - prefix baseURL
 *  - inject bearer token from utils/auth
 *  - attach Idempotency-Key on writes that require it
 *  - parse the `{ data, meta } | { error: { code, message, details } }` envelope
 *  - single-flight refresh-token on 401, then retry once
 *  - surface a normalized ApiError to callers
 *  - transparently serve fixtures when ENV.useMock is on
 */
const ENV = require('../config/env');
const auth = require('./auth');
const mock = require('../services/mock');

class ApiError extends Error {
  constructor(code, message, details, httpStatus) {
    super(message || code || 'REQUEST_FAILED');
    this.code = code || 'REQUEST_FAILED';
    this.details = details || null;
    this.httpStatus = httpStatus || 0;
    this.isApiError = true;
  }
}

let refreshing = null; // single-flight refresh promise

function uuid() {
  return 'xxxxxxxxyxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  }) + Date.now().toString(16);
}

function rawRequest(method, path, options) {
  const opts = options || {};
  const header = Object.assign(
    { 'content-type': 'application/json' },
    opts.header || {}
  );
  const token = auth.getAccessToken();
  if (token && !opts.noAuth) header.Authorization = 'Bearer ' + token;
  // Writes that mutate money/assets/verification require an idempotency key.
  if (opts.idempotent) header['Idempotency-Key'] = opts.idempotencyKey || uuid();

  return new Promise((resolve, reject) => {
    wx.request({
      url: ENV.apiBaseUrl + path,
      method,
      data: opts.data,
      header,
      timeout: ENV.requestTimeout,
      success: (res) => {
        const body = res.data || {};
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve({ data: body.data, meta: body.meta });
          return;
        }
        const err = body.error || {};
        reject(new ApiError(err.code, err.message, err.details, res.statusCode));
      },
      fail: (err) => {
        reject(new ApiError('NETWORK_ERROR', (err && err.errMsg) || '网络异常', null, 0));
      },
    });
  });
}

function doRefresh() {
  if (refreshing) return refreshing;
  const refreshToken = auth.getRefreshToken();
  if (!refreshToken) return Promise.reject(new ApiError('UNAUTHENTICATED', '登录已过期'));
  refreshing = rawRequest('POST', '/mini/auth/refresh', {
    noAuth: true,
    data: { refreshToken },
  })
    .then((res) => {
      auth.setTokens({
        accessToken: res.data.accessToken,
        refreshToken: res.data.refreshToken || refreshToken,
      });
      return true;
    })
    .finally(() => {
      refreshing = null;
    });
  return refreshing;
}

async function request(method, path, options) {
  const opts = options || {};

  if (ENV.useMock) {
    return mock.handle(method, path, opts);
  }

  try {
    return await rawRequest(method, path, opts);
  } catch (err) {
    if (err.isApiError && err.httpStatus === 401 && !opts.noAuth) {
      // First 401: try one silent refresh, then retry once.
      if (!opts._retried) {
        try {
          await doRefresh();
        } catch (e) {
          return failExpired(err);
        }
        return request(method, path, Object.assign({}, opts, { _retried: true }));
      }
      // Retried and still 401 → the session is truly invalid.
      return failExpired(err);
    }
    throw err;
  }
}

// failExpired clears the dead session and signals the app once (app.onSessionExpired)
// so it can drop any logged-in page UI and prompt a re-login, then rethrows so the
// caller's own error handling (empty-state fallback, etc.) still runs.
function failExpired(err) {
  auth.clear();
  const app = typeof getApp === 'function' ? getApp() : null;
  if (app && typeof app.onSessionExpired === 'function') app.onSessionExpired();
  throw err;
}

module.exports = {
  ApiError,
  request,
  get: (path, opts) => request('GET', path, opts),
  post: (path, data, opts) => request('POST', path, Object.assign({ data }, opts)),
  patch: (path, data, opts) => request('PATCH', path, Object.assign({ data }, opts)),
  put: (path, data, opts) => request('PUT', path, Object.assign({ data }, opts)),
  del: (path, opts) => request('DELETE', path, opts),
  uuid,
};
