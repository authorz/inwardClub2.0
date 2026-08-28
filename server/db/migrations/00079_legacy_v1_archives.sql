-- +goose Up
-- Preserve v1 records whose legacy tables overlap the canonical v2 ledger and
-- therefore cannot be replayed as additional balance-changing entries.

CREATE TABLE legacy_v1_archives (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_table VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NOT NULL,
  payload_json JSON NOT NULL,
  source_created_at DATETIME NULL,
  imported_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_legacy_v1_archive (source_table, source_id),
  KEY idx_legacy_v1_archive_created (source_table, source_created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS legacy_v1_archives;
