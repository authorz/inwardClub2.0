package payment

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// memStoreRepo is an in-memory StoreRepository for service-level tests.
type memStoreRepo struct {
	orders  []CollectionOrder
	refunds []Refund
	// payments maps a payment order id to its owning store scope (0 = missing).
	payments      map[int64]int64
	paymentOrders []PaymentOrder
	// membersByPhone maps a normalized phone to the member it resolves to.
	membersByPhone map[string]MemberMatch
	nextID         int64
}

func (r *memStoreRepo) ResolveMemberByPhone(_ context.Context, phone string) (MemberMatch, error) {
	if m, ok := r.membersByPhone[phone]; ok {
		return m, nil
	}
	return MemberMatch{}, apperr.New(apperr.CodeMemberNotFound, "member not found")
}

func (r *memStoreRepo) CreateCollectionOrder(_ context.Context, in CollectionOrderCreate) (CollectionOrder, error) {
	r.nextID++
	o := CollectionOrder{
		ID:                r.nextID,
		CollectionOrderNo: in.CollectionOrderNo,
		StoreID:           in.StoreID,
		PaymentOrderID:    r.nextID,
		PaymentOrderNo:    in.PaymentOrderNo,
		PayMethod:         wechatProvider,
		AmountCent:        in.AmountCent,
		Subject:           in.Subject,
		BusinessType:      in.BusinessType,
		Status:            CollectionPending,
		MemberID:          in.MemberID,
		MemberPhoneMasked: in.MemberPhoneMasked,
		AcquirerOrderNo:   in.AcquirerOrderNo,
		QRContent:         in.QRContent,
		ExpiresAt:         in.ExpiresAt,
		CreatedAt:         in.Now,
	}
	r.orders = append(r.orders, o)
	return o, nil
}

func (r *memStoreRepo) GetCollectionOrder(_ context.Context, storeID, id int64) (CollectionOrder, error) {
	for _, o := range r.orders {
		if o.ID == id && o.StoreID == storeID {
			return o, nil
		}
	}
	return CollectionOrder{}, apperr.NotFound("collection order not found")
}

func (r *memStoreRepo) CancelCollectionOrder(_ context.Context, storeID, id int64, now time.Time) error {
	for i := range r.orders {
		o := &r.orders[i]
		if o.ID == id && o.StoreID == storeID && o.Status == CollectionPending {
			o.Status = CollectionCancelled
			return nil
		}
	}
	return apperr.Conflict("collection order cannot be cancelled")
}

func (r *memStoreRepo) CreateRefund(_ context.Context, in RefundCreate) (Refund, error) {
	owner, ok := r.payments[in.PaymentOrderID]
	if !ok || owner != in.StoreID {
		return Refund{}, apperr.NotFound("payment order not found")
	}
	r.nextID++
	rf := Refund{
		ID:             r.nextID,
		RefundOrderNo:  in.RefundOrderNo,
		PaymentOrderID: in.PaymentOrderID,
		StoreID:        in.StoreID,
		AmountCent:     in.AmountCent,
		Status:         RefundPending,
		Reason:         in.Reason,
		CreatedAt:      in.Now,
	}
	r.refunds = append(r.refunds, rf)
	return rf, nil
}

func (r *memStoreRepo) CreateRefundAdmin(_ context.Context, in RefundCreate) (Refund, error) {
	owner, ok := r.payments[in.PaymentOrderID]
	if !ok {
		return Refund{}, apperr.NotFound("payment order not found")
	}
	if in.AmountCent > 500 {
		return Refund{}, apperr.Invalid("退款金额不能超过订单实付金额")
	}
	for _, existing := range r.refunds {
		if existing.PaymentOrderID == in.PaymentOrderID &&
			(existing.Status == RefundProcessing || existing.Status == RefundSucceeded) {
			return Refund{}, apperr.Conflict("该订单已发起退款")
		}
	}
	r.nextID++
	rf := Refund{
		ID:                r.nextID,
		RefundOrderNo:     in.RefundOrderNo,
		PaymentOrderID:    in.PaymentOrderID,
		StoreID:           owner,
		AmountCent:        in.AmountCent,
		PaymentAmountCent: 500,
		Channel:           "wechat",
		Status:            RefundProcessing,
		Reason:            in.Reason,
		CreatedAt:         in.Now,
		PaymentOrderNo:    "PO-55",
	}
	r.refunds = append(r.refunds, rf)
	return rf, nil
}

