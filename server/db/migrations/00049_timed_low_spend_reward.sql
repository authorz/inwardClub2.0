-- +goose Up
-- Headquarters-managed timed low-spend reward. Values are deliberately stored
-- in system_settings so operations can change them without a deployment.
INSERT INTO system_settings
  (setting_key, setting_value, updated_by, created_at, updated_at)
VALUES
  ('timed_low_spend_reward_enabled', 'true', NULL, NOW(), NOW()),
  ('timed_low_spend_reservation_cutoff', '20:00', NULL, NOW(), NOW()),
  ('timed_low_spend_consumption_cutoff', '20:30', NULL, NOW(), NOW()),
  ('timed_low_spend_minimum_amount', '88', NULL, NOW(), NOW()),
  ('timed_low_spend_reward_points', '2000', NULL, NOW(), NOW())
ON DUPLICATE KEY UPDATE setting_key = VALUES(setting_key);

-- +goose Down
DELETE FROM system_settings
WHERE setting_key IN (
  'timed_low_spend_reward_enabled',
  'timed_low_spend_reservation_cutoff',
  'timed_low_spend_consumption_cutoff',
  'timed_low_spend_minimum_amount',
  'timed_low_spend_reward_points'
);
