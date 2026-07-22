// Package db owns the MySQL connection pool and the single transaction helper
// every module uses. Modules never open their own connections; funds/payment/
// inventory writes always run through WithinTx so they share one atomic scope.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// DB wraps *sql.DB with the transaction helper.
type DB struct {
	*sql.DB
}

// Open configures and verifies a MySQL pool. The DSN must request parseTime and
// UTC so all times round-trip as RFC3339 UTC.
func Open(ctx context.Context, dsn string) (*DB, error) {
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &DB{DB: sqlDB}, nil
}

// WithinTx runs fn inside a transaction, retrying a bounded number of times on
// MySQL deadlocks (error 1213). It commits on success and rolls back on error.
func (d *DB) WithinTx(ctx context.Context, fn func(*sql.Tx) error) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := d.runTx(ctx, fn)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isDeadlock(err) {
			return err
		}
	}
	return fmt.Errorf("transaction failed after retries: %w", lastErr)
}

func (d *DB) runTx(ctx context.Context, fn func(*sql.Tx) error) (err error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func isDeadlock(err error) bool {
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == 1213 || myErr.Number == 1205
	}
	return false
}

// IsDuplicate reports whether err is a MySQL duplicate-key violation (1062).
// Idempotency keys and unique business numbers rely on this as the final guard.
func IsDuplicate(err error) bool {
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == 1062
	}
	return false
}
