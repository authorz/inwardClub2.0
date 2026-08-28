-- +goose Up
-- A coupon kind is the single business-facing definition selected by HQ,
-- store coupon products and VIP benefits. coupon_templates remain an internal
-- compatibility layer for existing entitlements and redemption joins.
ALTER TABLE coupon_categories
  ADD COLUMN description VARCHAR(500) NOT NULL DEFAULT '' AFTER business_type,
  ADD COLUMN admission_count INT UNSIGNED NOT NULL DEFAULT 1 AFTER description,
  ADD COLUMN default_validity_days INT UNSIGNED NOT NULL DEFAULT 30 AFTER admission_count,
  ADD COLUMN canonical_template_id BIGINT UNSIGNED NULL AFTER default_validity_days;

-- Preserve the original VIP JSON before repairing stale category references.
CREATE TABLE migration_00085_membership_benefit_backups (
  tier_id BIGINT UNSIGNED NOT NULL,
  benefit_config JSON NOT NULL,
  PRIMARY KEY (tier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE migration_00085_template_backups (
  template_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  description TEXT NULL,
  coupon_type VARCHAR(32) NOT NULL,
  admission_count INT UNSIGNED NOT NULL,
  validity_rule JSON NOT NULL,
  status VARCHAR(32) NOT NULL,
  PRIMARY KEY (template_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO migration_00085_template_backups
  (template_id, name, description, coupon_type, admission_count, validity_rule, status)
SELECT id, name, description, coupon_type, admission_count, validity_rule, status
FROM coupon_templates
WHERE scope_type = 'global' AND store_id IS NULL;

INSERT INTO migration_00085_membership_benefit_backups (tier_id, benefit_config)
SELECT id, benefit_config
FROM membership_tiers
WHERE CAST(benefit_config AS CHAR) LIKE '%"categoryId": 12%'
   OR CAST(benefit_config AS CHAR) LIKE '%"categoryId":12%'
   OR CAST(benefit_config AS CHAR) LIKE '%"categoryId": 13%'
   OR CAST(benefit_config AS CHAR) LIKE '%"categoryId":13%';

-- Keep the two concrete tournament ticket kinds that appear in configured VIP
-- benefits. They are created only when absent so fresh and existing databases
-- converge on the same model.
INSERT INTO coupon_categories
  (name, business_type, description, admission_count, default_validity_days,
   canonical_template_id, sort_order, status, created_at, updated_at)
SELECT '周赛门票', 'admission_ticket', '一张券兑换一张周赛门票', 1, 30,
       NULL, 15, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()
WHERE NOT EXISTS (SELECT 1 FROM coupon_categories WHERE name = '周赛门票');

INSERT INTO coupon_categories
  (name, business_type, description, admission_count, default_validity_days,
   canonical_template_id, sort_order, status, created_at, updated_at)
SELECT '月赛门票', 'admission_ticket', '一张券兑换一张月赛门票', 1, 30,
       NULL, 16, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP()
WHERE NOT EXISTS (SELECT 1 FROM coupon_categories WHERE name = '月赛门票');

-- Materialise one internal global definition for every coupon kind that does
-- not already have one. Operators no longer manage these rows directly.
INSERT INTO coupon_templates
  (scope_type, store_id, name, description, coupon_type, category_id,
   admission_count, value_cent, points_price, stock_quantity, issued_quantity,
   validity_rule, applicable_scope, per_member_limit, status, created_at, updated_at)
SELECT 'global', NULL, c.name,
       CASE WHEN c.description <> '' THEN c.description ELSE CONCAT('一张', c.name, '兑换一项对应权益') END,
       c.business_type, c.id, c.admission_count, 0, 0, 0, 0,
       JSON_OBJECT('days', c.default_validity_days), JSON_OBJECT(), 0,
       CASE WHEN c.status = 'active' THEN 'published' ELSE 'disabled' END,
       UTC_TIMESTAMP(), UTC_TIMESTAMP()
FROM coupon_categories c
WHERE NOT EXISTS (
  SELECT 1 FROM coupon_templates t
  WHERE t.category_id = c.id AND t.scope_type = 'global' AND t.store_id IS NULL
);

UPDATE coupon_categories c
JOIN (
  SELECT category_id, MIN(id) AS template_id
  FROM coupon_templates
  WHERE scope_type = 'global' AND store_id IS NULL
  GROUP BY category_id
) selected ON selected.category_id = c.id
JOIN coupon_templates t ON t.id = selected.template_id
SET c.canonical_template_id = t.id,
    c.description = COALESCE(NULLIF(t.description, ''), c.description),
    c.admission_count = GREATEST(t.admission_count, 1),
    c.default_validity_days = COALESCE(
      NULLIF(CAST(JSON_UNQUOTE(JSON_EXTRACT(t.validity_rule, '$.days')) AS UNSIGNED), 0),
      30
    );

-- The kind owns the internal definition's display and default validity.
UPDATE coupon_templates t
JOIN coupon_categories c ON c.canonical_template_id = t.id
SET t.name = c.name,
    t.description = c.description,
    t.coupon_type = c.business_type,
    t.admission_count = c.admission_count,
    t.validity_rule = JSON_OBJECT('days', c.default_validity_days),
    t.status = CASE WHEN c.status = 'active' THEN 'published' ELSE 'disabled' END,
    t.updated_at = UTC_TIMESTAMP();

ALTER TABLE coupon_categories
  ADD UNIQUE KEY uq_coupon_categories_canonical_template (canonical_template_id);

SET @weekly_ticket_kind_id = (
  SELECT id FROM coupon_categories WHERE name = '周赛门票' ORDER BY id LIMIT 1
);
SET @monthly_ticket_kind_id = (
  SELECT id FROM coupon_categories WHERE name = '月赛门票' ORDER BY id LIMIT 1
);

UPDATE membership_tiers
SET benefit_config = CAST(
  REPLACE(
    REPLACE(
      REPLACE(
        REPLACE(CAST(benefit_config AS CHAR), '"categoryId": 12', CONCAT('"categoryId": ', @weekly_ticket_kind_id)),
        '"categoryId":12', CONCAT('"categoryId":', @weekly_ticket_kind_id)
      ),
      '"categoryId": 13', CONCAT('"categoryId": ', @monthly_ticket_kind_id)
    ),
    '"categoryId":13', CONCAT('"categoryId":', @monthly_ticket_kind_id)
  ) AS JSON
)
WHERE CAST(benefit_config AS CHAR) LIKE '%"categoryId": 12%'
   OR CAST(benefit_config AS CHAR) LIKE '%"categoryId":12%'
   OR CAST(benefit_config AS CHAR) LIKE '%"categoryId": 13%'
   OR CAST(benefit_config AS CHAR) LIKE '%"categoryId":13%';

-- +goose Down
UPDATE membership_tiers t
JOIN migration_00085_membership_benefit_backups b ON b.tier_id = t.id
SET t.benefit_config = b.benefit_config;

UPDATE coupon_templates t
JOIN migration_00085_template_backups b ON b.template_id = t.id
SET t.name = b.name,
    t.description = b.description,
    t.coupon_type = b.coupon_type,
    t.admission_count = b.admission_count,
    t.validity_rule = b.validity_rule,
    t.status = b.status;

DROP TABLE migration_00085_membership_benefit_backups;
DROP TABLE migration_00085_template_backups;

ALTER TABLE coupon_categories
  DROP KEY uq_coupon_categories_canonical_template,
  DROP COLUMN canonical_template_id,
  DROP COLUMN default_validity_days,
  DROP COLUMN admission_count,
  DROP COLUMN description;
