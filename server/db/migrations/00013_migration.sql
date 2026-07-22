-- +goose Up
-- Data migration support: legacy id maps, migration runs, reconciliation results.

CREATE TABLE legacy_id_maps (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_system VARCHAR(32) NOT NULL DEFAULT 'v1',
  source_table VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NOT NULL,
  target_table VARCHAR(64) NOT NULL,
  target_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_legacy_map (source_system, source_table, source_id),
  KEY idx_legacy_target (target_table, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE migration_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  run_key VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'running',
  started_at DATETIME NOT NULL,
  finished_at DATETIME NULL,
  summary_json JSON NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_migration_run_key (run_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE reconciliation_results (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  run_key VARCHAR(64) NOT NULL,
  check_name VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL, -- ok / diff / error
  expected_json JSON NULL,
  actual_json JSON NULL,
  note VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_reconciliation_run (run_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS reconciliation_results;
DROP TABLE IF EXISTS migration_runs;
DROP TABLE IF EXISTS legacy_id_maps;
