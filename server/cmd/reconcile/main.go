// Command reconcile compares the v1 (source) and v2 (target) databases and emits
// a JSON report plus a human-readable Markdown summary. Phase-1 delivers the
// framework: table discovery, row counts and legacy-id-map coverage. Business
// balance/amount checks are added as their modules import data.
//
// Usage:
//
//	go run ./cmd/reconcile --source-dsn "$V1_MYSQL_DSN" --target-dsn "$MYSQL_DSN" --report ./tmp/reconciliation.json
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/inwardclub/server/internal/platform/config"
)

// Report is the machine-readable reconciliation output.
type Report struct {
	GeneratedAt   time.Time        `json:"generatedAt"`
	Source        *DatabaseSummary `json:"source,omitempty"`
	Target        *DatabaseSummary `json:"target,omitempty"`
	LegacyMapRows int64            `json:"legacyMapRows"`
	Diffs         []string         `json:"diffs"`
}

// DatabaseSummary lists per-table row counts for one database.
type DatabaseSummary struct {
	TableRowCounts map[string]int64 `json:"tableRowCounts"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "reconcile error:", err)
		os.Exit(1)
	}
}

func run() error {
	sourceDSN := flag.String("source-dsn", "", "v1 (source) MySQL DSN")
	targetDSN := flag.String("target-dsn", "", "v2 (target) MySQL DSN")
	reportPath := flag.String("report", "./tmp/reconciliation.json", "output JSON report path")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *sourceDSN == "" {
		*sourceDSN = os.Getenv("V1_MYSQL_DSN")
	}
	if *targetDSN == "" {
		*targetDSN = cfg.MySQLDSN
	}

	report := Report{GeneratedAt: time.Now().UTC(), Diffs: []string{}}

	if *sourceDSN != "" {
		summary, err := summarise(*sourceDSN)
		if err != nil {
			report.Diffs = append(report.Diffs, "source unreachable: "+err.Error())
		} else {
			report.Source = summary
		}
	} else {
		report.Diffs = append(report.Diffs, "source DSN not provided; skipped v1 discovery")
	}

	if *targetDSN != "" {
		summary, err := summarise(*targetDSN)
		if err != nil {
			report.Diffs = append(report.Diffs, "target unreachable: "+err.Error())
		} else {
			report.Target = summary
			report.LegacyMapRows = summary.TableRowCounts["legacy_id_maps"]
		}
	} else {
		report.Diffs = append(report.Diffs, "target DSN not provided; skipped v2 discovery")
	}

	if err := writeReport(*reportPath, report); err != nil {
		return err
	}
	printMarkdown(report)
	fmt.Printf("\nJSON report written to %s\n", *reportPath)
	return nil
}

// summarise discovers tables in the current schema and counts their rows.
func summarise(dsn string) (*DatabaseSummary, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(tables))
	for _, table := range tables {
		var count int64
		// #nosec G201 - table name comes from information_schema, not user input.
		if err := db.QueryRow("SELECT COUNT(*) FROM `" + table + "`").Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		counts[table] = count
	}
	return &DatabaseSummary{TableRowCounts: counts}, nil
}

func writeReport(path string, report Report) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func printMarkdown(report Report) {
	fmt.Println("# Reconciliation Report")
	fmt.Println("Generated:", report.GeneratedAt.Format(time.RFC3339))
	printSummary("Source (v1)", report.Source)
	printSummary("Target (v2)", report.Target)
	fmt.Printf("\nlegacy_id_maps rows: %d\n", report.LegacyMapRows)
	if len(report.Diffs) > 0 {
		fmt.Println("\n## Notes")
		for _, d := range report.Diffs {
			fmt.Println("-", d)
		}
	}
}

func printSummary(title string, summary *DatabaseSummary) {
	if summary == nil {
		return
	}
	fmt.Printf("\n## %s (%d tables)\n", title, len(summary.TableRowCounts))
	names := make([]string, 0, len(summary.TableRowCounts))
	for name := range summary.TableRowCounts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("- %s: %d\n", name, summary.TableRowCounts[name])
	}
}
