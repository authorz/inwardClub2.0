-- +goose Up
-- event_ticket keeps the established VIP event-benefit meaning. Activity ticket
-- exchange moves to a separate admission_ticket type.

UPDATE coupon_categories
SET name = '赛事券', updated_at = UTC_TIMESTAMP()
WHERE business_type = 'event_ticket' AND name = '赛事门票券';

INSERT INTO coupon_categories (name, business_type, sort_order, status, created_at, updated_at)
SELECT '门票券', 'admission_ticket', 15, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_categories WHERE business_type = 'admission_ticket'
);

-- +goose Down
DELETE FROM coupon_categories
WHERE business_type = 'admission_ticket';

UPDATE coupon_categories
SET name = '赛事门票券', updated_at = UTC_TIMESTAMP()
WHERE business_type = 'event_ticket' AND name = '赛事券';
