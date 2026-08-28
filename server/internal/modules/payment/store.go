package payment

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Offline collection order statuses (see 00006_payment.sql).
const (
	CollectionPending   = "pending"
	CollectionPaid      = "paid"
	CollectionCancelled = "cancelled"
	CollectionExpired   = "expired"
)

// Refund order statuses.
const (
	RefundPending    = "pending"
	RefundProcessing = "processing"
	RefundSucceeded  = "succeeded"
	RefundFailed     = "failed"
)

// CollectionOrder is a store-created offline aggregated collection order.
// MemberID and MemberPhoneMasked are the member binding locked at creation; both
// are empty for a walk-in (unbound) collection. The raw phone is never stored.
type CollectionOrder struct {
	ID                int64
	CollectionOrderNo string
	StoreID           int64
	PaymentOrderID    int64
	PaymentOrderNo    string
	PayMethod         string
	AmountCent        int64
	Subject           string
	BusinessType      string
	Status            string
	MemberID          *int64
	MemberPhoneMasked string
	AcquirerOrderNo   string
	QRContent         string
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// MemberMatch is the masked result of an offline-collection phone lookup. Only
// the id and masked identifiers ever leave the resolver; the raw phone is used
// solely to match and is never returned, persisted or logged.
type MemberMatch struct {
	ID          int64
	Nickname    string
	PhoneMasked string
}

// Refund is a store-initiated refund against a settled payment order.
type Refund struct {
	ID                int64
	RefundOrderNo     string
	PaymentOrderID    int64
	BusinessOrderID   int64
	StoreID           int64
	AmountCent        int64
	PaymentAmountCent int64
	Channel           string
	Status            string
	Reason            string
	CreatedAt         time.Time
	PaymentOrderNo    string
	PayMethod         string
	OrderType         string
	AcquirerOrderNo   string
}

// PaymentOrder is the read model for a payment_orders row, joined with its
// business order and store for console display. Unlike admin.PaymentTransaction
// (a reconciliation-oriented view), it surfaces the business order's own
// lifecycle state (BusinessStatus/PaymentStatus) and the payment order's
// last-updated time, which the transactions list does not carry.
type PaymentOrder struct {
	ID              int64
	PaymentOrderNo  string
	StoreID         *int64
	StoreName       string
	MemberID        *int64
	BusinessOrderID int64
	BusinessOrderNo string
	OrderType       string
	BusinessStatus  string
	PaymentStatus   string
	AmountCent      int64
	PayMethod       string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PaidAt          *time.Time
}

// PaymentOrderView is the console list representation of a payment order.
type PaymentOrderView struct {
	ID              int64      `json:"id"`
	PaymentOrderNo  string     `json:"paymentOrderNo"`
	StoreID         *int64     `json:"storeId,omitempty"`
	StoreName       string     `json:"storeName,omitempty"`
	MemberID        *int64     `json:"memberId,omitempty"`
	BusinessOrderID int64      `json:"businessOrderId"`
	BusinessOrderNo string     `json:"businessOrderNo"`
	OrderType       string     `json:"orderType"`
	BusinessStatus  string     `json:"businessStatus"`
	PaymentStatus   string     `json:"paymentStatus"`
	AmountCent      int64      `json:"amountCent"`
	PayMethod       string     `json:"payMethod"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	PaidAt          *time.Time `json:"paidAt,omitempty"`
}

// PaymentOrderFilter narrows a payment-orders list query. A nil StoreID means
// no scope filter (admin console); the store console always sets it from the
// JWT scope.
type PaymentOrderFilter struct {
	Page    httpx.Page
	StoreID *int64
	Status  string
}

// CollectionOrderView is the store-facing representation. The member fields are
// present only when a member was bound at creation; memberNickname is the masked
// nickname returned for operator confirmation on create and is omitted on reads
// (only the id and masked phone are persisted).
type CollectionOrderView struct {
	ID                int64     `json:"id"`
	CollectionOrderNo string    `json:"collectionOrderNo"`
	StoreID           int64     `json:"storeId"`
	AmountCent        int64     `json:"amountCent"`
	Subject           string    `json:"subject"`
	BusinessType      string    `json:"businessType"`
	Status            string    `json:"status"`
	PayChannel        string    `json:"payChannel"`
	MemberID          *int64    `json:"memberId,omitempty"`
	MemberNickname    string    `json:"memberNickname,omitempty"`
	MemberPhoneMasked string    `json:"memberPhoneMasked,omitempty"`
	QRContent         string    `json:"qrContent,omitempty"`
	ExpiresAt         time.Time `json:"expiresAt"`
	CreatedAt         time.Time `json:"createdAt"`
}

// RefundView is the store-facing refund representation.
type RefundView struct {
	ID             int64     `json:"id"`
	RefundOrderNo  string    `json:"refundOrderNo"`
	PaymentOrderID int64     `json:"paymentOrderId"`
	StoreID        int64     `json:"storeId"`
	AmountCent     int64     `json:"amountCent"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AdminRefundView is the admin-facing refund representation. Unlike the store
// console view it also exposes businessOrderId, storeId and channel since the
// admin console is not scoped to a single store.
type AdminRefundView struct {
	ID              int64     `json:"id"`
	RefundOrderNo   string    `json:"refundOrderNo"`
	PaymentOrderID  int64     `json:"paymentOrderId"`
	BusinessOrderID int64     `json:"businessOrderId"`
	StoreID         int64     `json:"storeId"`
	AmountCent      int64     `json:"amountCent"`
	Channel         string    `json:"channel"`
	Status          string    `json:"status"`
	Reason          string    `json:"reason,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// CreateCollectionOrderRequest is the store console create payload. storeId is
// never accepted from the client; it comes from the JWT scope. memberPhone is
// optional: when present it is used once to match a registered member and is
// never persisted; the caller may omit it to create a walk-in collection.
type CreateCollectionOrderRequest struct {
	AmountCent       int64  `json:"amountCent"`
	Subject          string `json:"subject"`
	BusinessType     string `json:"businessType"`
	ExpiresInSeconds int64  `json:"expiresInSeconds"`
	MemberPhone      string `json:"memberPhone"`
}

// CreateRefundRequest is the store console refund payload.
type CreateRefundRequest struct {
	PaymentOrderID     int64  `json:"paymentOrderId"`
	AmountCent         int64  `json:"amountCent"`
	Reason             string `json:"reason"`
	PasswordKeyID      string `json:"passwordKeyId"`
	PasswordCiphertext string `json:"passwordCiphertext"`
	Password           string `json:"-"`
}

// CollectionOrderCreate is the fully-resolved row set the repository persists in
// one transaction (business_order + payment_order + offline_collection_orders).
type CollectionOrderCreate struct {
	StoreID           int64
	AmountCent        int64
	Subject           string
	BusinessType      string
	BusinessOrderNo   string
	PaymentOrderNo    string
	CollectionOrderNo string
	AcquirerOrderNo   string
	QRContent         string
	ExpiresAt         time.Time
	CreatedByType     string
	CreatedByID       int64
	// Member binding, resolved and fixed at creation. MemberID is nil for a
	// walk-in collection; when set, the same operator (CreatedBy*) is recorded as
	// the binder and MemberPhoneMasked holds the masked snapshot. The raw phone is
	// never carried here.
	MemberID          *int64
	MemberPhoneMasked string
	IdemKey           string
	Now               time.Time
}

// RefundCreate is the resolved refund row plus its scope guard.
type RefundCreate struct {
	StoreID         int64
	PaymentOrderID  int64
	AmountCent      int64
	Reason          string
	RefundOrderNo   string
	RequestedByType string
	RequestedByID   int64
	IdemKey         string
	Now             time.Time
}

// StoreRepository is the store-scoped payment persistence port. Every read and
// write is filtered by the store scope passed from the service.
type StoreRepository interface {
	// ResolveMemberByPhone matches a registered member by normalized phone and
	// returns only masked identifiers. It returns a MEMBER_NOT_FOUND error when no
	// member matches; the raw phone is never returned, persisted or logged.
	ResolveMemberByPhone(ctx context.Context, phone string) (MemberMatch, error)
	CreateCollectionOrder(ctx context.Context, in CollectionOrderCreate) (CollectionOrder, error)
	GetCollectionOrder(ctx context.Context, storeID, id int64) (CollectionOrder, error)
	CancelCollectionOrder(ctx context.Context, storeID, id int64, now time.Time) error
	// ExpireCollections closes every pending collection order past its validity
	// window (offline-collection:expire, spec §11, §9.3.5): per order, in one
	// transaction, it expires the collection, its pending payment order and the
	// business order — guarded so a concurrent settlement always wins cleanly.
	// Returns the number expired.
	ExpireCollections(ctx context.Context, now time.Time) (int64, error)
	CreateRefund(ctx context.Context, in RefundCreate) (Refund, error)
	// CreateRefundAdmin inserts a pending refund for any store's payment order;
	// the store scope is resolved from the payment order itself rather than
	// verified against a caller-supplied store_id.
	CreateRefundAdmin(ctx context.Context, in RefundCreate) (Refund, error)
	CompleteRefundAdmin(ctx context.Context, refundID int64, externalRefundNo string, now time.Time) (Refund, error)
	FailRefundAdmin(ctx context.Context, refundID int64, now time.Time) error
	// ListPaymentOrders returns a page of payment_orders rows. A nil f.StoreID
	// means no scope filter (admin console); a set f.StoreID pins the query to
	// one store (store console).
	ListPaymentOrders(ctx context.Context, f PaymentOrderFilter) ([]PaymentOrder, int64, error)
	// GetPaymentOrder returns a single payment_orders row by id. A nil storeID
	// means no scope filter (admin console); a set storeID pins the lookup to
	// one store (store console) and returns NotFound if the order belongs to
	// another store.
	GetPaymentOrder(ctx context.Context, id int64, storeID *int64) (PaymentOrder, error)
}

// StoreService provides the store-console payment write operations.
type StoreService struct {
	repo                        StoreRepository
	wechat                      WeChatPayGateway
	passwords                   StoreAdminPasswordVerifier
	wechatPayAmountOverrideCent int64
	now                         func() time.Time
}

// StoreAdminPasswordVerifier re-authenticates an active store administrator
// from the same JWT-scoped store before a refund request is created.
type StoreAdminPasswordVerifier interface {
	VerifyStoreAdminPassword(ctx context.Context, storeID int64, password string) error
}

// NewStoreService builds the store payment service.
func NewStoreService(
	repo StoreRepository,
	wechat WeChatPayGateway,
	passwords StoreAdminPasswordVerifier,
	wechatPayAmountOverrideCent int64,
) *StoreService {
	return &StoreService{
		repo: repo, wechat: wechat, passwords: passwords,
		wechatPayAmountOverrideCent: wechatPayAmountOverrideCent,
		now:                         time.Now,
	}
}

// maxCollectionTTL caps the offline collection code validity. The window itself
// is client-supplied (expiresInSeconds); this is only a safety ceiling so the
// dynamic code cannot be turned into an effectively-static one.
const maxCollectionTTL = 24 * time.Hour

// CreateCollectionOrder validates the request, mints a dynamic QR via the
// acquirer and persists the order for the acting store. storeID comes from the
// JWT scope; the client cannot influence it.
func (s *StoreService) CreateCollectionOrder(ctx context.Context, storeID int64, byType string, byID int64, idemKey string, req CreateCollectionOrderRequest) (CollectionOrderView, error) {
	if req.AmountCent <= 0 {
		return CollectionOrderView{}, apperr.Invalid("amountCent must be positive")
	}
	req.Subject = strings.TrimSpace(req.Subject)
	if req.Subject == "" {
		return CollectionOrderView{}, apperr.Invalid("subject is required")
	}
	if utf8.RuneCountInString(req.Subject) > 127 {
		return CollectionOrderView{}, apperr.Invalid("subject must not exceed 127 characters")
	}
	if req.BusinessType == "" {
		return CollectionOrderView{}, apperr.Invalid("businessType is required")
	}
	if req.ExpiresInSeconds <= 0 {
		return CollectionOrderView{}, apperr.Invalid("expiresInSeconds must be positive")
	}
	ttl := time.Duration(req.ExpiresInSeconds) * time.Second
	if ttl > maxCollectionTTL {
		return CollectionOrderView{}, apperr.Invalid("expiresInSeconds exceeds the allowed maximum")
	}

	// Optional member binding: match once by normalized phone. A miss is a
	// controlled MEMBER_NOT_FOUND so the operator can retry without a member. The
	// raw phone is used only here and never persisted, logged or echoed back.
	var (
		match  MemberMatch
		bound  bool
		boundP *int64
	)
	if phone := normalizePhone(req.MemberPhone); phone != "" {
		m, err := s.repo.ResolveMemberByPhone(ctx, phone)
		if err != nil {
			return CollectionOrderView{}, err
		}
		match, bound = m, true
		boundP = &m.ID
	}

	now := s.now().UTC()
	expiresAt := now.Add(ttl)
	paymentOrderNo := newNo("PO", now)
	payAmountCent := req.AmountCent
	if s.wechatPayAmountOverrideCent > 0 {
		payAmountCent = s.wechatPayAmountOverrideCent
	}
	qrContent, err := s.wechat.CreateNativeOrder(ctx, paymentOrderNo, payAmountCent, req.Subject, expiresAt)
	if err != nil {
		return CollectionOrderView{}, apperr.From(err)
	}
	order, err := s.repo.CreateCollectionOrder(ctx, CollectionOrderCreate{
		StoreID:           storeID,
		AmountCent:        req.AmountCent,
		Subject:           req.Subject,
		BusinessType:      req.BusinessType,
		BusinessOrderNo:   newNo("BO", now),
		PaymentOrderNo:    paymentOrderNo,
		CollectionOrderNo: newNo("CO", now),
		QRContent:         qrContent,
		ExpiresAt:         expiresAt,
		CreatedByType:     byType,
		CreatedByID:       byID,
		MemberID:          boundP,
		MemberPhoneMasked: match.PhoneMasked,
		IdemKey:           idemKey,
		Now:               now,
	})
	if err != nil {
		// The WeChat order exists already. Best-effort compensation prevents an
		// untracked QR from remaining payable when the local transaction fails.
		_ = s.wechat.CloseOrder(ctx, paymentOrderNo)
		return CollectionOrderView{}, err
	}
	view := collectionOrderView(order)
	if bound {
		// The masked nickname is returned once for operator confirmation; it is
		// derived from the match and never persisted.
		view.MemberNickname = maskNickname(match.Nickname)
	}
	return view, nil
}

// GetCollectionOrder returns a store's own collection order.
func (s *StoreService) GetCollectionOrder(ctx context.Context, storeID, id int64) (CollectionOrderView, error) {
	order, err := s.repo.GetCollectionOrder(ctx, storeID, id)
	if err != nil {
		return CollectionOrderView{}, err
	}
	return collectionOrderView(order), nil
}

// CancelCollectionOrder cancels a still-pending collection order for the store.
func (s *StoreService) CancelCollectionOrder(ctx context.Context, storeID, id int64) error {
	order, err := s.repo.GetCollectionOrder(ctx, storeID, id)
	if err != nil {
		return err
	}
	if order.Status != CollectionPending {
		return apperr.Conflict("collection order cannot be cancelled")
	}
	if err := s.wechat.CloseOrder(ctx, order.PaymentOrderNo); err != nil {
		return apperr.From(err)
	}
	return s.repo.CancelCollectionOrder(ctx, storeID, id, s.now().UTC())
}

// CreateRefund verifies a manager password, then records a pending refund
// against a payment order that belongs to the acting store. Provider
// dispatch/settlement runs via the outbox worker and is documented as
// not-yet-implemented.
func (s *StoreService) CreateRefund(ctx context.Context, storeID int64, byType string, byID int64, idemKey string, req CreateRefundRequest) (RefundView, error) {
	if req.PaymentOrderID <= 0 {
		return RefundView{}, apperr.Invalid("paymentOrderId is required")
	}
	if req.AmountCent <= 0 {
		return RefundView{}, apperr.Invalid("amountCent must be positive")
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return RefundView{}, apperr.Invalid("请输入退款原因")
	}
	if utf8.RuneCountInString(req.Reason) > 255 {
		return RefundView{}, apperr.Invalid("退款原因不能超过 255 个字符")
	}
	if s.passwords == nil {
		return RefundView{}, apperr.Internal(fmt.Errorf("store admin password verifier is not configured"))
	}
	if err := s.passwords.VerifyStoreAdminPassword(ctx, storeID, req.Password); err != nil {
		return RefundView{}, err
	}
	now := s.now().UTC()
	refund, err := s.repo.CreateRefund(ctx, RefundCreate{
		StoreID:         storeID,
		PaymentOrderID:  req.PaymentOrderID,
		AmountCent:      req.AmountCent,
		Reason:          req.Reason,
		RefundOrderNo:   newNo("RF", now),
		RequestedByType: byType,
		RequestedByID:   byID,
		IdemKey:         idemKey,
		Now:             now,
	})
	if err != nil {
		return RefundView{}, err
	}
	return refundView(refund), nil
}

// ListPaymentOrders returns a page of payment orders scoped to the acting
// store; storeID is never client-supplied.
func (s *StoreService) ListPaymentOrders(ctx context.Context, storeID int64, status string, page httpx.Page) ([]PaymentOrderView, int64, error) {
	rows, total, err := s.repo.ListPaymentOrders(ctx, PaymentOrderFilter{Page: page, StoreID: &storeID, Status: status})
	if err != nil {
		return nil, 0, err
	}
	out := make([]PaymentOrderView, 0, len(rows))
	for _, r := range rows {
		out = append(out, paymentOrderView(r))
	}
	return out, total, nil
}

// GetPaymentOrder returns a single payment order scoped to the acting store;
// storeID is never client-supplied.
func (s *StoreService) GetPaymentOrder(ctx context.Context, storeID, id int64) (PaymentOrderView, error) {
	row, err := s.repo.GetPaymentOrder(ctx, id, &storeID)
	if err != nil {
		return PaymentOrderView{}, err
	}
	return paymentOrderView(row), nil
}

// AdminService provides the admin-console refund write operation. Unlike
// StoreService it is not scoped by a caller store_id.
type AdminService struct {
	repo                     StoreRepository
	wechat                   WeChatPayGateway
	offline                  OfflineAcquirer
	passwords                AdminPasswordVerifier
	refundAmountOverrideCent int64
	now                      func() time.Time
	channels                 *channelSettingsStore
}

// AdminPasswordVerifier re-authenticates the current admin before a refund.
type AdminPasswordVerifier interface {
	VerifyAccountPassword(ctx context.Context, accountID int64, password string) error
}

// NewAdminService builds the admin payment service.
func NewAdminService(
	repo StoreRepository,
	wechat WeChatPayGateway,
	offline OfflineAcquirer,
	passwords AdminPasswordVerifier,
	refundAmountOverrideCent int64,
) *AdminService {
	return &AdminService{
		repo: repo, wechat: wechat, offline: offline, passwords: passwords,
		refundAmountOverrideCent: refundAmountOverrideCent,
		now:                      time.Now, channels: newChannelSettingsStore(),
	}
}

// ListChannelSettings returns every payment channel's admin-configurable toggle.
func (s *AdminService) ListChannelSettings(ctx context.Context) []ChannelSetting {
	return s.channels.List()
}

// UpdateChannelSettings applies the requested channel toggles.
func (s *AdminService) UpdateChannelSettings(ctx context.Context, req UpdateChannelSettingsRequest) ([]ChannelSetting, error) {
	return s.channels.Update(req.Channels)
}

// CreateRefund verifies the administrator and executes a full refund through
// the payment order's original channel.
func (s *AdminService) CreateRefund(ctx context.Context, byType string, byID int64, idemKey string, req CreateRefundRequest) (AdminRefundView, error) {
	return s.createRefund(ctx, byType, byID, idemKey, req, true)
}

// CreateFoodOrderCancellationRefund executes a full, store-scoped food-order
// refund. Caller authentication and the optional force-cancel password check are
// owned by the order service; this method still re-checks the payment order's
// store and order type before touching money.
func (s *AdminService) CreateFoodOrderCancellationRefund(
	ctx context.Context,
	storeID int64,
	byType string,
	byID int64,
	idemKey string,
	req CreateRefundRequest,
) (AdminRefundView, error) {
	paymentOrder, err := s.repo.GetPaymentOrder(ctx, req.PaymentOrderID, &storeID)
	if err != nil {
		return AdminRefundView{}, err
	}
	if paymentOrder.OrderType != "food" {
		return AdminRefundView{}, apperr.Invalid("仅支持取消本店点餐订单")
	}
	return s.createRefund(ctx, byType, byID, idemKey, req, false)
}

func (s *AdminService) createRefund(ctx context.Context, byType string, byID int64, idemKey string, req CreateRefundRequest, verifyPassword bool) (AdminRefundView, error) {
	if req.PaymentOrderID <= 0 {
		return AdminRefundView{}, apperr.Invalid("paymentOrderId is required")
	}
	if req.AmountCent <= 0 {
		return AdminRefundView{}, apperr.Invalid("amountCent must be positive")
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return AdminRefundView{}, apperr.Invalid("请输入退款原因")
	}
	if utf8.RuneCountInString(req.Reason) > 255 {
		return AdminRefundView{}, apperr.Invalid("退款原因不能超过 255 个字符")
	}
	if verifyPassword {
		if s.passwords == nil {
			return AdminRefundView{}, apperr.Internal(fmt.Errorf("admin password verifier is not configured"))
		}
		if err := s.passwords.VerifyAccountPassword(ctx, byID, req.Password); err != nil {
			return AdminRefundView{}, err
		}
	}
	now := s.now().UTC()
	refund, err := s.repo.CreateRefundAdmin(ctx, RefundCreate{
		PaymentOrderID:  req.PaymentOrderID,
		AmountCent:      req.AmountCent,
		Reason:          req.Reason,
		RefundOrderNo:   newNo("RF", now),
		RequestedByType: byType,
		RequestedByID:   byID,
		IdemKey:         idemKey,
		Now:             now,
	})
	if err != nil {
		return AdminRefundView{}, err
	}

	var externalRefundNo string
	switch refund.Channel {
	case "wechat":
		refundCent, totalCent := refund.AmountCent, refund.PaymentAmountCent
		if s.refundAmountOverrideCent > 0 {
			refundCent, totalCent = s.refundAmountOverrideCent, s.refundAmountOverrideCent
		}
		externalRefundNo, err = s.wechat.Refund(
			ctx, refund.PaymentOrderNo, refund.RefundOrderNo, refundCent, totalCent,
		)
	case "coin":
		// Coin refunds are completed transactionally in the repository below.
	case "offline":
		externalRefundNo, err = s.offline.Refund(
			ctx, refund.AcquirerOrderNo, refund.RefundOrderNo, refund.AmountCent,
		)
	default:
		err = apperr.Invalid("该支付渠道暂不支持退款")
	}
	if err != nil {
		if rollbackErr := s.repo.FailRefundAdmin(ctx, refund.ID, now); rollbackErr != nil {
			return AdminRefundView{}, apperr.Internal(fmt.Errorf("退款渠道失败后回滚权益: %w", rollbackErr))
		}
		return AdminRefundView{}, apperr.From(err)
	}
	refund, err = s.repo.CompleteRefundAdmin(ctx, refund.ID, externalRefundNo, now)
	if err != nil {
		return AdminRefundView{}, err
	}
	return adminRefundView(refund), nil
}

// ListPaymentOrders returns a page of payment orders, optionally narrowed by
// f.StoreID; the admin console is not pinned to a single store.
func (s *AdminService) ListPaymentOrders(ctx context.Context, f PaymentOrderFilter) ([]PaymentOrderView, int64, error) {
	rows, total, err := s.repo.ListPaymentOrders(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PaymentOrderView, 0, len(rows))
	for _, r := range rows {
		out = append(out, paymentOrderView(r))
	}
	return out, total, nil
}

// GetPaymentOrder returns a single payment order by id; the admin console is
// not pinned to a single store.
func (s *AdminService) GetPaymentOrder(ctx context.Context, id int64) (PaymentOrderView, error) {
	row, err := s.repo.GetPaymentOrder(ctx, id, nil)
	if err != nil {
		return PaymentOrderView{}, err
	}
	return paymentOrderView(row), nil
}

func adminRefundView(r Refund) AdminRefundView {
	return AdminRefundView{
		ID:              r.ID,
		RefundOrderNo:   r.RefundOrderNo,
		PaymentOrderID:  r.PaymentOrderID,
		BusinessOrderID: r.BusinessOrderID,
		StoreID:         r.StoreID,
		AmountCent:      r.AmountCent,
		Channel:         r.Channel,
		Status:          r.Status,
		Reason:          r.Reason,
		CreatedAt:       r.CreatedAt,
	}
}

func collectionOrderView(o CollectionOrder) CollectionOrderView {
	return CollectionOrderView{
		ID:                o.ID,
		CollectionOrderNo: o.CollectionOrderNo,
		StoreID:           o.StoreID,
		AmountCent:        o.AmountCent,
		Subject:           o.Subject,
		BusinessType:      o.BusinessType,
		Status:            o.Status,
		PayChannel:        o.PayMethod,
		MemberID:          o.MemberID,
		MemberPhoneMasked: o.MemberPhoneMasked,
		QRContent:         o.QRContent,
		ExpiresAt:         o.ExpiresAt,
		CreatedAt:         o.CreatedAt,
	}
}

func paymentOrderView(o PaymentOrder) PaymentOrderView {
	return PaymentOrderView{
		ID:              o.ID,
		PaymentOrderNo:  o.PaymentOrderNo,
		StoreID:         o.StoreID,
		StoreName:       o.StoreName,
		MemberID:        o.MemberID,
		BusinessOrderID: o.BusinessOrderID,
		BusinessOrderNo: o.BusinessOrderNo,
		OrderType:       o.OrderType,
		BusinessStatus:  o.BusinessStatus,
		PaymentStatus:   o.PaymentStatus,
		AmountCent:      o.AmountCent,
		PayMethod:       o.PayMethod,
		Status:          o.Status,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
		PaidAt:          o.PaidAt,
	}
}

func refundView(r Refund) RefundView {
	return RefundView{
		ID:             r.ID,
		RefundOrderNo:  r.RefundOrderNo,
		PaymentOrderID: r.PaymentOrderID,
		StoreID:        r.StoreID,
		AmountCent:     r.AmountCent,
		Status:         r.Status,
		Reason:         r.Reason,
		CreatedAt:      r.CreatedAt,
	}
}

// newNo builds a human-readable, collision-resistant business number.
func newNo(prefix string, now time.Time) string {
	return fmt.Sprintf("%s%s%04d", prefix, now.Format("20060102150405"), rand.Intn(10000))
}

// normalizePhone trims surrounding whitespace and drops inner spaces/dashes so a
// pasted "138 0000 0000" matches a stored "13800000000". It does not attempt
// country-code rewriting; the match is against the stored value as-is.
func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	return phone
}

// maskPhone keeps the leading 3 and trailing 4 digits, masking the middle. Short
// values are returned unchanged so a malformed stored value is not exposed.
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// maskNickname reveals only the first rune, so the operator can confirm the
// matched member without the full identity leaking to the console.
func maskNickname(nickname string) string {
	r := []rune(nickname)
	if len(r) == 0 {
		return ""
	}
	if len(r) == 1 {
		return string(r[0]) + "*"
	}
	return string(r[0]) + "***"
}
