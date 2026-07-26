-- +goose Up
CREATE TABLE system_settings (
  setting_key VARCHAR(100) NOT NULL,
  setting_value TEXT NOT NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS system_settings;