func (r *memStoreRepo) CompleteRefundAdmin(_ context.Context, refundID int64, _ string, _ time.Time) (Refund, error) {
	for i := range r.refunds {
		if r.refunds[i].ID == refundID {
			r.refunds[i].Status = RefundSucceeded
			return r.refunds[i], nil
		}
	}
	return Refund{}, apperr.NotFound("refund order not found")
}

func (r *memStoreRepo) FailRefundAdmin(_ context.Context, refundID int64, _ time.Time) error {
	for i := range r.refunds {
		if r.refunds[i].ID == refundID {
			r.refunds[i].Status = RefundFailed
			return nil
		}
	}
	return apperr.NotFound("refund order not found")
}

func (r *memStoreRepo) ListPaymentOrders(_ context.Context, f PaymentOrderFilter) ([]PaymentOrder, int64, error) {
	var matched []PaymentOrder
	for _, p := range r.paymentOrders {
		if f.StoreID != nil && (p.StoreID == nil || *p.StoreID != *f.StoreID) {
			continue
		}
		if f.Status != "" && p.Status != f.Status {
			continue
		}
		matched = append(matched, p)
	}
	total := int64(len(matched))
	start := f.Page.Offset()
	if start > len(matched) {
		start = len(matched)
	}
	end := start + f.Page.Limit()
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

func (r *memStoreRepo) GetPaymentOrder(_ context.Context, id int64, storeID *int64) (PaymentOrder, error) {
	for _, p := range r.paymentOrders {
		if p.ID != id {
			continue
		}
		if storeID != nil && (p.StoreID == nil || *p.StoreID != *storeID) {
			return PaymentOrder{}, apperr.NotFound("payment order not found")
		}
		return p, nil
	}
	return PaymentOrder{}, apperr.NotFound("payment order not found")
}

func newTestStoreService() (*StoreService, *memStoreRepo) {
	repo := &memStoreRepo{payments: map[int64]int64{}, membersByPhone: map[string]MemberMatch{}}
	svc := NewStoreService(repo, NewFakeWeChatPayGateway(), allowStoreAdminPasswords{}, 0)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

type allowStoreAdminPasswords struct{}

func (allowStoreAdminPasswords) VerifyStoreAdminPassword(_ context.Context, _ int64, password string) error {
	if password != "secret" {
		return apperr.Forbidden("门店管理员登录密码错误")
	}
	return nil
}

type allowAdminPasswords struct{}

func (allowAdminPasswords) VerifyAccountPassword(_ context.Context, _ int64, password string) error {
	if password != "secret" {
		return apperr.Forbidden("管理员登录密码错误")
	}
	return nil
}

func newTestAdminService(repo StoreRepository) *AdminService {
	return NewAdminService(
		repo,
		NewFakeWeChatPayGateway(),
		NewFakeOfflineAcquirer(),
		allowAdminPasswords{},
		0,
	)
}

type recordingWeChatGateway struct {
	WeChatPayGateway
	refundAmount int64
	totalAmount  int64
	refundErr    error
}

func (g *recordingWeChatGateway) Refund(
	_ context.Context,
	_, _ string,
	amountCent, totalCent int64,
) (string, error) {
	g.refundAmount = amountCent
	g.totalAmount = totalCent
	if g.refundErr != nil {
		return "", g.refundErr
	}
	return "WX-RF-1", nil
}

func TestCreateCollectionOrder(t *testing.T) {
	svc, repo := newTestStoreService()
	view, err := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "idem-1",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food", ExpiresInSeconds: 900})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.StoreID != 7 || view.Status != CollectionPending {
		t.Fatalf("unexpected view: %+v", view)
	}
	if !strings.HasPrefix(view.QRContent, "weixin://wxpay/") || view.ExpiresAt.IsZero() {
		t.Fatalf("expected WeChat Native QR and expiry: %+v", view)
	}
	if view.PayChannel != wechatProvider {
		t.Fatalf("expected WeChat pay channel, got %+v", view)
	}
	// A walk-in collection (no memberPhone) binds no member.
	if view.MemberID != nil || view.MemberNickname != "" || view.MemberPhoneMasked != "" {
		t.Fatalf("walk-in must not bind a member: %+v", view)
	}
	// The client-supplied window drives the expiry.
	if got := view.ExpiresAt.Sub(view.CreatedAt); got != 900*time.Second {
		t.Fatalf("expected 900s TTL, got %v", got)
	}
	if len(repo.orders) != 1 {
		t.Fatalf("order not persisted")
	}
}

type recordingNativeGateway struct {
	WeChatPayGateway
	amountCent int64
	expiresAt  time.Time
	closedNo   string
}

func (g *recordingNativeGateway) CreateNativeOrder(_ context.Context, outTradeNo string, amountCent int64, _ string, expiresAt time.Time) (string, error) {
	g.amountCent = amountCent
	g.expiresAt = expiresAt
	return "weixin://wxpay/bizpayurl/up?pr=" + outTradeNo, nil
}

func (g *recordingNativeGateway) CloseOrder(_ context.Context, outTradeNo string) error {
	g.closedNo = outTradeNo
	return nil
}

func TestCreateCollectionOrderUsesWechatAmountOverride(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{}, membersByPhone: map[string]MemberMatch{}}
	gateway := &recordingNativeGateway{WeChatPayGateway: NewFakeWeChatPayGateway()}
	svc := NewStoreService(repo, gateway, allowStoreAdminPasswords{}, 1)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }

	view, err := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "idem-native",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food", ExpiresInSeconds: 300})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if gateway.amountCent != 1 {
		t.Fatalf("expected debug amount 1 cent, got %d", gateway.amountCent)
	}
	if !gateway.expiresAt.Equal(view.ExpiresAt) {
		t.Fatalf("gateway expiry %s != order expiry %s", gateway.expiresAt, view.ExpiresAt)
	}
}

