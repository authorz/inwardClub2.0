package wallet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

const testIdemKey = "idem-123"

// fakePointsRepo records calls and returns a NOT_IMPLEMENTED sentinel. The
// sentinel is a test double (the real repository is fully implemented); the
// tests use it to assert that a valid request reached the repository unchanged.
type fakePointsRepo struct {
	signInCalls    int
	saveAmount     int64
	withdrawAmount int64
	lastIdemKey    string
}

func (r *fakePointsRepo) GetSignInStatus(_ context.Context, _ int64) (SignInStatus, error) {
	return SignInStatus{
		Date:             "2026-07-25",
		RewardPoints:     100,
		NextRewardPoints: 200,
		DailyRewards:     []int64{100, 200, 300},
	}, nil
}

func (r *fakePointsRepo) RecordSignIn(_ context.Context, _ int64, idemKey string) (SignInResult, error) {
	r.signInCalls++
	r.lastIdemKey = idemKey
	return SignInResult{}, apperr.NotImplemented("daily sign-in is not available yet")
}

func (r *fakePointsRepo) SavePoints(_ context.Context, _, _, amount int64, idemKey string) (PointsTxnResult, error) {
	r.saveAmount = amount
	r.lastIdemKey = idemKey
	return PointsTxnResult{}, apperr.NotImplemented("points savings is not available yet")
}

func (r *fakePointsRepo) WithdrawPoints(_ context.Context, _, _, amount int64, idemKey string) (PointsTxnResult, error) {
	r.withdrawAmount = amount
	r.lastIdemKey = idemKey
	return PointsTxnResult{}, apperr.NotImplemented("points withdrawal is not available yet")
}

func codeOf(err error) apperr.Code { return apperr.From(err).Code }

func TestSignInStatusReachesRepository(t *testing.T) {
	svc := NewPointsService(&fakePointsRepo{})
	got, err := svc.SignInStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if got.Date != "2026-07-25" || got.RewardPoints != 100 || len(got.DailyRewards) != 3 {
		t.Fatalf("unexpected status: %+v", got)
	}
}

func TestSignInRequiresIdemKey(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)

	if _, err := svc.SignIn(context.Background(), 1, ""); codeOf(err) != apperr.CodeIdempotencyRequired {
		t.Fatalf("expected IDEMPOTENCY_KEY_REQUIRED, got %v", err)
	}
	if repo.signInCalls != 0 {
		t.Fatalf("repository must not be called without an idempotency key")
	}
}

func TestSignInReachesRepository(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)

	_, err := svc.SignIn(context.Background(), 1, testIdemKey)
	if codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("expected NOT_IMPLEMENTED, got %v", err)
	}
	if repo.signInCalls != 1 || repo.lastIdemKey != testIdemKey {
		t.Fatalf("expected repo call with idem key, got calls=%d key=%q", repo.signInCalls, repo.lastIdemKey)
	}
}

func TestSavePointsValidatesAmount(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)
	ctx := context.Background()

	cases := []int64{0, -5}
	for _, amount := range cases {
		if _, err := svc.SavePoints(ctx, 1, PointsAmountRequest{Amount: amount}, testIdemKey); codeOf(err) != apperr.CodeInvalidArgument {
			t.Fatalf("amount %d: expected INVALID_ARGUMENT, got %v", amount, err)
		}
	}
	if repo.saveAmount != 0 {
		t.Fatalf("repository must not be called for invalid amounts")
	}
}

func TestSavePointsMissingIdemKey(t *testing.T) {
	svc := NewPointsService(&fakePointsRepo{})
	if _, err := svc.SavePoints(context.Background(), 1, PointsAmountRequest{Amount: 100}, ""); codeOf(err) != apperr.CodeIdempotencyRequired {
		t.Fatalf("expected IDEMPOTENCY_KEY_REQUIRED, got %v", err)
	}
}

func TestSavePointsRequiresStore(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)
	if _, err := svc.SavePoints(
		context.Background(), 1, PointsAmountRequest{Amount: 100}, testIdemKey,
	); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
	if repo.saveAmount != 0 {
		t.Fatal("repository must not be called without a store")
	}
}

func TestSavePointsReachesRepository(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)

	_, err := svc.SavePoints(context.Background(), 1, PointsAmountRequest{Amount: 100, StoreID: 7}, testIdemKey)
	if codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("expected NOT_IMPLEMENTED, got %v", err)
	}
	if repo.saveAmount != 100 {
		t.Fatalf("expected amount forwarded, got %d", repo.saveAmount)
	}
}

func TestWithdrawPointsValidatesAmount(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)

	if _, err := svc.WithdrawPoints(context.Background(), 1, PointsAmountRequest{Amount: 0}, testIdemKey); codeOf(err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
	if repo.withdrawAmount != 0 {
		t.Fatalf("repository must not be called for invalid amounts")
	}
}

func TestWithdrawPointsReachesRepository(t *testing.T) {
	repo := &fakePointsRepo{}
	svc := NewPointsService(repo)

	_, err := svc.WithdrawPoints(context.Background(), 1, PointsAmountRequest{Amount: 50, StoreID: 7}, testIdemKey)
	if codeOf(err) != apperr.CodeNotImplemented {
		t.Fatalf("expected NOT_IMPLEMENTED, got %v", err)
	}
	if repo.withdrawAmount != 50 {
		t.Fatalf("expected amount forwarded, got %d", repo.withdrawAmount)
	}
}

// newPointsContext builds a gin context wired like the real chain: authenticated
// claims and the idempotency key already resolved into the context.
func newPointsContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Set(httpx.CtxClaims, &authn.Claims{})
	c.Set(httpx.CtxIdemKey, testIdemKey)
	return c, rec
}

func TestHandlerSavePointsRejectsInvalidBody(t *testing.T) {
	repo := &fakePointsRepo{}
	h := NewHandler(nil, NewPointsService(repo))
	c, rec := newPointsContext(`{`)

	h.SavePoints(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed body, got %d", rec.Code)
	}
	if repo.saveAmount != 0 {
		t.Fatalf("repository must not be called for a malformed body")
	}
}

func TestHandlerSavePointsBindsAndReachesRepository(t *testing.T) {
	repo := &fakePointsRepo{}
	h := NewHandler(nil, NewPointsService(repo))
	c, rec := newPointsContext(`{"amount":100,"storeId":7}`)

	h.SavePoints(c)

	// The repository is a controlled NOT_IMPLEMENTED, so the bound request that
	// passed validation surfaces as 501 while the amount and idem key reach it.
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 from the stub repository, got %d", rec.Code)
	}
	if repo.saveAmount != 100 || repo.lastIdemKey != testIdemKey {
		t.Fatalf("expected amount/idem key forwarded, got amount=%d key=%q", repo.saveAmount, repo.lastIdemKey)
	}
}

func TestHandlerSignInReachesRepository(t *testing.T) {
	repo := &fakePointsRepo{}
	h := NewHandler(nil, NewPointsService(repo))
	c, rec := newPointsContext("")

	h.SignIn(c)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 from the stub repository, got %d", rec.Code)
	}
	if repo.signInCalls != 1 || repo.lastIdemKey != testIdemKey {
		t.Fatalf("expected sign-in to reach repo with idem key, got calls=%d key=%q", repo.signInCalls, repo.lastIdemKey)
	}
}
