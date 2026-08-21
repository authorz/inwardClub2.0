// Package member owns the mini-program member self-service surface: profile,
// phone binding, invitations, plus the read-only membership tier / recharge
// product / ranking catalogues. Rankings are derived from settled recharge
// orders across all stores.
package member

import (
	"encoding/json"
	"time"
)

// Member is the member record backing the mini-program "me" surface.
type Member struct {
	ID                int64
	Nickname          string
	Gender            string
	Phone             string
	PhoneChangedAt    *time.Time
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
	Gender        *string
	AvatarAssetID *int64
	AvatarURL     *string
}

// PhoneChangeResult describes whether an authorised number actually changed
// and when a subsequent change becomes available.
type PhoneChangeResult struct {
	Changed       bool
	NextAllowedAt time.Time
}

// Invitee is a member that the current member has invited.
type Invitee struct {
	MemberID      int64
	Nickname      string
	AvatarAssetID *int64
	AvatarURL     string
	JoinedAt      time.Time
}

// MembershipTier is a configurable VIP level with structured benefits.
type MembershipTier struct {
	ID            int64
	Name          string
	Level         int
	Threshold     int64
	Benefits      string
	BenefitConfig json.RawMessage
	IconAssetID   *int64
	Status        string
}

type TierPointBenefit struct {
	Amount  int64  `json:"amount"`
	Period  string `json:"period"`
	Trigger string `json:"trigger"`
}

type TierCouponBenefit struct {
	CouponType string `json:"couponType"`
	Quantity   int    `json:"quantity"`
	Period     string `json:"period"`
	Trigger    string `json:"trigger"`
}

type TierBenefitConfig struct {
	Points       []TierPointBenefit  `json:"points"`
	Coupons      []TierCouponBenefit `json:"coupons"`
	Descriptions []string            `json:"descriptions"`
}

// MembershipTierCreate is the input to creating a new membership tier.
type MembershipTierCreate struct {
	Name          string
	Level         int
	Threshold     int64
	Benefits      string
	BenefitConfig json.RawMessage
	IconAssetID   *int64
	Status        string
}

// MembershipTierUpdate is a partial update to a membership tier; a nil field
// is left unchanged.
type MembershipTierUpdate struct {
	Name          *string
	Level         *int
	Threshold     *int64
	Benefits      *string
	BenefitConfig *json.RawMessage
	IconAssetID   *int64
	Status        *string
}

// RechargeProduct is a quick-recharge tier. Payment amount is stored in cents;
// credited coins and points are independent integer quantities.
type RechargeProduct struct {
	ID               int64
	AmountCent       int64
	CoinAmount       int64
	PointsAmount     int64
	CouponTemplateID *int64
	SortOrder        int
	Status           string
}

// RechargeProductCreate is the input to creating a new recharge package.
type RechargeProductCreate struct {
	AmountCent       int64
	CoinAmount       int64
	PointsAmount     int64
	CouponTemplateID *int64
	SortOrder        int
	Status           string
}

// RechargeProductUpdate is a partial update to a recharge package; a nil field
// is left unchanged.
type RechargeProductUpdate struct {
	AmountCent       *int64
	CoinAmount       *int64
	PointsAmount     *int64
	CouponTemplateID **int64
	SortOrder        *int
	Status           *string
}

// RankingEntry is a single row in a leaderboard snapshot.
type RankingEntry struct {
	Rank          int
	MemberID      int64
	Nickname      string
	AvatarAssetID *int64
	AvatarURL     string
	Gender        string
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
