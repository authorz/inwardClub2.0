-- +goose Up
-- Food orders. Item/price/pay-rule snapshot is captured at order time.

CREATE TABLE food_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  business_order_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  member_id BIGINT UNSIGNED NOT NULL,
  table_id BIGINT UNSIGNED NULL,
  total_amount_cent BIGINT NOT NULL DEFAULT 0,
  fulfillment_status VARCHAR(32) NOT NULL DEFAULT 'pending',
  remark VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_food_business_order (business_order_id),
  KEY idx_food_orders_store (store_id),
  KEY idx_food_orders_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE food_order_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  food_order_id BIGINT UNSIGNED NOT NULL,
  item_id BIGINT UNSIGNED NOT NULL,
  variant_id BIGINT UNSIGNED NULL,
  name_snapshot VARCHAR(128) NOT NULL,
  unit_price_cent BIGINT NOT NULL,
  quantity INT NOT NULL,
  pay_channels_snapshot JSON NOT NULL,
  subtotal_cent BIGINT NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_food_items_order (food_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS food_order_items;
DROP TABLE IF EXISTS food_orders;
