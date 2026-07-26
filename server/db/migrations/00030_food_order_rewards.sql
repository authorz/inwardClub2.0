-- +goose Up
-- Snapshot each catalog item's configured reward at order creation so later
-- catalog edits cannot change the entitlement of an already-created order.

ALTER TABLE food_order_items
  ADD COLUMN points_reward_snapshot BIGINT NOT NULL DEFAULT 0 AFTER pay_channels_snapshot;

ALTER TABLE food_orders
  ADD COLUMN points_earned BIGINT NOT NULL DEFAULT 0 AFTER total_amount_cent;

-- +goose Down
ALTER TABLE food_orders DROP COLUMN points_earned;
ALTER TABLE food_order_items DROP COLUMN points_reward_snapshot;
