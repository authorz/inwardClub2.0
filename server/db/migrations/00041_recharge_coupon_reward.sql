-- +goose Up
ALTER TABLE recharge_products
  ADD COLUMN coupon_template_id BIGINT UNSIGNED NULL AFTER points_amount,
  ADD KEY idx_recharge_products_coupon_template (coupon_template_id);

-- +goose Down
ALTER TABLE recharge_products
  DROP KEY idx_recharge_products_coupon_template,
  DROP COLUMN coupon_template_id;
