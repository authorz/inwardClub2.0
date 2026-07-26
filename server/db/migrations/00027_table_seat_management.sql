-- +goose Up
-- Admin table/seat management fields. Existing tables receive a stable,
-- per-store-safe code before the uniqueness constraint is applied.

ALTER TABLE tables
  ADD COLUMN code VARCHAR(64) NULL AFTER name,
  ADD COLUMN base_points INT UNSIGNED NOT NULL DEFAULT 0 AFTER capacity;

UPDATE tables SET code = CONCAT('T', id) WHERE code IS NULL OR code = '';

ALTER TABLE tables
  MODIFY COLUMN code VARCHAR(64) NOT NULL,
  ADD UNIQUE KEY uq_tables_store_code (store_id, code);

-- +goose Down
ALTER TABLE tables
  DROP INDEX uq_tables_store_code,
  DROP COLUMN base_points,
  DROP COLUMN code;
