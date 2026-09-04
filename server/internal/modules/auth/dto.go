package auth

import "github.com/inwardclub/server/internal/platform/authn"

// WeChatLoginRequest is the mini-program login body.
type WeChatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

// WeChatRegisterRequest completes a first-time member's registration. It is
// authorized by the register ticket returned from the phone-mask step, which
// already carries the authorized phone (no member exists until this succeeds).
// AvatarURL, Nickname, and Gender are all required before a full member session
// is issued; an OpenID-only pre-registration may already exist.
type WeChatRegisterRequest struct {
	RegisterTicket string `json:"registerTicket" binding:"required"`
	AvatarURL      string `json:"avatarUrl" binding:"required"`
	Nickname       string `json:"nickname" binding:"required"`
	Gender         string `json:"gender" binding:"required"`
	InviterCode    string `json:"inviterCode,omitempty"`
}

// PasswordLoginRequest is the back-office login body.
type PasswordLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest carries a refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// StaffStoreSwitchRequest selects one of the caller's existing staff bindings.
type StaffStoreSwitchRequest struct {
	StoreID int64 `json:"storeId" binding:"required"`
}

// LoginResponse is returned on successful authentication. IsNew is true only for
// a mini member's first-time registration login, so the client can prompt the
// new user for profile details (phone, avatar, nickname) and skip that prompt
// for returning members. It is always false for back-office logins.
type LoginResponse struct {
	Token       authn.TokenPair   `json:"token"`
	Profile     any               `json:"profile"`
	IsNew       bool              `json:"isNew"`
	SubjectType authn.SubjectType `json:"subjectType,omitempty"`
	StoreID     int64             `json:"storeId,omitempty"`
	// RegisterTicket is set (with an empty Token) only when a first-time member
	// must complete the profile form before any member row is created. The client
	// submits it back to /mini/auth/wechat/register.
	RegisterTicket string `json:"registerTicket,omitempty"`
}

// MemberProfile is the mini-program "me" payload. VipTier carries the member's
// current VIP level; it is nil when the member is not yet ranked.
type MemberProfile struct {
	ID           int64          `json:"id"`
	Nickname     string         `json:"nickname"`
	Gender       string         `json:"gender,omitempty"`
	AvatarURL    string         `json:"avatarUrl,omitempty"`
	MemberNo     string         `json:"memberNo,omitempty"`
	Phone        string         `json:"phone,omitempty"`
	InviteCode   string         `json:"inviteCode,omitempty"`
	InviterBound bool           `json:"inviterBound"`
	Status       string         `json:"status"`
	VipTier      *MemberVIPTier `json:"vipTier,omitempty"`
}

// MemberVIPTier is the member's current VIP level as surfaced on "me". It mirrors
// the tier catalogue view but is defined here so the auth module stays decoupled
// from the member module (the resolver is wired at composition time).
//
// Label is the short, user-facing VIP identity (e.g. "VIP1"), derived from the
// tier's level — never the full admin tier name (e.g. "VIP1 普通会员"), which is
// reserved for back-office management surfaces.
type MemberVIPTier struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	Level     int    `json:"level"`
	Threshold int64  `json:"threshold"`
	IconURL   string `json:"iconUrl,omitempty"`
}

// AccountProfile is the back-office "me" payload.
type AccountProfile struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	StoreID     int64  `json:"storeId,omitempty"`
	Status      string `json:"status"`
}
