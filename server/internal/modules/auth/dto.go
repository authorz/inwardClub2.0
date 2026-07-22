package auth

import "github.com/inwardclub/server/internal/platform/authn"

// WeChatLoginRequest is the mini-program login body.
type WeChatLoginRequest struct {
	Code string `json:"code" binding:"required"`
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

// LoginResponse is returned on successful authentication. IsNew is true only for
// a mini member's first-time registration login, so the client can prompt the
// new user for profile details (phone, avatar, nickname) and skip that prompt
// for returning members. It is always false for back-office logins.
type LoginResponse struct {
	Token   authn.TokenPair `json:"token"`
	Profile any             `json:"profile"`
	IsNew   bool            `json:"isNew"`
}

// MemberProfile is the mini-program "me" payload. VipTier carries the member's
// current VIP level (and its banner URL) so the home page can render the tier
// banner after login; it is nil when the member is not yet ranked.
type MemberProfile struct {
	ID         int64          `json:"id"`
	Nickname   string         `json:"nickname"`
	MemberNo   string         `json:"memberNo,omitempty"`
	Phone      string         `json:"phone,omitempty"`
	InviteCode string         `json:"inviteCode,omitempty"`
	Status     string         `json:"status"`
	VipTier    *MemberVIPTier `json:"vipTier,omitempty"`
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
	BannerURL string `json:"bannerUrl,omitempty"`
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
