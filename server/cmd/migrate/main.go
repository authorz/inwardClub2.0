// Command migrate applies goose migrations from the embedded filesystem.
//
// Usage:
//
//	go run ./cmd/migrate up|down|status|version|reset
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"

	appdb "github.com/inwardclub/server/db"
	"github.com/inwardclub/server/internal/platform/config"
)

const migrationsDir = "migrations"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.MySQLDSN == "" {
		return fmt.Errorf("MYSQL_DSN is required")
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(appdb.Migrations)
	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}

	switch command {
	case "up":
		return goose.Up(db, migrationsDir)
	case "down":
		return goose.Down(db, migrationsDir)
	case "status":
		return goose.Status(db, migrationsDir)
	case "version":
		return goose.Version(db, migrationsDir)
	case "reset":
		return goose.Reset(db, migrationsDir)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}
