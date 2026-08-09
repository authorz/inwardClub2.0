-- +goose Up
-- Every successful member-bound WeChat payment earns growth at ¥1 = 1. Runtime
-- settlement already covered recharge orders before this migration, so only
-- historical non-recharge WeChat orders need to be backfilled here.

CREATE TEMPORARY TABLE tmp_wechat_payment_growth_v52 AS
SELECT bo.member_id,
       SUM(FLOOR(GREATEST(bo.total_amount_cent - COALESCE(rf.refunded_cent, 0), 0) / 100)) AS growth_amount
FROM business_orders bo
JOIN (
  SELECT business_order_id
  FROM payment_orders
  WHERE pay_method = 'wechat'
    AND paid_at IS NOT NULL
    AND status IN ('paid', 'partially_refunded', 'refunded')
  GROUP BY business_order_id
) paid ON paid.business_order_id = bo.id
LEFT JOIN (
  SELECT business_order_id, SUM(amount_cent) AS refunded_cent
  FROM refund_orders
  WHERE status = 'succeeded'
  GROUP BY business_order_id
) rf ON rf.business_order_id = bo.id
WHERE bo.order_type <> 'recharge'
  AND bo.payment_status IN ('paid', 'partially_refunded')
  AND bo.member_id IS NOT NULL
GROUP BY bo.member_id;

INSERT INTO wallet_accounts
  (member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
SELECT member_id, 'growth_value', 0, 0, 0, NOW(), NOW()
FROM tmp_wechat_payment_growth_v52
WHERE growth_amount > 0
ON DUPLICATE KEY UPDATE updated_at = wallet_accounts.updated_at;

INSERT INTO wallet_ledger_entries
  (account_id, member_id, asset_type, direction, amount, balance_after,
   reason, source_type, source_id, idem_key, created_at)
SELECT wa.id,
       wa.member_id,
       'growth_value',
       'credit',
       t.growth_amount,
       wa.available_amount + t.growth_amount,
       'wechat_payment_growth',
       'wechat_payment_growth',
       NULL,
       CONCAT('growth_wechat_payment_backfill:v52:', wa.member_id),
       NOW()
FROM wallet_accounts wa
JOIN tmp_wechat_payment_growth_v52 t ON t.member_id = wa.member_id
WHERE wa.asset_type = 'growth_value'
  AND t.growth_amount > 0;

UPDATE wallet_accounts wa
JOIN tmp_wechat_payment_growth_v52 t ON t.member_id = wa.member_id
SET wa.available_amount = wa.available_amount + t.growth_amount,
    wa.version = wa.version + 1,
    wa.updated_at = NOW()
WHERE wa.asset_type = 'growth_value'
  AND t.growth_amount > 0;

-- Backfilled growth can cross a VIP threshold. Preserve the established
-- upgrade-only rule: never lower a member's currently held level.
UPDATE members m
JOIN wallet_accounts wa
  ON wa.member_id = m.id AND wa.asset_type = 'growth_value'
JOIN membership_tiers target
  ON target.id = (
    SELECT t.id
    FROM membership_tiers t
    WHERE t.status = 'active' AND t.threshold <= wa.available_amount
    ORDER BY t.threshold DESC, t.level DESC, t.id ASC
    LIMIT 1
  )
LEFT JOIN membership_tiers current_tier ON current_tier.id = m.current_tier_id
SET m.current_tier_id = target.id,
    m.updated_at = NOW()
WHERE m.current_tier_id IS NULL OR current_tier.level < target.level;

DROP TEMPORARY TABLE tmp_wechat_payment_growth_v52;

-- +goose Down
UPDATE wallet_accounts wa
JOIN wallet_ledger_entries le
  ON le.account_id = wa.id
 AND le.idem_key = CONCAT('growth_wechat_payment_backfill:v52:', wa.member_id)
SET wa.available_amount = GREATEST(wa.available_amount - le.amount, 0),
    wa.version = wa.version + 1,
    wa.updated_at = NOW()
WHERE wa.asset_type = 'growth_value';

DELETE FROM wallet_ledger_entries
WHERE idem_key LIKE 'growth_wechat_payment_backfill:v52:%';

-- VIP tiers remain upgrade-only on rollback, matching normal runtime behavior.
