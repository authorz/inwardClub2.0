-- +goose Up
-- Configurable rule engine. No business value (points/coins/vip/invite/recharge)
-- is hard-coded; rules are versioned, effective-dated and disabled by default.

CREATE TABLE rule_definitions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  rule_key VARCHAR(64) NOT NULL,
  scope_type VARCHAR(16) NOT NULL DEFAULT 'global',
  store_id BIGINT UNSIGNED NULL,
  version INT NOT NULL DEFAULT 1,
  config_json JSON NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 0,
  effective_from DATETIME NULL,
  effective_to DATETIME NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft', -- draft/published/disabled
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_rule_version (rule_key, scope_type, store_id, version),
  KEY idx_rules_key_status (rule_key, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE rule_executions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  rule_key VARCHAR(64) NOT NULL,
  rule_version INT NOT NULL,
  member_id BIGINT UNSIGNED NULL,
  source_type VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NULL,
  result_json JSON NULL,
  idem_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_rule_execution_idem (idem_key),
  KEY idx_rule_exec_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE benefit_grants (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  grant_no VARCHAR(64) NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  benefit_type VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL DEFAULT 0,
  rule_key VARCHAR(64) NOT NULL,
  rule_version INT NOT NULL,
  source_type VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'granted',
  idem_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_grant_no (grant_no),
  UNIQUE KEY uq_grant_idem (idem_key),
  KEY idx_grants_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS benefit_grants;
DROP TABLE IF EXISTS rule_executions;
DROP TABLE IF EXISTS rule_definitions;
