package member

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

// AssetResolver resolves assets to public URLs. Implemented by asset.Service.
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// PhoneResolver exchanges a WeChat phone authorisation code for the raw phone
// number. It is owned outside this module (the WeChat client) and wired by the
// bootstrap layer. When nil, phone binding reports NOT_IMPLEMENTED.
type PhoneResolver interface {
	ResolvePhone(ctx context.Context, code string) (string, error)
}

// MemberSettingsPolicy supplies headquarters-configured member-facing settings.
type MemberSettingsPolicy interface {
	PhoneChangeIntervalDays(ctx context.Context) (int, error)
	RechargeNotice(ctx context.Context) (string, error)
}

// Service provides the member self-service and catalogue read operations.
type Service struct {
	repo           Repository
	assets         AssetResolver
	phone          PhoneResolver
	settingsPolicy MemberSettingsPolicy
}

// NewService builds the member service. phone may be nil until the WeChat phone
// exchange is available.
func NewService(repo Repository, assets AssetResolver, phone PhoneResolver, policies ...MemberSettingsPolicy) *Service {
	svc := &Service{repo: repo, assets: assets, phone: phone}
	if len(policies) > 0 {
		svc.settingsPolicy = policies[0]
	}
	return svc
}

const (
	defaultRankingLimit            = 50
	defaultPhoneChangeIntervalDays = 30
)

// GetProfile returns the member profile view.
func (s *Service) GetProfile(ctx context.Context, memberID int64) (MemberView, error) {
	m, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return MemberView{}, err
	}
	return s.memberView(ctx, m), nil
}

// UpdateProfile applies a partial profile update and returns the fresh view.
func (s *Service) UpdateProfile(ctx context.Context, memberID int64, req UpdateProfileRequest) (MemberView, error) {
	if req.Nickname != nil {
		nickname, err := inputvalidation.PlainText(*req.Nickname, inputvalidation.TextOptions{
			Label: "昵称", MinRunes: 1, MaxRunes: 30,
		})
		if err != nil {
			return MemberView{}, apperr.Invalid(err.Error())
		}
		req.Nickname = &nickname
	}
	if req.Gender != nil {
		gender := *req.Gender
		if gender != "male" && gender != "female" && gender != "other" {
			return MemberView{}, apperr.Invalid("请选择正确的性别")
		}
	}
	if req.AssetID != nil && *req.AssetID <= 0 {
		return MemberView{}, apperr.Invalid("头像资源不正确")
	}
	if req.AvatarURL != nil {
		avatarURL, err := inputvalidation.HTTPURL("头像", *req.AvatarURL, true)
		if err != nil {
			return MemberView{}, apperr.Invalid(err.Error())
		}
		req.AvatarURL = &avatarURL
	}
	if req.Nickname == nil && req.Gender == nil && req.AssetID == nil && req.AvatarURL == nil {
		return MemberView{}, apperr.Invalid("没有需要更新的资料")
	}
	if err := s.repo.UpdateProfile(ctx, memberID, ProfileUpdate{Nickname: req.Nickname, Gender: req.Gender, AvatarAssetID: req.AssetID, AvatarURL: req.AvatarURL}); err != nil {
		return MemberView{}, err
	}
	return s.GetProfile(ctx, memberID)
}

// BindPhone exchanges a WeChat phone code and writes the member's phone number,
// returning only a masked representation to the client.
func (s *Service) BindPhone(ctx context.Context, memberID int64, req BindPhoneRequest) (PhoneBindingView, error) {
	if s.phone == nil {
		return PhoneBindingView{}, ErrPhoneBindingUnavailable
	}
	code, validationErr := inputvalidation.OpaqueToken("手机号授权凭证", req.Code, 2048)
	if validationErr != nil {
		return PhoneBindingView{}, apperr.Invalid(validationErr.Error())
	}
	phone, err := s.phone.ResolvePhone(ctx, code)
	if err != nil {
		return PhoneBindingView{}, apperr.Invalid("微信手机号授权失败，请重新授权").WithCause(err)
	}
	intervalDays := defaultPhoneChangeIntervalDays
	if s.settingsPolicy != nil {
		intervalDays, err = s.settingsPolicy.PhoneChangeIntervalDays(ctx)
		if err != nil {
			return PhoneBindingView{}, err
		}
	}
	change, err := s.repo.UpdatePhone(ctx, memberID, phone, intervalDays)
	if err != nil {
		return PhoneBindingView{}, err
	}
	view := PhoneBindingView{PhoneMasked: maskPhone(phone), Bound: true, Changed: change.Changed}
	if !change.NextAllowedAt.IsZero() {
		view.NextChangeAvailableAt = change.NextAllowedAt.UTC().Format(time.RFC3339)
	}
	return view, nil
}

