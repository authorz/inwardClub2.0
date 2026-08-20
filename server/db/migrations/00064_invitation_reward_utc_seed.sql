-- +goose Up
-- The default invitation rule in 00062 used NOW(), while runtime eligibility
-- compares against UTC timestamps. On databases configured for Asia/Shanghai
-- this placed the seed rule eight hours in the future. Only the untouched
-- system seed is normalized; administrator-authored schedules are preserved.
UPDATE rule_definitions
SET effective_from = TIMESTAMPADD(
      SECOND,
      -TIMESTAMPDIFF(SECOND, UTC_TIMESTAMP(), NOW()),
      effective_from
    ),
    updated_at = UTC_TIMESTAMP()
WHERE rule_key = 'invite_reward'
  AND scope_type = 'global'
  AND enabled = 1
  AND status = 'published'
  AND created_by IS NULL
  AND updated_by IS NULL
  AND JSON_EXTRACT(config_json, '$.schemaVersion') = 1
  AND effective_from > UTC_TIMESTAMP();

-- +goose Down
-- Data-only UTC normalization is intentionally not reversed.
SELECT 1;
