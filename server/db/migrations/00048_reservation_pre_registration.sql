-- +goose Up
-- Existing members are already registered; only new silent reservations create
-- profile_completed = 0 rows.
ALTER TABLE members
  ADD COLUMN profile_completed TINYINT(1) NOT NULL DEFAULT 1
  AFTER invited_by_member_id;

-- Preserve how the booking was made so public seat availability can show the
-- generic brand identity even when an existing member reserved while logged out.
ALTER TABLE reservations
  ADD COLUMN booked_as_guest TINYINT(1) NOT NULL DEFAULT 0
  AFTER member_id;

-- +goose Down
ALTER TABLE reservations DROP COLUMN booked_as_guest;
ALTER TABLE members DROP COLUMN profile_completed;
