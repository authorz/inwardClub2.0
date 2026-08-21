package order

import (
	"context"
	"fmt"
	"time"

	"github.com/inwardclub/server/internal/modules/payment"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

// MemberDirectory resolves the WeChat openid a payment needs for JSAPI prepay.
// It is a narrow port over the member module so order never imports it directly.
type MemberDirectory interface {
	OpenIDByMemberID(ctx context.Context, memberID int64) (string, error)
}

// AssetResolver resolves an asset id to a public URL (activity poster images).
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// Service provides the mini-program order read paths and payment initiation.
type Service struct {
	repo                        Repository
	wechat                      payment.WeChatPayGateway
	members                     MemberDirectory
	assets                      AssetResolver
	wechatPayAmountOverrideCent int64
}

// NewService builds the order service.
func NewService(
	repo Repository,
	wechat payment.WeChatPayGateway,
	members MemberDirectory,
	assets AssetResolver,
	wechatPayAmountOverrideCent int64,
) *Service {
	return &Service{
		repo: repo, wechat: wechat, members: members, assets: assets,
		wechatPayAmountOverrideCent: wechatPayAmountOverrideCent,
	}
}

// ---- Food orders ----

// ListFoodOrders returns the member's food orders (newest first).
func (s *Service) ListFoodOrders(ctx context.Context, memberID int64, page httpx.Page) ([]FoodOrderView, int64, error) {
	orders, total, err := s.repo.ListFoodOrders(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	orderIDs := make([]int64, 0, len(orders))
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
	}
	itemsByOrder, err := s.repo.ListFoodOrderItems(ctx, memberID, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	views := make([]FoodOrderView, 0, len(orders))
	for _, o := range orders {
		views = append(views, foodOrderView(o, itemsByOrder[o.ID]))
	}
	return views, total, nil
}

// GetFoodOrder returns a single food order with its lines.
func (s *Service) GetFoodOrder(ctx context.Context, memberID, id int64) (FoodOrderView, error) {
	o, items, err := s.repo.GetFoodOrder(ctx, memberID, id)
	if err != nil {
		return FoodOrderView{}, err
	}
	if s.assets != nil {
		for i := range items {
			if items[i].AssetID != nil {
				items[i].ImageURL, _ = s.assets.PublicURLByID(ctx, *items[i].AssetID)
			}
		}
	}
	return foodOrderView(o, items), nil
}

// CreateFoodOrder validates the request then, in one transaction, snapshots the
// catalog prices and creates the business + food + payment orders. The returned
// view carries the payment order so the client can continue to pay. idemKey is
// the request's Idempotency-Key and guards against duplicate placement.
func (s *Service) CreateFoodOrder(ctx context.Context, memberID int64, idemKey string, req CreateFoodOrderRequest) (FoodOrderView, error) {
	if req.StoreID <= 0 {
		return FoodOrderView{}, apperr.Invalid("请选择有效的门店")
	}
	if req.TableID != nil && *req.TableID <= 0 {
		return FoodOrderView{}, apperr.Invalid("桌台信息不正确")
	}
	if err := validatePayMethod(req.PayMethod); err != nil {
		return FoodOrderView{}, err
	}
	if len(req.Items) == 0 {
		return FoodOrderView{}, apperr.Invalid("at least one item is required")
	}
	if len(req.Items) > 100 {
		return FoodOrderView{}, apperr.Invalid("单笔订单最多包含100种餐品")
	}
	remark, validationErr := inputvalidation.PlainText(req.Remark, inputvalidation.TextOptions{
		Label: "订单备注", MaxRunes: 200, AllowEmpty: true, AllowNewlines: true,
	})
	if validationErr != nil {
		return FoodOrderView{}, apperr.Invalid(validationErr.Error())
	}
	lines, err := normalizeFoodLines(req.Items)
	if err != nil {
		return FoodOrderView{}, err
	}
	now := time.Now().UTC()
	o, items, po, err := s.repo.CreateFoodOrder(ctx, FoodOrderCreate{
		MemberID:        memberID,
		StoreID:         req.StoreID,
		TableID:         req.TableID,
		Remark:          remark,
		PayMethod:       req.PayMethod,
		Lines:           lines,
		BusinessOrderNo: newNo("BO", now),
		PaymentOrderNo:  newNo("PO", now),
		IdemKey:         idemKey,
		Now:             now,
	})
	if err != nil {
		return FoodOrderView{}, err
	}
	v := foodOrderView(o, items)
	v.PaymentOrderID = po.ID
	v.PaymentOrderNo = po.PaymentOrderNo
	v.PayMethod = po.PayMethod
	return v, nil
}

// normalizeFoodLines combines duplicate item/variant rows before inventory is
// checked, preventing repeated lines from bypassing the aggregate stock limit.
func normalizeFoodLines(lines []FoodLineItem) ([]FoodLineItem, error) {
	out := make([]FoodLineItem, 0, len(lines))
	index := make(map[string]int, len(lines))
	for _, line := range lines {
		if line.ItemID <= 0 || line.Quantity <= 0 {
			return nil, apperr.Invalid("itemId and quantity must be positive")
		}
		if line.Quantity > 99 {
			return nil, apperr.Invalid("单种餐品数量不能超过99")
		}
		variantID := int64(0)
		if line.VariantID != nil {
			if *line.VariantID <= 0 {
				return nil, apperr.Invalid("variantId must be positive")
			}
			variantID = *line.VariantID
		}
		key := fmt.Sprintf("%d:%d", line.ItemID, variantID)
		if i, ok := index[key]; ok {
			out[i].Quantity += line.Quantity
			if out[i].Quantity > 99 {
				return nil, apperr.Invalid("单种餐品数量不能超过99")
			}
			continue
		}
		index[key] = len(out)
		out = append(out, line)
	}
	return out, nil
}

// ---- Recharge orders ----

// ListRechargeOrders returns the member's recharge (top-up) orders.
func (s *Service) ListRechargeOrders(ctx context.Context, memberID int64, page httpx.Page) ([]RechargeOrderView, int64, error) {
	orders, total, err := s.repo.ListRechargeOrders(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]RechargeOrderView, 0, len(orders))
	for _, o := range orders {
		views = append(views, rechargeOrderView(o))
	}
	return views, total, nil
}

// GetRechargeOrder returns a single recharge order.
func (s *Service) GetRechargeOrder(ctx context.Context, memberID, id int64) (RechargeOrderView, error) {
	o, err := s.repo.GetRechargeOrder(ctx, memberID, id)
	if err != nil {
		return RechargeOrderView{}, err
	}
	return rechargeOrderView(o), nil
}

// CreateRechargeOrder validates the request then creates the recharge business
// order + payment order in one transaction. Bonus tiers governed by the rule
// engine are applied on settlement (wired in a later milestone); the top-up
// amount itself is paid here. The returned view carries the payment order.
func (s *Service) CreateRechargeOrder(ctx context.Context, memberID int64, idemKey string, req CreateRechargeOrderRequest) (RechargeOrderView, error) {
	if err := validatePayMethod(req.PayMethod); err != nil {
		return RechargeOrderView{}, err
	}
	if req.AmountCent <= 0 {
		return RechargeOrderView{}, apperr.Invalid("amountCent must be positive")
	}
	if req.AmountCent > 100_000_000 {
		return RechargeOrderView{}, apperr.Invalid("单次充值金额不能超过100万元")
	}
	now := time.Now().UTC()
	o, po, err := s.repo.CreateRechargeOrder(ctx, RechargeOrderCreate{
		MemberID:        memberID,
		AmountCent:      req.AmountCent,
		PayMethod:       req.PayMethod,
		BusinessOrderNo: newNo("BO", now),
		PaymentOrderNo:  newNo("PO", now),
		IdemKey:         idemKey,
		Now:             now,
	})
	if err != nil {
		return RechargeOrderView{}, err
	}
	v := rechargeOrderView(o)
	v.PaymentOrderID = po.ID
	v.PaymentOrderNo = po.PaymentOrderNo
	v.PayMethod = po.PayMethod
	return v, nil
}

// ---- Activity orders ----

// ListActivityOrders returns the member's activity (ticket) orders.
func (s *Service) ListActivityOrders(ctx context.Context, memberID int64, page httpx.Page) ([]ActivityOrderView, int64, error) {
	orders, total, err := s.repo.ListActivityOrders(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]ActivityOrderView, 0, len(orders))
	for _, o := range orders {
		views = append(views, activityOrderView(o, nil))
	}
	return views, total, nil
}

