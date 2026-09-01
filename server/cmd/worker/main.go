// Command worker runs the asynq background task server. It also runs the outbox
// dispatcher, which relays committed outbox_events into the asynq queue, and a
// scheduler that enqueues the daily reporting rollup. Every task type now has a
// concrete handler: print:receipt, report:rollup, the expiry sweeps,
// payment:post-process, asset:pending-cleanup, the recurring VIP benefit sweep,
// and the legacy rule:post-process compatibility evaluator.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/inwardclub/server/internal/modules/asset"
	"github.com/inwardclub/server/internal/modules/coupon"
	"github.com/inwardclub/server/internal/modules/order"
	"github.com/inwardclub/server/internal/modules/payment"
	"github.com/inwardclub/server/internal/modules/printer"
	"github.com/inwardclub/server/internal/modules/reporting"
	"github.com/inwardclub/server/internal/modules/reservation"
	"github.com/inwardclub/server/internal/modules/rule"
	"github.com/inwardclub/server/internal/modules/systemsetting"
	"github.com/inwardclub/server/internal/modules/vipbenefit"
	"github.com/inwardclub/server/internal/platform/config"
	platdb "github.com/inwardclub/server/internal/platform/db"
	"github.com/inwardclub/server/internal/platform/logger"
	"github.com/inwardclub/server/internal/platform/outbox"
	platredis "github.com/inwardclub/server/internal/platform/redis"
)

// workerMaxRetry is the asynq per-task retry budget the dispatcher sets when
// enqueueing outbox events.
const workerMaxRetry = 25

// reportRollupCron enqueues the daily reporting rollup shortly after the
// business midnight, so each completed day is aggregated into reporting_daily
// once. The scheduler resolves the cron in the business zone.
const reportRollupCron = "5 0 * * *"

// vipMonthlyBenefitCron enqueues the daily benefit:vip-monthly evaluation just
// after the business midnight (following the rollup). Resolved in the business
// zone. Natural-period idempotency makes a missed or doubled tick safe.
const vipMonthlyBenefitCron = "30 0 * * *"

