-- +goose Up
-- Invitation rewards are driven by the published invite_reward rule. The
-- accrual account keeps sub-coin commission precision so an integer coin wallet
-- can still deliver the configured percentage exactly over time.

ALTER TABLE members
  ADD COLUMN invited_at DATETIME NULL AFTER invited_by_member_id;

UPDATE members
SET invited_at = COALESCE(updated_at, created_at)
WHERE invited_by_member_id IS NOT NULL AND invited_at IS NULL;

CREATE TABLE invitation_reward_accounts (
  invitee_member_id BIGINT UNSIGNED NOT NULL,
  inviter_member_id BIGINT UNSIGNED NOT NULL,
  commission_remainder_numerator BIGINT NOT NULL DEFAULT 0,
  first_reward_active TINYINT(1) NOT NULL DEFAULT 0,
  first_reward_payment_order_id BIGINT UNSIGNED NULL,
  first_reward_rule_version INT NULL,
  first_reward_store_id BIGINT UNSIGNED NULL,
  first_reward_paid_at DATETIME NULL,
  first_reward_coins BIGINT NOT NULL DEFAULT 0,
  first_reward_points BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (invitee_member_id),
  KEY idx_invitation_reward_inviter (inviter_member_id),
  KEY idx_invitation_reward_first_payment (first_reward_payment_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE invitation_reward_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event_type VARCHAR(32) NOT NULL,
  invitee_member_id BIGINT UNSIGNED NOT NULL,
  inviter_member_id BIGINT UNSIGNED NOT NULL,
  payment_order_id BIGINT UNSIGNED NULL,
  refund_order_id BIGINT UNSIGNED NULL,
  order_type VARCHAR(32) NOT NULL DEFAULT '',
  amount_cent BIGINT NOT NULL DEFAULT 0,
  rule_version INT NOT NULL,
  commission_rate_basis_points INT NOT NULL DEFAULT 0,
  commission_numerator_delta BIGINT NOT NULL DEFAULT 0,
  coin_delta BIGINT NOT NULL DEFAULT 0,
  points_delta BIGINT NOT NULL DEFAULT 0,
  idem_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_invitation_reward_event_idem (idem_key),
  KEY idx_invitation_reward_payment (payment_order_id),
  KEY idx_invitation_reward_refund (refund_order_id),
  KEY idx_invitation_reward_invitee (invitee_member_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO rule_definitions
  (rule_key, scope_type, store_id, version, config_json, enabled,
   effective_from, effective_to, status, created_by, updated_by, created_at, updated_at)
SELECT
  'invite_reward',
  'global',
  NULL,
  COALESCE(MAX(version), 0) + 1,
  JSON_OBJECT(
    'schemaVersion', 1,
    'firstLowSpendRewardCoins', 50,
    'firstLowSpendRewardPoints', 2000,
    'commissionRateBasisPoints', 1000
  ),
  1,
  NOW(),
  NULL,
  'published',
  NULL,
  NULL,
  NOW(),
  NOW()
FROM rule_definitions
WHERE rule_key = 'invite_reward';

-- +goose Down
DELETE FROM rule_definitions
WHERE rule_key = 'invite_reward'
  AND JSON_EXTRACT(config_json, '$.schemaVersion') = 1;

DROP TABLE IF EXISTS invitation_reward_events;
DROP TABLE IF EXISTS invitation_reward_accounts;

ALTER TABLE members DROP COLUMN invited_at;