// GetActivityOrder returns a single activity order with its tickets.
func (s *Service) GetActivityOrder(ctx context.Context, memberID, id int64) (ActivityOrderView, error) {
	o, tickets, err := s.repo.GetActivityOrder(ctx, memberID, id)
	if err != nil {
		return ActivityOrderView{}, err
	}
	return activityOrderView(o, tickets), nil
}

// CreateActivityOrder validates the request then, in one transaction, reserves
// ticket-type stock, issues ticket instances with verification codes and creates
// the business + activity + payment orders. The returned view carries the issued
// tickets and the payment order.
func (s *Service) CreateActivityOrder(ctx context.Context, memberID int64, idemKey string, req CreateActivityOrderRequest) (ActivityOrderView, error) {
	if req.ActivityID <= 0 || req.TicketTypeID <= 0 {
		return ActivityOrderView{}, apperr.Invalid("活动或票档信息不正确")
	}
	if err := validatePayMethod(req.PayMethod); err != nil {
		return ActivityOrderView{}, err
	}
	if req.Quantity <= 0 {
		return ActivityOrderView{}, apperr.Invalid("quantity must be positive")
	}
	if req.Quantity > 99 {
		return ActivityOrderView{}, apperr.Invalid("单次购票数量不能超过99张")
	}
	if req.PayMethod == PayMethodCoupon {
		if req.Quantity != 1 || req.CouponEntitlementID == nil || *req.CouponEntitlementID <= 0 {
			return ActivityOrderView{}, apperr.Invalid("一张赛事门票券只能兑换一张门票")
		}
	} else if req.CouponEntitlementID != nil {
		return ActivityOrderView{}, apperr.Invalid("当前支付方式不能使用门票券")
	}
	now := time.Now().UTC()
	o, tickets, po, err := s.repo.CreateActivityOrder(ctx, ActivityOrderCreate{
		ActivityID:          req.ActivityID,
		TicketTypeID:        req.TicketTypeID,
		Quantity:            req.Quantity,
		MemberID:            memberID,
		PayMethod:           req.PayMethod,
		CouponEntitlementID: req.CouponEntitlementID,
		BusinessOrderNo:     newNo("BO", now),
		PaymentOrderNo:      newNo("PO", now),
		IdemKey:             idemKey,
		Now:                 now,
	})
	if err != nil {
		return ActivityOrderView{}, err
	}
	v := activityOrderView(o, tickets)
	v.PaymentOrderID = po.ID
	v.PaymentOrderNo = po.PaymentOrderNo
	v.PayMethod = po.PayMethod
	return v, nil
}

