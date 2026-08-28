-- +goose Up
CREATE TABLE migration_00087_recharge_store_backups (
  business_order_id BIGINT UNSIGNED NOT NULL,
  business_order_store_id BIGINT UNSIGNED NULL,
  payment_order_id BIGINT UNSIGNED NOT NULL,
  payment_order_store_id BIGINT UNSIGNED NULL,
  assigned_store_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (business_order_id),
  UNIQUE KEY uq_migration_00087_payment_order (payment_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO migration_00087_recharge_store_backups
  (business_order_id, business_order_store_id, payment_order_id,
   payment_order_store_id, assigned_store_id)
SELECT bo.id, bo.store_id, po.id, po.store_id, only_store.id
FROM business_orders bo
JOIN payment_orders po ON po.business_order_id = bo.id
JOIN (
  SELECT MIN(id) AS id
  FROM stores
  WHERE status = 'active'
  HAVING COUNT(*) = 1
) only_store
WHERE bo.order_type = 'recharge'
  AND bo.store_id IS NULL
  AND po.store_id IS NULL;

UPDATE business_orders bo
JOIN migration_00087_recharge_store_backups backup
  ON backup.business_order_id = bo.id
SET bo.store_id = backup.assigned_store_id;

UPDATE payment_orders po
JOIN migration_00087_recharge_store_backups backup
  ON backup.payment_order_id = po.id
SET po.store_id = backup.assigned_store_id;

-- +goose Down
UPDATE payment_orders po
JOIN migration_00087_recharge_store_backups backup
  ON backup.payment_order_id = po.id
SET po.store_id = backup.payment_order_store_id
WHERE po.store_id = backup.assigned_store_id;

UPDATE business_orders bo
JOIN migration_00087_recharge_store_backups backup
  ON backup.business_order_id = bo.id
SET bo.store_id = backup.business_order_store_id
WHERE bo.store_id = backup.assigned_store_id;

DROP TABLE migration_00087_recharge_store_backups;
