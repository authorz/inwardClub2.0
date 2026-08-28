# InwardClub 1.0 production data migration

`cmd/import-v1` replaces v2 operational data with the final v1 snapshot while
retaining v2-only configuration. The source must be restored into a read-only
MySQL database before running the command.

## Approved policy

- v1 is authoritative for members, balances, orders, recharge history, coupons,
  points workflows, store, catalog, tables and seats.
- Keep v2 admin accounts, roles, system/rule settings, membership tiers,
  recharge products, coupon categories/templates, assets and printer devices.
- Reassign all retained printer devices and store-scoped admin accounts to v1
  store ID 1. Delete every other store.
- Exclude all activity data. The snapshot contains zero activity definitions and
  1,479 orphan activity orders; reconciliation records this approved difference.
- Map `users.balance` to integer coins with `FLOOR`, `users.points` to points and
  `users.all_balance` to non-spendable growth value.
- Map legacy 88-yuan coupons to `alcohol`. Map the new-member package and old
  weekly/monthly event coupons to `event_ticket`, which uses direct redemption
  and receipt printing.
- Archive overlapping `balance_consumption_records` and `user_points` rows in
  `legacy_v1_archives`; `transaction_records` is the canonical v2 wallet ledger.

## Safety controls

The default command is read-only. Execution requires both `-execute` and the
literal confirmation value. A production execution cannot use `-skip-backup`.
The command creates a gzip-compressed consistent `mysqldump` and SHA-256 before
uploading assets or opening the destructive database transaction.

```bash
go run ./cmd/migrate up

go run ./cmd/import-v1 \
  -source-dsn "$V1_MYSQL_DSN" \
  -report ./tmp/v1-import-preflight.json

go run ./cmd/import-v1 \
  -source-dsn "$V1_MYSQL_DSN" \
  -execute \
  -confirm CLEAR_V2_OPERATIONAL_DATA \
  -run-key inwardclub-v1-final-20260828 \
  -backup-dir ./tmp/production-backups \
  -report ./tmp/v1-import-production.json
```

`-skip-assets` and `-skip-backup` exist only for isolated rehearsal databases.
Never use them for production.

## Required verification

Run the independent source/target verifier after the import commits:

```bash
go run ./cmd/verify-v1-import -source-dsn "$V1_MYSQL_DSN"
```

- Compare source and target counts for members, catalog, food orders/items,
  points workflows and coupon entitlements.
- Compare food and recharge amount totals in integer RMB cents.
- Compare wallet points, floored coins and growth-value totals.
- Require zero orphan references for members, orders, order items, coupon
  templates and wallet accounts.
- Verify one store remains, retained printers use store ID 1, activities are
  empty, and the activity exclusion is present in `reconciliation_results`.
- Verify active legacy alcohol and event coupons can be viewed; an event coupon
  use must create a redemption and receipt print job exactly once.