// ---- Payment initiation ----

// CreateWeChatJSAPI initiates a WeChat JSAPI payment for a pending payment order
// and returns the prepay parameters. It never settles the order — settlement is
// driven by the WeChat notify callback owned by the payment module.
func (s *Service) CreateWeChatJSAPI(ctx context.Context, memberID, paymentOrderID int64) (WeChatJSAPIResponse, error) {
	po, err := s.loadPayablePaymentOrder(ctx, memberID, paymentOrderID, PayMethodWeChat)
	if err != nil {
		return WeChatJSAPIResponse{}, err
	}
	openID, err := s.members.OpenIDByMemberID(ctx, memberID)
	if err != nil {
		return WeChatJSAPIResponse{}, err
	}
	payAmountCent := po.AmountCent
	if s.wechatPayAmountOverrideCent > 0 {
		payAmountCent = s.wechatPayAmountOverrideCent
	}
	prepay, err := s.wechat.CreateJSAPIPrepay(ctx, po.PaymentOrderNo, payAmountCent, openID, "InwardClub order")
	if err != nil {
		return WeChatJSAPIResponse{}, apperr.From(err)
	}
	return WeChatJSAPIResponse{PaymentOrderID: po.ID, Prepay: prepay}, nil
}

// PayByCoin pays a pending payment order from the member's coin balance. The
// ownership/state check, wallet debit, ledger append, payment transaction and
// status flip all run in one transaction inside the repository, keyed by the
// request's Idempotency-Key so a retry never double-debits.
func (s *Service) PayByCoin(ctx context.Context, memberID, paymentOrderID int64, idemKey string) error {
	// Cheap pre-check for a friendly error before opening the settlement tx; the
	// repository re-validates under a row lock as the authoritative guard.
	if _, err := s.loadPayablePaymentOrder(ctx, memberID, paymentOrderID, PayMethodCoin); err != nil {
		return err
	}
	return s.repo.SettleByCoin(ctx, CoinPayment{
		MemberID:       memberID,
		PaymentOrderID: paymentOrderID,
		IdemKey:        idemKey,
		Now:            time.Now().UTC(),
	})
}

