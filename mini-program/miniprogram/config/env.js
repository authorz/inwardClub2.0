/**
 * Runtime configuration. Single source of truth for the API base URL and the
 * mock switch. No page or service hardcodes a host.
 *
 * `useMock` lets the whole app run inside WeChat DevTools without a live
 * backend: the request client transparently serves fixtures from services/mock.
 * Flip to false (and set apiBaseUrl) once the Go `/api/v2/mini/*` server is up.
 */
const ENV = {
  // API root per docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md §5.1.
  // Member paths are prefixed with /mini and staff paths with /store in
  // services/api.js.
  // Local dev backend (Go /api/v2/mini/*). Point this at wherever the API runs;
  // for production set the real host. The mini now consumes live interfaces, not
  // mock fixtures. Staff (/store/*) pages are deferred — with no staff signal in
  // the member login the 工作人员入口 hides itself in live mode.
  // NOTE: the API must run against the LOCAL migrated docker DB, never server/.env's
  // remote DSN (which lacks recent migrations). See docs/acceptance boot recipe.
  apiBaseUrl: 'http://127.0.0.1:18477/api/v2',
  useMock: false,
  requestTimeout: 15000,
  // simulated latency for mock responses (ms)
  mockLatency: 220,
};

module.exports = ENV;
