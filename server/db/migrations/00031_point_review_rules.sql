-- +goose Up
-- Configurable headquarters rule plus immutable calculation snapshots for the
-- staff-reviewed point-saving flow.

CREATE TABLE point_review_settings (
  id TINYINT UNSIGNED NOT NULL,
  points_divisor BIGINT UNSIGNED NOT NULL DEFAULT 5,
  coin_points_divisor BIGINT UNSIGNED NOT NULL DEFAULT 2000,
  version BIGINT UNSIGNED NOT NULL DEFAULT 1,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO point_review_settings
  (id, points_divisor, coin_points_divisor, version, created_at, updated_at)
VALUES (1, 5, 2000, 1, NOW(), NOW());

ALTER TABLE point_savings
  ADD COLUMN base_points BIGINT NOT NULL DEFAULT 0 AFTER points,
  ADD COLUMN excess_points BIGINT NOT NULL DEFAULT 0 AFTER base_points,
  ADD COLUMN awarded_points BIGINT NOT NULL DEFAULT 0 AFTER excess_points,
  ADD COLUMN coin_base_points BIGINT NOT NULL DEFAULT 0 AFTER awarded_points,
  ADD COLUMN awarded_coins BIGINT NOT NULL DEFAULT 0 AFTER coin_base_points,
  ADD COLUMN rule_version BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER awarded_coins,
  ADD COLUMN points_divisor BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER rule_version,
  ADD COLUMN coin_points_divisor BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER points_divisor,
  ADD COLUMN business_date DATE NULL AFTER coin_points_divisor,
  ADD COLUMN business_start_at DATETIME NULL AFTER business_date,
  ADD COLUMN business_end_at DATETIME NULL AFTER business_start_at,
  ADD COLUMN calculation_start_at DATETIME NULL AFTER business_end_at,
  ADD COLUMN calculation_end_at DATETIME NULL AFTER calculation_start_at,
  ADD COLUMN last_approved_saving_id BIGINT UNSIGNED NULL AFTER calculation_end_at,
  ADD COLUMN calculation_description VARCHAR(512) NOT NULL DEFAULT '' AFTER last_approved_saving_id;

-- +goose Down
ALTER TABLE point_savings
  DROP COLUMN calculation_description,
  DROP COLUMN last_approved_saving_id,
  DROP COLUMN calculation_end_at,
  DROP COLUMN calculation_start_at,
  DROP COLUMN business_end_at,
  DROP COLUMN business_start_at,
  DROP COLUMN business_date,
  DROP COLUMN coin_points_divisor,
  DROP COLUMN points_divisor,
  DROP COLUMN rule_version,
  DROP COLUMN awarded_coins,
  DROP COLUMN coin_base_points,
  DROP COLUMN awarded_points,
  DROP COLUMN excess_points,
  DROP COLUMN base_points;

DROP TABLE IF EXISTS point_review_settings;
