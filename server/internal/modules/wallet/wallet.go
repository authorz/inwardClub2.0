// Package wallet owns the append-only member ledger. Balances are the sum of
// effective ledger rows; corrections are new opposite rows, never UPDATEs.
// Phase-1 exposes read paths (wallet + ledger) for the mini program.
package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/inwardclub/server/internal/platform/audit"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Asset types held in member wallets. Definitions/consumability are governed by
// configurable rules, never hard-coded amounts.
const (
	AssetPoints      = "points"
	AssetCoins       = "coins"
	AssetCashBalance = "cash_balance"
	AssetGrowthValue = "growth_value"
)

// assetOrder is the stable display order for wallet accounts.
var assetOrder = []string{AssetPoints, AssetCoins, AssetCashBalance, AssetGrowthValue}

// Account is a member's balance for one asset type.
type Account struct {
	AssetType       string `json:"assetType"`
	AvailableAmount int64  `json:"availableAmount"`
	HeldAmount      int64  `json:"heldAmount"`
}

// LedgerEntry is one append-only ledger row.
type LedgerEntry struct {
	ID           int64     `json:"id"`
	AssetType    string    `json:"assetType"`
	Direction    string    `json:"direction"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balanceAfter"`
	Reason       string    `json:"reason"`
	SourceType   string    `json:"sourceType"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Adjustment directions for AdjustBalance.
const (
	DirectionCredit = "credit"
	DirectionDebit  = "debit"
)

// AdjustmentRequest is an admin-initiated wallet balance correction (e.g. a
// store console goodwill credit/debit), applied outside the member-facing
// sign-in/savings flows.
type AdjustmentRequest struct {
	AssetType string
	Direction string
	Amount    int64
	Reason    string
}

// Repository is the wallet persistence port.
type Repository interface {
	ListAccounts(ctx context.Context, memberID int64) (map[string]Account, error)
	ListLedger(ctx context.Context, memberID int64, assetType string, limit, offset int) ([]LedgerEntry, int64, error)
	// AdjustBalance applies an admin wallet adjustment inside a transaction,
	// crediting or debiting the member's account and appending the matching
	// ledger row. storeID identifies the store console that made the
	// adjustment (recorded as the ledger source); idemKey guards against a
	// retried request applying the adjustment twice.
	AdjustBalance(ctx context.Context, memberID, storeID int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error)
	// AdjustBalanceForAdmin applies the same adjustment on behalf of the
	// headquarters console, which is not scoped to a single store; the
	// ledger row's source_id is left NULL rather than pinned to a store.
	AdjustBalanceForAdmin(ctx context.Context, memberID int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL wallet repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) ListAccounts(ctx context.Context, memberID int64) (map[string]Account, error) {
	const q = `SELECT asset_type, available_amount, held_amount FROM wallet_accounts WHERE member_id = ?`
	rows, err := r.db.QueryContext(ctx, q, memberID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make(map[string]Account)
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.AssetType, &a.AvailableAmount, &a.HeldAmount); err != nil {
			return nil, apperr.Internal(err)
		}
		out[a.AssetType] = a
	}
	return out, rows.Err()
}

func (r *sqlRepository) ListLedger(ctx context.Context, memberID int64, assetType string, limit, offset int) ([]LedgerEntry, int64, error) {
	where := `member_id = ?`
	args := []any{memberID}
	if assetType != "" {
		where += ` AND asset_type = ?`
		args = append(args, assetType)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wallet_ledger_entries WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	q := `SELECT id, asset_type, direction, amount, balance_after, reason, source_type, created_at
		FROM wallet_ledger_entries WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.ID, &e.AssetType, &e.Direction, &e.Amount, &e.BalanceAfter, &e.Reason, &e.SourceType, &e.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// AdjustBalance applies an admin-initiated credit or debit to the member's
// wallet, scoped to the requesting store, and returns the affected account.
func (r *sqlRepository) AdjustBalance(ctx context.Context, memberID, storeID int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error) {
	return r.adjustBalance(ctx, memberID, &storeID, req, idemKey, auditEntry)
}

// AdjustBalanceForAdmin applies a headquarters-initiated credit or debit, not
// attributed to a single store.
func (r *sqlRepository) AdjustBalanceForAdmin(ctx context.Context, memberID int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error) {
	return r.adjustBalance(ctx, memberID, nil, req, idemKey, auditEntry)
}

// adjustBalance is the shared transactional core for AdjustBalance and
// AdjustBalanceForAdmin. A nil storeID is recorded as a NULL ledger source_id.
func (r *sqlRepository) adjustBalance(ctx context.Context, memberID int64, storeID *int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error) {
	now := time.Now().UTC()
	var out Account
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var accountID, available, held int64
		const selAcct = `SELECT id, available_amount, held_amount FROM wallet_accounts
			WHERE member_id = ? AND asset_type = ? FOR UPDATE`
		switch err := tx.QueryRowContext(ctx, selAcct, memberID, req.AssetType).Scan(&accountID, &available, &held); {
		case errors.Is(err, sql.ErrNoRows):
			const insAcct = `INSERT INTO wallet_accounts
				(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
				VALUES (?, ?, 0, 0, 0, ?, ?)`
			res, err := tx.ExecContext(ctx, insAcct, memberID, req.AssetType, now, now)
			if err != nil {
				return apperr.Internal(err)
			}
			if accountID, err = res.LastInsertId(); err != nil {
				return apperr.Internal(err)
			}
			available, held = 0, 0
		case err != nil:
			return apperr.Internal(err)
		}

		var newBalance int64
		switch req.Direction {
		case DirectionCredit:
			newBalance = available + req.Amount
		case DirectionDebit:
			if available < req.Amount {
				return apperr.Invalid("insufficient balance")
			}
			newBalance = available - req.Amount
		default:
			return apperr.Invalid("invalid direction")
		}

		const upd = `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`
		if _, err := tx.ExecContext(ctx, upd, newBalance, now, accountID); err != nil {
			return apperr.Internal(err)
		}

		reason := req.Reason
		if reason == "" {
			reason = "admin_adjustment"
		}
		const insLedger = `INSERT INTO wallet_ledger_entries
			(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'admin_adjustment', ?, ?, ?)`
		if _, err := tx.ExecContext(ctx, insLedger, accountID, memberID, req.AssetType, req.Direction, req.Amount, newBalance, reason, storeID, idemKey, now); err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("adjustment already recorded")
			}
			return apperr.Internal(err)
		}
		auditEntry.Before = Account{AssetType: req.AssetType, AvailableAmount: available, HeldAmount: held}
		auditEntry.After = Account{AssetType: req.AssetType, AvailableAmount: newBalance, HeldAmount: held}
		auditEntry.Reason = reason
		if err := audit.RecordTx(ctx, tx, auditEntry); err != nil {
			return err
		}
		out = Account{AssetType: req.AssetType, AvailableAmount: newBalance, HeldAmount: held}
		return nil
	})
	if err != nil {
		return Account{}, err
	}
	return out, nil
}

