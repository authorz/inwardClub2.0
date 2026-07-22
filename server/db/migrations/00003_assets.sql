-- +goose Up
-- Assets table per QINIU_ASSET_SERVICE_SPEC. Business tables store asset_id only.

CREATE TABLE assets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  bucket VARCHAR(100) NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  etag VARCHAR(128) NULL,
  original_filename VARCHAR(255) NOT NULL,
  content_type VARCHAR(100) NOT NULL,
  size_bytes BIGINT UNSIGNED NOT NULL,
  width INT UNSIGNED NULL,
  height INT UNSIGNED NULL,
  purpose VARCHAR(32) NOT NULL,
  visibility VARCHAR(16) NOT NULL DEFAULT 'public',
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  uploaded_by_type VARCHAR(20) NOT NULL,
  uploaded_by_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL,
  uploaded_at DATETIME NULL,
  deleted_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_assets_object_key (object_key),
  KEY idx_assets_owner_purpose (uploaded_by_type, uploaded_by_id, purpose),
  KEY idx_assets_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS assets;
