-- +goose Up
ALTER TABLE members
  ADD COLUMN phone_changed_at DATETIME NULL AFTER phone;

-- Existing members inherit their account creation time. Accounts older than the
-- configured interval can therefore change immediately after this migration.
UPDATE members
SET phone_changed_at = created_at
WHERE phone IS NOT NULL
  AND TRIM(phone) <> ''
  AND phone_changed_at IS NULL;

INSERT INTO system_settings
  (setting_key, setting_value, updated_by, created_at, updated_at)
VALUES
  ('phone_change_interval_days', '30', NULL, NOW(), NOW())
ON DUPLICATE KEY UPDATE setting_key = VALUES(setting_key);

-- +goose Down
DELETE FROM system_settings WHERE setting_key = 'phone_change_interval_days';
ALTER TABLE members DROP COLUMN phone_changed_at;
