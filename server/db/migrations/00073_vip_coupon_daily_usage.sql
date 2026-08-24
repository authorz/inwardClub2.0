-- +goose Up
-- VIP benefit coupons may be redeemed at most once per member and Beijing
-- calendar day. Purchased coupons never write this table and remain unlimited.
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

-- Preserve today's limit when this migration is applied after a VIP coupon has
-- already been consumed. Historical days that used more than one VIP coupon
-- retain the earliest known usage; INSERT IGNORE resolves the new unique key.
INSERT IGNORE INTO vip_coupon_daily_usages
  (member_id, usage_date, entitlement_id, created_at)
SELECT used.member_id,
       DATE(CONVERT_TZ(used.used_at, '+00:00', '+08:00')),
       used.entitlement_id,
       used.used_at
FROM (
  SELECT r.member_id, r.entitlement_id, r.created_at AS used_at
  FROM coupon_redemptions r
  JOIN coupon_entitlements e ON e.id = r.entitlement_id
  WHERE e.granted_reason = 'VIP等级福利' AND e.granted_by_type = 'system'
  UNION ALL
  SELECT fo.member_id, fo.coupon_entitlement_id, fo.created_at
  FROM food_orders fo
  JOIN coupon_entitlements e ON e.id = fo.coupon_entitlement_id
  WHERE fo.coupon_entitlement_id IS NOT NULL
    AND e.granted_reason = 'VIP等级福利' AND e.granted_by_type = 'system'
  UNION ALL
  SELECT ao.member_id, ao.coupon_entitlement_id, ao.created_at
  FROM activity_orders ao
  JOIN coupon_entitlements e ON e.id = ao.coupon_entitlement_id
  WHERE ao.coupon_entitlement_id IS NOT NULL
    AND e.granted_reason = 'VIP等级福利' AND e.granted_by_type = 'system'
) AS used
ORDER BY used.used_at, used.entitlement_id;

-- +goose Down
DROP TABLE IF EXISTS vip_coupon_daily_usages;
