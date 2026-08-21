-- +goose Up
-- Product coupon redemption uses the normal food-order payment spine. The
-- entitlement link keeps the consumed asset traceable and prevents reuse.

ALTER TABLE food_orders
  ADD COLUMN coupon_entitlement_id BIGINT UNSIGNED NULL AFTER member_id,
  ADD UNIQUE KEY uq_food_order_coupon (coupon_entitlement_id);

-- +goose Down
ALTER TABLE food_orders
  DROP INDEX uq_food_order_coupon,
  DROP COLUMN coupon_entitlement_id;
