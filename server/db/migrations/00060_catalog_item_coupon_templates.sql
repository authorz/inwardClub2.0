-- +goose Up
-- Products are redeemable by specific coupon templates, not by the generic
-- coupon implementation type (exchange / discount / cash). The legacy values
-- cannot be mapped safely to a concrete template. Keep the legacy column only
-- for rollback/audit compatibility; application code no longer reads it.

ALTER TABLE catalog_items
  ADD COLUMN coupon_template_ids JSON NULL AFTER coupon_redeem_types;

-- +goose Down
ALTER TABLE catalog_items
  DROP COLUMN coupon_template_ids;
