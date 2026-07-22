-- +goose Up
-- Member-facing read-only catalogues: configurable VIP tiers and quick-recharge
-- coin packages. Both are surfaced by the mini-program member self-service pages
-- and are populated with minimal seed data so the pages have readable content.

CREATE TABLE membership_tiers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  level INT NOT NULL DEFAULT 0,
  threshold BIGINT NOT NULL DEFAULT 0, -- points required to reach this tier
  benefits VARCHAR(1024) NOT NULL DEFAULT '',
  icon_asset_id BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active', -- active/disabled
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_membership_tiers_status (status),
  KEY idx_membership_tiers_level (level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE recharge_products (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL, -- paid coins/points
  bonus_amount BIGINT NOT NULL DEFAULT 0, -- extra coins/points granted
  asset_type VARCHAR(32) NOT NULL DEFAULT 'coin', -- coin/point
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active', -- active/disabled
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_recharge_products_sort (status, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO membership_tiers (name, level, threshold, benefits, status, created_at, updated_at) VALUES
  ('普通会员', 1, 0,     '基础权益', 'active', NOW(), NOW()),
  ('白银会员', 2, 1000,  '9.5 折优惠', 'active', NOW(), NOW()),
  ('黄金会员', 3, 5000,  '9 折优惠 + 生日礼', 'active', NOW(), NOW());

INSERT INTO recharge_products (name, amount, bonus_amount, asset_type, sort_order, status, created_at, updated_at) VALUES
  ('充值 10 元',  1000,  0,    'coin', 1, 'active', NOW(), NOW()),
  ('充值 50 元',  5000,  500,  'coin', 2, 'active', NOW(), NOW()),
  ('充值 100 元', 10000, 1500, 'coin', 3, 'active', NOW(), NOW());

-- +goose Down
DROP TABLE IF EXISTS recharge_products;
DROP TABLE IF EXISTS membership_tiers;
