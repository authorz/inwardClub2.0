package order

import (
	"context"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/modules/payment"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// memRepo is an in-memory order repository for tests. Only the fields the tests
// exercise are populated.
type memRepo struct {
	food      map[int64]FoodOrder
	foodItems map[int64][]FoodOrderItem
	payments  map[int64]PaymentOrder
	tickets   map[int64][]MemberTicket

	// Expiry-sweep fixtures, exercised by expiry_test.go; unused by the other
	// tests. activityOrders/sold back ExpireActivityOrders; sweepTickets backs
	// ExpireTickets.
	activityOrders []fakeActivityOrder
	sold           map[int64]int64
	sweepTickets   []fakeTicket
}

func newMemRepo() *memRepo {
	return &memRepo{
		food:      map[int64]FoodOrder{},
		foodItems: map[int64][]FoodOrderItem{},
		payments:  map[int64]PaymentOrder{},
		tickets:   map[int64][]MemberTicket{},
	}
}

func (r *memRepo) ListMemberTickets(_ context.Context, memberID int64) ([]MemberTicket, error) {
	return r.tickets[memberID], nil
}

func (r *memRepo) ListFoodOrders(_ context.Context, memberID int64, _, _ int) ([]FoodOrder, int64, error) {
	var out []FoodOrder
	for _, o := range r.food {
		if o.MemberID == memberID {
			out = append(out, o)
		}
	}
	return out, int64(len(out)), nil
}

func (r *memRepo) GetFoodOrder(_ context.Context, memberID, id int64) (FoodOrder, []FoodOrderItem, error) {
	o, ok := r.food[id]
	if !ok || o.MemberID != memberID {
		return FoodOrder{}, nil, apperr.NotFound("food order not found")
	}
	return o, r.foodItems[id], nil
}

func (r *memRepo) ListRechargeOrders(_ context.Context, _ int64, _, _ int) ([]RechargeOrder, int64, error) {
	return nil, 0, nil
}
func (r *memRepo) GetRechargeOrder(_ context.Context, _, _ int64) (RechargeOrder, error) {
	return RechargeOrder{}, apperr.NotFound("recharge order not found")
}
func (r *memRepo) ListActivityOrders(_ context.Context, _ int64, _, _ int) ([]ActivityOrder, int64, error) {
	return nil, 0, nil
}
func (r *memRepo) GetActivityOrder(_ context.Context, _, _ int64) (ActivityOrder, []Ticket, error) {
	return ActivityOrder{}, nil, apperr.NotFound("activity order not found")
}

func (r *memRepo) GetPaymentOrder(_ context.Context, id int64) (PaymentOrder, error) {
	po, ok := r.payments[id]
	if !ok {
		return PaymentOrder{}, apperr.NotFound("payment order not found")
	}
	return po, nil
}

// The create/settle write methods are covered end-to-end against MySQL; the
// in-memory repo returns canned success so the service wiring can be unit-tested.
func (r *memRepo) CreateFoodOrder(_ context.Context, in FoodOrderCreate) (FoodOrder, []FoodOrderItem, PaymentOrder, error) {
	var total int64
	items := make([]FoodOrderItem, 0, len(in.Lines))
	for _, ln := range in.Lines {
		sub := int64(ln.Quantity) * 1000
		total += sub
		items = append(items, FoodOrderItem{ItemID: ln.ItemID, UnitPriceCent: 1000, Quantity: ln.Quantity, SubtotalCent: sub})
	}
	food := FoodOrder{ID: 1, BusinessOrderID: 1, StoreID: in.StoreID, MemberID: in.MemberID, TotalAmountCent: total, FulfillmentStatus: "pending"}
	po := PaymentOrder{ID: 1, PaymentOrderNo: in.PaymentOrderNo, AmountCent: total, PayMethod: in.PayMethod, Status: PaymentStatusPending}
	return food, items, po, nil
}

func (r *memRepo) CreateRechargeOrder(_ context.Context, in RechargeOrderCreate) (RechargeOrder, PaymentOrder, error) {
	ro := RechargeOrder{ID: 1, BusinessOrderNo: in.BusinessOrderNo, MemberID: in.MemberID, TotalAmountCent: in.AmountCent, OrderStatus: "created", PaymentStatus: "unpaid"}
	po := PaymentOrder{ID: 1, PaymentOrderNo: in.PaymentOrderNo, AmountCent: in.AmountCent, PayMethod: in.PayMethod, Status: PaymentStatusPending}
	return ro, po, nil
}

func (r *memRepo) CreateActivityOrder(_ context.Context, in ActivityOrderCreate) (ActivityOrder, []Ticket, PaymentOrder, error) {
	total := int64(in.Quantity) * 2000
	tickets := make([]Ticket, 0, in.Quantity)
	for i := 0; i < in.Quantity; i++ {
		tickets = append(tickets, Ticket{ID: int64(i + 1), TicketNo: "TK-1", ActivityID: in.ActivityID, TicketTypeID: in.TicketTypeID, PriceCent: 2000, Status: "pending"})
	}
	ao := ActivityOrder{ID: 1, BusinessOrderID: 1, ActivityID: in.ActivityID, MemberID: in.MemberID, TicketCount: in.Quantity, TotalAmountCent: total, Status: "created"}
	po := PaymentOrder{ID: 1, PaymentOrderNo: in.PaymentOrderNo, AmountCent: total, PayMethod: in.PayMethod, Status: PaymentStatusPending}
	return ao, tickets, po, nil
}

func (r *memRepo) SettleByCoin(_ context.Context, _ CoinPayment) error { return nil }

type fakeMembers struct{ openID string }

func (f fakeMembers) OpenIDByMemberID(_ context.Context, _ int64) (string, error) {
	return f.openID, nil
}

type fakeAssets struct{ url string }

func (f fakeAssets) PublicURLByID(_ context.Context, _ int64) (string, error) {
	return f.url, nil
}

type amountCapturingWeChatGateway struct {
	*payment.FakeWeChatPayGateway
	amountCent int64
}

func (g *amountCapturingWeChatGateway) CreateJSAPIPrepay(
	ctx context.Context,
	outTradeNo string,
	amountCent int64,
	openID, description string,
) (payment.WeChatPrepay, error) {
	g.amountCent = amountCent
	return g.FakeWeChatPayGateway.CreateJSAPIPrepay(ctx, outTradeNo, amountCent, openID, description)
}

func newService(repo Repository) *Service {
	return NewService(repo, payment.NewFakeWeChatPayGateway(), fakeMembers{openID: "open-123"}, fakeAssets{url: "https://cdn/poster.png"}, 0)
}

func ptr(v int64) *int64 { return &v }

func codeOf(t *testing.T, err error) apperr.Code {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	return apperr.From(err).Code
}

func TestGetFoodOrderScopedToMember(t *testing.T) {
	repo := newMemRepo()
	repo.food[1] = FoodOrder{ID: 1, MemberID: 10, StoreID: 5, TotalAmountCent: 1500, FulfillmentStatus: "pending"}
	repo.foodItems[1] = []FoodOrderItem{{ID: 1, FoodOrderID: 1, ItemID: 7, NameSnapshot: "Latte", UnitPriceCent: 1500, Quantity: 1, SubtotalCent: 1500}}
	svc := newService(repo)
	ctx := context.Background()

	view, err := svc.GetFoodOrder(ctx, 10, 1)
	if err != nil {
		t.Fatalf("owner read: %v", err)
	}
	if len(view.Items) != 1 || view.Items[0].Name != "Latte" {
		t.Fatalf("unexpected items: %+v", view.Items)
	}

	// A different member must not see the order.
	if code := codeOf(t, mustErr(svc.GetFoodOrder(ctx, 99, 1))); code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for other member, got %s", code)
	}
}

