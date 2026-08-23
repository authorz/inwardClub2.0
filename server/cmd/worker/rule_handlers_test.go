package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/inwardclub/server/internal/modules/rule"
)

// stubRuleRepo is a rule.Repository that returns a canned active rule (or none)
// and an optional error, so the worker's rule handlers can be exercised without
// a DB. quietLogger is shared from main_test.go (same package).
type stubRuleRepo struct {
	active map[string]rule.Definition
	err    error
}

type stubVIPBenefitService struct {
	granted int64
	err     error
}

func (s *stubVIPBenefitService) SweepScheduled(context.Context) (int64, error) {
	return s.granted, s.err
}

func (s *stubRuleRepo) ActiveRule(_ context.Context, key string, _ time.Time) (rule.Definition, bool, error) {
	if s.err != nil {
		return rule.Definition{}, false, s.err
	}
	d, ok := s.active[key]
	return d, ok, nil
}

func TestVipMonthlyHandlerCompletes(t *testing.T) {
	svc := &stubVIPBenefitService{granted: 3}
	handler := vipMonthlyHandler(quietLogger(), svc)
	if err := handler(context.Background(), asynq.NewTask(TaskVipMonthlyBenefit, nil)); err != nil {
		t.Fatalf("vipMonthlyHandler: %v", err)
	}
}

// TestVipMonthlyHandlerPropagatesError: a repository error is returned so asynq
// retries the daily evaluation.
func TestVipMonthlyHandlerPropagatesError(t *testing.T) {
	svc := &stubVIPBenefitService{err: errors.New("db down")}
	handler := vipMonthlyHandler(quietLogger(), svc)
	if err := handler(context.Background(), asynq.NewTask(TaskVipMonthlyBenefit, nil)); err == nil {
		t.Fatal("expected propagated error")
	}
}

// TestRulePostProcessHandlerNoRule: a well-formed payload with no enabled invite
// rule evaluates to a no-op and returns nil.
func TestRulePostProcessHandlerNoRule(t *testing.T) {
	svc := rule.NewPostProcessService(&stubRuleRepo{active: map[string]rule.Definition{}}, quietLogger())
	handler := rulePostProcessHandler(quietLogger(), svc)

	payload, _ := json.Marshal(rule.EvalInput{MemberID: 7, PaymentOrderID: 42, AmountCent: 1000})
	if err := handler(context.Background(), asynq.NewTask(TaskRulePostProcess, payload)); err != nil {
		t.Fatalf("rulePostProcessHandler: %v", err)
	}
}

// TestRulePostProcessHandlerBadPayload: undecodable JSON is dropped with
// asynq.SkipRetry (a retry can never fix it).
func TestRulePostProcessHandlerBadPayload(t *testing.T) {
	svc := rule.NewPostProcessService(&stubRuleRepo{active: map[string]rule.Definition{}}, quietLogger())
	handler := rulePostProcessHandler(quietLogger(), svc)

	err := handler(context.Background(), asynq.NewTask(TaskRulePostProcess, []byte("{bad")))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("expected SkipRetry, got %v", err)
	}
}

// TestParsePostProcessInput: nil/empty payload is a zero input; valid JSON parses
// its fields; malformed JSON errors.
func TestParsePostProcessInput(t *testing.T) {
	for _, payload := range [][]byte{nil, {}} {
		in, err := parsePostProcessInput(payload)
		if err != nil || in != (rule.EvalInput{}) {
			t.Fatalf("empty payload %q: got (%+v, %v), want (zero, nil)", payload, in, err)
		}
	}
	in, err := parsePostProcessInput([]byte(`{"memberId":7,"paymentOrderId":42,"amountCent":1000}`))
	if err != nil {
		t.Fatalf("valid payload: %v", err)
	}
	if in.MemberID != 7 || in.PaymentOrderID != 42 || in.AmountCent != 1000 {
		t.Fatalf("parsed input = %+v", in)
	}
	if _, err := parsePostProcessInput([]byte(`{bad`)); err == nil {
		t.Fatal("expected error for malformed json")
	}
}
