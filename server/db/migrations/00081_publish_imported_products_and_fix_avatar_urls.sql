-- +goose Up
-- The v1 importer originally copied product status "active" verbatim even
-- though the v2 public catalog only exposes "published" items. Limit the
-- correction to rows traced to v1 products.

UPDATE catalog_items AS item
JOIN legacy_id_maps AS legacy_map
  ON legacy_map.source_system = 'v1'
  AND legacy_map.source_table = 'products'
  AND legacy_map.target_table = 'catalog_items'
  AND legacy_map.target_id = item.id
SET item.status = 'published',
    item.updated_at = UTC_TIMESTAMP()
WHERE item.status = 'active';

-- V1 member avatars were copied as relative storage paths. Preserve direct
-- HTTP(S) URLs and only qualify non-empty relative values on traced v1 users.

UPDATE members AS member
JOIN legacy_id_maps AS legacy_map
  ON legacy_map.source_system = 'v1'
  AND legacy_map.source_table = 'users'
  AND legacy_map.target_table = 'members'
  AND legacy_map.target_id = member.id
SET member.avatar_url = CONCAT(
      'https://api.inwardclub.com/storage/',
      TRIM(LEADING '/' FROM TRIM(member.avatar_url))
    ),
    member.updated_at = UTC_TIMESTAMP()
WHERE member.avatar_url IS NOT NULL
  AND TRIM(member.avatar_url) <> ''
  AND member.avatar_url NOT LIKE 'http://%'
  AND member.avatar_url NOT LIKE 'https://%';

-- +goose Down
-- Only reverse values that still match the exact normalization performed by
-- this migration; later manual status or avatar changes remain untouched.

UPDATE catalog_items AS item
JOIN legacy_id_maps AS legacy_map
  ON legacy_map.source_system = 'v1'
  AND legacy_map.source_table = 'products'
  AND legacy_map.target_table = 'catalog_items'
  AND legacy_map.target_id = item.id
SET item.status = 'active',
    item.updated_at = UTC_TIMESTAMP()
WHERE item.status = 'published';

UPDATE members AS member
JOIN legacy_id_maps AS legacy_map
  ON legacy_map.source_system = 'v1'
  AND legacy_map.source_table = 'users'
  AND legacy_map.target_table = 'members'
  AND legacy_map.target_id = member.id
SET member.avatar_url = SUBSTRING(
      member.avatar_url,
      CHAR_LENGTH('https://api.inwardclub.com/storage/') + 1
    ),
    member.updated_at = UTC_TIMESTAMP()
WHERE member.avatar_url LIKE 'https://api.inwardclub.com/storage/%';
