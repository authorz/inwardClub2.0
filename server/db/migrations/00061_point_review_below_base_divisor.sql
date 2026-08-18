-- +goose Up
ALTER TABLE point_review_settings
  ADD COLUMN below_base_points_divisor BIGINT UNSIGNED NOT NULL DEFAULT 2 AFTER points_divisor;

ALTER TABLE point_savings
  ADD COLUMN below_base_points_divisor BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER points_divisor;

-- +goose Down
ALTER TABLE point_savings
  DROP COLUMN below_base_points_divisor;

ALTER TABLE point_review_settings
  DROP COLUMN below_base_points_divisor;
