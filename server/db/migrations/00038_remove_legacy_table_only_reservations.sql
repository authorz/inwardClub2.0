-- +goose Up
-- Seat reservation now always requires a concrete table and seat. Legacy
-- table-only rows cannot be represented on the seat map and can incorrectly
-- block a member from choosing a visible seat.
DELETE FROM reservations
WHERE status = 'booked' AND (table_id IS NULL OR seat_id IS NULL);

DELETE c
FROM reservation_daily_claims c
LEFT JOIN reservations r
  ON r.member_id = c.member_id
 AND r.status = 'booked'
 AND r.seat_id IS NOT NULL
 AND r.created_at >= c.daily_start
 AND r.created_at < DATE_ADD(c.daily_start, INTERVAL 1 DAY)
WHERE r.id IS NULL;

-- +goose Down
-- Removed legacy rows cannot be reconstructed.
SELECT 1;