func TestCreateCollectionOrderValidation(t *testing.T) {
	svc, _ := newTestStoreService()
	cases := map[string]CreateCollectionOrderRequest{
		"zero amount": {Subject: "x", BusinessType: "food", ExpiresInSeconds: 900},
		"no subject":  {AmountCent: 100, BusinessType: "food", ExpiresInSeconds: 900},
		"no business": {AmountCent: 100, Subject: "x", ExpiresInSeconds: 900},
		"no expiry":   {AmountCent: 100, Subject: "x", BusinessType: "food"},
		"expiry >24h": {AmountCent: 100, Subject: "x", BusinessType: "food", ExpiresInSeconds: 90000},
	}
	for name, req := range cases {
		if _, err := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "", req); apperr.From(err).Code != apperr.CodeInvalidArgument {
			t.Fatalf("%s: expected INVALID_ARGUMENT, got %v", name, err)
		}
	}
}

func TestCreateCollectionOrderBindsMember(t *testing.T) {
	svc, repo := newTestStoreService()
	repo.membersByPhone["13800000000"] = MemberMatch{ID: 42, Nickname: "Alice", PhoneMasked: "138****0000"}

	view, err := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "idem-b",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food",
			ExpiresInSeconds: 900, MemberPhone: " 138-0000-0000 "})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.MemberID == nil || *view.MemberID != 42 {
		t.Fatalf("expected member 42 bound, got %+v", view)
	}
	// Masked identifiers are returned for operator confirmation; the raw phone is
	// never echoed back.
	if view.MemberPhoneMasked != "138****0000" || view.MemberNickname != "A***" {
		t.Fatalf("expected masked member confirmation, got %+v", view)
	}
	if len(repo.orders) != 1 || repo.orders[0].MemberID == nil || *repo.orders[0].MemberID != 42 {
		t.Fatalf("member binding not persisted: %+v", repo.orders)
	}
	if repo.orders[0].MemberPhoneMasked != "138****0000" {
		t.Fatalf("masked phone snapshot not persisted: %+v", repo.orders[0])
	}
}

