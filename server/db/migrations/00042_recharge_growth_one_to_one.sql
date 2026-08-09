-- +goose Up
-- Growth value is no longer an independently configured recharge-product
-- grant. The authoritative rule is ¥1 recharge face value = 1 growth value.
-- Keep the legacy column aligned for compatibility, then reconcile existing
-- 2.0 paid recharge members to the same rule with one auditable ledger entry.

UPDATE recharge_products
SET growth_amount = FLOOR(amount_cent / 100);

CREATE TEMPORARY TABLE tmp_recharge_growth_target AS
SELECT bo.member_id,
       SUM(FLOOR(GREATEST(bo.total_amount_cent - COALESCE(rf.refunded_cent, 0), 0) / 100)) AS target_growth
FROM business_orders bo
LEFT JOIN (
  SELECT business_order_id, SUM(amount_cent) AS refunded_cent
  FROM refund_orders
  WHERE status = 'succeeded'
  GROUP BY business_order_id
) rf ON rf.business_order_id = bo.id
WHERE bo.order_type = 'recharge'
  AND bo.payment_status IN ('paid', 'partially_refunded')
  AND bo.member_id IS NOT NULL
GROUP BY bo.member_id;

INSERT INTO wallet_accounts
  (member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
SELECT member_id, 'growth_value', 0, 0, 0, NOW(), NOW()
FROM tmp_recharge_growth_target
WHERE target_growth > 0
ON DUPLICATE KEY UPDATE updated_at = wallet_accounts.updated_at;

INSERT INTO wallet_ledger_entries
  (account_id, member_id, asset_type, direction, amount, balance_after,
   reason, source_type, source_id, idem_key, created_at)
SELECT wa.id,
       wa.member_id,
       'growth_value',
       IF(t.target_growth >= wa.available_amount, 'credit', 'debit'),
       ABS(t.target_growth - wa.available_amount),
       t.target_growth,
       'recharge_growth_reconcile',
       'migration',
       NULL,
       CONCAT('growth_recharge_reconcile:v42:', wa.member_id),
       NOW()
FROM wallet_accounts wa
JOIN tmp_recharge_growth_target t ON t.member_id = wa.member_id
WHERE wa.asset_type = 'growth_value'
  AND t.target_growth <> wa.available_amount;

UPDATE wallet_accounts wa
JOIN tmp_recharge_growth_target t ON t.member_id = wa.member_id
SET wa.available_amount = t.target_growth,
    wa.version = wa.version + 1,
    wa.updated_at = NOW()
WHERE wa.asset_type = 'growth_value'
  AND wa.available_amount <> t.target_growth;

DROP TEMPORARY TABLE tmp_recharge_growth_target;

-- +goose Down
UPDATE wallet_accounts wa
JOIN wallet_ledger_entries le
  ON le.account_id = wa.id
 AND le.idem_key = CONCAT('growth_recharge_reconcile:v42:', wa.member_id)
SET wa.available_amount = CASE
      WHEN le.direction = 'credit' THEN GREATEST(wa.available_amount - le.amount, 0)
      ELSE wa.available_amount + le.amount
    END,
    wa.version = wa.version + 1,
    wa.updated_at = NOW()
WHERE wa.asset_type = 'growth_value';

DELETE FROM wallet_ledger_entries
WHERE idem_key LIKE 'growth_recharge_reconcile:v42:%';