// loadPayablePaymentOrder fetches a payment order and enforces that it belongs to
// the member, is still pending, and matches the requested pay method.
func (s *Service) loadPayablePaymentOrder(ctx context.Context, memberID, id int64, payMethod string) (PaymentOrder, error) {
	po, err := s.repo.GetPaymentOrder(ctx, id)
	if err != nil {
		return PaymentOrder{}, err
	}
	if po.MemberID == nil || *po.MemberID != memberID {
		// Do not disclose existence of another member's order.
		return PaymentOrder{}, apperr.NotFound("payment order not found")
	}
	if po.Status != PaymentStatusPending {
		return PaymentOrder{}, apperr.Conflict("payment order is not payable")
	}
	if po.PayMethod != payMethod {
		return PaymentOrder{}, apperr.Invalid("payment order pay method mismatch")
	}
	return po, nil
}

func validatePayMethod(m string) error {
	switch m {
	case PayMethodWeChat, PayMethodCoin, PayMethodCoupon:
		return nil
	default:
		return apperr.Invalid("unsupported pay method")
	}
}

func foodOrderView(o FoodOrder, items []FoodOrderItem) FoodOrderView {
	v := FoodOrderView{
		ID:                o.ID,
		BusinessOrderID:   o.BusinessOrderID,
		OrderNo:           o.BusinessOrderNo,
		StoreID:           o.StoreID,
		TableID:           o.TableID,
		TotalAmountCent:   o.TotalAmountCent,
		PointsEarned:      o.PointsEarned,
		PaymentStatus:     o.PaymentStatus,
		PayMethod:         o.PayMethod,
		PaidAmountCent:    o.PaidAmountCent,
		RefundAmountCent:  o.RefundAmountCent,
		StoreName:         o.StoreName,
		TableName:         o.TableName,
		FulfillmentStatus: o.FulfillmentStatus,
		Remark:            o.Remark,
		CreatedAt:         o.CreatedAt,
	}
	for _, it := range items {
		v.Items = append(v.Items, FoodOrderItemView{
			ItemID:        it.ItemID,
			VariantID:     it.VariantID,
			Name:          it.NameSnapshot,
			UnitPriceCent: it.UnitPriceCent,
			Quantity:      it.Quantity,
			PointsReward:  it.PointsReward,
			SubtotalCent:  it.SubtotalCent,
			ImageURL:      it.ImageURL,
		})
	}
	return v
}

func rechargeOrderView(o RechargeOrder) RechargeOrderView {
	return RechargeOrderView{
		ID:              o.ID,
		BusinessOrderNo: o.BusinessOrderNo,
		TotalAmountCent: o.TotalAmountCent,
		OrderStatus:     o.OrderStatus,
		PaymentStatus:   o.PaymentStatus,
		CreatedAt:       o.CreatedAt,
	}
}

func activityOrderView(o ActivityOrder, tickets []Ticket) ActivityOrderView {
	v := ActivityOrderView{
		ID:              o.ID,
		BusinessOrderID: o.BusinessOrderID,
		ActivityID:      o.ActivityID,
		TicketCount:     o.TicketCount,
		TotalAmountCent: o.TotalAmountCent,
		Status:          o.Status,
		CreatedAt:       o.CreatedAt,
	}
	for _, t := range tickets {
		v.Tickets = append(v.Tickets, TicketView{
			ID:           t.ID,
			TicketNo:     t.TicketNo,
			TicketTypeID: t.TicketTypeID,
			SessionID:    t.SessionID,
			PriceCent:    t.PriceCent,
			Status:       t.Status,
		})
	}
	return v
}
