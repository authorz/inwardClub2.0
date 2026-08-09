-- +goose Up
-- Tournament events are informational competition promotions shown from the
-- reservation page. They are deliberately separate from ticketed activities.
CREATE TABLE tournament_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(128) NOT NULL,
  summary VARCHAR(500) NULL,
  content MEDIUMTEXT NULL,
  asset_id BIGINT UNSIGNED NULL,
  start_at DATETIME NOT NULL,
  end_at DATETIME NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'published',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_tournament_events_store_time (store_id, status, start_at, end_at),
  KEY idx_tournament_events_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS tournament_events;
