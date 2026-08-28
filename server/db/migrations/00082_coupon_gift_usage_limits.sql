-- +goose Up
-- Headquarters configures how many gifted coupons of each category one member
-- may consume per Beijing calendar day. Zero means unlimited. Purchased coupon
-- entitlements bypass this limit.

ALTER TABLE coupon_categories
  ADD COLUMN gift_daily_usage_limit INT UNSIGNED NOT NULL DEFAULT 1 AFTER status;

CREATE TABLE gift_coupon_daily_usages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  member_id BIGINT UNSIGNED NOT NULL,
  category_id BIGINT UNSIGNED NOT NULL,
  usage_date DATE NOT NULL,
  slot_number INT UNSIGNED NOT NULL,
  entitlement_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_gift_coupon_usage_slot (member_id, category_id, usage_date, slot_number),
  UNIQUE KEY uq_gift_coupon_usage_entitlement (entitlement_id),
  KEY idx_gift_coupon_usage_member_day (member_id, usage_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO gift_coupon_daily_usages
  (member_id, category_id, usage_date, slot_number, entitlement_id, created_at)
SELECT u.member_id, t.category_id, u.usage_date, 1, u.entitlement_id, u.created_at
FROM vip_coupon_daily_usages u
JOIN coupon_entitlements e ON e.id = u.entitlement_id
JOIN coupon_templates t ON t.id = e.coupon_template_id;

DROP TABLE vip_coupon_daily_usages;

-- +goose Down
CREATE TABLE vip_coupon_daily_usages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  member_id BIGINT UNSIGNED NOT NULL,
  usage_date DATE NOT NULL,
  entitlement_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_vip_coupon_usage_member_day (member_id, usage_date),
  UNIQUE KEY uq_vip_coupon_usage_entitlement (entitlement_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO vip_coupon_daily_usages
  (member_id, usage_date, entitlement_id, created_at)
SELECT member_id, usage_date, entitlement_id, created_at
FROM gift_coupon_daily_usages
ORDER BY created_at, entitlement_id;

DROP TABLE gift_coupon_daily_usages;

ALTER TABLE coupon_categories
  DROP COLUMN gift_daily_usage_limit;
