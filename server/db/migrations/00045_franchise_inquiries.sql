-- +goose Up
CREATE TABLE franchise_inquiries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  contact_name VARCHAR(100) NOT NULL,
  phone VARCHAR(32) NOT NULL,
  expected_region VARCHAR(255) NOT NULL,
  source VARCHAR(100) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_franchise_inquiries_phone (phone),
  KEY idx_franchise_inquiries_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO system_settings
  (setting_key, setting_value, updated_by, created_at, updated_at)
VALUES
  ('franchise_inquiry_sources', '["美团","抖音","小红书","店员","微信小程序"]', NULL, UTC_TIMESTAMP(3), UTC_TIMESTAMP(3))
ON DUPLICATE KEY UPDATE setting_key = VALUES(setting_key);

-- +goose Down
DELETE FROM system_settings WHERE setting_key = 'franchise_inquiry_sources';
DROP TABLE franchise_inquiries;
