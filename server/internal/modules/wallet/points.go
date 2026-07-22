package wallet

// Sign-in, points-savings and points-withdrawal operations for the mini program.
// They all mutate the member's points balance in the append-only ledger.
//
// Daily sign-in awards points immediately against sign_in_records + the wallet
// ledger (see points_repository.go). Points savings and withdrawals insert a
// 'pending' request row (point_savings / point_withdrawals), claimed on the
// idempotency key, for store-console review; no balance moves until approval.
// Request validation, member ownership (the id comes from the token, never the
// body) and idempotency-key presence are enforced here so the handlers stay thin.

import (
	"context"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// pointsMinAmount is the smallest points amount that may be saved or withdrawn.
const pointsMinAmount = 1

// Idempotency scopes for the points endpoints. The sign-in and savings/withdrawal
// request rows are claimed on the idempotency key, so a retried create is idempotent.
const (
	ScopeSignIn          = "mini/sign-ins"
	ScopePointSavings    = "mini/point-savings"
	ScopePointWithdrawal = "mini/point-withdrawals"
)

// SignInResult is returned after a daily sign-in.
type SignInResult struct {
	Date         string `json:"date"`
	PointsEarned int64  `json:"pointsEarned"`
	StreakDays   int    `json:"streakDays"`
}

// PointsTxnResult is returned after a points save or withdrawal request is
// created. The request is created pending review, so no balance has moved yet:
// Amount echoes the requested points, BalanceAfter is 0, and RequestID is the id
// of the created point_savings / point_withdrawals row.
type PointsTxnResult struct {
	AssetType    string `json:"assetType"`
	Amount       int64  `json:"amount"`
	BalanceAfter int64  `json:"balanceAfter"`
	RequestID    int64  `json:"requestId"`
	Status       string `json:"status"`
}

// PointsAmountRequest is the body for point-savings and point-withdrawals. The
// amount is a positive count of points; required rejects a zero/missing value
// and the service rejects anything below pointsMinAmount. StoreID is optional;
// when set the request is scoped to that store so its console can review it.
type PointsAmountRequest struct {
	Amount  int64 `json:"amount" binding:"required"`
	StoreID int64 `json:"storeId,omitempty"`
}

// PointsRepository is the write persistence port for the points operations. The
// idempotency key is threaded through so the implementation can claim it via the
// unique key on the request row.
type PointsRepository interface {
	RecordSignIn(ctx context.Context, memberID int64, idemKey string) (SignInResult, error)
	SavePoints(ctx context.Context, memberID, storeID, amount int64, idemKey string) (PointsTxnResult, error)
	WithdrawPoints(ctx context.Context, memberID, storeID, amount int64, idemKey string) (PointsTxnResult, error)
}

// PointsService validates and drives the points operations.
type PointsService struct {
	repo PointsRepository
}

// NewPointsService builds the points service.
func NewPointsService(repo PointsRepository) *PointsService { return &PointsService{repo: repo} }

// SignIn records the member's daily sign-in. Idempotency is enforced per member
// per day by the repository; the header is required so a retried request cannot
// double-award points.
func (s *PointsService) SignIn(ctx context.Context, memberID int64, idemKey string) (SignInResult, error) {
	if err := requireIdemKey(idemKey); err != nil {
		return SignInResult{}, err
	}
	return s.repo.RecordSignIn(ctx, memberID, idemKey)
}

// SavePoints creates a pending points-savings request for store-console review.
// No balance moves on creation; the request row is recorded against
// point_savings and settled when the store approves it.
func (s *PointsService) SavePoints(ctx context.Context, memberID int64, req PointsAmountRequest, idemKey string) (PointsTxnResult, error) {
	if err := requireIdemKey(idemKey); err != nil {
		return PointsTxnResult{}, err
	}
	if err := validatePointsAmount(req.Amount); err != nil {
		return PointsTxnResult{}, err
	}
	return s.repo.SavePoints(ctx, memberID, req.StoreID, req.Amount, idemKey)
}

// WithdrawPoints creates a pending points-withdrawal request for store-console
// review. Like SavePoints, no balance moves on creation; the request row is
// recorded against point_withdrawals and settled when the store approves it.
func (s *PointsService) WithdrawPoints(ctx context.Context, memberID int64, req PointsAmountRequest, idemKey string) (PointsTxnResult, error) {
	if err := requireIdemKey(idemKey); err != nil {
		return PointsTxnResult{}, err
	}
	if err := validatePointsAmount(req.Amount); err != nil {
		return PointsTxnResult{}, err
	}
	return s.repo.WithdrawPoints(ctx, memberID, req.StoreID, req.Amount, idemKey)
}

func requireIdemKey(key string) error {
	if key == "" {
		return apperr.New(apperr.CodeIdempotencyRequired, "Idempotency-Key header is required")
	}
	return nil
}

func validatePointsAmount(amount int64) error {
	if amount < pointsMinAmount {
		return apperr.Invalid("amount must be a positive number of points")
	}
	return nil
}
