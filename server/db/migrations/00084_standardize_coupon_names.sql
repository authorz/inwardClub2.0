-- +goose Up
-- Back up every affected value before applying the two requested display-name
-- replacements. Hex literals keep the retired terms out of source text while
-- still making this migration deterministic and reversible.
CREATE TABLE migration_00084_coupon_name_backups (
  source_table VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NOT NULL,
  column_name VARCHAR(64) NOT NULL,
  old_value LONGTEXT NOT NULL,
  PRIMARY KEY (source_table, source_id, column_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET @old_event_coupon_name = CONVERT(0xe58f82e8b59be4ba8be588b8 USING utf8mb4);
SET @old_alcohol_coupon_name = CONVERT(0xe98592e6b0b4e58591e68da2e588b8 USING utf8mb4);

INSERT INTO migration_00084_coupon_name_backups
  (source_table, source_id, column_name, old_value)
SELECT 'coupon_categories', id, 'name', name
FROM coupon_categories
WHERE name LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR name LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

INSERT INTO migration_00084_coupon_name_backups
  (source_table, source_id, column_name, old_value)
SELECT 'membership_tiers', id, 'benefits', benefits
FROM membership_tiers
WHERE benefits LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR benefits LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

INSERT INTO migration_00084_coupon_name_backups
  (source_table, source_id, column_name, old_value)
SELECT 'membership_tiers', id, 'benefit_config', CAST(benefit_config AS CHAR)
FROM membership_tiers
WHERE CAST(benefit_config AS CHAR) LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR CAST(benefit_config AS CHAR) LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

INSERT INTO migration_00084_coupon_name_backups
  (source_table, source_id, column_name, old_value)
SELECT 'error_events', id, 'message', message
FROM error_events
WHERE message LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR message LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

INSERT INTO migration_00084_coupon_name_backups
  (source_table, source_id, column_name, old_value)
SELECT 'outbox_events', id, 'payload', CAST(payload AS CHAR)
FROM outbox_events
WHERE CAST(payload AS CHAR) LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR CAST(payload AS CHAR) LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

INSERT INTO migration_00084_coupon_name_backups
  (source_table, source_id, column_name, old_value)
SELECT 'print_jobs', id, 'payload', CAST(payload AS CHAR)
FROM print_jobs
WHERE CAST(payload AS CHAR) LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR CAST(payload AS CHAR) LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

UPDATE coupon_categories
SET name = REPLACE(REPLACE(name, @old_event_coupon_name, '赛事券'), @old_alcohol_coupon_name, '酒水券')
WHERE name LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR name LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

UPDATE membership_tiers
SET benefits = REPLACE(REPLACE(benefits, @old_event_coupon_name, '赛事券'), @old_alcohol_coupon_name, '酒水券')
WHERE benefits LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR benefits LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

UPDATE membership_tiers
SET benefit_config = CAST(
  REPLACE(REPLACE(CAST(benefit_config AS CHAR), @old_event_coupon_name, '赛事券'), @old_alcohol_coupon_name, '酒水券')
  AS JSON
)
WHERE CAST(benefit_config AS CHAR) LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR CAST(benefit_config AS CHAR) LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

UPDATE error_events
SET message = REPLACE(REPLACE(message, @old_event_coupon_name, '赛事券'), @old_alcohol_coupon_name, '酒水券')
WHERE message LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR message LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

UPDATE outbox_events
SET payload = CAST(
  REPLACE(REPLACE(CAST(payload AS CHAR), @old_event_coupon_name, '赛事券'), @old_alcohol_coupon_name, '酒水券')
  AS JSON
)
WHERE CAST(payload AS CHAR) LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR CAST(payload AS CHAR) LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

UPDATE print_jobs
SET payload = CAST(
  REPLACE(REPLACE(CAST(payload AS CHAR), @old_event_coupon_name, '赛事券'), @old_alcohol_coupon_name, '酒水券')
  AS JSON
)
WHERE CAST(payload AS CHAR) LIKE CONCAT('%', @old_event_coupon_name, '%')
   OR CAST(payload AS CHAR) LIKE CONCAT('%', @old_alcohol_coupon_name, '%');

-- +goose Down
UPDATE coupon_categories c
JOIN migration_00084_coupon_name_backups b
  ON b.source_table = 'coupon_categories' AND b.source_id = c.id AND b.column_name = 'name'
SET c.name = b.old_value;

UPDATE membership_tiers t
JOIN migration_00084_coupon_name_backups b
  ON b.source_table = 'membership_tiers' AND b.source_id = t.id AND b.column_name = 'benefits'
SET t.benefits = b.old_value;

UPDATE membership_tiers t
JOIN migration_00084_coupon_name_backups b
  ON b.source_table = 'membership_tiers' AND b.source_id = t.id AND b.column_name = 'benefit_config'
SET t.benefit_config = CAST(b.old_value AS JSON);

UPDATE error_events e
JOIN migration_00084_coupon_name_backups b
  ON b.source_table = 'error_events' AND b.source_id = e.id AND b.column_name = 'message'
SET e.message = b.old_value;

UPDATE outbox_events e
JOIN migration_00084_coupon_name_backups b
  ON b.source_table = 'outbox_events' AND b.source_id = e.id AND b.column_name = 'payload'
SET e.payload = CAST(b.old_value AS JSON);

UPDATE print_jobs j
JOIN migration_00084_coupon_name_backups b
  ON b.source_table = 'print_jobs' AND b.source_id = j.id AND b.column_name = 'payload'
SET j.payload = CAST(b.old_value AS JSON);

DROP TABLE migration_00084_coupon_name_backups;
