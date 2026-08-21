-- +goose Up
-- Earlier store deletion removed stores but only disabled their access/device
-- rows. Purge those orphans so administrator, staff and printer lists cannot
-- expose records belonging to a store that no longer exists.
DELETE aa
FROM admin_accounts aa
LEFT JOIN stores s ON s.id = aa.store_id
WHERE aa.store_id IS NOT NULL
  AND s.id IS NULL;

DELETE sa
FROM staff_accounts sa
LEFT JOIN stores s ON s.id = sa.store_id
WHERE s.id IS NULL;

DELETE pd
FROM printer_devices pd
LEFT JOIN stores s ON s.id = pd.store_id
WHERE s.id IS NULL;

-- +goose Down
-- Deleted orphaned access/device rows cannot be reconstructed safely.
SELECT 1;
