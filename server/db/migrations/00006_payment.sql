-- +goose Up
-- Unified payment: business_orders -> payment_orders -> payment_transactions;
-- refunds and offline aggregated collection. External trade numbers are unique.

CREATE TABLE business_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  business_order_no VARCHAR(64) NOT NULL,
  order_type VARCHAR(32) NOT NULL, -- food / activity / recharge / coupon / offline_collection
  store_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NULL,
  total_amount_cent BIGINT NOT NULL DEFAULT 0,
  order_status VARCHAR(32) NOT NULL DEFAULT 'created',
  payment_status VARCHAR(32) NOT NULL DEFAULT 'unpaid',
  snapshot_json JSON NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_business_order_no (business_order_no),
  KEY idx_business_orders_member (member_id),
  KEY idx_business_orders_store (store_id),
  KEY idx_business_orders_type_status (order_type, payment_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE payment_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  payment_order_no VARCHAR(64) NOT NULL,
  business_order_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NULL,
  amount_cent BIGINT NOT NULL,
  -- pay_method: wechat / coin (mini). Offline channel resolved at callback.
  pay_method VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  paid_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payment_order_no (payment_order_no),
  UNIQUE KEY uq_payment_idem (idem_key),
  KEY idx_payment_business (business_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE payment_transactions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  payment_order_id BIGINT UNSIGNED NOT NULL,
  provider VARCHAR(32) NOT NULL, -- wechat / coin / offline_acquirer
  channel VARCHAR(32) NOT NULL DEFAULT '', -- wechat / alipay for offline
  out_trade_no VARCHAR(64) NOT NULL,
  external_transaction_no VARCHAR(128) NULL,
  amount_cent BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'success',
  raw_payload JSON NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_txn_provider_out_trade (provider, out_trade_no),
  UNIQUE KEY uq_txn_external (external_transaction_no),
  KEY idx_txn_payment (payment_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE refund_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  refund_order_no VARCHAR(64) NOT NULL,
  payment_order_id BIGINT UNSIGNED NOT NULL,
  business_order_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NULL,
  amount_cent BIGINT NOT NULL,
  channel VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  reason VARCHAR(255) NOT NULL DEFAULT '',
  external_refund_no VARCHAR(128) NULL,
  idem_key VARCHAR(128) NULL,
  requested_by_type VARCHAR(20) NOT NULL DEFAULT '',
  requested_by_id BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_refund_order_no (refund_order_no),
  UNIQUE KEY uq_refund_idem (idem_key),
  UNIQUE KEY uq_refund_external (external_refund_no),
  KEY idx_refund_payment (payment_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Offline aggregated collection orders. member_id is nullable and locked at
-- creation; only masked phone snapshot is stored, never the raw phone.
CREATE TABLE offline_collection_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  collection_order_no VARCHAR(64) NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  payment_order_id BIGINT UNSIGNED NOT NULL,
  amount_cent BIGINT NOT NULL,
  subject VARCHAR(128) NOT NULL,
  business_type VARCHAR(32) NOT NULL,
  member_id BIGINT UNSIGNED NULL,
  member_phone_masked VARCHAR(20) NULL,
  bound_by_type VARCHAR(20) NULL,
  bound_by_id BIGINT UNSIGNED NULL,
  bound_at DATETIME NULL,
  acquirer_order_no VARCHAR(128) NULL,
  qr_content VARCHAR(512) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  created_by_type VARCHAR(20) NOT NULL DEFAULT '',
  created_by_id BIGINT UNSIGNED NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_collection_order_no (collection_order_no),
  KEY idx_collection_store (store_id),
  KEY idx_collection_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS offline_collection_orders;
DROP TABLE IF EXISTS refund_orders;
DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS payment_orders;
DROP TABLE IF EXISTS business_orders;
