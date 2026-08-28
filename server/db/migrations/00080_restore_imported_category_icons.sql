-- +goose Up
-- Rebind the curated category SVGs retained during the final v1 import. The
-- imported categories intentionally keep their v1 IDs, so match both ID and
-- name and only fill missing bindings.

UPDATE catalog_categories AS category
JOIN (
  SELECT 1 AS category_id, '酒券兑换' AS category_name, 'public/menu-category/white-v1/voucher-redemption.svg' AS object_key
  UNION ALL SELECT 26, '特调系列', 'public/menu-category/white-v1/signature-drinks.svg'
  UNION ALL SELECT 27, '主题特调', 'public/menu-category/white-v1/signature-drinks.svg'
  UNION ALL SELECT 4, '经典啤酒', 'public/menu-category/white-v1/classic-beer.svg'
  UNION ALL SELECT 24, '经典系列', 'public/menu-category/white-v1/classic-cocktails.svg'
  UNION ALL SELECT 2, '纯饮系列', 'public/menu-category/white-v1/whisky-neat.svg'
  UNION ALL SELECT 3, '软饮系列', 'public/menu-category/white-v1/soft-drinks.svg'
  UNION ALL SELECT 6, '酒券套餐', 'public/menu-category/white-v1/voucher-sets.svg'
) AS icon_mapping
  ON icon_mapping.category_id = category.id
  AND icon_mapping.category_name = TRIM(category.name)
JOIN assets AS asset
  ON asset.object_key = icon_mapping.object_key
  AND asset.status IN ('uploaded', 'bound')
  AND asset.deleted_at IS NULL
SET category.asset_id = asset.id,
    category.updated_at = UTC_TIMESTAMP()
WHERE category.asset_id IS NULL;

-- +goose Down
-- Only undo bindings that still point to the exact curated asset selected by
-- this migration, so a later manual icon change is never overwritten.

UPDATE catalog_categories AS category
JOIN (
  SELECT 1 AS category_id, '酒券兑换' AS category_name, 'public/menu-category/white-v1/voucher-redemption.svg' AS object_key
  UNION ALL SELECT 26, '特调系列', 'public/menu-category/white-v1/signature-drinks.svg'
  UNION ALL SELECT 27, '主题特调', 'public/menu-category/white-v1/signature-drinks.svg'
  UNION ALL SELECT 4, '经典啤酒', 'public/menu-category/white-v1/classic-beer.svg'
  UNION ALL SELECT 24, '经典系列', 'public/menu-category/white-v1/classic-cocktails.svg'
  UNION ALL SELECT 2, '纯饮系列', 'public/menu-category/white-v1/whisky-neat.svg'
  UNION ALL SELECT 3, '软饮系列', 'public/menu-category/white-v1/soft-drinks.svg'
  UNION ALL SELECT 6, '酒券套餐', 'public/menu-category/white-v1/voucher-sets.svg'
) AS icon_mapping
  ON icon_mapping.category_id = category.id
  AND icon_mapping.category_name = TRIM(category.name)
JOIN assets AS asset
  ON asset.object_key = icon_mapping.object_key
  AND category.asset_id = asset.id
SET category.asset_id = NULL,
    category.updated_at = UTC_TIMESTAMP();
