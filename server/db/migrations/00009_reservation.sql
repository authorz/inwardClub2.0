-- +goose Up
-- Reservation, waitlist and arrival as three independent state machines.

CREATE TABLE reservations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  reservation_no VARCHAR(64) NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  table_id BIGINT UNSIGNED NULL,
  seat_id BIGINT UNSIGNED NULL,
  party_size INT NOT NULL DEFAULT 1,
  reserved_at DATETIME NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'booked', -- booked/cancelled/expired/arrived
  remark VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_reservation_no (reservation_no),
  KEY idx_reservations_store (store_id),
  KEY idx_reservations_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE waitlist_entries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  party_size INT NOT NULL DEFAULT 1,
  status VARCHAR(32) NOT NULL DEFAULT 'waiting', -- waiting/called/seated/left
  queued_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_waitlist_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE arrival_records (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  reservation_id BIGINT UNSIGNED NULL,
  arrived_at DATETIME NOT NULL,
  recorded_by_type VARCHAR(20) NOT NULL,
  recorded_by_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_arrivals_store (store_id),
  KEY idx_arrivals_reservation (reservation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS arrival_records;
DROP TABLE IF EXISTS waitlist_entries;
DROP TABLE IF EXISTS reservations;