// ListInvitations returns the members invited by the current member.
func (s *Service) ListInvitations(ctx context.Context, memberID int64, page httpx.Page) ([]InvitationView, int64, error) {
	invitees, total, err := s.repo.ListInvitees(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]InvitationView, 0, len(invitees))
	for _, iv := range invitees {
		avatarURL := iv.AvatarURL
		if avatarURL == "" {
			avatarURL = s.assetURL(ctx, iv.AvatarAssetID)
		}
		views = append(views, InvitationView{
			MemberID:  iv.MemberID,
			Nickname:  iv.Nickname,
			AvatarURL: avatarURL,
			JoinedAt:  iv.JoinedAt.UTC().Format(time.RFC3339),
		})
	}
	return views, total, nil
}

// BindInvitation binds the current member to an inviter by invite code. A member
// may bind exactly once and never to themselves.
func (s *Service) BindInvitation(ctx context.Context, memberID int64, req BindInvitationRequest) error {
	inviteCode, validationErr := inputvalidation.InviteCode(req.InviteCode, false)
	if validationErr != nil {
		return apperr.Invalid(validationErr.Error())
	}
	inviter, err := s.repo.GetByInviteCode(ctx, inviteCode)
	if err != nil {
		return err
	}
	if inviter.ID == memberID {
		return ErrSelfInvite
	}
	return s.repo.BindInviter(ctx, memberID, inviter.ID)
}

// ListMembershipTiers returns the active VIP tiers.
func (s *Service) ListMembershipTiers(ctx context.Context) ([]MembershipTierView, error) {
	tiers, err := s.repo.ListMembershipTiers(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]MembershipTierView, 0, len(tiers))
	for _, t := range tiers {
		views = append(views, s.membershipTierView(ctx, t))
	}
	return views, nil
}

// AdminListMembershipTiers returns every VIP tier regardless of status.
func (s *Service) AdminListMembershipTiers(ctx context.Context) ([]MembershipTierView, error) {
	tiers, err := s.repo.ListAllMembershipTiers(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]MembershipTierView, 0, len(tiers))
	for _, t := range tiers {
		views = append(views, s.membershipTierView(ctx, t))
	}
	return views, nil
}

// AdminGetMembershipTier returns a single VIP tier regardless of status.
func (s *Service) AdminGetMembershipTier(ctx context.Context, tierID int64) (MembershipTierView, error) {
	t, err := s.repo.GetMembershipTier(ctx, tierID)
	if err != nil {
		return MembershipTierView{}, err
	}
	return s.membershipTierView(ctx, t), nil
}

// CreateMembershipTier creates a new VIP tier.
func (s *Service) CreateMembershipTier(ctx context.Context, req MembershipTierCreateRequest) (MembershipTierView, error) {
	if req.Name == "" {
		return MembershipTierView{}, apperr.Invalid("member: name is required")
	}
	benefitConfig, err := normalizeBenefitConfig(req.BenefitConfig)
	if err != nil {
		return MembershipTierView{}, err
	}
	t, err := s.repo.CreateMembershipTier(ctx, MembershipTierCreate{
		Name: req.Name, Level: req.Level, Threshold: req.Threshold,
		Benefits: req.Benefits, BenefitConfig: benefitConfig,
		IconAssetID: req.IconAssetID, Status: req.Status,
	})
	if err != nil {
		return MembershipTierView{}, err
	}
	return s.membershipTierView(ctx, t), nil
}

