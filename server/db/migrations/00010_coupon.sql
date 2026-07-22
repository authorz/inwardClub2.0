-- +goose Up
-- Coupons reuse the catalog; templates reference items/categories rather than
-- copying products. Entitlements are member-held; redemptions record the hit.

CREATE TABLE coupon_templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  scope_type VARCHAR(16) NOT NULL DEFAULT 'store',
  store_id BIGINT UNSIGNED NULL,
  name VARCHAR(128) NOT NULL,
  description TEXT NULL,
  coupon_type VARCHAR(32) NOT NULL, -- exchange / discount / cash
  value_cent BIGINT NOT NULL DEFAULT 0,
  points_price BIGINT NOT NULL DEFAULT 0,
  stock_quantity BIGINT NOT NULL DEFAULT 0,
  issued_quantity BIGINT NOT NULL DEFAULT 0,
  validity_rule JSON NOT NULL,
  applicable_scope JSON NOT NULL,
  per_member_limit INT NOT NULL DEFAULT 0,
  assigned_store_ids JSON NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_coupon_templates_scope (scope_type, store_id),
  KEY idx_coupon_templates_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE coupon_entitlements (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  entitlement_no VARCHAR(64) NOT NULL,
  coupon_template_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active', -- active/used/void/expired
  rule_version INT NOT NULL DEFAULT 1,
  granted_reason VARCHAR(255) NOT NULL DEFAULT '',
  granted_by_type VARCHAR(20) NOT NULL DEFAULT '',
  granted_by_id BIGINT UNSIGNED NULL,
  expires_at DATETIME NULL,
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_entitlement_no (entitlement_no),
  UNIQUE KEY uq_entitlement_idem (idem_key),
  KEY idx_entitlements_member (member_id),
  KEY idx_entitlements_template (coupon_template_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE coupon_redemptions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  redemption_no VARCHAR(64) NOT NULL,
  entitlement_id BIGINT UNSIGNED NOT NULL,
  coupon_template_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  matched_rule_json JSON NULL,
  item_snapshot_json JSON NULL,
  verified_by_type VARCHAR(20) NOT NULL DEFAULT '',
  verified_by_id BIGINT UNSIGNED NULL,
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_redemption_no (redemption_no),
  UNIQUE KEY uq_redemption_entitlement (entitlement_id),
  UNIQUE KEY uq_redemption_idem (idem_key),
  KEY idx_redemptions_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS coupon_redemptions;
DROP TABLE IF EXISTS coupon_entitlements;
DROP TABLE IF EXISTS coupon_templates;
