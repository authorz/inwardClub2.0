-- +goose Up
ALTER TABLE food_order_cancellations
  DROP INDEX uq_food_cancel_order,
  ADD KEY idx_food_cancel_order_status (food_order_id, status);

-- +goose Down
ALTER TABLE food_order_cancellations
  DROP INDEX idx_food_cancel_order_status,
  ADD UNIQUE KEY uq_food_cancel_order (food_order_id);
