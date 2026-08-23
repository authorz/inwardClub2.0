package rule

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeRepo is an in-memory Repository keyed by rule_key.
type fakeRepo struct {
	active map[string]Definition
	err    error
}

func (r *fakeRepo) ActiveRule(_ context.Context, ruleKey string, _ time.Time) (Definition, bool, error) {
	if r.err != nil {
		return Definition{}, false, r.err
	}
	d, ok := r.active[ruleKey]
	return d, ok, nil
}

// TestPostProcessNoRule: with no enabled 邀请 reward rule the evaluator grants
// nothing without error.
func TestPostProcessNoRule(t *testing.T) {
	svc := NewPostProcessService(&fakeRepo{active: map[string]Definition{}}, quietLogger())
	n, err := svc.Evaluate(context.Background(), EvalInput{MemberID: 1, PaymentOrderID: 9})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if n != 0 {
		t.Fatalf("granted = %d, want 0", n)
	}
}

// TestPostProcessEnabledStillNoGrant: an enabled invite-reward rule is surfaced
// but grants nothing (application pending business confirmation).
func TestPostProcessEnabledStillNoGrant(t *testing.T) {
	repo := &fakeRepo{active: map[string]Definition{
		KeyInviteReward: {RuleKey: KeyInviteReward, Version: 1, ConfigJSON: []byte(`{}`)},
	}}
	svc := NewPostProcessService(repo, quietLogger())
	n, err := svc.Evaluate(context.Background(), EvalInput{MemberID: 1, PaymentOrderID: 9})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if n != 0 {
		t.Fatalf("granted = %d, want 0 (grant application pending)", n)
	}
}

// TestPostProcessRepoError: a repository error propagates so asynq retries.
func TestPostProcessRepoError(t *testing.T) {
	svc := NewPostProcessService(&fakeRepo{err: errors.New("db down")}, quietLogger())
	if _, err := svc.Evaluate(context.Background(), EvalInput{}); err == nil {
		t.Fatal("expected propagated error")
	}
}
