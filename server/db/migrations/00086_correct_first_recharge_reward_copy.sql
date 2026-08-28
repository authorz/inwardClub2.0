-- +goose Up
CREATE TABLE migration_00086_system_setting_backups (
  setting_key VARCHAR(100) NOT NULL,
  setting_value TEXT NOT NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO migration_00086_system_setting_backups
  (setting_key, setting_value, updated_by, created_at, updated_at)
SELECT setting_key, setting_value, updated_by, created_at, updated_at
FROM system_settings
WHERE setting_key = 'recharge_notice'
  AND setting_value IN (
    '新用户首充积分赠送双倍',
    '新用户首充积分赠送双倍，充值一千及以上都赠送双倍积分，不与新用户首充赠送双倍同享。'
  );

UPDATE system_settings
SET setting_value = '新用户全平台首次成功充值低于 1000 元，按到账金币数赠送 2 倍积分。',
    updated_at = UTC_TIMESTAMP()
WHERE setting_key = 'recharge_notice'
  AND setting_value IN (
    '新用户首充积分赠送双倍',
    '新用户首充积分赠送双倍，充值一千及以上都赠送双倍积分，不与新用户首充赠送双倍同享。'
  );

-- +goose Down
UPDATE system_settings current_setting
JOIN migration_00086_system_setting_backups backup
  ON backup.setting_key = current_setting.setting_key
SET current_setting.setting_value = backup.setting_value,
    current_setting.updated_by = backup.updated_by,
    current_setting.created_at = backup.created_at,
    current_setting.updated_at = backup.updated_at
WHERE current_setting.setting_value = '新用户全平台首次成功充值低于 1000 元，按到账金币数赠送 2 倍积分。';

DROP TABLE migration_00086_system_setting_backups;
