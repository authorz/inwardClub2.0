package member

// UpdateProfileRequest is the PATCH /mini/me body. Fields are optional; only the
// provided ones are updated.
type UpdateProfileRequest struct {
	Nickname  *string `json:"nickname"`
	Gender    *string `json:"gender"`
	AssetID   *int64  `json:"assetId"`
	AvatarURL *string `json:"avatarUrl"`
}

// BindPhoneRequest is the POST /mini/me/phone-bindings body. Code is the WeChat
// phone authorisation code exchanged server-side for the real phone number.
type BindPhoneRequest struct {
	Code string `json:"code" binding:"required"`
}

// BindInvitationRequest is the POST /mini/invitations/bind body.
type BindInvitationRequest struct {
	InviteCode string `json:"inviteCode" binding:"required"`
}

// MemberView is the public member profile payload.
type MemberView struct {
	ID         int64  `json:"id"`
	Nickname   string `json:"nickname"`
	Gender     string `json:"gender,omitempty"`
	Phone      string `json:"phone,omitempty"`
	AvatarURL  string `json:"avatarUrl,omitempty"`
	InviteCode string `json:"inviteCode,omitempty"`
	Status     string `json:"status"`
}

// PhoneBindingView is returned after a phone binding. PhoneMasked never exposes
// the raw phone number to clients that did not already have it.
type PhoneBindingView struct {
	PhoneMasked           string `json:"phoneMasked"`
	Bound                 bool   `json:"bound"`
	Changed               bool   `json:"changed"`
	NextChangeAvailableAt string `json:"nextChangeAvailableAt,omitempty"`
}

// InvitationView is one invited member in GET /mini/invitations.
type InvitationView struct {
	MemberID  int64  `json:"memberId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	JoinedAt  string `json:"joinedAt"`
}

// MembershipTierView is the VIP tier representation returned by both the admin
// and mini read endpoints. BannerPath is the raw Qiniu object key/path stored in
// the DB; BannerURL is the fully-resolved public URL (QINIU_PUBLIC_DOMAIN + path).
type MembershipTierView struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Level      int    `json:"level"`
	Threshold  int64  `json:"threshold"`
	Benefits   string `json:"benefits,omitempty"`
	IconURL    string `json:"iconUrl,omitempty"`
	BannerPath string `json:"bannerPath,omitempty"`
	BannerURL  string `json:"bannerUrl,omitempty"`
	Status     string `json:"status"`
}

// MembershipTierCreateRequest is the POST /admin/membership-tiers body. Status
// defaults to "active" when omitted. BannerPath is the uploaded VIP banner's
// object key/path (not a URL, not an asset id).
type MembershipTierCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Level       int    `json:"level"`
	Threshold   int64  `json:"threshold"`
	Benefits    string `json:"benefits,omitempty"`
	IconAssetID *int64 `json:"iconAssetId,omitempty"`
	BannerPath  string `json:"bannerPath,omitempty"`
	Status      string `json:"status,omitempty"`
}

// MembershipTierUpdateRequest is the PATCH /admin/membership-tiers/:tierID
// body. Omitted fields are left unchanged. BannerPath is the uploaded VIP
// banner's object key/path (not a URL, not an asset id).
type MembershipTierUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Level       *int    `json:"level,omitempty"`
	Threshold   *int64  `json:"threshold,omitempty"`
	Benefits    *string `json:"benefits,omitempty"`
	IconAssetID *int64  `json:"iconAssetId,omitempty"`
	BannerPath  *string `json:"bannerPath,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// RechargeProductView is the public recharge package representation.
type RechargeProductView struct {
	ID               int64  `json:"id"`
	AmountCent       int64  `json:"amountCent"`
	CoinAmount       int64  `json:"coinAmount"`
	PointsAmount     int64  `json:"pointsAmount"`
	CouponTemplateID *int64 `json:"couponTemplateId,omitempty"`
	SortOrder        int    `json:"sortOrder"`
	Status           string `json:"status"`
}

// RechargeProductCreateRequest is the POST /admin/recharge-products body.
// Status defaults to "active" when omitted.
type RechargeProductCreateRequest struct {
	AmountCent       int64  `json:"amountCent" binding:"required"`
	CoinAmount       int64  `json:"coinAmount" binding:"required"`
	PointsAmount     int64  `json:"pointsAmount,omitempty"`
	CouponTemplateID *int64 `json:"couponTemplateId,omitempty"`
	SortOrder        int    `json:"sortOrder,omitempty"`
	Status           string `json:"status,omitempty"`
}

// RechargeProductUpdateRequest is the PATCH /admin/recharge-products/:productID
// body. Omitted fields are left unchanged.
type RechargeProductUpdateRequest struct {
	AmountCent       *int64  `json:"amountCent,omitempty"`
	CoinAmount       *int64  `json:"coinAmount,omitempty"`
	PointsAmount     *int64  `json:"pointsAmount,omitempty"`
	CouponTemplateID *int64  `json:"couponTemplateId,omitempty"`
	SortOrder        *int    `json:"sortOrder,omitempty"`
	Status           *string `json:"status,omitempty"`
}

// RankingEntryView is one leaderboard row.
type RankingEntryView struct {
	Rank      int    `json:"rank"`
	MemberID  int64  `json:"memberId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Gender    string `json:"gender,omitempty"`
	Score     int64  `json:"score"`
}
