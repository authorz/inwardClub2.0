-- +goose Up
-- Move the timed reservation/waitlist low-spend reward from one headquarters
-- switch to an independently managed rule for every existing store.
INSERT INTO store_rules (store_id, rule_key, config_json, enabled, created_at, updated_at)
SELECT
  s.id,
  'timed_low_spend_reward',
  JSON_OBJECT(
    'reservationCutoff', COALESCE(
      (SELECT setting_value FROM system_settings WHERE setting_key = 'timed_low_spend_reservation_cutoff'),
      '20:00'
    ),
    'consumptionCutoff', COALESCE(
      (SELECT setting_value FROM system_settings WHERE setting_key = 'timed_low_spend_consumption_cutoff'),
      '20:30'
    ),
    'minimumAmountCent', CAST(COALESCE(
      (SELECT setting_value FROM system_settings WHERE setting_key = 'timed_low_spend_minimum_amount'),
      '88'
    ) AS UNSIGNED) * 100,
    'rewardPoints', CAST(COALESCE(
      (SELECT setting_value FROM system_settings WHERE setting_key = 'timed_low_spend_reward_points'),
      '2000'
    ) AS UNSIGNED)
  ),
  CASE LOWER(COALESCE(
    (SELECT setting_value FROM system_settings WHERE setting_key = 'timed_low_spend_reward_enabled'),
    'true'
  )) WHEN 'true' THEN 1 ELSE 0 END,
  NOW(),
  NOW()
FROM stores s
ON DUPLICATE KEY UPDATE rule_key = VALUES(rule_key);

DELETE FROM system_settings
WHERE setting_key IN (
  'timed_low_spend_reward_enabled',
  'timed_low_spend_reservation_cutoff',
  'timed_low_spend_consumption_cutoff',
  'timed_low_spend_minimum_amount',
  'timed_low_spend_reward_points'
);

-- +goose Down
INSERT INTO system_settings (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT 'timed_low_spend_reward_enabled', IF(sr.enabled = 1, 'true', 'false'), NULL, NOW(), NOW()
FROM store_rules sr WHERE sr.rule_key = 'timed_low_spend_reward' ORDER BY sr.store_id LIMIT 1
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value), updated_at = VALUES(updated_at);

INSERT INTO system_settings (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT 'timed_low_spend_reservation_cutoff', JSON_UNQUOTE(JSON_EXTRACT(sr.config_json, '$.reservationCutoff')), NULL, NOW(), NOW()
FROM store_rules sr WHERE sr.rule_key = 'timed_low_spend_reward' ORDER BY sr.store_id LIMIT 1
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value), updated_at = VALUES(updated_at);

INSERT INTO system_settings (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT 'timed_low_spend_consumption_cutoff', JSON_UNQUOTE(JSON_EXTRACT(sr.config_json, '$.consumptionCutoff')), NULL, NOW(), NOW()
FROM store_rules sr WHERE sr.rule_key = 'timed_low_spend_reward' ORDER BY sr.store_id LIMIT 1
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value), updated_at = VALUES(updated_at);

INSERT INTO system_settings (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT 'timed_low_spend_minimum_amount', CAST(JSON_EXTRACT(sr.config_json, '$.minimumAmountCent') / 100 AS CHAR), NULL, NOW(), NOW()
FROM store_rules sr WHERE sr.rule_key = 'timed_low_spend_reward' ORDER BY sr.store_id LIMIT 1
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value), updated_at = VALUES(updated_at);

INSERT INTO system_settings (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT 'timed_low_spend_reward_points', JSON_UNQUOTE(JSON_EXTRACT(sr.config_json, '$.rewardPoints')), NULL, NOW(), NOW()
FROM store_rules sr WHERE sr.rule_key = 'timed_low_spend_reward' ORDER BY sr.store_id LIMIT 1
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value), updated_at = VALUES(updated_at);

DELETE FROM store_rules WHERE rule_key = 'timed_low_spend_reward';
