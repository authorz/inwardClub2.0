-- +goose Up
-- Catalog categories distinguish ordinary products from coupon products. A
-- coupon product snapshots its granted template onto the order line so later
-- edits cannot change what a paid customer receives.

ALTER TABLE catalog_categories
  ADD COLUMN category_type VARCHAR(16) NOT NULL DEFAULT 'product' AFTER name,
  ADD KEY idx_catalog_categories_type (store_id, category_type, status);

ALTER TABLE catalog_items
  ADD COLUMN grant_coupon_template_id BIGINT UNSIGNED NULL AFTER coupon_template_ids,
  ADD KEY idx_catalog_items_grant_coupon (grant_coupon_template_id);

ALTER TABLE food_order_items
  ADD COLUMN coupon_template_id_snapshot BIGINT UNSIGNED NULL AFTER points_reward_snapshot,
  ADD KEY idx_food_order_items_coupon_snapshot (coupon_template_id_snapshot);

-- +goose Down
ALTER TABLE food_order_items
  DROP KEY idx_food_order_items_coupon_snapshot,
  DROP COLUMN coupon_template_id_snapshot;

ALTER TABLE catalog_items
  DROP KEY idx_catalog_items_grant_coupon,
  DROP COLUMN grant_coupon_template_id;

ALTER TABLE catalog_categories
  DROP KEY idx_catalog_categories_type,
  DROP COLUMN category_type;