// Task type names mirror the scheduled jobs in the spec (§11).
const (
	TaskPaymentPostProcess   = "payment:post-process"
	TaskOfflineExpire        = "offline-collection:expire"
	TaskPrint                = "print:receipt"
	TaskReservationExpire    = "reservation:expire"
	TaskSeatReservationReset = "reservation:seat-reset"
	TaskActivityOrderExpire  = "activity-order:expire"
	TaskTicketCouponExpire   = "ticket-coupon:expire"
	TaskVipMonthlyBenefit    = "benefit:vip-monthly"
	TaskRulePostProcess      = "rule:post-process"
	TaskAssetPendingCleanup  = "asset:pending-cleanup"
	TaskReportRollup         = "report:rollup"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "worker error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel)
	if cfg.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required for the worker")
	}
	if cfg.MySQLDSN == "" {
		return fmt.Errorf("MYSQL_DSN is required for the worker outbox dispatcher")
	}

	// ctx is cancelled on SIGINT/SIGTERM and stops the dispatcher loop; the asynq
	// server installs its own signal handling and shuts down on the same signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := platdb.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer database.Close()

	client := asynq.NewClient(platredis.AsynqOpt(cfg.RedisAddr))
	defer client.Close()

	dispatcher := outbox.NewDispatcher(
		outbox.NewSQLStore(database),
		outbox.NewAsynqEnqueuer(client, workerMaxRetry),
		log,
	)
	go dispatcher.Run(ctx)

	srv := asynq.NewServer(
		platredis.AsynqOpt(cfg.RedisAddr),
		asynq.Config{
			Concurrency: 10,
			Logger:      &asynqLogger{log: log},
		},
	)

	// Printer: real Xpyun client when fakes are off, in-process fake otherwise.
	// The print:receipt handler executes jobs through it.
	globalSettingsSvc := systemsetting.NewService(systemsetting.NewRepository(database))
	printerMode := "fake"
	if cfg.PrinterReal() {
		printerMode = "real"
	} else {
		log.Warn("printer adapter is fake; Xpyun will not receive print jobs")
	}
	log.Info("printer adapter configured", "mode", printerMode)
	receiptPrinter := printer.SelectWithSettings(globalSettingsSvc, cfg.Xpyun, !cfg.PrinterReal())
	printJobRecorder := printer.NewJobRepository(database)

	// Payment post-process: evaluates a settled, member-bound offline collection
	// against the enabled low_spend_reward rule and grants coins/points/growth
	// (spec §9.3.4). With no enabled rule it acknowledges the event without
	// granting; see docs/payment-post-process.md for the §13 seam.
	postProcessSvc := payment.NewPostProcessService(payment.NewPostProcessRepository(database))

	// Reporting rollup: refreshes the reporting_daily pre-aggregates that back
	// the admin/store revenue and reservation reports.
	rollupSvc := reporting.NewRollupService(reporting.NewRollupRepository(database))

	// Pure-DB expiry sweeps (spec §11). Each is built from just its repository —
	// the sweeps only transition DB state and need no adapters. ticket/coupon
	// covers both member-held vouchers: activity tickets and coupon entitlements.
	reservationRepo := reservation.NewRepository(database)
	reservationExpiry := reservation.NewExpiryService(reservationRepo)
	seatReset := reservation.NewSeatResetService(reservationRepo, businessLocation(cfg, log))
	couponExpiry := coupon.NewExpiryService(coupon.NewRepository(database))
	orderExpiry := order.NewExpiryService(order.NewRepository(database))
	collectionExpiry := payment.NewCollectionExpiryService(payment.NewStoreRepository(database))
	ticketCouponSweep := func(ctx context.Context) (int64, error) {
		tickets, err := orderExpiry.SweepExpiredTickets(ctx)
		if err != nil {
			return tickets, err
		}
		coupons, err := couponExpiry.SweepExpired(ctx)
		return tickets + coupons, err
	}

	// Asset pending-cleanup: hourly GC that abandons assets stuck 'pending' past
	// the upload window (Qiniu spec §8). Pure-DB and idempotent like the sweeps.
	assetCleanup := asset.NewCleanupService(asset.NewRepository(database))

	// The daily VIP sweep materialises configured daily/weekly/monthly grants.
	// rule:post-process remains a no-producer compatibility handler; WeChat
	// invitation rewards are applied synchronously during settlement.
	ruleRepo := rule.NewRepository(database)
	vipMonthly := vipbenefit.NewService(database)
	invitePostProcess := rule.NewPostProcessService(ruleRepo, log)

	mux := asynq.NewServeMux()
	for _, task := range allTasks() {
		switch task {
		case TaskPaymentPostProcess:
			mux.HandleFunc(task, postProcessHandler(log, postProcessSvc))
		case TaskPrint:
			mux.HandleFunc(task, printHandler(log, receiptPrinter, printJobRecorder))
		case TaskReportRollup:
			mux.HandleFunc(task, rollupHandler(log, rollupSvc))
		case TaskReservationExpire:
			mux.HandleFunc(task, sweepHandler(log, task, reservationExpiry.SweepExpired))
		case TaskSeatReservationReset:
			mux.HandleFunc(task, sweepHandler(log, task, seatReset.Sweep))
		case TaskActivityOrderExpire:
			mux.HandleFunc(task, sweepHandler(log, task, orderExpiry.SweepExpiredActivityOrders))
		case TaskOfflineExpire:
			mux.HandleFunc(task, sweepHandler(log, task, collectionExpiry.SweepExpired))
		case TaskTicketCouponExpire:
			mux.HandleFunc(task, sweepHandler(log, task, ticketCouponSweep))
		case TaskAssetPendingCleanup:
			mux.HandleFunc(task, sweepHandler(log, task, assetCleanup.SweepPending))
		case TaskVipMonthlyBenefit:
			mux.HandleFunc(task, vipMonthlyHandler(log, vipMonthly))
		case TaskRulePostProcess:
			mux.HandleFunc(task, rulePostProcessHandler(log, invitePostProcess))
		default:
			mux.HandleFunc(task, logHandler(log, task))
		}
	}

	// Scheduler: enqueue report:rollup daily in the business zone, plus once now
	// so a freshly started worker populates reporting_daily without waiting for
	// midnight. Both carry an empty payload, i.e. a full recompute.
	scheduler := asynq.NewScheduler(platredis.AsynqOpt(cfg.RedisAddr), &asynq.SchedulerOpts{Location: businessLocation(cfg, log)})
	if _, err := scheduler.Register(reportRollupCron, asynq.NewTask(TaskReportRollup, nil)); err != nil {
		return fmt.Errorf("register report:rollup schedule: %w", err)
	}
	if _, err := scheduler.Register(vipMonthlyBenefitCron, asynq.NewTask(TaskVipMonthlyBenefit, nil)); err != nil {
		return fmt.Errorf("register benefit:vip-monthly schedule: %w", err)
	}
	if _, err := scheduler.Register("0 0 * * *", asynq.NewTask(TaskSeatReservationReset, nil)); err != nil {
		return fmt.Errorf("register reservation:seat-reset schedule: %w", err)
	}
	// Expiry sweeps on their spec §11 cadence: per-minute expiries, hourly
	// ticket/coupon and asset pending-cleanup. Each sweep is idempotent, so a
	// missed or doubled tick is safe.
	for _, e := range expirySchedule() {
		if _, err := scheduler.Register(e.cron, asynq.NewTask(e.task, nil)); err != nil {
			return fmt.Errorf("register %s schedule: %w", e.task, err)
		}
	}
	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}
	defer scheduler.Shutdown()

	if _, err := client.Enqueue(asynq.NewTask(TaskReportRollup, nil)); err != nil {
		log.Warn("startup report:rollup enqueue skipped", "error", err)
	}
	if _, err := client.Enqueue(asynq.NewTask(TaskVipMonthlyBenefit, nil)); err != nil {
		log.Warn("startup benefit:vip-monthly enqueue skipped", "error", err)
	}
	if _, err := client.Enqueue(asynq.NewTask(TaskSeatReservationReset, nil)); err != nil {
		log.Warn("startup reservation:seat-reset enqueue skipped", "error", err)
	}

	log.Info("worker starting", "redis", cfg.RedisAddr, "tasks", len(allTasks()))
	return srv.Run(mux)
}

