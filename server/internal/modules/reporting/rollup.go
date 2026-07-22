package reporting

import (
	"context"
	"time"
)

// Daily rollup metric names stored in reporting_daily.metric. Each metric is one
// pre-aggregated read model refreshed by the report:rollup worker.
const (
	MetricRevenue      = "revenue"
	MetricReservations = "reservations"
)

// RollupRequest bounds a rollup run. From/To are inclusive report-date edges in
// the stored (UTC) calendar; a nil edge is unbounded, so a zero-value request
// recomputes every date. A set StoreID pins the run to one store; a nil StoreID
// recomputes every store, including the store-less bucket for orders with no
// store (e.g. wallet recharges).
type RollupRequest struct {
	From    *time.Time
	To      *time.Time
	StoreID *int64
}

// RollupResult reports how many reporting_daily rows each metric wrote.
type RollupResult struct {
	RevenueRows     int64
	ReservationRows int64
}

// RollupRepository is the write port for the reporting_daily pipeline.
type RollupRepository interface {
	RollupDaily(ctx context.Context, req RollupRequest) (RollupResult, error)
}

// RollupService recomputes the reporting_daily pre-aggregates that back the
// revenue and reservation read models. It is invoked by the report:rollup worker
// (daily, plus a run on worker start) and is safe to re-run: the pipeline clears
// and rewrites the affected (date, store) partition, so any window converges to
// the live aggregate for that window.
type RollupService struct {
	repo RollupRepository
}

// NewRollupService builds the reporting rollup pipeline service.
func NewRollupService(repo RollupRepository) *RollupService { return &RollupService{repo: repo} }

// Rollup recomputes the daily aggregates for the requested bounds.
func (s *RollupService) Rollup(ctx context.Context, req RollupRequest) (RollupResult, error) {
	return s.repo.RollupDaily(ctx, req)
}
