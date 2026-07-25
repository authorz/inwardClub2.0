-- +goose Up
-- Dashboard overview reads paid orders and wallet debits by recent time window.

ALTER TABLE payment_orders
  ADD KEY idx_payment_orders_dashboard (status, paid_at);

ALTER TABLE wallet_ledger_entries
  ADD KEY idx_wallet_ledger_dashboard (asset_type, direction, created_at);

ALTER TABLE members
  ADD KEY idx_members_created_at (created_at);

-- +goose Down
ALTER TABLE members DROP INDEX idx_members_created_at;
ALTER TABLE wallet_ledger_entries DROP INDEX idx_wallet_ledger_dashboard;
ALTER TABLE payment_orders DROP INDEX idx_payment_orders_dashboard;
