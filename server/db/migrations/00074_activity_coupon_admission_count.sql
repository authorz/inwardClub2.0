-- +goose Up
-- Activity ticket tiers and event-ticket coupons must agree on the number of
-- people admitted. Existing tiers/templates/entitlements remain single-person.

ALTER TABLE activity_ticket_types
  ADD COLUMN admission_count INT NOT NULL DEFAULT 1 AFTER name;

ALTER TABLE coupon_templates
  ADD COLUMN admission_count INT NOT NULL DEFAULT 1 AFTER coupon_type;

ALTER TABLE coupon_entitlements
  ADD COLUMN admission_count INT NOT NULL DEFAULT 1 AFTER coupon_template_id;

UPDATE coupon_entitlements e
JOIN coupon_templates t ON t.id = e.coupon_template_id
SET e.admission_count = t.admission_count;

-- +goose Down
ALTER TABLE coupon_entitlements
  DROP COLUMN admission_count;

ALTER TABLE coupon_templates
  DROP COLUMN admission_count;

ALTER TABLE activity_ticket_types
  DROP COLUMN admission_count;
