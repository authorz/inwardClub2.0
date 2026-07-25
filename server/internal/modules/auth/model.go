package auth

import "time"

// Member is a mini-program member identity.
type Member struct {
	ID                int64
	WeChatOpenID      string
	Nickname          string
	AvatarURL         string
	Gender            string
	Phone             string
	InviteCode        string
	InvitedByMemberID *int64
	Status            string
	TokenVersion      int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
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

// Staff is a mini-program staff binding. The token subject remains MemberID so
// member-facing mini endpoints continue to resolve the underlying member.
type Staff struct {
	ID           int64
	MemberID     int64
	StoreID      int64
	Status       string
	TokenVersion int64
}

// Account status values.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)
