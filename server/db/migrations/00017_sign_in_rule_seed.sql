-- +goose Up
-- Seed the default daily sign-in rule so the headquarters console has a real,
-- editable sign_in rule out of the box. The config_json fully describes the
-- reward curve (7-day ladder, capped at 700 from day 7 onward) and whether the
-- rule is active; the enabled column mirrors config for cheap filtering.

INSERT INTO rule_definitions
  (rule_key, scope_type, store_id, version, config_json, enabled, status, created_at, updated_at)
VALUES
  ('sign_in', 'global', NULL, 1,
   '{"dailyRewards":[100,200,300,400,500,600,700],"capDay":7,"capReward":700,"enabled":true}',
   1, 'published', NOW(), NOW());

-- +goose Down
DELETE FROM rule_definitions WHERE rule_key = 'sign_in' AND scope_type = 'global' AND version = 1;