// UpdateMembershipTier applies a partial update to a VIP tier.
func (s *Service) UpdateMembershipTier(ctx context.Context, tierID int64, req MembershipTierUpdateRequest) (MembershipTierView, error) {
	u := MembershipTierUpdate{
		Name:        req.Name,
		Level:       req.Level,
		Threshold:   req.Threshold,
		Benefits:    req.Benefits,
		IconAssetID: req.IconAssetID,
		Status:      req.Status,
	}
	if req.BenefitConfig != nil {
		benefitConfig, err := normalizeBenefitConfig(*req.BenefitConfig)
		if err != nil {
			return MembershipTierView{}, err
		}
		u.BenefitConfig = &benefitConfig
	}
	if u.Name == nil && u.Level == nil && u.Threshold == nil && u.Benefits == nil && u.BenefitConfig == nil && u.IconAssetID == nil && u.Status == nil {
		return MembershipTierView{}, apperr.Invalid("member: no tier fields to update")
	}
	t, err := s.repo.UpdateMembershipTier(ctx, tierID, u)
	if err != nil {
		return MembershipTierView{}, err
	}
	return s.membershipTierView(ctx, t), nil
}

// DisableMembershipTier moves a VIP tier to the disabled status.
func (s *Service) DisableMembershipTier(ctx context.Context, tierID int64) (MembershipTierView, error) {
	status := StatusDisabled
	t, err := s.repo.UpdateMembershipTier(ctx, tierID, MembershipTierUpdate{Status: &status})
	if err != nil {
		return MembershipTierView{}, err
	}
	return s.membershipTierView(ctx, t), nil
}

// ListRechargeProducts returns the active recharge packages.
func (s *Service) ListRechargeProducts(ctx context.Context) ([]RechargeProductView, error) {
	products, err := s.repo.ListRechargeProducts(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]RechargeProductView, 0, len(products))
	for _, p := range products {
		views = append(views, rechargeProductView(p))
	}
	return views, nil
}

// RechargeNotice returns the configured copy shipped alongside the public recharge list.
func (s *Service) RechargeNotice(ctx context.Context) (string, error) {
	if s.settingsPolicy == nil {
		return "", nil
	}
	return s.settingsPolicy.RechargeNotice(ctx)
}

// AdminListRechargeProducts returns every recharge package regardless of status.
func (s *Service) AdminListRechargeProducts(ctx context.Context) ([]RechargeProductView, error) {
	products, err := s.repo.ListAllRechargeProducts(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]RechargeProductView, 0, len(products))
	for _, p := range products {
		views = append(views, rechargeProductView(p))
	}
	return views, nil
}

// AdminGetRechargeProduct returns a single recharge package regardless of status.
func (s *Service) AdminGetRechargeProduct(ctx context.Context, productID int64) (RechargeProductView, error) {
	p, err := s.repo.GetRechargeProduct(ctx, productID)
	if err != nil {
		return RechargeProductView{}, err
	}
	return rechargeProductView(p), nil
}

// CreateRechargeProduct creates a new recharge package.
func (s *Service) CreateRechargeProduct(ctx context.Context, req RechargeProductCreateRequest) (RechargeProductView, error) {
	if req.AmountCent <= 0 {
		return RechargeProductView{}, apperr.Invalid("member: amountCent must be positive")
	}
	if req.CoinAmount <= 0 {
		return RechargeProductView{}, apperr.Invalid("member: coinAmount must be positive")
	}
	if req.PointsAmount < 0 {
		return RechargeProductView{}, apperr.Invalid("member: pointsAmount cannot be negative")
	}
	if req.CouponTemplateID != nil {
		if *req.CouponTemplateID <= 0 {
			req.CouponTemplateID = nil
		} else if err := s.repo.ValidateRechargeCouponTemplate(ctx, *req.CouponTemplateID); err != nil {
			return RechargeProductView{}, err
		}
	}
	p, err := s.repo.CreateRechargeProduct(ctx, RechargeProductCreate{
		AmountCent:       req.AmountCent,
		CoinAmount:       req.CoinAmount,
		PointsAmount:     req.PointsAmount,
		CouponTemplateID: req.CouponTemplateID,
		SortOrder:        req.SortOrder,
		Status:           req.Status,
	})
	if err != nil {
		return RechargeProductView{}, err
	}
	return rechargeProductView(p), nil
}

