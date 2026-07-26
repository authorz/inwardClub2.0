-- +goose Up
-- Supports current-seat occupancy lookup, duplicate-booking checks and the
-- daily 04:00 seat reset without scanning the full reservation history.
ALTER TABLE reservations
  ADD INDEX idx_reservations_seat_status_created (seat_id, status, created_at, id);

-- +goose Down
ALTER TABLE reservations
  DROP INDEX idx_reservations_seat_status_created;
