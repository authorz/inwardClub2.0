-- +goose Up
-- Older cancellation logic deleted the visible reservation but intentionally
-- retained the daily claim. Remove those orphaned claims so the UI and the
-- one-active-reservation rule agree for members who already cancelled today.
DELETE c
FROM reservation_daily_claims c
LEFT JOIN reservations r
  ON r.member_id = c.member_id
 AND r.status = 'booked'
 AND r.created_at >= c.daily_start
 AND r.created_at < DATE_ADD(c.daily_start, INTERVAL 1 DAY)
WHERE r.id IS NULL;

-- +goose Down
-- Released claims cannot be reconstructed once the reservation was deleted.
SELECT 1;
