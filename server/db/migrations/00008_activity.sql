-- +goose Up
-- Activities: one activity -> many sessions/ticket types; one order -> many
-- ticket instances, each with its own verification code and record.

CREATE TABLE activities (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  scope_type VARCHAR(16) NOT NULL DEFAULT 'store',
  store_id BIGINT UNSIGNED NULL,
  source_activity_id BIGINT UNSIGNED NULL,
  title VARCHAR(128) NOT NULL,
  description TEXT NULL,
  content MEDIUMTEXT NULL,
  asset_id BIGINT UNSIGNED NULL,
  start_at DATETIME NULL,
  end_at DATETIME NULL,
  pay_channels JSON NOT NULL,
  purchase_limit_per_member INT NOT NULL DEFAULT 0,
  assigned_store_ids JSON NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_activities_scope (scope_type, store_id),
  KEY idx_activities_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE activity_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  activity_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NULL,
  name VARCHAR(128) NOT NULL,
  start_at DATETIME NOT NULL,
  end_at DATETIME NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_sessions_activity (activity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE activity_ticket_types (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  activity_id BIGINT UNSIGNED NOT NULL,
  session_id BIGINT UNSIGNED NULL,
  store_id BIGINT UNSIGNED NULL,
  name VARCHAR(128) NOT NULL,
  price_cent BIGINT NOT NULL DEFAULT 0,
  stock_quantity BIGINT NOT NULL DEFAULT 0,
  sold_quantity BIGINT NOT NULL DEFAULT 0,
  sale_start_at DATETIME NULL,
  sale_end_at DATETIME NULL,
  pay_channels JSON NOT NULL,
  max_tickets_per_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_ticket_types_activity (activity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE activity_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  business_order_id BIGINT UNSIGNED NOT NULL,
  activity_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  ticket_count INT NOT NULL DEFAULT 0,
  total_amount_cent BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'created',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_activity_business_order (business_order_id),
  KEY idx_activity_orders_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE tickets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  ticket_no VARCHAR(64) NOT NULL,
  verification_code VARCHAR(32) NOT NULL,
  activity_order_id BIGINT UNSIGNED NOT NULL,
  activity_id BIGINT UNSIGNED NOT NULL,
  ticket_type_id BIGINT UNSIGNED NOT NULL,
  session_id BIGINT UNSIGNED NULL,
  store_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  price_cent BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending/active/used/refunded/expired
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ticket_no (ticket_no),
  UNIQUE KEY uq_ticket_code (verification_code),
  KEY idx_tickets_order (activity_order_id),
  KEY idx_tickets_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE verifications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  verification_no VARCHAR(64) NOT NULL,
  target_type VARCHAR(32) NOT NULL, -- ticket / coupon
  target_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  verified_by_type VARCHAR(20) NOT NULL,
  verified_by_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NULL,
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_verification_no (verification_no),
  UNIQUE KEY uq_verification_target (target_type, target_id),
  UNIQUE KEY uq_verification_idem (idem_key),
  KEY idx_verifications_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS activity_orders;
DROP TABLE IF EXISTS activity_ticket_types;
DROP TABLE IF EXISTS activity_sessions;
DROP TABLE IF EXISTS activities;
