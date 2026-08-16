-- +goose Up
ALTER TABLE point_savings
  ADD COLUMN reviewed_by_type VARCHAR(32) NULL AFTER reviewed_by,
  ADD COLUMN reviewer_snapshot_json JSON NULL AFTER reviewed_by_type;

UPDATE point_savings ps
JOIN admin_accounts aa
  ON aa.id = ps.reviewed_by
  AND aa.store_id = ps.store_id
SET ps.reviewed_by_type = aa.role,
    ps.reviewer_snapshot_json = JSON_OBJECT(
      'type', aa.role,
      'id', aa.id,
      'role', aa.role,
      'username', aa.username,
      'displayName', aa.display_name,
      'source', 'backfill_current_state'
    )
WHERE ps.reviewed_by IS NOT NULL
  AND ps.reviewer_snapshot_json IS NULL
  AND aa.role IN ('store_admin', 'cashier');

UPDATE point_savings ps
JOIN staff_accounts sa
  ON sa.member_id = ps.reviewed_by
  AND sa.store_id = ps.store_id
JOIN members m ON m.id = sa.member_id
SET ps.reviewed_by_type = 'staff',
    ps.reviewer_snapshot_json = JSON_OBJECT(
      'type', 'staff',
      'id', m.id,
      'staffName', sa.name,
      'nickname', m.nickname,
      'phone', COALESCE(m.phone, ''),
      'avatarUrl', COALESCE(m.avatar_url, ''),
      'source', 'backfill_current_state'
    )
WHERE ps.reviewed_by IS NOT NULL
  AND ps.reviewer_snapshot_json IS NULL;

-- +goose Down
ALTER TABLE point_savings
  DROP COLUMN reviewer_snapshot_json,
  DROP COLUMN reviewed_by_type;
