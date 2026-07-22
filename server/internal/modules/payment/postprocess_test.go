package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

// --- pure evaluation: config parsing + grant computation ---

func TestComputeGrantsFixedAndPermille(t *testing.T) {
	cfg := consumeRewardConfig{Grants: []consumeGrantConfig{
		{Asset: pointsAsset, Mode: grantModeFixed, Value: 50},
		{Asset: coinsAsset, Mode: grantModePermille, Value: 10}, // 10 per 1000 cent
		{Asset: growthAsset, Mode: grantModePermille, Value: 5},
	}}
	// amount 3000 cent: coins = 3000*10/1000 = 30, growth = 3000*5/1000 = 15.
	got := computeGrants(cfg, 3000)
	want := []benefitGrant{
		{Asset: pointsAsset, Amount: 50},
		{Asset: coinsAsset, Amount: 30},
		{Asset: growthAsset, Amount: 15},
	}
	if len(got) != len(want) {
		t.Fatalf("grant count: got %d want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("grant %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestComputeGrantsPermilleFloorsAndDropsZero(t *testing.T) {
	cfg := consumeRewardConfig{Grants: []consumeGrantConfig{
		{Asset: coinsAsset, Mode: grantModePermille, Value: 10},
	}}
	// 1550 * 10 / 1000 = 15 (floor of 15.5), a real grant.
	if got := computeGrants(cfg, 1550); len(got) != 1 || got[0].Amount != 15 {
		t.Fatalf("expected coins 15, got %+v", got)
	}
	// 90 * 10 / 1000 = 0 -> dropped so no zero ledger entry is written.
	if got := computeGrants(cfg, 90); len(got) != 0 {
		t.Fatalf("sub-unit permille grant must be dropped, got %+v", got)
	}
}

func TestComputeGrantsSkipsUnknownAssetAndMode(t *testing.T) {
	cfg := consumeRewardConfig{Grants: []consumeGrantConfig{
		{Asset: "cash_balance", Mode: grantModeFixed, Value: 100}, // not earnable
		{Asset: pointsAsset, Mode: "ratio", Value: 100},           // unknown mode
		{Asset: pointsAsset, Mode: grantModeFixed, Value: 100},    // the only valid line
	}}
	got := computeGrants(cfg, 5000)
	if len(got) != 1 || got[0].Asset != pointsAsset || got[0].Amount != 100 {
		t.Fatalf("only the valid points line should survive, got %+v", got)
	}
}

func TestComputeGrantsEmptyConfig(t *testing.T) {
	if got := computeGrants(consumeRewardConfig{}, 5000); len(got) != 0 {
		t.Fatalf("empty config must grant nothing, got %+v", got)
	}
}

func TestParseConsumeRewardConfigMalformed(t *testing.T) {
	if _, err := parseConsumeRewardConfig([]byte(`{"grants": not-json}`)); err == nil {
		t.Fatalf("malformed config must return an error")
	}
	cfg, err := parseConsumeRewardConfig([]byte(`{"grants":[{"asset":"points","mode":"fixed","value":7}]}`))
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if len(cfg.Grants) != 1 || cfg.Grants[0].Value != 7 {
		t.Fatalf("config not parsed: %+v", cfg)
	}
}

// --- payload decoding + validation ---

func TestParsePostProcessPayloadUndecodable(t *testing.T) {
	_, err := parsePostProcessPayload([]byte(`{"memberId":`))
	if !errors.Is(err, ErrUndecodablePostProcess) {
		t.Fatalf("expected ErrUndecodablePostProcess, got %v", err)
	}
}

func TestPostProcessPayloadValid(t *testing.T) {
	if (postProcessPayload{MemberID: 1, PaymentOrderID: 2}).valid() != true {
		t.Fatalf("member-bound payload must be valid")
	}
	if (postProcessPayload{MemberID: 0, PaymentOrderID: 2}).valid() != false {
		t.Fatalf("missing member must be invalid")
	}
	if (postProcessPayload{MemberID: 1, PaymentOrderID: 0}).valid() != false {
		t.Fatalf("missing payment order must be invalid")
	}
}

// --- service orchestration + idempotency contract (fake repository) ---

// fakePostProcessRepo mirrors the SQL repository contract branch-for-branch: it
// gates on an enabled rule, records the (member, payment order, rule version)
// execution marker once, and applies each computed grant at most once. Tests
// assert on this spine so the worker semantics are covered without a live MySQL,
// matching the settlement spine convention.
type fakePostProcessRepo struct {
	rule     *consumeRewardConfig // nil => no enabled consume_reward rule
	version  int
	executed map[string]bool  // rule_executions idem_key -> processed
	grants   map[string]int64 // benefit_grants idem_key -> amount
	upgrades int              // growth grants that re-resolved the VIP tier
	calls    int              // Process invocations, to prove the service delegates
}

func newFakePostProcessRepo() *fakePostProcessRepo {
	return &fakePostProcessRepo{executed: map[string]bool{}, grants: map[string]int64{}}
}

func (f *fakePostProcessRepo) Process(_ context.Context, p postProcessPayload, _ time.Time) (PostProcessResult, error) {
	f.calls++
	if f.rule == nil {
		return PostProcessResult{RuleMatched: false}, nil
	}
	res := PostProcessResult{RuleMatched: true, RuleVersion: f.version}
	execIdem := fmt.Sprintf("rule:%d:%d", f.version, p.PaymentOrderID)
	if f.executed[execIdem] {
		res.AlreadyDone = true
		return res, nil
	}
	f.executed[execIdem] = true
	for _, g := range computeGrants(*f.rule, p.AmountCent) {
		gi := fmt.Sprintf("rule:%d:%d:%s", f.version, p.PaymentOrderID, g.Asset)
		if _, ok := f.grants[gi]; ok {
			continue
		}
		f.grants[gi] = g.Amount
		res.GrantsApplied++
		if g.Asset == growthAsset {
			f.upgrades++
		}
	}
	return res, nil
}

func rawPayload(t *testing.T, p postProcessPayload) []byte {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}

func TestServiceRejectsUndecodablePayload(t *testing.T) {
	repo := newFakePostProcessRepo()
	svc := NewPostProcessService(repo)
	if _, err := svc.Process(context.Background(), []byte(`{bad`)); !errors.Is(err, ErrUndecodablePostProcess) {
		t.Fatalf("expected ErrUndecodablePostProcess, got %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("undecodable payload must not reach the repository, calls=%d", repo.calls)
	}
}

func TestServiceRejectsInvalidPayload(t *testing.T) {
	repo := newFakePostProcessRepo()
	svc := NewPostProcessService(repo)
	// Structurally valid JSON but missing the member binding.
	raw := rawPayload(t, postProcessPayload{PaymentOrderID: 9, AmountCent: 1000})
	if _, err := svc.Process(context.Background(), raw); !errors.Is(err, ErrUndecodablePostProcess) {
		t.Fatalf("expected ErrUndecodablePostProcess for member-less payload, got %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("invalid payload must not reach the repository, calls=%d", repo.calls)
	}
}

func TestServiceNoEnabledRuleGrantsNothing(t *testing.T) {
	repo := newFakePostProcessRepo() // rule stays nil
	svc := NewPostProcessService(repo)
	raw := rawPayload(t, postProcessPayload{Source: collectionType, MemberID: 7, PaymentOrderID: 3, StoreID: 1, AmountCent: 5000})
	res, err := svc.Process(context.Background(), raw)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if res.RuleMatched || res.GrantsApplied != 0 {
		t.Fatalf("no enabled rule must match nothing and grant nothing, got %+v", res)
	}
	if repo.calls != 1 {
		t.Fatalf("valid payload must reach the repository exactly once, calls=%d", repo.calls)
	}
}

func TestServiceGrantsOncePerEvent(t *testing.T) {
	repo := newFakePostProcessRepo()
	repo.version = 4
	repo.rule = &consumeRewardConfig{Grants: []consumeGrantConfig{
		{Asset: pointsAsset, Mode: grantModeFixed, Value: 100},
		{Asset: growthAsset, Mode: grantModePermille, Value: 10}, // 6000*10/1000 = 60
	}}
	svc := NewPostProcessService(repo)
	raw := rawPayload(t, postProcessPayload{Source: collectionType, MemberID: 7, PaymentOrderID: 3, StoreID: 1, AmountCent: 6000})

	// First delivery grants both lines; replays are idempotent no-ops.
	first, err := svc.Process(context.Background(), raw)
	if err != nil {
		t.Fatalf("first process: %v", err)
	}
	if !first.RuleMatched || first.GrantsApplied != 2 || first.AlreadyDone {
		t.Fatalf("first delivery must apply both grants, got %+v", first)
	}
	for i := 0; i < 2; i++ {
		replay, err := svc.Process(context.Background(), raw)
		if err != nil {
			t.Fatalf("replay %d: %v", i, err)
		}
		if !replay.AlreadyDone || replay.GrantsApplied != 0 {
			t.Fatalf("replay %d must be an idempotent no-op, got %+v", i, replay)
		}
	}
	if len(repo.grants) != 2 {
		t.Fatalf("exactly two benefit grants must persist, got %d", len(repo.grants))
	}
	if repo.grants["rule:4:3:points"] != 100 || repo.grants["rule:4:3:growth_value"] != 60 {
		t.Fatalf("grant amounts wrong: %+v", repo.grants)
	}
	if repo.upgrades != 1 {
		t.Fatalf("a growth grant must re-resolve the VIP tier exactly once, got %d", repo.upgrades)
	}
}

func TestServiceDistinctOrdersGrantIndependently(t *testing.T) {
	repo := newFakePostProcessRepo()
	repo.version = 1
	repo.rule = &consumeRewardConfig{Grants: []consumeGrantConfig{{Asset: pointsAsset, Mode: grantModeFixed, Value: 100}}}
	svc := NewPostProcessService(repo)
	for _, po := range []int64{10, 11} {
		raw := rawPayload(t, postProcessPayload{Source: collectionType, MemberID: 7, PaymentOrderID: po, StoreID: 1, AmountCent: 1000})
		res, err := svc.Process(context.Background(), raw)
		if err != nil {
			t.Fatalf("process po %d: %v", po, err)
		}
		if res.GrantsApplied != 1 {
			t.Fatalf("each distinct order must grant once, po %d got %+v", po, res)
		}
	}
	if len(repo.grants) != 2 {
		t.Fatalf("two distinct orders must produce two grants, got %d", len(repo.grants))
	}
}
