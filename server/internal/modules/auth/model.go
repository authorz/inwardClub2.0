package auth

import "time"

// Member is a mini-program member identity.
type Member struct {
	ID           int64
	WeChatOpenID string
	Nickname     string
	Phone        string
	InviteCode   string
	Status       string
	TokenVersion int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Account is a back-office account (admin console or store console).
type Account struct {
	ID           int64
	Username     string
	PasswordHash string
	DisplayName  string
	Role         string
	StoreID      int64 // 0 for global super_admin
	Status       string
	TokenVersion int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Account status values.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)
