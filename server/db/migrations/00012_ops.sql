-- +goose Up
-- Operations & observability: banners, printers, print jobs, audit logs,
-- outbox, idempotency keys and daily reporting rollups.

CREATE TABLE banners (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  scope_type VARCHAR(16) NOT NULL DEFAULT 'store',
  store_id BIGINT UNSIGNED NULL,
  title VARCHAR(128) NOT NULL DEFAULT '',
  asset_id BIGINT UNSIGNED NOT NULL,
  link_url VARCHAR(512) NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_banners_scope (scope_type, store_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE printer_devices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  provider VARCHAR(32) NOT NULL DEFAULT 'xpyun',
  device_sn VARCHAR(64) NOT NULL,
  device_key VARCHAR(64) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_printer_sn (provider, device_sn),
  KEY idx_printers_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE print_jobs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  device_id BIGINT UNSIGNED NULL,
  template VARCHAR(64) NOT NULL,
  business_order_no VARCHAR(64) NOT NULL,
  payload JSON NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  idem_key VARCHAR(128) NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_print_idem (idem_key),
  KEY idx_print_jobs_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE audit_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  actor_type VARCHAR(20) NOT NULL,
  actor_id BIGINT UNSIGNED NOT NULL,
  actor_role VARCHAR(32) NOT NULL,
  store_id BIGINT UNSIGNED NULL,
  action VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NOT NULL,
  target_id BIGINT UNSIGNED NOT NULL,
  before_json JSON NULL,
  after_json JSON NULL,
  reason VARCHAR(512) NOT NULL DEFAULT '',
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_audit_store (store_id),
  KEY idx_audit_actor (actor_type, actor_id),
  KEY idx_audit_target (target_type, target_id),
  KEY idx_audit_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE outbox_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  topic VARCHAR(64) NOT NULL,
  payload JSON NOT NULL,
  idem_key VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  available_at DATETIME NOT NULL,
  dispatched_at DATETIME NULL,
  last_error VARCHAR(512) NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_outbox_status_available (status, available_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE idempotency_keys (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  idem_key VARCHAR(128) NOT NULL,
  scope VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NOT NULL DEFAULT '',
  target_id BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_idempotency (scope, idem_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE reporting_daily (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NULL,
  report_date DATE NOT NULL,
  metric VARCHAR(64) NOT NULL,
  amount_cent BIGINT NOT NULL DEFAULT 0,
  quantity BIGINT NOT NULL DEFAULT 0,
  detail_json JSON NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_reporting (store_id, report_date, metric)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS reporting_daily;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS print_jobs;
DROP TABLE IF EXISTS printer_devices;
DROP TABLE IF EXISTS banners;
