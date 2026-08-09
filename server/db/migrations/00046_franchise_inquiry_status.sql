-- +goose Up
ALTER TABLE franchise_inquiries
  ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'unprocessed' AFTER source,
  ADD COLUMN processed_at DATETIME(3) NULL AFTER status,
  ADD KEY idx_franchise_inquiries_status_created (status, created_at);

-- +goose Down
ALTER TABLE franchise_inquiries
  DROP KEY idx_franchise_inquiries_status_created,
  DROP COLUMN processed_at,
  DROP COLUMN status;
