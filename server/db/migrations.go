// Package db exposes the embedded goose migration filesystem so cmd/migrate and
// integration tests share one source of truth for schema.
package db

import "embed"

// Migrations holds the SQL migration files applied by goose.
//
//go:embed migrations/*.sql
var Migrations embed.FS
