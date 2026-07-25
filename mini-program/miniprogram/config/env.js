/**
 * Runtime configuration. Single source of truth for the API base URL.
 * No page or service hardcodes a host.
 */
const ENV = {
  // API root per docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md §5.1.
  // Member paths are prefixed with /mini and staff paths with /store in
  // services/api.js.
  // Local dev backend (Go /api/v2/mini/*). Point this at wherever the API runs;
  // for production set the real host.
  // NOTE: the API must run against the LOCAL migrated docker DB, never server/.env's
  // remote DSN (which lacks recent migrations). See docs/acceptance boot recipe.
  apiBaseUrl: 'http://192.168.0.161:8081/api/v2',
  requestTimeout: 15000,
};

module.exports = ENV;