// UpdateRechargeProduct applies a partial update to a recharge package.
func (s *Service) UpdateRechargeProduct(ctx context.Context, productID int64, req RechargeProductUpdateRequest) (RechargeProductView, error) {
	u := RechargeProductUpdate{
		AmountCent:   req.AmountCent,
		CoinAmount:   req.CoinAmount,
		PointsAmount: req.PointsAmount,
		SortOrder:    req.SortOrder,
		Status:       req.Status,
	}
	if req.CouponTemplateID != nil {
		var couponID *int64
		if *req.CouponTemplateID > 0 {
			if err := s.repo.ValidateRechargeCouponTemplate(ctx, *req.CouponTemplateID); err != nil {
				return RechargeProductView{}, err
			}
			couponID = req.CouponTemplateID
		}
		u.CouponTemplateID = &couponID
	}
	if u.AmountCent == nil && u.CoinAmount == nil && u.PointsAmount == nil && u.CouponTemplateID == nil && u.SortOrder == nil && u.Status == nil {
		return RechargeProductView{}, apperr.Invalid("member: no recharge product fields to update")
	}
	if u.AmountCent != nil && *u.AmountCent <= 0 {
		return RechargeProductView{}, apperr.Invalid("member: amountCent must be positive")
	}
	if u.CoinAmount != nil && *u.CoinAmount <= 0 {
		return RechargeProductView{}, apperr.Invalid("member: coinAmount must be positive")
	}
	if u.PointsAmount != nil && *u.PointsAmount < 0 {
		return RechargeProductView{}, apperr.Invalid("member: pointsAmount cannot be negative")
	}
	p, err := s.repo.UpdateRechargeProduct(ctx, productID, u)
	if err != nil {
		return RechargeProductView{}, err
	}
	return rechargeProductView(p), nil
}

// DisableRechargeProduct moves a recharge package to the disabled status.
func (s *Service) DisableRechargeProduct(ctx context.Context, productID int64) (RechargeProductView, error) {
	status := StatusDisabled
	p, err := s.repo.UpdateRechargeProduct(ctx, productID, RechargeProductUpdate{Status: &status})
	if err != nil {
		return RechargeProductView{}, err
	}
	return rechargeProductView(p), nil
}

// ListRankings returns a leaderboard snapshot for the requested period.
func (s *Service) ListRankings(ctx context.Context, period string) ([]RankingEntryView, error) {
	period, err := normalizeRankingPeriod(period)
	if err != nil {
		return nil, err
	}
	entries, err := s.repo.ListRankings(ctx, period, defaultRankingLimit)
	if err != nil {
		return nil, err
	}
	views := make([]RankingEntryView, 0, len(entries))
	for _, e := range entries {
		avatarURL := e.AvatarURL
		if avatarURL == "" {
			avatarURL = s.assetURL(ctx, e.AvatarAssetID)
		}
		views = append(views, RankingEntryView{
			Rank:        e.Rank,
			MemberID:    e.MemberID,
			Nickname:    e.Nickname,
			AvatarURL:   avatarURL,
			Gender:      e.Gender,
			Score:       e.Score,
			GrowthValue: e.GrowthValue,
		})
	}
	return views, nil
}

func (s *Service) memberView(ctx context.Context, m Member) MemberView {
	return MemberView{
		ID:         m.ID,
		Nickname:   m.Nickname,
		Gender:     m.Gender,
		Phone:      maskPhone(m.Phone),
		AvatarURL:  s.assetURL(ctx, m.AvatarAssetID),
		InviteCode: m.InviteCode,
		Status:     m.Status,
	}
}

// VIPShortLabel is the member-facing short VIP identity for a tier, derived from
// its level (e.g. level 1 -> "VIP1"). The full tier name (e.g. "VIP1 普通会员")
// is reserved for admin management surfaces; the mini "me"/home profile exposes
// only this short label.
func VIPShortLabel(level int) string {
	if level < 1 {
		return "VIP"
	}
	return "VIP" + strconv.Itoa(level)
}

func (s *Service) membershipTierView(ctx context.Context, t MembershipTier) MembershipTierView {
	var config TierBenefitConfig
	_ = json.Unmarshal(t.BenefitConfig, &config)
	if config.Points == nil {
		config.Points = []TierPointBenefit{}
	}
	if config.Coupons == nil {
		config.Coupons = []TierCouponBenefit{}
	}
	if config.Descriptions == nil {
		config.Descriptions = []string{}
	}
	return MembershipTierView{
		ID: t.ID, Name: t.Name, Level: t.Level, Threshold: t.Threshold,
		Benefits: t.Benefits, BenefitConfig: config,
		IconURL: s.assetURL(ctx, t.IconAssetID), Status: t.Status,
	}
}