func TestListFoodOrdersReturnsViews(t *testing.T) {
	repo := newMemRepo()
	repo.food[1] = FoodOrder{ID: 1, MemberID: 10, StoreID: 5}
	repo.food[2] = FoodOrder{ID: 2, MemberID: 20, StoreID: 5}
	svc := newService(repo)

	views, total, err := svc.ListFoodOrders(context.Background(), 10, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 order for member 10, got total=%d len=%d", total, len(views))
	}
}

func TestCreateOrdersReturnPayableViews(t *testing.T) {
	svc := newService(newMemRepo())
	ctx := context.Background()

	food, err := svc.CreateFoodOrder(ctx, 10, "idem-food", CreateFoodOrderRequest{
		StoreID: 5, PayMethod: PayMethodWeChat, Items: []FoodLineItem{{ItemID: 7, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("food create: %v", err)
	}
	if food.PaymentOrderID == 0 || food.PayMethod != PayMethodWeChat || food.TotalAmountCent == 0 {
		t.Fatalf("unexpected food view: %+v", food)
	}
	rec, err := svc.CreateRechargeOrder(ctx, 10, "idem-rec", CreateRechargeOrderRequest{AmountCent: 5000, PayMethod: PayMethodWeChat})
	if err != nil {
		t.Fatalf("recharge create: %v", err)
	}
	if rec.PaymentOrderID == 0 || rec.TotalAmountCent != 5000 {
		t.Fatalf("unexpected recharge view: %+v", rec)
	}
	act, err := svc.CreateActivityOrder(ctx, 10, "idem-act", CreateActivityOrderRequest{ActivityID: 3, TicketTypeID: 4, Quantity: 2, PayMethod: PayMethodCoin})
	if err != nil {
		t.Fatalf("activity create: %v", err)
	}
	if act.PaymentOrderID == 0 || act.TicketCount != 2 || len(act.Tickets) != 2 {
		t.Fatalf("unexpected activity view: %+v", act)
	}
}

func TestListTicketsBuildsDisplayViews(t *testing.T) {
	repo := newMemRepo()
	start := time.Date(2026, 7, 20, 19, 30, 0, 0, time.UTC)
	repo.tickets[10] = []MemberTicket{
		{ID: 1, ActivityID: 3, Title: "会员私享局", ScopeType: "global", AssetID: ptr(9),
			StartAt: &start, StoreName: "三里屯店", TicketName: "双人票", Status: "active", Code: "5561 2200"},
		{ID: 2, ActivityID: 4, Title: "城市夜赛", ScopeType: "store",
			StoreName: "望京店", TicketName: "VIP票", Status: "used", Code: "2231 8890"},
	}
	svc := newService(repo)

	views, err := svc.ListTickets(context.Background(), 10)
	if err != nil {
		t.Fatalf("list tickets: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(views))
	}
	v0 := views[0]
	if v0.Tone != "member" || v0.ImageURL != "https://cdn/poster.png" || v0.Status != "unused" ||
		v0.TimeText != "2026.07.20 19:30" || v0.Qty != 1 || v0.TicketName != "双人票" || v0.StoreName != "三里屯店" {
		t.Fatalf("unexpected first ticket view: %+v", v0)
	}
	if views[1].Tone != "" || views[1].ImageURL != "" || views[1].Status != "used" {
		t.Fatalf("unexpected second ticket view: %+v", views[1])
	}
}

func TestCreateFoodOrderValidatesInput(t *testing.T) {
	svc := newService(newMemRepo())
	// Empty item list is rejected by the service before it opens the create tx.
	_, err := svc.CreateFoodOrder(context.Background(), 10, "idem-1", CreateFoodOrderRequest{StoreID: 5, PayMethod: PayMethodWeChat})
	if code := codeOf(t, err); code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %s", code)
	}
}

func TestCreateFoodOrderCombinesDuplicateLines(t *testing.T) {
	svc := newService(newMemRepo())
	view, err := svc.CreateFoodOrder(context.Background(), 10, "idem-duplicates", CreateFoodOrderRequest{
		StoreID:   5,
		PayMethod: PayMethodWeChat,
		Items: []FoodLineItem{
			{ItemID: 7, Quantity: 1},
			{ItemID: 7, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(view.Items) != 1 || view.Items[0].Quantity != 3 || view.TotalAmountCent != 3000 {
		t.Fatalf("duplicate lines were not combined: %+v", view)
	}
}

func TestWeChatJSAPIReturnsPrepay(t *testing.T) {
	repo := newMemRepo()
	repo.payments[1] = PaymentOrder{ID: 1, PaymentOrderNo: "PO-1", MemberID: ptr(10), AmountCent: 1500, PayMethod: PayMethodWeChat, Status: PaymentStatusPending}
	svc := newService(repo)

	resp, err := svc.CreateWeChatJSAPI(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("jsapi: %v", err)
	}
	if resp.PaymentOrderID != 1 || resp.Prepay.PrepayID == "" {
		t.Fatalf("unexpected prepay response: %+v", resp)
	}
}

func TestWeChatJSAPIDebugAmountKeepsBusinessAmount(t *testing.T) {
	repo := newMemRepo()
	repo.payments[1] = PaymentOrder{
		ID: 1, PaymentOrderNo: "PO-DEBUG", MemberID: ptr(10),
		AmountCent: 50000, PayMethod: PayMethodWeChat, Status: PaymentStatusPending,
	}
	gateway := &amountCapturingWeChatGateway{FakeWeChatPayGateway: payment.NewFakeWeChatPayGateway()}
	svc := NewService(repo, gateway, fakeMembers{openID: "open-123"}, fakeAssets{}, 1)

	if _, err := svc.CreateWeChatJSAPI(context.Background(), 10, 1); err != nil {
		t.Fatalf("jsapi: %v", err)
	}
	if gateway.amountCent != 1 {
		t.Fatalf("wechat amount = %d, want 1", gateway.amountCent)
	}
	if repo.payments[1].AmountCent != 50000 {
		t.Fatalf("business amount = %d, want 50000", repo.payments[1].AmountCent)
	}
}

func TestWeChatJSAPIRejectsForeignOrder(t *testing.T) {
	repo := newMemRepo()
	repo.payments[1] = PaymentOrder{ID: 1, MemberID: ptr(99), AmountCent: 1500, PayMethod: PayMethodWeChat, Status: PaymentStatusPending}
	svc := newService(repo)

	if code := codeOf(t, mustErrResp(svc.CreateWeChatJSAPI(context.Background(), 10, 1))); code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for foreign order, got %s", code)
	}
}

func TestWeChatJSAPIRejectsPayMethodMismatch(t *testing.T) {
	repo := newMemRepo()
	repo.payments[1] = PaymentOrder{ID: 1, MemberID: ptr(10), AmountCent: 1500, PayMethod: PayMethodCoin, Status: PaymentStatusPending}
	svc := newService(repo)

	if code := codeOf(t, mustErrResp(svc.CreateWeChatJSAPI(context.Background(), 10, 1))); code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for pay-method mismatch, got %s", code)
	}
}

func TestPayByCoinValidatesThenSettles(t *testing.T) {
	repo := newMemRepo()
	repo.payments[1] = PaymentOrder{ID: 1, MemberID: ptr(10), AmountCent: 1500, PayMethod: PayMethodCoin, Status: PaymentStatusPending}
	repo.payments[2] = PaymentOrder{ID: 2, MemberID: ptr(10), AmountCent: 1500, PayMethod: PayMethodCoin, Status: PaymentStatusPaid}
	svc := newService(repo)
	ctx := context.Background()

	// Payable coin order settles through the repository.
	if err := svc.PayByCoin(ctx, 10, 1, "idem-coin"); err != nil {
		t.Fatalf("expected settlement, got %v", err)
	}
	// Already-paid order is rejected as a conflict before settlement.
	if code := codeOf(t, svc.PayByCoin(ctx, 10, 2, "idem-coin-2")); code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT for paid order, got %s", code)
	}
}

// mustErr / mustErrResp discard the value of a (value, error) return so the error
// can be asserted inline.
func mustErr(_ FoodOrderView, err error) error           { return err }
func mustErrResp(_ WeChatJSAPIResponse, err error) error { return err }
