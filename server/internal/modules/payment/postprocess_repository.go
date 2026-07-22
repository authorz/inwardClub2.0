package payment

// The payment:post-process worker. A settled, member-bound offline collection
// (spec §9.3.4) may earn the member configured benefits — coins/points/growth —
// evaluated asynchronously so the collection callback stays fast. This file is
// the DB-facing half: it loads the enabled low_spend_reward rule, computes its
// grants (postprocess.go) and applies them exactly once, keyed on
// member_id + payment_order_id + rule_version (spec §11: rule:{ruleVersion}:{order}).
//
// It reuses the recharge settlement's proven machinery: the wallet credit follows
// creditGrowthValue's account-upsert + idempotent-ledger shape, and a growth grant
// re-resolves the member's VIP tier through the same applyTierUpgrade / resolveTier
// path the recharge flow uses.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// PostProcessService evaluates one payment:post-process outbox hand-off. It owns
// payload decoding and validation, then delegates the load-rule-and-grant work to
// the repository, whose effects are idempotent under retry.
type PostProcessService struct {
	repo PostProcessRepository
	now  func() time.Time
}

// PostProcessRepository is the persistence port. The SQL implementation loads the
// rule and applies grants transactionally; tests supply an in-memory fake that
// mirrors that contract branch-for-branch.
type PostProcessRepository interface {
	// Process loads the applicable enabled rule and applies its grants for one
	// settled collection, idempotently, reporting what it did.
	Process(ctx context.Context, p postProcessPayload, now time.Time) (PostProcessResult, error)
}

// NewPostProcessService builds the service over a repository.
func NewPostProcessService(repo PostProcessRepository) *PostProcessService {
	return &PostProcessService{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// Process decodes a raw payment:post-process task payload and applies any earned
// benefits. A payload that cannot be decoded, or that is missing its member /
// payment identifiers, returns ErrUndecodablePostProcess so the worker drops it
// without retry. Everything else delegates to the repository.
func (s *PostProcessService) Process(ctx context.Context, raw []byte) (PostProcessResult, error) {
	p, err := parsePostProcessPayload(raw)
	if err != nil {
		return PostProcessResult{}, err
	}
	if !p.valid() {
		return PostProcessResult{}, fmt.Errorf("%w: missing memberId/paymentOrderId", ErrUndecodablePostProcess)
	}
	return s.repo.Process(ctx, p, s.now())
}

type postProcessSQLRepository struct{ db *platdb.DB }

// NewPostProcessRepository builds the MySQL post-process repository.
func NewPostProcessRepository(db *platdb.DB) PostProcessRepository {
	return &postProcessSQLRepository{db: db}
}

// postProcessRule is the minimal projection of the enabled low_spend_reward rule the
// worker needs: its version (part of the idempotency key) and raw config.
type postProcessRule struct {
	version int
	config  []byte
}

// Process loads the enabled low_spend_reward rule and applies its grants for one
// settled collection. With no enabled rule (the current production state, spec
// §13) it acknowledges the event without granting — a real terminal decision, not
// a skeleton. With a rule, all effects ride one transaction anchored on the
// rule_executions idempotency marker.
func (r *postProcessSQLRepository) Process(ctx context.Context, p postProcessPayload, now time.Time) (PostProcessResult, error) {
	rule, ok, err := r.loadConsumeRule(ctx, p.StoreID, now)
	if err != nil {
		return PostProcessResult{}, err
	}
	if !ok {
		return PostProcessResult{RuleMatched: false}, nil
	}
	cfg, err := parseConsumeRewardConfig(rule.config)
	if err != nil {
		// A published-but-malformed config is an operator error a retry cannot fix;
		// acknowledge (no grant) so the queue is not wedged on one bad rule.
		return PostProcessResult{RuleMatched: true, RuleVersion: rule.version}, nil
	}
	grants := computeGrants(cfg, p.AmountCent)

	res := PostProcessResult{RuleMatched: true, RuleVersion: rule.version}
	err = r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		// The rule_executions row is the spec's member_id + payment_order_id +
		// rule_version idempotency marker. Its unique idem_key means a replayed or
		// doubled event finds it already present and grants nothing.
		first, err := insertRuleExecution(ctx, tx, rule.version, p, now)
		if err != nil {
			return err
		}
		if !first {
			res.AlreadyDone = true
			return nil
		}
		for _, g := range grants {
			applied, err := applyConsumeGrant(ctx, tx, rule.version, p, g, now)
			if err != nil {
				return err
			}
			if applied {
				res.GrantsApplied++
			}
		}
		return nil
	})
	if err != nil {
		return PostProcessResult{}, err
	}
	return res, nil
}

