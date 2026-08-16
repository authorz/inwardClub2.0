-- +goose Up
ALTER TABLE audit_logs
  ADD COLUMN actor_snapshot_json JSON NULL AFTER actor_role,
  ADD COLUMN target_snapshot_json JSON NULL AFTER target_id,
  ADD COLUMN scope_snapshot_json JSON NULL AFTER store_id;

-- +goose Down
ALTER TABLE audit_logs
  DROP COLUMN scope_snapshot_json,
  DROP COLUMN target_snapshot_json,
  DROP COLUMN actor_snapshot_json;