// businessLocation resolves the configured business zone for the scheduler,
// falling back to UTC (with a warning) if the clock cannot be built.
func businessLocation(cfg *config.Config, log *slog.Logger) *time.Location {
	clock, err := cfg.BusinessClock()
	if err != nil {
		log.Warn("business clock unavailable; scheduling rollup in UTC", "error", err)
		return time.UTC
	}
	return clock.Location()
}

func allTasks() []string {
	return []string{
		TaskPaymentPostProcess, TaskOfflineExpire, TaskPrint, TaskReservationExpire, TaskSeatReservationReset,
		TaskActivityOrderExpire, TaskTicketCouponExpire, TaskVipMonthlyBenefit,
		TaskRulePostProcess, TaskAssetPendingCleanup, TaskReportRollup,
	}
}

// scheduleEntry pairs a cron spec with the task type it enqueues.
type scheduleEntry struct {
	cron string
	task string
}

// expirySchedule is the periodic-enqueue plan for the pure-DB expiry batch,
// matching the triggers in spec §11.
func expirySchedule() []scheduleEntry {
	return []scheduleEntry{
		{"* * * * *", TaskReservationExpire},
		{"* * * * *", TaskActivityOrderExpire},
		{"* * * * *", TaskOfflineExpire},
		{"@every 1h", TaskTicketCouponExpire},
		{"@every 1h", TaskAssetPendingCleanup},
	}
}

// sweepHandler adapts an expiry sweep (which returns the number of rows it
// transitioned) to an asynq handler. A failed sweep returns the error so asynq
// retries it; every sweep is idempotent, so a retry or a doubled schedule tick
// never double-applies.
func sweepHandler(log *slog.Logger, task string, sweep func(context.Context) (int64, error)) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, _ *asynq.Task) error {
		n, err := sweep(ctx)
		if err != nil {
			log.Error("expiry sweep failed", "type", task, "error", err.Error())
			return err
		}
		log.Info("expiry sweep complete", "type", task, "expired", n)
		return nil
	}
}

type scheduledVIPBenefitService interface {
	SweepScheduled(context.Context) (int64, error)
}

// vipMonthlyHandler runs the daily configured VIP benefit sweep. Natural-period
// idempotency makes daily retries safe for daily, weekly and monthly rules.
func vipMonthlyHandler(log *slog.Logger, svc scheduledVIPBenefitService) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, _ *asynq.Task) error {
		granted, err := svc.SweepScheduled(ctx)
		if err != nil {
			log.Error("benefit:vip-monthly failed", "error", err.Error())
			return err
		}
		log.Info("benefit:vip-monthly complete", "granted", granted)
		return nil
	}
}

