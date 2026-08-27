-- +goose Up
-- The event/admission split renamed the category but left the legacy global
-- event template with the obsolete combined name.
UPDATE coupon_templates
SET name = '赛事券', updated_at = UTC_TIMESTAMP()
WHERE scope_type = 'global'
  AND store_id IS NULL
  AND coupon_type = 'event_ticket'
  AND name = '赛事门票券';

-- +goose Down
UPDATE coupon_templates
SET name = '赛事门票券', updated_at = UTC_TIMESTAMP()
WHERE scope_type = 'global'
  AND store_id IS NULL
  AND coupon_type = 'event_ticket'
  AND name = '赛事券';
