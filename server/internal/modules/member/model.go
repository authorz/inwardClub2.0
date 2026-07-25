// Package member owns the mini-program member self-service surface: profile,
// phone binding, invitations, plus the read-only membership tier / recharge
// product / ranking catalogues. The member profile itself lives in the members
// table (shared with auth); tiers/products come from the membership_tiers /
// recharge_products tables and rankings from approved point_savings.
package member

import "time"

// Member is the member record backing the mini-program "me" surface.
type Member struct {
	ID                int64
	Nickname          string
	Phone             string
	InviteCode        string
	AvatarAssetID     *int64
	InvitedByMemberID *int64
	CurrentTierID     *int64
	Status            string
	CreatedAt         time.Time
}

// ProfileUpdate carries the mutable member profile fields. A nil pointer means
// "leave unchanged"; only non-nil fields are written.
type ProfileUpdate struct {
	Nickname      *string
	AvatarAssetID *int64
	AvatarURL     *string
}

// Invitee is a member that the current member has invited.
type Invitee struct {
	MemberID      int64
	Nickname      string
	AvatarAssetID *int64
	AvatarURL     string
	JoinedAt      time.Time
}

// MembershipTier is a configurable VIP level. BannerPath is the uploaded VIP
// banner's object key/path (resolved to QINIU_PUBLIC_DOMAIN + path on read);
// BannerAssetID is the legacy asset reference, retained only for reading tiers
// configured before the banner_path migration.
type MembershipTier struct {
	ID            int64
	Name          string
	Level         int
	Threshold     int64
	Benefits      string
	IconAssetID   *int64
	BannerAssetID *int64
	BannerPath    string
	Status        string
}

// MembershipTierCreate is the input to creating a new membership tier.
type MembershipTierCreate struct {
	Name        string
	Level       int
	Threshold   int64
	Benefits    string
	IconAssetID *int64
	BannerPath  string
	Status      string
}

// MembershipTierUpdate is a partial update to a membership tier; a nil field
// is left unchanged.
type MembershipTierUpdate struct {
	Name        *string
	Level       *int
	Threshold   *int64
	Benefits    *string
	IconAssetID *int64
	BannerPath  *string
	Status      *string
}

// RechargeProduct is a quick-recharge tier. Payment amount is stored in cents;
// credited coins and points are independent integer quantities.
type RechargeProduct struct {
	ID           int64
	AmountCent   int64
	CoinAmount   int64
	PointsAmount int64
	SortOrder    int
	Status       string
}

// RechargeProductCreate is the input to creating a new recharge package.
type RechargeProductCreate struct {
	AmountCent   int64
	CoinAmount   int64
	PointsAmount int64
	SortOrder    int
	Status       string
}

// RechargeProductUpdate is a partial update to a recharge package; a nil field
// is left unchanged.
type RechargeProductUpdate struct {
	AmountCent   *int64
	CoinAmount   *int64
	PointsAmount *int64
	SortOrder    *int
	Status       *string
}

// RankingEntry is a single row in a leaderboard snapshot.
type RankingEntry struct {
	Rank          int
	MemberID      int64
	Nickname      string
	AvatarAssetID *int64
	Score         int64
}

// Member status values.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// Ranking periods.
const (
	RankingWeek  = "week"
	RankingMonth = "month"
	RankingAll   = "all"
)

// pointSavingApproved is the point_savings status counted toward rankings. It
// mirrors activity.PointSavingApproved without importing that module.
const pointSavingApproved = "approved"
