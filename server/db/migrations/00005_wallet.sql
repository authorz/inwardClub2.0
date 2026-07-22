-- +goose Up
-- Wallet: one account per (member, asset). Ledger is append-only; available
-- balance equals the sum of effective ledger rows. Holds reserve funds.

CREATE TABLE wallet_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  member_id BIGINT UNSIGNED NOT NULL,
  -- asset_type: points / coins / cash_balance / growth_value
  asset_type VARCHAR(32) NOT NULL,
  available_amount BIGINT NOT NULL DEFAULT 0,
  held_amount BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_wallet_account (member_id, asset_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Append-only ledger. Corrections are new opposite-direction rows; never UPDATE.
CREATE TABLE wallet_ledger_entries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  asset_type VARCHAR(32) NOT NULL,
  -- direction: credit (+) or debit (-)
  direction VARCHAR(8) NOT NULL,
  amount BIGINT NOT NULL,
  balance_after BIGINT NOT NULL,
  reason VARCHAR(64) NOT NULL,
  source_type VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NULL,
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ledger_idem (idem_key),
  KEY idx_ledger_account (account_id),
  KEY idx_ledger_member (member_id, asset_type),
  KEY idx_ledger_source (source_type, source_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE wallet_holds (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  asset_type VARCHAR(32) NOT NULL,
  amount BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  source_type VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NULL,
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  released_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_hold_idem (idem_key),
  KEY idx_holds_account (account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS wallet_holds;
DROP TABLE IF EXISTS wallet_ledger_entries;
DROP TABLE IF EXISTS wallet_accounts;
