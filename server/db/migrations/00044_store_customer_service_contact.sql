-- +goose Up
ALTER TABLE stores
  ADD COLUMN customer_service_qr_asset_id BIGINT UNSIGNED NULL AFTER phone,
  ADD KEY idx_stores_customer_service_qr_asset (customer_service_qr_asset_id);

-- +goose Down
ALTER TABLE stores
  DROP KEY idx_stores_customer_service_qr_asset,
  DROP COLUMN customer_service_qr_asset_id;
