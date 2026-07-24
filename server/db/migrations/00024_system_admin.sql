-- +goose Up
-- Mark the built-in headquarters account explicitly so account-management
-- rules do not depend on its username after this migration.
ALTER TABLE admin_accounts
  ADD COLUMN is_system TINYINT(1) NOT NULL DEFAULT 0 AFTER role;

UPDATE admin_accounts
SET is_system = 1
WHERE username = 'superadmin' AND role = 'super_admin';

-- +goose Down
ALTER TABLE admin_accounts
  DROP COLUMN is_system;
