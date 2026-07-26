-- +goose Up
-- Reservations are deleted when cancelled or cleared, while the one-booking-
-- per-business-day rule must remain enforceable independently of visible rows.

CREATE TABLE reservation_daily_claims (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  member_id BIGINT UNSIGNED NOT NULL,
  daily_start DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_reservation_daily_member (member_id, daily_start)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Existing timestamps are UTC. For Asia/Shanghai with a 04:00 business-day
-- boundary, adding four hours and truncating to the date yields that day's UTC
-- boundary after subtracting four hours again.
INSERT IGNORE INTO reservation_daily_claims (member_id, daily_start, created_at)
SELECT member_id,
       TIMESTAMP(DATE(DATE_ADD(created_at, INTERVAL 4 HOUR))) - INTERVAL 4 HOUR,
       MIN(created_at)
FROM reservations
GROUP BY member_id, TIMESTAMP(DATE(DATE_ADD(created_at, INTERVAL 4 HOUR))) - INTERVAL 4 HOUR;

-- +goose Down
DROP TABLE IF EXISTS reservation_daily_claims;
