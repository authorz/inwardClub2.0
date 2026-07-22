-- +goose Up
-- VIP banner path: store the uploaded banner's object key/path directly on the
-- membership tier, resolved to a public URL (QINIU_PUBLIC_DOMAIN + path) on read.
-- Supersedes banner_asset_id for new writes; banner_asset_id (00020) is retained
-- only for reading tiers configured before this migration. NULL means "no banner".

ALTER TABLE membership_tiers
  ADD COLUMN banner_path VARCHAR(512) NULL -- VIP banner object key/path; NULL = none
  AFTER banner_asset_id;

-- +goose Down
ALTER TABLE membership_tiers DROP COLUMN banner_path;
