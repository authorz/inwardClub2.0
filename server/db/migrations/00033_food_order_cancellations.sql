-- +goose Up
CREATE TABLE food_order_cancellations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  food_order_id BIGINT UNSIGNED NOT NULL,
  payment_order_id BIGINT UNSIGNED NOT NULL,
  refund_order_id BIGINT UNSIGNED NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  original_status VARCHAR(32) NOT NULL,
  points_earned BIGINT NOT NULL DEFAULT 0,
  points_recovered BIGINT NOT NULL DEFAULT 0,
  points_shortfall BIGINT NOT NULL DEFAULT 0,
  forced TINYINT(1) NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'processing',
  failure_reason VARCHAR(255) NOT NULL DEFAULT '',
  requested_by_type VARCHAR(20) NOT NULL,
  requested_by_id BIGINT UNSIGNED NOT NULL,
  idem_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_food_cancel_order (food_order_id),
  UNIQUE KEY uq_food_cancel_idem (idem_key),
  KEY idx_food_cancel_store_created (store_id, created_at),
  KEY idx_food_cancel_refund (refund_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS food_order_cancellations;
