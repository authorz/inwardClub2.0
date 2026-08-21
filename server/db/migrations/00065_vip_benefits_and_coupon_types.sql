-- +goose Up
-- VIP benefits are structured so the admin console can edit points, coupon
-- grants and display-only physical benefits independently. Coupon templates
-- are category definitions only: one entitlement exchanges one product/ticket.

ALTER TABLE membership_tiers
  ADD COLUMN benefit_config JSON NULL AFTER benefits;

ALTER TABLE activity_orders
  ADD COLUMN coupon_entitlement_id BIGINT UNSIGNED NULL AFTER member_id,
  ADD UNIQUE KEY uq_activity_order_coupon (coupon_entitlement_id);

UPDATE coupon_templates
SET coupon_type = CASE
      WHEN name LIKE '%赛事%' OR name LIKE '%门票%' THEN 'event_ticket'
      WHEN name LIKE '%小吃%' THEN 'snack'
      WHEN name LIKE '%酒%' OR name LIKE '%啤%' THEN 'alcohol'
      WHEN name LIKE '%餐食%' OR name LIKE '%餐券%' THEN 'meal'
      ELSE 'beverage'
    END,
    value_cent = 0,
    points_price = 0,
    stock_quantity = 0,
    per_member_limit = 0,
    updated_at = NOW();

UPDATE coupon_entitlements
SET expires_at = DATE_ADD(created_at, INTERVAL 30 DAY),
    updated_at = NOW()
WHERE expires_at IS NULL OR expires_at <> DATE_ADD(created_at, INTERVAL 30 DAY);

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '赛事门票券', '一张券兑换一张活动门票', 'event_ticket', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'event_ticket'
);

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '小吃券', '一张券兑换一份已关联小吃', 'snack', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'snack'
);

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '酒水券', '一张券兑换一份已关联酒水', 'alcohol', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'alcohol'
);

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '饮料券', '一张券兑换一份已关联饮料', 'beverage', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'beverage'
);

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '餐食券', '一张券兑换一份已关联餐食', 'meal', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'meal'
);

UPDATE membership_tiers
SET benefits = '每日低消酒券赠送1000积分',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 1000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE level = 2;

UPDATE membership_tiers
SET benefits = '每日低消酒券赠送1500积分；每日到店无酒精饮品或啤酒一杯',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 1500, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(JSON_OBJECT('couponType', 'alcohol', 'quantity', 1, 'period', 'daily', 'trigger', 'visit')),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE level = 3;

UPDATE membership_tiers
SET benefits = '每日首次下单酒券赠送3000积分；每日到店任意酒水一杯；小吃一份',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 3000, 'period', 'daily', 'trigger', 'first_order')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 1, 'period', 'daily', 'trigger', 'visit')
      ),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE level = 4;

UPDATE membership_tiers
SET benefits = '每日首次低消酒券赠送4000积分；每日到店任意酒水一杯、小吃一份、餐食一份；每周赠送工作日期间赛事票1张；每月赠送周赛门票1张',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 4000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'meal', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 1, 'period', 'weekly', 'trigger', 'weekday_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 1, 'period', 'monthly', 'trigger', 'weekly_event')
      ),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE level = 5;

UPDATE membership_tiers
SET benefits = '当天首次低消赠送6000积分；每日到店任意酒水2杯、小吃2份、餐食1份；每周赠送工作日期间赛事票3张；每月赠送周赛门票2张；每月赠送礼品区任选1份',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 6000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'meal', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 3, 'period', 'weekly', 'trigger', 'weekday_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 2, 'period', 'monthly', 'trigger', 'weekly_event')
      ),
      'descriptions', JSON_ARRAY('每月赠送礼品区任选1份')
    ),
    updated_at = NOW()
WHERE level = 6;

UPDATE membership_tiers
SET benefits = '当天首次低消酒券赠送8000积分；每日到店任意酒水2杯、小吃2份、餐食2份；每周赠送工作日期间赛事门票5张；每月赠送周赛门票3张、月赛门票1张；每周赠送店内礼品区任选1份；专属定制酒杯',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 8000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'meal', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 5, 'period', 'weekly', 'trigger', 'weekday_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 3, 'period', 'monthly', 'trigger', 'weekly_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 1, 'period', 'monthly', 'trigger', 'monthly_event')
      ),
      'descriptions', JSON_ARRAY('每周赠送店内礼品区任选1份', '专属定制酒杯')
    ),
    updated_at = NOW()
WHERE level = 7;

UPDATE membership_tiers
SET benefits = '当天首次低消酒券赠送10000积分；每日到店任意酒水3杯、小吃5份、餐食2份；每周赠送工作日期间赛事门票5张；每月赠送周赛门票4张、月赛门票2张；每天赠送店内礼品区任选1份；专属定制酒杯；免费布置生日宴会并赠酒水套餐（持有效证明）',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 10000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 3, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 5, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'meal', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 5, 'period', 'weekly', 'trigger', 'weekday_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 4, 'period', 'monthly', 'trigger', 'weekly_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 2, 'period', 'monthly', 'trigger', 'monthly_event')
      ),
      'descriptions', JSON_ARRAY('每天赠送店内礼品区任选1份', '专属定制酒杯', '免费布置生日宴会并赠酒水套餐（持有效证明）')
    ),
    updated_at = NOW()
WHERE level = 8;

-- +goose Down
ALTER TABLE activity_orders
  DROP INDEX uq_activity_order_coupon,
  DROP COLUMN coupon_entitlement_id;
ALTER TABLE membership_tiers DROP COLUMN benefit_config;
