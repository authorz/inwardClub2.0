-- +goose Up
CREATE TABLE migration_00090_coupon_source_backups (
  entitlement_id BIGINT UNSIGNED NOT NULL,
  previous_granted_by_id BIGINT UNSIGNED NULL,
  previous_updated_at DATETIME NOT NULL,
  PRIMARY KEY (entitlement_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO migration_00090_coupon_source_backups
  (entitlement_id, previous_granted_by_id, previous_updated_at)
SELECT e.id, e.granted_by_id, e.updated_at
FROM coupon_entitlements e
JOIN food_order_items source_line
  ON source_line.id = CAST(
    SUBSTRING_INDEX(SUBSTRING_INDEX(e.idem_key, ':', 3), ':', -1) AS UNSIGNED
  )
  AND source_line.coupon_template_id_snapshot = e.coupon_template_id
WHERE e.granted_by_type = 'purchase'
  AND e.granted_by_id IS NULL
  AND e.idem_key REGEXP '^food_coupon:[0-9]+:[0-9]+:[0-9]+$';

UPDATE coupon_entitlements e
JOIN migration_00090_coupon_source_backups backup
  ON backup.entitlement_id = e.id
JOIN food_order_items source_line
  ON source_line.id = CAST(
    SUBSTRING_INDEX(SUBSTRING_INDEX(e.idem_key, ':', 3), ':', -1) AS UNSIGNED
  )
SET e.granted_by_id = source_line.id,
    e.updated_at = UTC_TIMESTAMP();

-- +goose Down
UPDATE coupon_entitlements e
JOIN migration_00090_coupon_source_backups backup
  ON backup.entitlement_id = e.id
SET e.granted_by_id = backup.previous_granted_by_id,
    e.updated_at = backup.previous_updated_at;

DROP TABLE migration_00090_coupon_source_backups;
