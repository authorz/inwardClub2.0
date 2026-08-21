-- +goose Up
-- Keep the provider failure visible to headquarters operators alongside each
-- durable print job. The existing status/attempt counters describe lifecycle;
-- this column carries the actionable failure detail.
ALTER TABLE print_jobs
  ADD COLUMN last_error VARCHAR(512) NOT NULL DEFAULT '' AFTER attempts;

-- +goose Down
ALTER TABLE print_jobs
  DROP COLUMN last_error;
