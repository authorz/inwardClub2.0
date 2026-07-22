-- +goose Up
-- Recharge VIP-growth settlement: give recharge products a configurable
-- growth_value grant and give members a persisted resolved VIP tier.
--
-- growth_amount mirrors bonus_amount: it is per-product, admin-configured, and
-- defaults to 0 so existing products grant no growth until a value is set. This
-- keeps the growth amount config-driven (never hard-coded in Go, per spec §1)
-- while preserving current coins-bonus behaviour byte-for-byte.
--
-- current_tier_id is the highest membership_tiers.id the member has reached; it
-- is advanced upgrade-only by the recharge settlement as growth crosses the
-- configured thresholds. NULL means "not yet ranked". Downgrade / expiry /
-- monthly benefits stay unimplemented pending business rules (spec §13, VIP row).

ALTER TABLE recharge_products
  ADD COLUMN growth_amount BIGINT NOT NULL DEFAULT 0 -- growth_value granted on top-up (0 = none)
  AFTER bonus_amount;

ALTER TABLE members
  ADD COLUMN current_tier_id BIGINT UNSIGNED NULL -- resolved VIP tier (membership_tiers.id); NULL = unranked
  AFTER status;

-- +goose Down
ALTER TABLE members DROP COLUMN current_tier_id;
ALTER TABLE recharge_products DROP COLUMN growth_amount;