// Service provides wallet read operations.
type Service struct {
	repo Repository
}

// NewService builds the wallet service.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// GetWallet returns all four asset accounts, defaulting missing ones to zero so
// the client always sees a stable shape.
func (s *Service) GetWallet(ctx context.Context, memberID int64) ([]Account, error) {
	existing, err := s.repo.ListAccounts(ctx, memberID)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(assetOrder))
	for _, asset := range assetOrder {
		if a, ok := existing[asset]; ok {
			out = append(out, a)
		} else {
			out = append(out, Account{AssetType: asset})
		}
	}
	return out, nil
}

// Ledger returns a paginated ledger for the member.
func (s *Service) Ledger(ctx context.Context, memberID int64, assetType string, page httpx.Page) ([]LedgerEntry, int64, error) {
	return s.repo.ListLedger(ctx, memberID, assetType, page.Limit(), page.Offset())
}

// AdjustBalance validates and applies an admin-initiated wallet adjustment for
// a member, scoped to the requesting store. idemKey guards against a retried
// request applying the adjustment twice.
func (s *Service) AdjustBalance(ctx context.Context, memberID, storeID int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error) {
	if err := validateAdjustment(req, idemKey); err != nil {
		return Account{}, err
	}
	return s.repo.AdjustBalance(ctx, memberID, storeID, req, idemKey, auditEntry)
}

// AdjustBalanceForAdmin validates and applies a headquarters-initiated wallet
// adjustment for a member, not scoped to a single store. idemKey guards
// against a retried request applying the adjustment twice.
func (s *Service) AdjustBalanceForAdmin(ctx context.Context, memberID int64, req AdjustmentRequest, idemKey string, auditEntry audit.Entry) (Account, error) {
	if err := validateAdjustment(req, idemKey); err != nil {
		return Account{}, err
	}
	return s.repo.AdjustBalanceForAdmin(ctx, memberID, req, idemKey, auditEntry)
}

func validateAdjustment(req AdjustmentRequest, idemKey string) error {
	if idemKey == "" {
		return apperr.New(apperr.CodeIdempotencyRequired, "Idempotency-Key header is required")
	}
	if !validAsset(req.AssetType) {
		return apperr.Invalid("invalid assetType")
	}
	if req.Direction != DirectionCredit && req.Direction != DirectionDebit {
		return apperr.Invalid("direction must be credit or debit")
	}
	if req.Amount <= 0 {
		return apperr.Invalid("amount must be a positive number")
	}
	return nil
}

func validAsset(assetType string) bool {
	for _, a := range assetOrder {
		if a == assetType {
			return true
		}
	}
	return false
}
