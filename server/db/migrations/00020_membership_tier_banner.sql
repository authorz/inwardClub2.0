-- +goose Up
-- VIP level banner: give each membership tier an optional uploaded banner asset,
-- rendered on the mini-program home page for the member's current tier. It
-- mirrors icon_asset_id: an optional reference into the assets table (never a raw
-- URL), resolved to a public URL on read. NULL means "no banner configured".

ALTER TABLE membership_tiers
  ADD COLUMN banner_asset_id BIGINT UNSIGNED NULL -- VIP banner asset (assets.id); NULL = none
  AFTER icon_asset_id;

-- +goose Down
ALTER TABLE membership_tiers DROP COLUMN banner_asset_id;
