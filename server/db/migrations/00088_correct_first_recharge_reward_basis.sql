-- +goose Up
CREATE TABLE migration_00088_first_recharge_reward_backups (
  reward_ledger_id BIGINT UNSIGNED NOT NULL,
  account_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  payment_order_id BIGINT UNSIGNED NOT NULL,
  business_order_id BIGINT UNSIGNED NOT NULL,
  actual_bonus BIGINT NOT NULL,
  expected_bonus BIGINT NOT NULL,
  correction_delta BIGINT NOT NULL,
  account_balance_before BIGINT NOT NULL,
  PRIMARY KEY (reward_ledger_id),
  UNIQUE KEY uq_migration_00088_payment_order (payment_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO migration_00088_first_recharge_reward_backups
  (reward_ledger_id, account_id, member_id, payment_order_id,
   business_order_id, actual_bonus, expected_bonus, correction_delta,
   account_balance_before)
SELECT reward.id, reward.account_id, reward.member_id, po.id,
       bo.id, reward.amount, COALESCE(base.amount, 0),
       COALESCE(base.amount, 0) - reward.amount, wa.available_amount
FROM wallet_ledger_entries reward
JOIN payment_orders po
  ON reward.idem_key = CONCAT('first_recharge_reward:', po.id)
JOIN business_orders bo ON bo.id = po.business_order_id
JOIN wallet_accounts wa ON wa.id = reward.account_id
LEFT JOIN wallet_ledger_entries base
  ON base.idem_key = CONCAT('recharge_points:', po.id)
WHERE reward.asset_type = 'points'
  AND reward.direction = 'credit'
  AND reward.reason = 'first_recharge_reward'
  AND COALESCE(base.amount, 0) <> reward.amount;

INSERT INTO wallet_ledger_entries
  (account_id, member_id, asset_type, direction, amount, balance_after,
   reason, source_type, source_id, idem_key, created_at)
SELECT account_id, member_id, 'points',
       CASE WHEN correction_delta > 0 THEN 'credit' ELSE 'debit' END,
       ABS(correction_delta), account_balance_before + correction_delta,
       'first_recharge_reward_correction', 'first_recharge_reward',
       business_order_id,
       CONCAT('first_recharge_reward_correction:v88:', payment_order_id),
       UTC_TIMESTAMP()
FROM migration_00088_first_recharge_reward_backups
WHERE correction_delta <> 0;

UPDATE wallet_accounts account
JOIN (
  SELECT account_id, SUM(correction_delta) AS correction_delta
  FROM migration_00088_first_recharge_reward_backups
  GROUP BY account_id
) correction ON correction.account_id = account.id
SET account.available_amount = account.available_amount + correction.correction_delta,
    account.version = account.version + 1,
    account.updated_at = UTC_TIMESTAMP();

CREATE TABLE migration_00088_system_setting_backups (
  setting_key VARCHAR(100) NOT NULL,
  setting_value TEXT NOT NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO migration_00088_system_setting_backups
  (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT setting_key, setting_value, updated_by, created_at, updated_at
FROM system_settings
WHERE setting_key = 'recharge_notice'
  AND setting_value = '新用户全平台首次成功充值低于 1000 元，按到账金币数赠送 2 倍积分。';

UPDATE system_settings
SET setting_value = '新用户全平台首次成功充值低于 1000 元，按对应充值档位配置获得双倍赠送积分。',
    updated_at = UTC_TIMESTAMP()
WHERE setting_key = 'recharge_notice'
  AND setting_value = '新用户全平台首次成功充值低于 1000 元，按到账金币数赠送 2 倍积分。';

-- +goose Down
UPDATE system_settings current_setting
JOIN migration_00088_system_setting_backups backup
  ON backup.setting_key = current_setting.setting_key
SET current_setting.setting_value = backup.setting_value,
    current_setting.updated_by = backup.updated_by,
    current_setting.created_at = backup.created_at,
    current_setting.updated_at = backup.updated_at
WHERE current_setting.setting_value = '新用户全平台首次成功充值低于 1000 元，按对应充值档位配置获得双倍赠送积分。';

UPDATE wallet_accounts account
JOIN (
  SELECT account_id, SUM(correction_delta) AS correction_delta
  FROM migration_00088_first_recharge_reward_backups
  GROUP BY account_id
) correction ON correction.account_id = account.id
SET account.available_amount = account.available_amount - correction.correction_delta,
    account.version = account.version + 1,
    account.updated_at = UTC_TIMESTAMP();

DELETE ledger
FROM wallet_ledger_entries ledger
JOIN migration_00088_first_recharge_reward_backups backup
  ON ledger.idem_key = CONCAT('first_recharge_reward_correction:v88:', backup.payment_order_id);

DROP TABLE migration_00088_system_setting_backups;
DROP TABLE migration_00088_first_recharge_reward_backups;
