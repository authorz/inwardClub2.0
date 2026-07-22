-- +goose Up
-- Catalog: global/store scoped categories, items, variants and store overrides.

CREATE TABLE catalog_categories (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  scope_type VARCHAR(16) NOT NULL DEFAULT 'store',
  store_id BIGINT UNSIGNED NULL,
  parent_id BIGINT UNSIGNED NULL,
  name VARCHAR(128) NOT NULL,
  asset_id BIGINT UNSIGNED NULL,
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_categories_scope (scope_type, store_id),
  KEY idx_categories_parent (parent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE catalog_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  scope_type VARCHAR(16) NOT NULL DEFAULT 'store',
  store_id BIGINT UNSIGNED NULL,
  source_item_id BIGINT UNSIGNED NULL,
  category_id BIGINT UNSIGNED NULL,
  name VARCHAR(128) NOT NULL,
  description TEXT NULL,
  asset_id BIGINT UNSIGNED NULL,
  item_type VARCHAR(32) NOT NULL DEFAULT 'food',
  price_cent BIGINT NOT NULL DEFAULT 0,
  stock_quantity BIGINT NOT NULL DEFAULT 0,
  pay_channels JSON NOT NULL,
  points_reward BIGINT NOT NULL DEFAULT 0,
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_items_scope (scope_type, store_id),
  KEY idx_items_category (category_id),
  KEY idx_items_type_status (item_type, status),
  KEY idx_items_source (source_item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE catalog_variants (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  item_id BIGINT UNSIGNED NOT NULL,
  sku_code VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  price_cent BIGINT NOT NULL DEFAULT 0,
  stock_quantity BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_variant_sku (item_id, sku_code),
  KEY idx_variants_item (item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- store_item_overrides holds a store's adoption of a global item: only display,
-- price, stock, pay channels, sort and status may differ from the template.
CREATE TABLE store_item_overrides (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  item_id BIGINT UNSIGNED NOT NULL,
  category_id BIGINT UNSIGNED NULL,
  price_cent BIGINT NULL,
  stock_quantity BIGINT NULL,
  pay_channels JSON NULL,
  asset_id BIGINT UNSIGNED NULL,
  sort_order INT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_store_item_override (store_id, item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS store_item_overrides;
DROP TABLE IF EXISTS catalog_variants;
DROP TABLE IF EXISTS catalog_items;
DROP TABLE IF EXISTS catalog_categories;
