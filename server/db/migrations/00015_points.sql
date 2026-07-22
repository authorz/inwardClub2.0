-- +goose Up
-- Points savings / withdrawals: member-initiated requests reviewed by the store
-- console. A request is created 'pending' by the mini program and later
-- approved or rejected; the reviewing staff and time are captured in place. The
-- idempotency key is unique so a retried create returns the same request.
-- Approved point_savings back the monthly points leaderboard.

CREATE TABLE point_savings (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  points BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending/approved/rejected
  remark VARCHAR(255) NOT NULL DEFAULT '',
  idem_key VARCHAR(128) NULL,
  reviewed_by BIGINT UNSIGNED NULL,
  reviewed_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_point_savings_idem (idem_key),
  KEY idx_point_savings_store (store_id, status),
  KEY idx_point_savings_member (member_id),
  KEY idx_point_savings_ranking (status, reviewed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE point_withdrawals (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  points BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending/approved/rejected
  remark VARCHAR(255) NOT NULL DEFAULT '',
  idem_key VARCHAR(128) NULL,
  reviewed_by BIGINT UNSIGNED NULL,
  reviewed_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_point_withdrawals_idem (idem_key),
  KEY idx_point_withdrawals_store (store_id, status),
  KEY idx_point_withdrawals_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS point_withdrawals;
DROP TABLE IF EXISTS point_savings;
