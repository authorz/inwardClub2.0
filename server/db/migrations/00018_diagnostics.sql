-- +goose Up
-- Diagnostics: durable admin-facing error-events feed. Persists 5xx responses
-- and handler-attached errors captured by the diagnostics Capture middleware so
-- the feed survives process restarts (previously an in-memory ring buffer). The
-- table is bounded by a retention cap pruned on each write (see the diagnostics
-- module); the id/created_at ordering backs the newest-first paged read.

CREATE TABLE error_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  method VARCHAR(16) NOT NULL DEFAULT '',
  path VARCHAR(255) NOT NULL DEFAULT '',
  status INT NOT NULL,
  message VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_error_events_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS error_events;