// rulePostProcessHandler keeps legacy invite-reward tasks parseable. The payload
// is a rule.EvalInput (the settled-order facts). Malformed JSON is
// dropped without retry (asynq.SkipRetry) since a retry cannot fix it; an empty
// payload decodes to a zero input and evaluates to a no-op. Nothing enqueues
// rule:post-process yet, so today this path is exercised by tests.
func rulePostProcessHandler(log *slog.Logger, svc *rule.PostProcessService) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		in, err := parsePostProcessInput(t.Payload())
		if err != nil {
			log.Error("rule:post-process: undecodable payload dropped", "error", err, "payload", string(t.Payload()))
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}
		granted, err := svc.Evaluate(ctx, in)
		if err != nil {
			log.Error("rule:post-process failed", "error", err.Error())
			return err
		}
		log.Info("rule:post-process complete", "granted", granted, "paymentOrderId", in.PaymentOrderID)
		return nil
	}
}

// parsePostProcessInput decodes a rule:post-process task payload into a
// rule.EvalInput. An empty payload is a zero input (a no-op evaluation); invalid
// JSON is an error the handler maps to asynq.SkipRetry.
func parsePostProcessInput(payload []byte) (rule.EvalInput, error) {
	var in rule.EvalInput
	if len(payload) == 0 {
		return in, nil
	}
	if err := json.Unmarshal(payload, &in); err != nil {
		return in, err
	}
	return in, nil
}

func logHandler(log *slog.Logger, task string) func(context.Context, *asynq.Task) error {
	return func(_ context.Context, t *asynq.Task) error {
		log.Info("task received", "type", task, "payload", string(t.Payload()))
		return nil
	}
}

// printHandler executes a print:receipt task through the configured Printer. The
// payload is a printer.Job. A bad payload is dropped (returning nil so asynq does
// not retry a message it can never decode); a provider error is returned so
// asynq retries per the dispatcher's retry budget. The producer is wired: the
// SettleWeChat / SettleOffline / SettleByCoin settlement paths append a
// print:receipt outbox event (printer.WriteReceipt) for store-bound orders, which
// the dispatcher relays here (see docs/adapters.md §3.1).
func printHandler(log *slog.Logger, p printer.Printer, recorders ...printer.JobRecorder) func(context.Context, *asynq.Task) error {
	var recorder printer.JobRecorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}
	return func(ctx context.Context, t *asynq.Task) error {
		var job printer.Job
		if err := json.Unmarshal(t.Payload(), &job); err != nil {
			log.Error("print task: undecodable payload dropped", "error", err, "payload", string(t.Payload()))
			return nil
		}
		if recorder != nil && job.ID <= 0 {
			if taskID, ok := asynq.GetTaskID(ctx); ok {
				jobID, err := recorder.Ensure(ctx, taskID, job)
				if err != nil {
					log.Error("print task: could not create legacy log", "error", err, "task_id", taskID)
					return err
				}
				job.ID = jobID
			}
		}
		if recorder != nil && job.ID > 0 {
			if err := recorder.StartAttempt(ctx, job.ID); err != nil {
				log.Error("print task: could not record attempt", "error", err, "job_id", job.ID)
				return err
			}
		}
		if err := p.Print(ctx, job); err != nil {
			if recorder != nil && job.ID > 0 {
				if recordErr := recorder.MarkFailed(ctx, job.ID, err); recordErr != nil {
					log.Error("print task: could not record failure", "error", recordErr, "job_id", job.ID)
				}
			}
			log.Error("print task: print failed", "error", err, "sn", job.DeviceSN)
			return err
		}
		if recorder != nil && job.ID > 0 {
			if err := recorder.MarkPrinted(ctx, job.ID); err != nil {
				// The receipt has already reached the provider. A log-write failure
				// must not retry and risk printing the same receipt twice.
				log.Error("print task: printed but status update failed", "error", err, "job_id", job.ID)
			}
		}
		message := "print task: printed"
		if _, simulated := p.(*printer.FakePrinter); simulated {
			message = "print task: simulated"
		}
		log.Info(message, "sn", job.DeviceSN, "template", job.Template)
		return nil
	}
}

