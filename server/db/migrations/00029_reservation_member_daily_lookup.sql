-- +goose Up
-- Supports the per-member business-day reservation limit check.
ALTER TABLE reservations
  ADD INDEX idx_reservations_member_created (member_id, created_at);

-- +goose Down
ALTER TABLE reservations
  DROP INDEX idx_reservations_member_created;
