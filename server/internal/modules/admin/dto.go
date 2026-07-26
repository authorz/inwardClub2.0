package admin

import (
	"encoding/json"
	"time"

	"github.com/inwardclub/server/internal/modules/wallet"
)

// StoreView is the console list representation of a store.
type StoreView struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// CatalogItemView is the console list representation of a catalog item.
type CatalogItemView struct {
	ID            int64     `json:"id"`
	ScopeType     string    `json:"scopeType"`
	StoreID       *int64    `json:"storeId,omitempty"`
	CategoryID    *int64    `json:"categoryId,omitempty"`
	Name          string    `json:"name"`
	ItemType      string    `json:"itemType"`
	PriceCent     int64     `json:"priceCent"`
	StockQuantity int64     `json:"stockQuantity"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// CouponTemplateView is the console list representation of a coupon template.
type CouponTemplateView struct {
	ID          int64     `json:"id"`
	ScopeType   string    `json:"scopeType"`
	StoreID     *int64    `json:"storeId,omitempty"`
	Name        string    `json:"name"`
	CouponType  string    `json:"couponType"`
	ValueCent   int64     `json:"valueCent"`
	TotalStock  int64     `json:"totalStock"`
	IssuedCount int64     `json:"issuedCount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ActivityView is the console list representation of an activity.
type ActivityView struct {
	ID        int64      `json:"id"`
	ScopeType string     `json:"scopeType"`
	StoreID   *int64     `json:"storeId,omitempty"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	AssetID   *int64     `json:"assetId,omitempty"`
	ImageURL  string     `json:"imageUrl,omitempty"`
	StartAt   *time.Time `json:"startAt,omitempty"`
	EndAt     *time.Time `json:"endAt,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// OrderView is the console list representation of an order (the unified
// order-center read model shared by GET /admin/orders and GET /store/orders).
// businessOrderId/amountCent/orderType/paymentStatus/orderStatus/storeName/
// member* are what the admin and store console order-center tables render;
// orderNo/totalCent/status are retained as compatibility aliases.
type OrderView struct {
	ID                int64      `json:"id"`
	BusinessOrderID   int64      `json:"businessOrderId"`
	OrderNo           string     `json:"orderNo"`
	OrderType         string     `json:"orderType"`
	StoreID           *int64     `json:"storeId,omitempty"`
	StoreName         string     `json:"storeName,omitempty"`
	MemberID          *int64     `json:"memberId,omitempty"`
	MemberNickname    string     `json:"memberNickname,omitempty"`
	MemberPhone       string     `json:"memberPhone,omitempty"`
	MemberAvatarURL   string     `json:"memberAvatarUrl,omitempty"`
	MemberPhoneMasked string     `json:"memberPhoneMasked,omitempty"`
	PaymentOrderID    int64      `json:"paymentOrderId,omitempty"`
	TotalCent         int64      `json:"totalCent"`
	AmountCent        int64      `json:"amountCent"`
	PayChannel        string     `json:"payChannel,omitempty"`
	RefundStatus      string     `json:"refundStatus,omitempty"`
	PaymentStatus     string     `json:"paymentStatus"`
	OrderStatus       string     `json:"orderStatus"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"createdAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
}

// MemberView is the console list representation of a member.
type MemberView struct {
	ID            int64     `json:"id"`
	Nickname      string    `json:"nickname,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	AvatarURL     string    `json:"avatarUrl,omitempty"`
	Gender        string    `json:"gender,omitempty"`
	PointsBalance int64     `json:"pointsBalance"`
	CoinsBalance  int64     `json:"coinsBalance"`
	VIPTierName   string    `json:"vipTierName,omitempty"`
	VIPLevel      int       `json:"vipLevel"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// MemberDetailView is the store console single-member detail: the member
// summary plus its wallet balances.
type MemberDetailView struct {
	MemberView
	Wallet []wallet.Account `json:"wallet"`
}

// WalletAdjustmentRequest is the POST /store/members/:memberID/wallet-adjustments
// body: a store-initiated correction to one of the member's wallet balances.
type WalletAdjustmentRequest struct {
	AssetType string `json:"assetType" binding:"required"`
	Direction string `json:"direction" binding:"required"`
	Amount    int64  `json:"amount" binding:"required"`
	Reason    string `json:"reason,omitempty"`
}

// WalletAdjustmentView is the result of applying a wallet adjustment.
type WalletAdjustmentView struct {
	AssetType    string `json:"assetType"`
	Direction    string `json:"direction"`
	Amount       int64  `json:"amount"`
	BalanceAfter int64  `json:"balanceAfter"`
	Reason       string `json:"reason,omitempty"`
}

// WalletLedgerEntryView is the console list representation of a wallet ledger
// entry, carrying the fields useful for audit review.
type WalletLedgerEntryView struct {
	ID              int64     `json:"id"`
	RecordKey       string    `json:"recordKey"`
	MemberID        int64     `json:"memberId"`
	MemberNickname  string    `json:"memberNickname,omitempty"`
	MemberPhone     string    `json:"memberPhone,omitempty"`
	MemberAvatarURL string    `json:"memberAvatarUrl,omitempty"`
	StoreID         *int64    `json:"storeId,omitempty"`
	StoreName       string    `json:"storeName,omitempty"`
	AssetType       string    `json:"assetType"`
	Direction       string    `json:"direction"`
	Amount          int64     `json:"amount"`
	BalanceAfter    *int64    `json:"balanceAfter,omitempty"`
	Status          string    `json:"status"`
	Reason          string    `json:"reason"`
	SourceType      string    `json:"sourceType"`
	SourceID        *int64    `json:"sourceId,omitempty"`
	RelatedOrderNo  string    `json:"relatedOrderNo,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// PaymentTransactionView is the console list representation of a payment order.
type PaymentTransactionView struct {
	ID              int64      `json:"id"`
	PaymentOrderNo  string     `json:"paymentOrderNo"`
	StoreID         *int64     `json:"storeId,omitempty"`
	StoreName       string     `json:"storeName,omitempty"`
	BusinessOrderID int64      `json:"businessOrderId"`
	BusinessOrderNo string     `json:"businessOrderNo"`
	OrderType       string     `json:"orderType"`
	AmountCent      int64      `json:"amountCent"`
	PayMethod       string     `json:"payMethod"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	PaidAt          *time.Time `json:"paidAt,omitempty"`
}

// RefundView is the console list representation of a refund order.
type RefundView struct {
	ID              int64     `json:"id"`
	RefundOrderNo   string    `json:"refundOrderNo"`
	PaymentOrderID  int64     `json:"paymentOrderId"`
	BusinessOrderID int64     `json:"businessOrderId"`
	StoreID         *int64    `json:"storeId,omitempty"`
	StoreName       string    `json:"storeName,omitempty"`
	BusinessOrderNo string    `json:"businessOrderNo"`
	OrderAmountCent int64     `json:"orderAmountCent"`
	MemberID        *int64    `json:"memberId,omitempty"`
	MemberNickname  string    `json:"memberNickname,omitempty"`
	MemberPhone     string    `json:"memberPhone,omitempty"`
	MemberAvatarURL string    `json:"memberAvatarUrl,omitempty"`
	AmountCent      int64     `json:"amountCent"`
	Channel         string    `json:"channel"`
	Status          string    `json:"status"`
	Reason          string    `json:"reason,omitempty"`
	OrderCreatedAt  time.Time `json:"orderCreatedAt"`
	OperatedAt      time.Time `json:"operatedAt"`
}

// AuditLogView is the console list representation of an audit-trail row.
type AuditLogView struct {
	ID         int64     `json:"id"`
	ActorType  string    `json:"actorType"`
	ActorID    int64     `json:"actorId"`
	Action     string    `json:"action"`
	TargetType string    `json:"targetType,omitempty"`
	TargetID   string    `json:"targetId,omitempty"`
	StoreID    *int64    `json:"storeId,omitempty"`
	RequestID  string    `json:"requestId,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// RuleDefinitionView is the console representation of a business rule.
type RuleDefinitionView struct {
	ID         int64           `json:"id"`
	Key        string          `json:"ruleKey"`
	ScopeType  string          `json:"scopeType"`
	StoreID    *int64          `json:"storeId,omitempty"`
	Version    int             `json:"version"`
	ConfigJSON json.RawMessage `json:"configJson"`
	Enabled    bool            `json:"enabled"`
	Status     string          `json:"status"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// RuleDefinitionUpdateRequest is the PATCH body for a rule definition. Omitted
// fields are left unchanged.
type RuleDefinitionUpdateRequest struct {
	ConfigJSON json.RawMessage `json:"configJson,omitempty"`
	Enabled    *bool           `json:"enabled,omitempty"`
	Status     *string         `json:"status,omitempty"`
}

// RuleDefinitionCreateRequest is the POST /admin/rule-definitions body. New
// rows always start in the 'draft' status; scopeType defaults to "global" and
// version defaults to 1 when omitted.
type RuleDefinitionCreateRequest struct {
	RuleKey    string          `json:"ruleKey"`
	ScopeType  string          `json:"scopeType,omitempty"`
	StoreID    *int64          `json:"storeId,omitempty"`
	Version    int             `json:"version,omitempty"`
	ConfigJSON json.RawMessage `json:"configJson"`
	Enabled    bool            `json:"enabled,omitempty"`
}

// AdminAccountView is the console list representation of a back-office login
// account (admin_accounts row).
type AdminAccountView struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	IsSystem    bool      `json:"isSystem"`
	StoreID     *int64    `json:"storeId,omitempty"`
	StoreName   string    `json:"storeName,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AdminAccountCreateRequest is the POST /admin/admin-accounts body.
type AdminAccountCreateRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"displayName"`
}

// AdminAccountUpdateRequest is the PATCH /admin/admin-accounts/:accountID
// body. Username and system status are immutable.
type AdminAccountUpdateRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Password    *string `json:"password,omitempty"`
}

// StaffAccountView is the console list representation of a staff_accounts row.
// MemberID/Phone identify the bound mini-program member.
type StaffAccountView struct {
	ID        int64     `json:"id"`
	MemberID  int64     `json:"memberId,omitempty"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	StoreID   int64     `json:"storeId"`
	StoreName string    `json:"storeName,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// CashierCreateRequest is the POST /store/cashiers body.
type CashierCreateRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

// CashierUpdateRequest is the PATCH /store/cashiers/:cashierID body. Only
// displayName is editable; username/role/store are fixed at creation and
// status changes go through the dedicated disable action.
type CashierUpdateRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
}

// CashierCredentialView is returned once, on create and on password reset,
// carrying the plaintext initial password so the console can hand it to the
// operator immediately; it is never retrievable again afterwards.
type CashierCredentialView struct {
	AdminAccountView
	InitialPassword string `json:"initialPassword"`
}

// StaffAccountCreateRequest is the POST /store/staff-accounts body. A staff
// member must already be a registered mini-program member; the store binds them
// by memberId (resolved via GET /store/member-lookup?phone=). Name is the staff
// display name entered on the create form; when blank it falls back to the
// member's nickname.
type StaffAccountCreateRequest struct {
	MemberID int64  `json:"memberId" binding:"required"`
	Name     string `json:"name"`
}

// StaffAccountUpdateRequest is the PATCH /store/staff-accounts/:staffID body.
type StaffAccountUpdateRequest struct {
	Name *string `json:"name,omitempty"`
}

// AdminStaffAccountCreateRequest is the POST /admin/staff-accounts body. Unlike
// the store console, the admin console is not pinned to one store, so the target
// store is supplied by the caller. The staff member must already be a registered
// mini-program member, bound by memberId (resolved via
// GET /admin/member-lookup?phone=).
type AdminStaffAccountCreateRequest struct {
	StoreID  int64  `json:"storeId" binding:"required"`
	MemberID int64  `json:"memberId" binding:"required"`
	Name     string `json:"name"`
}

// AdminStaffAccountUpdateRequest is the PATCH /admin/staff-accounts/:staffID
// body. A nil field is left unchanged; storeId lets headquarters reassign the
// staff member to a different store.
type AdminStaffAccountUpdateRequest struct {
	StoreID *int64  `json:"storeId,omitempty"`
	Name    *string `json:"name,omitempty"`
}

// StoreAdminCreateRequest is the POST /admin/store-admin-accounts body: a
// headquarters-created store_admin login account, pinned to storeId and using
// the caller-supplied initial password.
type StoreAdminCreateRequest struct {
	StoreID     int64  `json:"storeId"`
	Username    string `json:"username"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"displayName"`
}

// StoreAdminUpdateRequest is the PATCH /admin/store-admin-accounts/:accountID
// body. A nil field is left unchanged; storeId reassigns the account and
// password replaces the login password.
type StoreAdminUpdateRequest struct {
	StoreID     *int64  `json:"storeId,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Password    *string `json:"password,omitempty"`
}

// StoreProfileView is the single store profile returned to the store console.
type StoreProfileView struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	LogoURL       string `json:"logoUrl,omitempty"`
	Phone         string `json:"phone,omitempty"`
	Address       string `json:"address"`
	BusinessHours string `json:"businessHours,omitempty"`
	Status        string `json:"status"`
}
