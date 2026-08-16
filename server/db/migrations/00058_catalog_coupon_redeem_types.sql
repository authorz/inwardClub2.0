-- +goose Up
-- A product is maintained once; this field declares which coupon types may
-- redeem it. NULL/[] means the product is not available on the coupon page.

ALTER TABLE catalog_items
  ADD COLUMN coupon_redeem_types JSON NULL AFTER pay_channels;

-- +goose Down
ALTER TABLE catalog_items
  DROP COLUMN coupon_redeem_types;