func normalizeBenefitConfig(config TierBenefitConfig) (json.RawMessage, error) {
	for _, benefit := range config.Points {
		if benefit.Amount <= 0 || !validBenefitPeriod(benefit.Period) || !validBenefitTrigger(benefit.Trigger) {
			return nil, apperr.Invalid("积分福利配置不正确")
		}
	}
	for _, benefit := range config.Coupons {
		if benefit.Quantity <= 0 || benefit.Quantity > 99 || (benefit.CategoryID <= 0 && !validTierCouponType(benefit.CouponType)) ||
			!validBenefitPeriod(benefit.Period) || !validBenefitTrigger(benefit.Trigger) {
			return nil, apperr.Invalid("券福利类型或数量不正确")
		}
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, apperr.Invalid("VIP权益配置不正确")
	}
	return raw, nil
}

func validBenefitPeriod(value string) bool {
	switch value {
	case "once", "daily", "weekly", "monthly":
		return true
	default:
		return false
	}
}

func validBenefitTrigger(value string) bool {
	switch value {
	case "tier_achieved", "low_spend", "first_order", "visit", "period_start", "weekday_event", "weekly_event", "monthly_event":
		return true
	default:
		return false
	}
}

func validTierCouponType(value string) bool {
	switch value {
	case "event_ticket", "admission_ticket", "snack", "alcohol", "beverage", "drink", "meal", "gift":
		return true
	default:
		return false
	}
}

// CurrentTierView returns the VIP tier the member currently holds
// (members.current_tier_id), resolved to a view with its icon URL. It
// returns a nil pointer when the member is unranked or the held tier reference
// is dangling, so the "me" surface degrades to "no tier" rather than failing.
func (s *Service) CurrentTierView(ctx context.Context, memberID int64) (*MembershipTierView, error) {
	m, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if m.CurrentTierID == nil {
		// Unranked: current_tier_id is only written once a member crosses a paid
		// growth threshold. Every member is at least the base tier (lowest active
		// level, threshold 0), so "me" still surfaces a VIP level.
		return s.baseTierView(ctx)
	}
	t, err := s.repo.GetMembershipTier(ctx, *m.CurrentTierID)
	if err != nil {
		if ae := apperr.From(err); ae != nil && ae.Code == apperr.CodeNotFound {
			return s.baseTierView(ctx) // dangling reference: fall back to base tier
		}
		return nil, err
	}
	view := s.membershipTierView(ctx, t)
	return &view, nil
}

// baseTierView resolves the base VIP tier — the lowest active level (VIP1) — to a
// view, or nil when no tiers are configured at all. ListMembershipTiers returns
// active tiers ordered by level ASC, so the first entry is the base tier.
func (s *Service) baseTierView(ctx context.Context) (*MembershipTierView, error) {
	tiers, err := s.repo.ListMembershipTiers(ctx)
	if err != nil {
		return nil, err
	}
	if len(tiers) == 0 {
		return nil, nil
	}
	view := s.membershipTierView(ctx, tiers[0])
	return &view, nil
}

func rechargeProductView(p RechargeProduct) RechargeProductView {
	return RechargeProductView{
		ID:               p.ID,
		AmountCent:       p.AmountCent,
		CoinAmount:       p.CoinAmount,
		PointsAmount:     p.PointsAmount,
		CouponTemplateID: p.CouponTemplateID,
		SortOrder:        p.SortOrder,
		Status:           p.Status,
	}
}

// assetURL resolves an optional asset id to a public URL, tolerating misses.
func (s *Service) assetURL(ctx context.Context, id *int64) string {
	if id == nil {
		return ""
	}
	url, err := s.assets.PublicURLByID(ctx, *id)
	if err != nil {
		return ""
	}
	return url
}

func normalizeRankingPeriod(period string) (string, error) {
	switch period {
	case "":
		return RankingAll, nil
	case RankingMonth, RankingAll, RankingWater:
		return period, nil
	default:
		return "", apperr.Invalid("invalid ranking period")
	}
}

// maskPhone masks the middle of a phone number, keeping the leading 3 and
// trailing 4 digits. Short or empty values are returned unchanged/empty.
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
