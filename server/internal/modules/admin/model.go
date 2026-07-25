// Package admin owns the headquarters console read models. Phase-1 exposes the
// list/read paths that back the admin console (and, via store scope, the
// single-store console). Console CRUD and writes are layered on later.
//
// Every list read flows through a single ListFilter so the admin (all scopes)
// and store (pinned scope) consoles share one code path: a nil StoreID means
// "no scope filter" (admin), a set StoreID pins the query to one store.
package admin

import (
	"encoding/json"
	"time"
)

// StoreSummary is the console list representation of a store.
type StoreSummary struct {
	ID        int64
	Name      string
	Phone     string
	Address   string
	Status    string
	CreatedAt time.Time
}

// CatalogItem is the console read model for a catalog item.
type CatalogItem struct {
	ID            int64
	ScopeType     string
	StoreID       *int64
	CategoryID    *int64
	Name          string
	ItemType      string
	PriceCent     int64
	StockQuantity int64
	Status        string
	CreatedAt     time.Time
}

// CouponTemplate is the console read model for a coupon template.
type CouponTemplate struct {
	ID          int64
	ScopeType   string
	StoreID     *int64
	Name        string
	CouponType  string
	ValueCent   int64
	TotalStock  int64
	IssuedCount int64
	Status      string
	CreatedAt   time.Time
}

// Activity is the console read model for a marketing activity.
type Activity struct {
	ID        int64
	ScopeType string
	StoreID   *int64
	Name      string
	Type      string
	AssetID   *int64
	StartAt   *time.Time
	EndAt     *time.Time
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Order is the console read model for an order. It carries the joined store and
// member display fields plus the order/payment status pair the order-center
// pages render.
type Order struct {
	ID             int64
	OrderNo        string
	OrderType      string
	StoreID        *int64
	StoreName      string
	MemberID       *int64
	MemberNickname string
	MemberPhone    string
	TotalCent      int64
	PayChannel     string
	PaymentStatus  string
	OrderStatus    string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

// Member is the console read model for a member.
type Member struct {
	ID            int64
	Nickname      string
	Phone         string
	AvatarURL     string
	Gender        string
	PointsBalance int64
	CoinsBalance  int64
	VIPTierName   string
	VIPLevel      int
	Status        string
	CreatedAt     time.Time
}

// WalletLedgerEntry is the console read model for a wallet_ledger_entries row.
type WalletLedgerEntry struct {
	ID           int64
	MemberID     int64
	AssetType    string
	Direction    string
	Amount       int64
	BalanceAfter int64
	Reason       string
	SourceType   string
	SourceID     *int64
	CreatedAt    time.Time
}

// AuditLog is the console read model for an audit-trail row.
type AuditLog struct {
	ID         int64
	ActorType  string
	ActorID    int64
	Action     string
	TargetType string
	TargetID   string
	StoreID    *int64
	RequestID  string
	CreatedAt  time.Time
}

// RuleDefinition is the console read model for a configurable business rule.
// ConfigJSON carries the rule's raw config document (e.g. the sign_in daily
// rewards table) verbatim so the console can render and edit it.
type RuleDefinition struct {
	ID         int64
	Key        string
	ScopeType  string
	StoreID    *int64
	Version    int
	ConfigJSON json.RawMessage
	Enabled    bool
	Status     string
	UpdatedAt  time.Time
}

// RuleDefinitionUpdate is a partial update to a rule definition; a nil field is
// left unchanged. At least one field must be set.
type RuleDefinitionUpdate struct {
	ConfigJSON json.RawMessage
	Enabled    *bool
	Status     *string
}

// RuleDefinitionCreate is the input to creating a new rule definition version.
// ScopeType defaults to "global" and Version defaults to 1 when left zero.
type RuleDefinitionCreate struct {
	Key        string
	ScopeType  string
	StoreID    *int64
	Version    int
	ConfigJSON json.RawMessage
	Enabled    bool
}

// PaymentTransaction is the console read model for a payment order, joined with
// its business order and store for display.
type PaymentTransaction struct {
	ID              int64
	PaymentOrderNo  string
	StoreID         *int64
	StoreName       string
	BusinessOrderID int64
	BusinessOrderNo string
	OrderType       string
	AmountCent      int64
	PayMethod       string
	Status          string
	CreatedAt       time.Time
	PaidAt          *time.Time
}

// Refund is the console read model for a refund order, joined with its payment
// order, business order and store for display.
type Refund struct {
	ID              int64
	RefundOrderNo   string
	PaymentOrderID  int64
	BusinessOrderID int64
	StoreID         *int64
	StoreName       string
	AmountCent      int64
	Channel         string
	Status          string
	Reason          string
	CreatedAt       time.Time
}

// AdminAccount is the console read model for a back-office login account
// (admin_accounts row): super_admin, store_admin or cashier.
type AdminAccount struct {
	ID          int64
	Username    string
	DisplayName string
	Role        string
	IsSystem    bool
	StoreID     *int64
	StoreName   string
	Status      string
	CreatedAt   time.Time
}

// StaffAccount is the console read model for a staff_accounts row: a
// WeChat-bound store staff identity, distinct from admin_accounts login accounts.
// MemberID/Phone come from the bound member (staff_accounts.member_id → members).
type StaffAccount struct {
	ID        int64
	MemberID  int64
	Name      string
	Phone     string
	StoreID   int64
	StoreName string
	Status    string
	CreatedAt time.Time
}

// Status values shared by console list endpoints.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)