// postProcessHandler evaluates a settled, member-bound offline collection against
// the enabled consume_reward rule and grants the member its configured benefits —
// coins/points/growth (spec §9.3.4). The payload is the settlement's
// postProcessPayload. An undecodable or structurally-invalid payload is dropped
// without retry (asynq.SkipRetry, since a retry cannot fix it); a DB error is
// returned so asynq retries under the dispatcher's budget. When no consume_reward
// rule is enabled (the current state, spec §13) the handler acknowledges the event
// after recording that no rule applied. All grant effects are idempotent, so a
// retried or doubled event never double-credits.
func postProcessHandler(log *slog.Logger, svc *payment.PostProcessService) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		res, err := svc.Process(ctx, t.Payload())
		if err != nil {
			if errors.Is(err, payment.ErrUndecodablePostProcess) {
				log.Error("post-process task: undecodable payload dropped", "error", err, "payload", string(t.Payload()))
				return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
			}
			log.Error("post-process task: failed", "error", err)
			return err
		}
		log.Info("post-process task: complete",
			"ruleMatched", res.RuleMatched, "ruleVersion", res.RuleVersion,
			"grantsApplied", res.GrantsApplied, "alreadyDone", res.AlreadyDone)
		return nil
	}
}

// rollupHandler recomputes the reporting_daily pre-aggregates for a
// report:rollup task. An empty payload (the scheduled/startup case) recomputes
// every date and store; a payload may pin the window (see rollupPayload). An
// undecodable payload is dropped without retry (asynq.SkipRetry) since a retry
// cannot fix it; a pipeline error is returned so asynq retries.
func rollupHandler(log *slog.Logger, svc *reporting.RollupService) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		req, err := parseRollupRequest(t.Payload())
		if err != nil {
			log.Error("rollup task: undecodable payload dropped", "error", err, "payload", string(t.Payload()))
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}
		res, err := svc.Rollup(ctx, req)
		if err != nil {
			log.Error("rollup task: failed", "error", err)
			return err
		}
		log.Info("rollup task: complete",
			"revenueRows", res.RevenueRows, "reservationRows", res.ReservationRows,
			"from", boundStr(req.From), "to", boundStr(req.To), "storeId", storeStr(req.StoreID))
		return nil
	}
}

// rollupPayload is the optional report:rollup task payload. Date pins a single
// day; From/To pin an inclusive range; StoreID pins one store. Dates are
// calendar days (YYYY-MM-DD). Any field left empty widens the window, so an
// empty payload is a full recompute.
type rollupPayload struct {
	Date    string `json:"date"`
	From    string `json:"from"`
	To      string `json:"to"`
	StoreID *int64 `json:"storeId"`
}

const rollupDateLayout = "2006-01-02"

// parseRollupRequest maps a raw task payload to a reporting.RollupRequest.
func parseRollupRequest(payload []byte) (reporting.RollupRequest, error) {
	var req reporting.RollupRequest
	if len(payload) == 0 {
		return req, nil
	}
	var p rollupPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return req, err
	}
	if p.Date != "" {
		d, err := time.Parse(rollupDateLayout, p.Date)
		if err != nil {
			return req, err
		}
		req.From, req.To = &d, &d
	} else {
		if p.From != "" {
			d, err := time.Parse(rollupDateLayout, p.From)
			if err != nil {
				return req, err
			}
			req.From = &d
		}
		if p.To != "" {
			d, err := time.Parse(rollupDateLayout, p.To)
			if err != nil {
				return req, err
			}
			req.To = &d
		}
	}
	req.StoreID = p.StoreID
	return req, nil
}

func boundStr(t *time.Time) string {
	if t == nil {
		return "*"
	}
	return t.Format(rollupDateLayout)
}

func storeStr(id *int64) string {
	if id == nil {
		return "*"
	}
	return fmt.Sprintf("%d", *id)
}

// asynqLogger adapts slog to the asynq.Logger interface.
type asynqLogger struct{ log *slog.Logger }

func (l *asynqLogger) Debug(args ...any) { l.log.Debug(fmt.Sprint(args...)) }
func (l *asynqLogger) Info(args ...any)  { l.log.Info(fmt.Sprint(args...)) }
func (l *asynqLogger) Warn(args ...any)  { l.log.Warn(fmt.Sprint(args...)) }
func (l *asynqLogger) Error(args ...any) { l.log.Error(fmt.Sprint(args...)) }
func (l *asynqLogger) Fatal(args ...any) { l.log.Error(fmt.Sprint(args...)) }
