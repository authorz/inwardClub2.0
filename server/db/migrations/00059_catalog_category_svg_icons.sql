-- +goose Up
-- Register the white SVG category icons already uploaded to Qiniu, then bind
-- every matching store category by its normalized category name.

INSERT INTO assets (
  bucket,
  object_key,
  etag,
  original_filename,
  content_type,
  size_bytes,
  width,
  height,
  purpose,
  visibility,
  status,
  uploaded_by_type,
  uploaded_by_id,
  created_at,
  uploaded_at,
  deleted_at
) VALUES
  ('inward-mini', 'public/menu-category/white-v1/voucher-redemption.svg', NULL, 'voucher-redemption.svg', 'image/svg+xml', 269, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/signature-drinks.svg', NULL, 'signature-drinks.svg', 'image/svg+xml', 355, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/classic-beer.svg', NULL, 'classic-beer.svg', 'image/svg+xml', 460, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/classic-cocktails.svg', NULL, 'classic-cocktails.svg', 'image/svg+xml', 314, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/whisky-neat.svg', NULL, 'whisky-neat.svg', 'image/svg+xml', 298, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/soft-drinks.svg', NULL, 'soft-drinks.svg', 'image/svg+xml', 337, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/bar-snacks.svg', NULL, 'bar-snacks.svg', 'image/svg+xml', 294, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/voucher-sets.svg', NULL, 'voucher-sets.svg', 'image/svg+xml', 342, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL),
  ('inward-mini', 'public/menu-category/white-v1/fine-dining.svg', NULL, 'fine-dining.svg', 'image/svg+xml', 335, NULL, NULL, 'category', 'public', 'bound', 'super_admin', 1, NOW(), NOW(), NULL)
ON DUPLICATE KEY UPDATE
  original_filename = VALUES(original_filename),
  content_type = VALUES(content_type),
  size_bytes = VALUES(size_bytes),
  purpose = VALUES(purpose),
  visibility = VALUES(visibility),
  status = VALUES(status),
  uploaded_at = COALESCE(uploaded_at, VALUES(uploaded_at)),
  deleted_at = NULL;

UPDATE catalog_categories AS category
JOIN assets AS asset
  ON asset.object_key = CASE TRIM(category.name)
    WHEN '酒券兑换区' THEN 'public/menu-category/white-v1/voucher-redemption.svg'
    WHEN '特调饮品' THEN 'public/menu-category/white-v1/signature-drinks.svg'
    WHEN '经典啤酒' THEN 'public/menu-category/white-v1/classic-beer.svg'
    WHEN '经典鸡尾酒' THEN 'public/menu-category/white-v1/classic-cocktails.svg'
    WHEN '纯饮威士忌' THEN 'public/menu-category/white-v1/whisky-neat.svg'
    WHEN '软饮系列' THEN 'public/menu-category/white-v1/soft-drinks.svg'
    WHEN '佐食小吃' THEN 'public/menu-category/white-v1/bar-snacks.svg'
    WHEN '酒券套餐' THEN 'public/menu-category/white-v1/voucher-sets.svg'
    WHEN '精致餐食' THEN 'public/menu-category/white-v1/fine-dining.svg'
    ELSE NULL
  END
SET category.asset_id = asset.id,
    category.updated_at = NOW();

-- +goose Down
-- Restore the prior curated PNG bindings when those legacy assets exist.

UPDATE catalog_categories AS category
JOIN assets AS asset
  ON asset.object_key = CASE TRIM(category.name)
    WHEN '特调饮品' THEN 'inwardclub/development/category/curated-20260812/signature-drinks.png'
    WHEN '经典啤酒' THEN 'inwardclub/development/category/curated-20260812/classic-beer.png'
    WHEN '经典鸡尾酒' THEN 'inwardclub/development/category/curated-20260812/classic-cocktails.png'
    WHEN '纯饮威士忌' THEN 'inwardclub/development/category/curated-20260812/whisky-neat.png'
    WHEN '软饮系列' THEN 'inwardclub/development/category/curated-20260812/soft-drinks.png'
    WHEN '佐食小吃' THEN 'inwardclub/development/category/curated-20260812/bar-snacks.png'
    WHEN '酒券套餐' THEN 'inwardclub/development/category/curated-20260812/voucher-sets.png'
    WHEN '精致餐食' THEN 'inwardclub/development/category/curated-20260812/fine-dining.png'
    ELSE NULL
  END
SET category.asset_id = asset.id,
    category.updated_at = NOW();

DELETE asset
FROM assets AS asset
LEFT JOIN catalog_categories AS category ON category.asset_id = asset.id
WHERE asset.object_key IN (
  'public/menu-category/white-v1/voucher-redemption.svg',
  'public/menu-category/white-v1/signature-drinks.svg',
  'public/menu-category/white-v1/classic-beer.svg',
  'public/menu-category/white-v1/classic-cocktails.svg',
  'public/menu-category/white-v1/whisky-neat.svg',
  'public/menu-category/white-v1/soft-drinks.svg',
  'public/menu-category/white-v1/bar-snacks.svg',
  'public/menu-category/white-v1/voucher-sets.svg',
  'public/menu-category/white-v1/fine-dining.svg'
)
  AND category.id IS NULL;
