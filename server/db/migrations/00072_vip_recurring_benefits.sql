-- +goose Up
-- Materialise the confirmed VIP2-VIP8 recurring benefit rules. Period/trigger
-- fields are executed by internal/modules/vipbenefit; event-ticket expiry is
-- derived from the natural business week/month at issue time.

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '饮品或啤酒券', '一张券兑换一份已关联的无酒精饮品或啤酒', 'drink', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'drink'
);

INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, value_cent, points_price,
   stock_quantity, issued_quantity, validity_rule, applicable_scope,
   per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, '礼品区任选券', '一张券兑换一份已关联的礼品区商品', 'gift', 0, 0,
       0, 0, JSON_OBJECT('days', 30), JSON_OBJECT(), 0, 'published', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates WHERE scope_type = 'global' AND coupon_type = 'gift'
);

-- Original confirmed growth thresholds from docs/InwardCLub2.0.xlsx.
UPDATE membership_tiers
SET threshold = CASE level
      WHEN 1 THEN 0
      WHEN 2 THEN 888
      WHEN 3 THEN 1888
      ELSE threshold
    END,
    updated_at = NOW()
WHERE status = 'active' AND level BETWEEN 1 AND 3;

INSERT INTO membership_tiers
  (name, level, threshold, benefits, benefit_config, icon_asset_id, status, created_at, updated_at)
SELECT 'VIP4 黄金会员', 4, 2888, '', JSON_OBJECT(), NULL, 'active', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM membership_tiers WHERE level = 4 AND status = 'active');

INSERT INTO membership_tiers
  (name, level, threshold, benefits, benefit_config, icon_asset_id, status, created_at, updated_at)
SELECT 'VIP5 铂金会员', 5, 4888, '', JSON_OBJECT(), NULL, 'active', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM membership_tiers WHERE level = 5 AND status = 'active');

INSERT INTO membership_tiers
  (name, level, threshold, benefits, benefit_config, icon_asset_id, status, created_at, updated_at)
SELECT 'VIP6 钻石会员', 6, 6888, '', JSON_OBJECT(), NULL, 'active', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM membership_tiers WHERE level = 6 AND status = 'active');

INSERT INTO membership_tiers
  (name, level, threshold, benefits, benefit_config, icon_asset_id, status, created_at, updated_at)
SELECT 'VIP7 星耀会员', 7, 8888, '', JSON_OBJECT(), NULL, 'active', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM membership_tiers WHERE level = 7 AND status = 'active');

INSERT INTO membership_tiers
  (name, level, threshold, benefits, benefit_config, icon_asset_id, status, created_at, updated_at)
SELECT 'VIP8 大师会员', 8, 12888, '', JSON_OBJECT(), NULL, 'active', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM membership_tiers WHERE level = 8 AND status = 'active');

UPDATE membership_tiers
SET benefits = '每日低消酒券赠送1000积分',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 1000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE status = 'active' AND level = 2;

UPDATE membership_tiers
SET benefits = '每日低消酒券赠送1500积分；每日到店无酒精饮品或啤酒一杯',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 1500, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(JSON_OBJECT('couponType', 'drink', 'quantity', 1, 'period', 'daily', 'trigger', 'visit')),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE status = 'active' AND level = 3;

UPDATE membership_tiers
SET benefits = '每日首次下单酒券赠送3000积分；每日到店任意酒水一杯、小吃一份',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 3000, 'period', 'daily', 'trigger', 'first_order')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 1, 'period', 'daily', 'trigger', 'visit')
      ),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE status = 'active' AND level = 4;

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
WHERE status = 'active' AND level = 5;

UPDATE membership_tiers
SET benefits = '当天首次低消赠送6000积分；每日到店任意酒水2杯、小吃2份、餐食1份；每周赠送工作日期间赛事票3张；每月赠送周赛门票2张；每月赠送礼品区任选1份',
    benefit_config = JSON_OBJECT(
      'points', JSON_ARRAY(JSON_OBJECT('amount', 6000, 'period', 'daily', 'trigger', 'low_spend')),
      'coupons', JSON_ARRAY(
        JSON_OBJECT('couponType', 'alcohol', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'snack', 'quantity', 2, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'meal', 'quantity', 1, 'period', 'daily', 'trigger', 'visit'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 3, 'period', 'weekly', 'trigger', 'weekday_event'),
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 2, 'period', 'monthly', 'trigger', 'weekly_event'),
        JSON_OBJECT('couponType', 'gift', 'quantity', 1, 'period', 'monthly', 'trigger', 'period_start')
      ),
      'descriptions', JSON_ARRAY()
    ),
    updated_at = NOW()
WHERE status = 'active' AND level = 6;

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
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 1, 'period', 'monthly', 'trigger', 'monthly_event'),
        JSON_OBJECT('couponType', 'gift', 'quantity', 1, 'period', 'weekly', 'trigger', 'period_start')
      ),
      'descriptions', JSON_ARRAY('专属定制酒杯')
    ),
    updated_at = NOW()
WHERE status = 'active' AND level = 7;

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
        JSON_OBJECT('couponType', 'event_ticket', 'quantity', 2, 'period', 'monthly', 'trigger', 'monthly_event'),
        JSON_OBJECT('couponType', 'gift', 'quantity', 1, 'period', 'daily', 'trigger', 'period_start')
      ),
      'descriptions', JSON_ARRAY('专属定制酒杯', '免费布置生日宴会并赠酒水套餐（持有效证明）')
    ),
    updated_at = NOW()
WHERE status = 'active' AND level = 8;

-- Existing members may already have enough growth for one of the newly-created
-- tiers. Advance only upward; recurring grants are then picked up idempotently by
-- the daily worker sweep.
UPDATE members m
JOIN wallet_accounts wa ON wa.member_id = m.id AND wa.asset_type = 'growth_value'
JOIN membership_tiers target ON target.id = (
  SELECT t.id
  FROM membership_tiers t
  WHERE t.status = 'active' AND t.threshold <= wa.available_amount
  ORDER BY t.threshold DESC, t.level DESC, t.id ASC
  LIMIT 1
)
LEFT JOIN membership_tiers current_tier ON current_tier.id = m.current_tier_id
SET m.current_tier_id = target.id,
    m.updated_at = NOW()
WHERE m.current_tier_id IS NULL OR current_tier.level < target.level;

-- +goose Down
UPDATE membership_tiers
SET benefit_config = JSON_OBJECT('points', JSON_ARRAY(), 'coupons', JSON_ARRAY(), 'descriptions', JSON_ARRAY()),
    updated_at = NOW()
WHERE status = 'active' AND level BETWEEN 2 AND 8;

DELETE FROM coupon_templates
WHERE scope_type = 'global' AND coupon_type IN ('drink', 'gift') AND issued_quantity = 0;
