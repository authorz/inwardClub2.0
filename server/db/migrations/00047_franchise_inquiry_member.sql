-- +goose Up
-- MySQL DDL auto-commits. The guards keep this migration retryable if an
-- earlier attempt added the column before a later statement failed.
SET @member_column_exists = (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'franchise_inquiries'
    AND column_name = 'member_id'
);
SET @add_member_column_sql = IF(
  @member_column_exists = 0,
  'ALTER TABLE franchise_inquiries ADD COLUMN member_id BIGINT UNSIGNED NULL AFTER id',
  'SELECT 1'
);
PREPARE add_member_column_stmt FROM @add_member_column_sql;
EXECUTE add_member_column_stmt;
DEALLOCATE PREPARE add_member_column_stmt;

SET @member_index_exists = (
  SELECT COUNT(*) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'franchise_inquiries'
    AND index_name = 'idx_franchise_inquiries_member_created'
);
SET @add_member_index_sql = IF(
  @member_index_exists = 0,
  'ALTER TABLE franchise_inquiries ADD KEY idx_franchise_inquiries_member_created (member_id, created_at)',
  'SELECT 1'
);
PREPARE add_member_index_stmt FROM @add_member_index_sql;
EXECUTE add_member_index_stmt;
DEALLOCATE PREPARE add_member_index_stmt;

-- Historical inquiries predate member attribution. Only backfill an exact
-- phone match; unmatched inquiries remain anonymous instead of guessing.
UPDATE franchise_inquiries fi
JOIN members m ON m.phone COLLATE utf8mb4_unicode_ci = fi.phone COLLATE utf8mb4_unicode_ci
SET fi.member_id = m.id
WHERE fi.member_id IS NULL
  AND fi.phone <> '';

-- +goose Down
ALTER TABLE franchise_inquiries
  DROP KEY idx_franchise_inquiries_member_created,
  DROP COLUMN member_id;