func TestCreateCollectionOrderMemberNotFound(t *testing.T) {
	svc, repo := newTestStoreService()
	// A supplied-but-unmatched phone is a controlled MEMBER_NOT_FOUND; nothing is
	// persisted so the operator can retry without binding.
	_, err := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "idem-nf",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food",
			ExpiresInSeconds: 900, MemberPhone: "13900000000"})
	if apperr.From(err).Code != apperr.CodeMemberNotFound {
		t.Fatalf("expected MEMBER_NOT_FOUND, got %v", err)
	}
	if len(repo.orders) != 0 {
		t.Fatalf("no order must be persisted on member-not-found, got %+v", repo.orders)
	}

	// Retrying without the phone succeeds as a walk-in collection.
	view, err := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "idem-nf2",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food", ExpiresInSeconds: 900})
	if err != nil {
		t.Fatalf("walk-in retry: %v", err)
	}
	if view.MemberID != nil {
		t.Fatalf("walk-in retry must not bind a member: %+v", view)
	}
}

func TestGetCollectionOrderScope(t *testing.T) {
	svc, _ := newTestStoreService()
	view, _ := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food", ExpiresInSeconds: 900})

	if _, err := svc.GetCollectionOrder(context.Background(), 7, view.ID); err != nil {
		t.Fatalf("owner get: %v", err)
	}
	// A different store scope must not read the order.
	if _, err := svc.GetCollectionOrder(context.Background(), 99, view.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for other store, got %v", err)
	}
}

func TestCancelCollectionOrder(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{}, membersByPhone: map[string]MemberMatch{}}
	gateway := &recordingNativeGateway{WeChatPayGateway: NewFakeWeChatPayGateway()}
	svc := NewStoreService(repo, gateway, allowStoreAdminPasswords{}, 0)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	view, _ := svc.CreateCollectionOrder(context.Background(), 7, "cashier", 3, "",
		CreateCollectionOrderRequest{AmountCent: 1500, Subject: "coffee", BusinessType: "food", ExpiresInSeconds: 900})

	if err := svc.CancelCollectionOrder(context.Background(), 7, view.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if gateway.closedNo == "" {
		t.Fatal("expected WeChat order to be closed")
	}
	// Second cancel conflicts; wrong scope conflicts too.
	if err := svc.CancelCollectionOrder(context.Background(), 7, view.ID); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT on double cancel, got %v", err)
	}
}

func TestCreateRefundScope(t *testing.T) {
	svc, repo := newTestStoreService()
	repo.payments[55] = 7 // payment order 55 belongs to store 7

	view, err := svc.CreateRefund(context.Background(), 7, "store_admin", 2, "idem-r",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "damaged", Password: "secret"})
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if view.Status != RefundPending || view.StoreID != 7 {
		t.Fatalf("unexpected refund view: %+v", view)
	}
	// A store cannot refund another store's payment order.
	if _, err := svc.CreateRefund(context.Background(), 99, "store_admin", 2, "",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "damaged", Password: "secret"}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for cross-store refund, got %v", err)
	}
}

func TestCreateRefundRequiresStoreAdminPassword(t *testing.T) {
	svc, repo := newTestStoreService()
	repo.payments[55] = 7

	for _, password := range []string{"", "wrong"} {
		_, err := svc.CreateRefund(context.Background(), 7, "cashier", 3, "idem-r",
			CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "顾客申请", Password: password})
		if code := apperr.From(err).Code; code != apperr.CodeInvalidArgument && code != apperr.CodePermissionDenied {
			t.Fatalf("password %q should be rejected, got %v", password, err)
		}
	}
	if len(repo.refunds) != 0 {
		t.Fatalf("invalid manager password must not create a refund, got %+v", repo.refunds)
	}
}

func TestCreateRefundValidation(t *testing.T) {
	svc, _ := newTestStoreService()
	if _, err := svc.CreateRefund(context.Background(), 7, "store_admin", 2, "",
		CreateRefundRequest{AmountCent: 500}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing payment order, got %v", err)
	}
	if _, err := svc.CreateRefund(context.Background(), 7, "store_admin", 2, "",
		CreateRefundRequest{PaymentOrderID: 1}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for zero amount, got %v", err)
	}
}

func TestAdminServiceCreateRefund(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	svc := newTestAdminService(repo)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }

	view, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-admin-r",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "damaged", Password: "secret"})
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if view.Status != RefundSucceeded || view.StoreID != 7 {
		t.Fatalf("unexpected admin refund view: %+v", view)
	}

	if _, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "",
		CreateRefundRequest{PaymentOrderID: 999, AmountCent: 500, Reason: "damaged", Password: "secret"}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for unknown payment order, got %v", err)
	}
}