// loadConsumeRule returns the enabled, published, currently-effective
// low_spend_reward rule that applies to this collection's store, or ok=false when
// none is enabled. It mirrors the sign-in rule lookup (wallet/points_repository.go)
// and extends it to the scope_type / store_id the rule_definitions table models: a
// store-scoped rule for this store wins over a global one, then the highest version.
func (r *postProcessSQLRepository) loadConsumeRule(ctx context.Context, storeID int64, now time.Time) (postProcessRule, bool, error) {
	const q = `SELECT version, config_json FROM rule_definitions
		WHERE rule_key = ? AND enabled = 1 AND status = 'published'
		  AND (scope_type = 'global' OR (scope_type = 'store' AND store_id = ?))
		  AND (effective_from IS NULL OR effective_from <= ?)
		  AND (effective_to IS NULL OR effective_to > ?)
		ORDER BY (scope_type = 'store') DESC, version DESC
		LIMIT 1`
	var rule postProcessRule
	switch err := r.db.QueryRowContext(ctx, q, lowSpendRewardRuleKey, storeID, now, now).Scan(&rule.version, &rule.config); {
	case errors.Is(err, sql.ErrNoRows):
		return postProcessRule{}, false, nil
	case err != nil:
		return postProcessRule{}, false, apperr.Internal(err)
	}
	return rule, true, nil
}

// insertRuleExecution records the idempotent processing marker for this
// (member, payment order, rule version). It reports first=false when the marker
// already exists — the event was already processed — via the unique idem_key
// (spec §11: rule:{ruleVersion}:{order}).
func insertRuleExecution(ctx context.Context, tx *sql.Tx, version int, p postProcessPayload, now time.Time) (bool, error) {
	idem := fmt.Sprintf("rule:%d:%d", version, p.PaymentOrderID)
	const q = `INSERT INTO rule_executions
		(rule_key, rule_version, member_id, source_type, source_id, result_json, idem_key, created_at)
		VALUES (?, ?, ?, ?, ?, NULL, ?, ?)`
	if _, err := tx.ExecContext(ctx, q, lowSpendRewardRuleKey, version, p.MemberID, p.Source, p.PaymentOrderID, idem, now); err != nil {
		if platdb.IsDuplicate(err) {
			return false, nil
		}
		return false, apperr.Internal(err)
	}
	return true, nil
}

// applyConsumeGrant records one benefit_grants row and credits the member's
// wallet for it, inside the caller's transaction. Both the grant row and the
// wallet ledger entry are keyed on rule:{version}:{paymentOrder}:{asset}, so a
// concurrent or replayed apply credits at most once. A growth grant additionally
// re-resolves the member's VIP tier from the new balance (upgrade-only), reusing
// the recharge settlement's tier machinery. It reports applied=false when the
// grant was already present.
func applyConsumeGrant(ctx context.Context, tx *sql.Tx, version int, p postProcessPayload, g benefitGrant, now time.Time) (bool, error) {
	idem := fmt.Sprintf("rule:%d:%d:%s", version, p.PaymentOrderID, g.Asset)
	const insGrant = `INSERT INTO benefit_grants
		(grant_no, member_id, benefit_type, amount, rule_key, rule_version, source_type, source_id, status, idem_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'granted', ?, ?)`
	if _, err := tx.ExecContext(ctx, insGrant, idem, p.MemberID, g.Asset, g.Amount, lowSpendRewardRuleKey, version, p.Source, p.PaymentOrderID, idem, now); err != nil {
		if platdb.IsDuplicate(err) {
			return false, nil // already granted for this (member, order, version, asset)
		}
		return false, apperr.Internal(err)
	}
	balance, err := creditConsumeAsset(ctx, tx, p.MemberID, g.Asset, g.Amount, p.PaymentOrderID, idem, now)
	if err != nil {
		return false, err
	}
	if g.Asset == growthAsset {
		if err := applyTierUpgrade(ctx, tx, p.MemberID, balance, now); err != nil {
			return false, err
		}
	}
	return true, nil
}

// creditConsumeAsset credits amount into the member's asset wallet (creating the
// account on first use) and appends the matching ledger entry, returning the new
// available balance. It mirrors the recharge settlement's wallet credit
// (creditGrowthValue) but is asset-generic and keyed on the caller's idem_key —
// the unique wallet_ledger_entries.idem_key is the final guard against a double
// credit.
func creditConsumeAsset(ctx context.Context, tx *sql.Tx, memberID int64, asset string, amount, sourceID int64, idemKey string, now time.Time) (int64, error) {
	var accountID, available int64
	const selAcct = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selAcct, memberID, asset).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		const insAcct = `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)`
		res, err := tx.ExecContext(ctx, insAcct, memberID, asset, now, now)
		if err != nil {
			return 0, apperr.Internal(err)
		}
		if accountID, err = res.LastInsertId(); err != nil {
			return 0, apperr.Internal(err)
		}
		available = 0
	case err != nil:
		return 0, apperr.Internal(err)
	}

	newBalance := available + amount
	const insLedger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, 'low_spend_reward', ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insLedger, accountID, memberID, asset, amount, newBalance, collectionType, sourceID, idemKey, now); err != nil {
		if platdb.IsDuplicate(err) {
			// The ledger already has this credit (a replay racing the grant row);
			// report the existing balance without re-applying.
			return available, nil
		}
		return 0, apperr.Internal(err)
	}

	const updateAcct = `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ?
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, updateAcct, newBalance, now, accountID); err != nil {
		return 0, apperr.Internal(err)
	}
	return newBalance, nil
}
