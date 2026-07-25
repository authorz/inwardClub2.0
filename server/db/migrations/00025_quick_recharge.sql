-- +goose Up
-- Quick recharge tiers have no display name. Payment amount, credited coins and
-- credited points are independent quantities with explicit units.

ALTER TABLE recharge_products
  CHANGE COLUMN amount amount_cent BIGINT NOT NULL COMMENT 'payment amount in cents',
  ADD COLUMN coin_amount BIGINT NOT NULL DEFAULT 0 COMMENT 'total coins credited after payment' AFTER amount_cent,
  ADD COLUMN points_amount BIGINT NOT NULL DEFAULT 0 COMMENT 'points credited after payment' AFTER coin_amount;

UPDATE recharge_products
SET coin_amount = (amount_cent DIV 100) +
    CASE WHEN asset_type = 'coin' THEN bonus_amount ELSE 0 END,
    points_amount =
    CASE WHEN asset_type IN ('point', 'points') THEN bonus_amount ELSE 0 END;

ALTER TABLE recharge_products
  DROP COLUMN name,
  DROP COLUMN bonus_amount,
  DROP COLUMN asset_type;

-- +goose Down
ALTER TABLE recharge_products
  ADD COLUMN name VARCHAR(64) NULL AFTER id,
  ADD COLUMN bonus_amount BIGINT NOT NULL DEFAULT 0 AFTER amount_cent,
  ADD COLUMN asset_type VARCHAR(32) NOT NULL DEFAULT 'coin' AFTER growth_amount;

UPDATE recharge_products
SET name = CONCAT('充值 ', amount_cent / 100, ' 元');

ALTER TABLE recharge_products
  MODIFY COLUMN name VARCHAR(64) NOT NULL,
  DROP COLUMN coin_amount,
  DROP COLUMN points_amount,
  CHANGE COLUMN amount_cent amount BIGINT NOT NULL COMMENT 'paid amount in cents';
