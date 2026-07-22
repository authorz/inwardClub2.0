-- +goose Up
-- Stores and their per-store settings, rule overrides, tables and seats.

CREATE TABLE stores (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  legacy_store_id BIGINT UNSIGNED NULL,
  name VARCHAR(128) NOT NULL,
  logo_asset_id BIGINT UNSIGNED NULL,
  phone VARCHAR(20) NULL,
  address VARCHAR(255) NOT NULL DEFAULT '',
  latitude DECIMAL(10,7) NULL,
  longitude DECIMAL(10,7) NULL,
  business_hours VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_stores_legacy (legacy_store_id),
  KEY idx_stores_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE store_settings (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  settings_json JSON NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_store_settings (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- store_rules overrides global rule_definitions per store (values are never
-- hard-coded in Go; a NULL/absent row means "inherit global").
CREATE TABLE store_rules (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  rule_key VARCHAR(64) NOT NULL,
  config_json JSON NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_store_rule (store_id, rule_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE tables (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  capacity INT UNSIGNED NOT NULL DEFAULT 0,
  layout_asset_id BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'available',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_tables_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE seats (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  table_id BIGINT UNSIGNED NULL,
  name VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'available',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_seats_store (store_id),
  KEY idx_seats_table (table_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS seats;
DROP TABLE IF EXISTS tables;
DROP TABLE IF EXISTS store_rules;
DROP TABLE IF EXISTS store_settings;
DROP TABLE IF EXISTS stores;
