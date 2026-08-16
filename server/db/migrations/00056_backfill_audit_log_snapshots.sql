-- +goose Up
UPDATE audit_logs al
JOIN admin_accounts aa ON aa.id = al.actor_id
SET al.actor_snapshot_json = JSON_OBJECT(
  'type', al.actor_type,
  'id', al.actor_id,
  'role', al.actor_role,
  'username', aa.username,
  'displayName', aa.display_name,
  'source', 'backfill_current_state'
)
WHERE al.actor_snapshot_json IS NULL
  AND al.actor_type IN ('super_admin', 'store_admin', 'cashier');

UPDATE audit_logs al
JOIN members m ON m.id = al.target_id
SET al.target_snapshot_json = JSON_OBJECT(
  'type', al.target_type,
  'id', al.target_id,
  'nickname', m.nickname,
  'phone', COALESCE(m.phone, ''),
  'avatarUrl', COALESCE(m.avatar_url, ''),
  'source', 'backfill_current_state'
)
WHERE al.target_snapshot_json IS NULL
  AND al.target_type = 'member';

UPDATE audit_logs al
JOIN stores s ON s.id = al.store_id
SET al.scope_snapshot_json = JSON_OBJECT(
  'storeId', al.store_id,
  'storeName', s.name,
  'source', 'backfill_current_state'
)
WHERE al.scope_snapshot_json IS NULL
  AND al.store_id IS NOT NULL;

-- +goose Down
SELECT 1;
