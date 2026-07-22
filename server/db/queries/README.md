# db/queries

This directory is reserved for `sqlc` query definitions.

Current state (M0): repositories are written directly against `database/sql`
with hand-written SQL colocated in each module's `repository.go`. This is
wire-compatible with what `sqlc` generates and keeps funds/payment/inventory
writes on explicit SQL (no ORM auto-writes), satisfying the technical constraint.

Follow-up (tracked in the delivery report): introduce `sqlc.yaml` and move the
canonical SQL here so query structs are code-generated. No behavioural change is
expected — the repository interfaces already isolate SQL from services.
