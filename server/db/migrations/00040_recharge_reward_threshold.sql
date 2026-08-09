-- +goose Up
INSERT INTO system_settings
  (setting_key, setting_value, updated_by, created_at, updated_at)
VALUES
  ('recharge_double_points_threshold_amount', '1000', NULL, NOW(), NOW())
ON DUPLICATE KEY UPDATE setting_key = VALUES(setting_key);

-- +goose Down
DELETE FROM system_settings
WHERE setting_key = 'recharge_double_points_threshold_amount';
