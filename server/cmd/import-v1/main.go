// Command import-v1 replaces v2 operational data with the final v1 snapshot.
// It preserves v2-only configuration, records legacy ID mappings, and emits a
// reconciliation report. The command is read-only unless both destructive
// confirmation flags are supplied.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/inwardclub/server/internal/platform/config"
	appdb "github.com/inwardclub/server/internal/platform/db"
)

const clearConfirmation = "CLEAR_V2_OPERATIONAL_DATA"

type options struct {
	sourceDSN  string
	sourceBase string
	execute    bool
	confirm    string
	runKey     string
	report     string
	skipAssets bool
	backupDir  string
	skipBackup bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "import-v1 error:", err)
		os.Exit(1)
	}
}

func run() error {
	var opts options
	flag.StringVar(&opts.sourceDSN, "source-dsn", os.Getenv("V1_MYSQL_DSN"), "read-only v1 MySQL DSN")
	flag.StringVar(&opts.sourceBase, "source-base", "https://api.inwardclub.com/storage/", "base URL for v1 images")
	flag.BoolVar(&opts.execute, "execute", false, "perform the destructive target replacement")
	flag.StringVar(&opts.confirm, "confirm", "", "must equal "+clearConfirmation+" when -execute is set")
	flag.StringVar(&opts.runKey, "run-key", "", "migration run key (required for execution)")
	flag.StringVar(&opts.report, "report", "./tmp/v1-import-report.json", "JSON report path")
	flag.BoolVar(&opts.skipAssets, "skip-assets", false, "skip image transfer (rehearsal only)")
	flag.StringVar(&opts.backupDir, "backup-dir", "./tmp/production-backups", "directory for the pre-migration target backup")
	flag.BoolVar(&opts.skipBackup, "skip-backup", false, "skip target backup (rehearsal only)")
	flag.Parse()

	if opts.sourceDSN == "" {
		return errors.New("-source-dsn or V1_MYSQL_DSN is required")
	}
	if opts.execute && opts.confirm != clearConfirmation {
		return fmt.Errorf("-execute requires -confirm %s", clearConfirmation)
	}
	if opts.execute && opts.runKey == "" {
		return errors.New("-run-key is required with -execute")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.MySQLDSN == "" {
		return errors.New("MYSQL_DSN is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()
	source, err := sql.Open("mysql", opts.sourceDSN)
	if err != nil {
		return fmt.Errorf("open v1 source: %w", err)
	}
	defer source.Close()
	if err := source.PingContext(ctx); err != nil {
		return fmt.Errorf("ping v1 source: %w", err)
	}
	target, err := appdb.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		return fmt.Errorf("open v2 target: %w", err)
	}
	defer target.Close()

	importer := newImporter(source, target, opts.runKey, cfg, opts.sourceBase, opts.skipAssets)
	if err := importer.preflight(ctx); err != nil {
		return err
	}
	if opts.execute {
		if cfg.AppEnv == "production" && opts.skipBackup {
			return errors.New("-skip-backup is forbidden when APP_ENV=production")
		}
		if !opts.skipBackup {
			backup, err := backupMySQL(ctx, cfg.MySQLDSN, opts.backupDir)
			if err != nil {
				return err
			}
			importer.backupPath, importer.backupSHA256 = backup.path, backup.sha256
		}
		if err := importer.execute(ctx); err != nil {
			return err
		}
	}
	report := importer.report(opts.execute)
	if err := writeJSON(opts.report, report); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(data))
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}
