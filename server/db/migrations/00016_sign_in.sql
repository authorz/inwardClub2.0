-- +goose Up
-- Daily sign-in records. One row per (member, sign_date); the unique key makes a
-- second sign-in on the same day a no-op instead of a double award. streak_days
-- is the consecutive-day count that produced points_awarded, and idem_key ties
-- the row to the wallet ledger entry created in the same transaction.

CREATE TABLE sign_in_records (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  member_id BIGINT UNSIGNED NOT NULL,
  sign_date DATE NOT NULL,
  streak_days INT NOT NULL,
  points_awarded BIGINT NOT NULL,
  idem_key VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_sign_in_member_date (member_id, sign_date),
  KEY idx_sign_in_member (member_id, sign_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS sign_in_records;
