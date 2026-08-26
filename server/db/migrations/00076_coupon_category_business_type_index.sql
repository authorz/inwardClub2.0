-- +goose Up
-- Multiple admin-managed categories may share one fulfillment behavior. Every
-- new template and VIP coupon benefit selects an exact category_id.

ALTER TABLE coupon_categories
  DROP KEY uq_coupon_categories_business_type,
  ADD KEY idx_coupon_categories_business_type (business_type);

-- +goose Down
ALTER TABLE coupon_categories
  DROP KEY idx_coupon_categories_business_type,
  ADD UNIQUE KEY uq_coupon_categories_business_type (business_type);
