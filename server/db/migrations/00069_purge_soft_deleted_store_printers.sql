-- +goose Up
-- Legacy store deletion marked stores as deleted but left their printer rows
-- behind. Those hidden rows still occupy the global provider/device_sn unique
-- key and prevent a live store from registering the released physical device.
DELETE pd
FROM printer_devices pd
INNER JOIN stores s ON s.id = pd.store_id
WHERE s.status = 'deleted';

-- +goose Down
-- Deleted operational credentials cannot be reconstructed safely.
SELECT 1;
