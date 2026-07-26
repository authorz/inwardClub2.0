-- +goose Up
UPDATE food_orders fo
JOIN business_orders bo ON bo.id = fo.business_order_id
SET fo.fulfillment_status = 'completed',
    fo.updated_at = NOW(),
    bo.order_status = 'completed',
    bo.updated_at = NOW()
WHERE bo.payment_status = 'paid'
  AND fo.fulfillment_status <> 'cancelled';

-- +goose Down
-- Existing fulfillment history cannot be reconstructed safely.
SELECT 1;