func TestFoodOrderCancellationRefundIsStoreScopedAndSkipsSecondPassword(t *testing.T) {
	storeID := int64(7)
	repo := &memStoreRepo{
		payments:      map[int64]int64{55: storeID},
		paymentOrders: []PaymentOrder{{ID: 55, StoreID: &storeID, OrderType: "food"}},
	}
	svc := newTestAdminService(repo)
	view, err := svc.CreateFoodOrderCancellationRefund(context.Background(), storeID, "store_admin", 2, "cancel-refund",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "门店取消点餐订单"})
	if err != nil {
		t.Fatalf("food cancellation refund: %v", err)
	}
	if view.Status != RefundSucceeded {
		t.Fatalf("unexpected refund: %+v", view)
	}
	if _, err := svc.CreateFoodOrderCancellationRefund(context.Background(), 99, "store_admin", 2, "foreign-refund",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "门店取消点餐订单"}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("foreign store refund must be hidden, got %v", err)
	}
}

func TestAdminServiceRefundRequiresValidPassword(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	svc := newTestAdminService(repo)

	_, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-admin-r",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "damaged", Password: "wrong"})
	if apperr.From(err).Code != apperr.CodePermissionDenied {
		t.Fatalf("expected forbidden for wrong password, got %v", err)
	}
	if len(repo.refunds) != 0 {
		t.Fatalf("wrong password must not create a refund, got %+v", repo.refunds)
	}
}

func TestAdminServiceRefundRequiresReason(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	svc := newTestAdminService(repo)

	_, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-admin-r",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Password: "secret"})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty reason, got %v", err)
	}
	if len(repo.refunds) != 0 {
		t.Fatalf("empty reason must not create a refund, got %+v", repo.refunds)
	}
}

func TestAdminServiceRefundUsesDebugGatewayAmount(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	gateway := &recordingWeChatGateway{WeChatPayGateway: NewFakeWeChatPayGateway()}
	svc := NewAdminService(repo, gateway, NewFakeOfflineAcquirer(), allowAdminPasswords{}, 1)

	view, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-admin-r",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "damaged", Password: "secret"})
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if gateway.refundAmount != 1 || gateway.totalAmount != 1 {
		t.Fatalf("expected debug gateway amount 1/1, got %d/%d", gateway.refundAmount, gateway.totalAmount)
	}
	if view.AmountCent != 500 {
		t.Fatalf("business refund amount must remain 500, got %d", view.AmountCent)
	}
}

func TestAdminServiceSupportsOnePartialRefund(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	gateway := &recordingWeChatGateway{WeChatPayGateway: NewFakeWeChatPayGateway()}
	svc := NewAdminService(repo, gateway, NewFakeOfflineAcquirer(), allowAdminPasswords{}, 0)

	view, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-partial",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 200, Reason: "partial", Password: "secret"})
	if err != nil {
		t.Fatalf("partial refund: %v", err)
	}
	if gateway.refundAmount != 200 || gateway.totalAmount != 500 {
		t.Fatalf("expected partial/total gateway amount 200/500, got %d/%d", gateway.refundAmount, gateway.totalAmount)
	}
	if view.AmountCent != 200 || view.Status != RefundSucceeded {
		t.Fatalf("unexpected partial refund view: %+v", view)
	}

	_, err = svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-duplicate",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 100, Reason: "again", Password: "secret"})
	if apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected duplicate refund conflict, got %v", err)
	}
}

func TestAdminServiceRejectsRefundAbovePaidAmount(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	svc := newTestAdminService(repo)

	_, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-over",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 501, Reason: "too much", Password: "secret"})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected over-refund invalid argument, got %v", err)
	}
}

