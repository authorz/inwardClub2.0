-- +goose Up
-- Gifted-coupon limits are headquarters-level rules, not properties of coupon
-- categories. A missing rule or a NULL daily_limit means unrestricted use.
-- Purchased coupon products always bypass these rules in application code.
CREATE TABLE gift_coupon_usage_rules (
  coupon_category_id BIGINT UNSIGNED NOT NULL,
  daily_limit INT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (coupon_category_id),
  CONSTRAINT fk_gift_coupon_usage_rule_category
    FOREIGN KEY (coupon_category_id) REFERENCES coupon_categories(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Migration 82 added a default limit of one to every category. That implicit
-- value was not a valid global configuration, so no rules are copied here:
-- after this migration every gifted coupon is unrestricted until HQ explicitly
-- adds a rule.
ALTER TABLE coupon_categories
  DROP COLUMN gift_daily_usage_limit;

-- +goose Down
ALTER TABLE coupon_categories
  ADD COLUMN gift_daily_usage_limit INT UNSIGNED NOT NULL DEFAULT 1 AFTER status;

UPDATE coupon_categories c
JOIN gift_coupon_usage_rules r ON r.coupon_category_id = c.id
SET c.gift_daily_usage_limit = COALESCE(r.daily_limit, 0);

DROP TABLE gift_coupon_usage_rules;
