-- +goose Up
ALTER TABLE printer_devices
  ADD COLUMN sound_enabled TINYINT(1) NOT NULL DEFAULT 1 AFTER status;

-- +goose Down
ALTER TABLE printer_devices
  DROP COLUMN sound_enabled;