func TestAdminServiceMarksRefundFailedWhenGatewayRejects(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{55: 7}}
	gateway := &recordingWeChatGateway{
		WeChatPayGateway: NewFakeWeChatPayGateway(),
		refundErr:        errors.New("gateway rejected"),
	}
	svc := NewAdminService(repo, gateway, NewFakeOfflineAcquirer(), allowAdminPasswords{}, 0)

	_, err := svc.CreateRefund(context.Background(), "platform_admin", 1, "idem-admin-r",
		CreateRefundRequest{PaymentOrderID: 55, AmountCent: 500, Reason: "damaged", Password: "secret"})
	if err == nil {
		t.Fatal("expected gateway error")
	}
	if len(repo.refunds) != 1 || repo.refunds[0].Status != RefundFailed {
		t.Fatalf("expected failed refund persisted, got %+v", repo.refunds)
	}
}

func storeIDPtr(id int64) *int64 { return &id }

func TestStoreServiceListPaymentOrders(t *testing.T) {
	svc, repo := newTestStoreService()
	repo.paymentOrders = []PaymentOrder{
		{ID: 1, PaymentOrderNo: "PO1", StoreID: storeIDPtr(7), Status: "paid"},
		{ID: 2, PaymentOrderNo: "PO2", StoreID: storeIDPtr(7), Status: "pending"},
		{ID: 3, PaymentOrderNo: "PO3", StoreID: storeIDPtr(99), Status: "paid"},
	}

	views, total, err := svc.ListPaymentOrders(context.Background(), 7, "", httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected 2 orders scoped to store 7, got total=%d len=%d", total, len(views))
	}
	for _, v := range views {
		if v.StoreID == nil || *v.StoreID != 7 {
			t.Fatalf("leaked cross-store order: %+v", v)
		}
	}

	views, total, err = svc.ListPaymentOrders(context.Background(), 7, "paid", httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if total != 1 || len(views) != 1 || views[0].PaymentOrderNo != "PO1" {
		t.Fatalf("expected 1 paid order for store 7, got %+v (total=%d)", views, total)
	}
}

func TestAdminServiceListPaymentOrders(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{}}
	repo.paymentOrders = []PaymentOrder{
		{ID: 1, PaymentOrderNo: "PO1", StoreID: storeIDPtr(7), Status: "paid"},
		{ID: 2, PaymentOrderNo: "PO2", StoreID: storeIDPtr(99), Status: "paid"},
	}
	svc := newTestAdminService(repo)

	views, total, err := svc.ListPaymentOrders(context.Background(), PaymentOrderFilter{Page: httpx.Page{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected all stores' orders, got total=%d len=%d", total, len(views))
	}

	storeID := int64(7)
	views, total, err = svc.ListPaymentOrders(context.Background(), PaymentOrderFilter{Page: httpx.Page{Page: 1, PageSize: 20}, StoreID: &storeID})
	if err != nil {
		t.Fatalf("list scoped: %v", err)
	}
	if total != 1 || len(views) != 1 || views[0].PaymentOrderNo != "PO1" {
		t.Fatalf("expected 1 order for store 7, got %+v (total=%d)", views, total)
	}
}

func TestStoreServiceGetPaymentOrder(t *testing.T) {
	svc, repo := newTestStoreService()
	repo.paymentOrders = []PaymentOrder{
		{ID: 1, PaymentOrderNo: "PO1", StoreID: storeIDPtr(7), Status: "paid"},
		{ID: 2, PaymentOrderNo: "PO2", StoreID: storeIDPtr(99), Status: "paid"},
	}

	view, err := svc.GetPaymentOrder(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.PaymentOrderNo != "PO1" {
		t.Fatalf("expected PO1, got %+v", view)
	}

	if _, err := svc.GetPaymentOrder(context.Background(), 7, 2); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected not-found for cross-store order, got %v", err)
	}

	if _, err := svc.GetPaymentOrder(context.Background(), 7, 999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected not-found for missing order, got %v", err)
	}
}

func TestAdminServiceGetPaymentOrder(t *testing.T) {
	repo := &memStoreRepo{payments: map[int64]int64{}}
	repo.paymentOrders = []PaymentOrder{
		{ID: 1, PaymentOrderNo: "PO1", StoreID: storeIDPtr(7), Status: "paid"},
	}
	svc := newTestAdminService(repo)

	view, err := svc.GetPaymentOrder(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.PaymentOrderNo != "PO1" {
		t.Fatalf("expected PO1, got %+v", view)
	}

	if _, err := svc.GetPaymentOrder(context.Background(), 999); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected not-found for missing order, got %v", err)
	}
}
