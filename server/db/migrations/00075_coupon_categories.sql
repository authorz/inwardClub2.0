-- +goose Up
-- Coupon display categories are managed by the HQ console. Each category is
-- bound to one fixed fulfillment behavior so renaming/reordering never changes
-- redemption semantics for existing templates and entitlements.

CREATE TABLE coupon_categories (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  business_type VARCHAR(32) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_coupon_categories_name (name),
  UNIQUE KEY uq_coupon_categories_business_type (business_type),
  KEY idx_coupon_categories_status_sort (status, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO coupon_categories (name, business_type, sort_order, status, created_at, updated_at) VALUES
  ('赛事门票券', 'event_ticket', 10, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
  ('小吃券', 'snack', 20, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
  ('酒水券', 'alcohol', 30, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
  ('饮料券', 'beverage', 40, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
  ('饮品或啤酒券', 'drink', 50, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
  ('餐食券', 'meal', 60, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
  ('礼品券', 'gift', 70, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP());

ALTER TABLE coupon_templates
  ADD COLUMN category_id BIGINT UNSIGNED NULL AFTER coupon_type;

UPDATE coupon_templates t
JOIN coupon_categories c ON c.business_type = t.coupon_type
SET t.category_id = c.id;

ALTER TABLE coupon_templates
  MODIFY COLUMN category_id BIGINT UNSIGNED NOT NULL,
  ADD KEY idx_coupon_templates_category (category_id);

-- +goose Down
ALTER TABLE coupon_templates
  DROP KEY idx_coupon_templates_category,
  DROP COLUMN category_id;

DROP TABLE IF EXISTS coupon_categories;
