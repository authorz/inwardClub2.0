package member

import (
	"context"
	"strconv"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// AssetResolver resolves assets to public URLs. Implemented by asset.Service.
// PublicURLByID resolves a stored asset id; PublicURL resolves a raw object
// key/path (QINIU_PUBLIC_DOMAIN + path).
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
	PublicURL(objectKey string) string
}

// PhoneResolver exchanges a WeChat phone authorisation code for the raw phone
// number. It is owned outside this module (the WeChat client) and wired by the
// bootstrap layer. When nil, phone binding reports NOT_IMPLEMENTED.
type PhoneResolver interface {
	ResolvePhone(ctx context.Context, code string) (string, error)
}

// Service provides the member self-service and catalogue read operations.
type Service struct {
	repo   Repository
	assets AssetResolver
	phone  PhoneResolver
}

// NewService builds the member service. phone may be nil until the WeChat phone
// exchange is available.
func NewService(repo Repository, assets AssetResolver, phone PhoneResolver) *Service {
	return &Service{repo: repo, assets: assets, phone: phone}
}

const defaultRankingLimit = 20

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
	if err := s.repo.UpdateProfile(ctx, memberID, ProfileUpdate{Nickname: req.Nickname, AvatarAssetID: req.AssetID}); err != nil {
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
	phone, err := s.phone.ResolvePhone(ctx, req.Code)
	if err != nil {
		return PhoneBindingView{}, apperr.Unauthenticated("wechat phone authorization failed")
	}
	if err := s.repo.UpdatePhone(ctx, memberID, phone); err != nil {
		return PhoneBindingView{}, err
	}
	return PhoneBindingView{PhoneMasked: maskPhone(phone), Bound: true}, nil
}

// ListInvitations returns the members invited by the current member.
func (s *Service) ListInvitations(ctx context.Context, memberID int64, page httpx.Page) ([]InvitationView, int64, error) {
	invitees, total, err := s.repo.ListInvitees(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]InvitationView, 0, len(invitees))
	for _, iv := range invitees {
		views = append(views, InvitationView{
			MemberID:  iv.MemberID,
			Nickname:  iv.Nickname,
			AvatarURL: s.assetURL(ctx, iv.AvatarAssetID),
			JoinedAt:  iv.JoinedAt.UTC().Format(time.RFC3339),
		})
	}
	return views, total, nil
}

// BindInvitation binds the current member to an inviter by invite code. A member
// may bind exactly once and never to themselves.
func (s *Service) BindInvitation(ctx context.Context, memberID int64, req BindInvitationRequest) error {
	inviter, err := s.repo.GetByInviteCode(ctx, req.InviteCode)
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
	t, err := s.repo.CreateMembershipTier(ctx, MembershipTierCreate{
		Name:        req.Name,
		Level:       req.Level,
		Threshold:   req.Threshold,
		Benefits:    req.Benefits,
		IconAssetID: req.IconAssetID,
		BannerPath:  req.BannerPath,
		Status:      req.Status,
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
		BannerPath:  req.BannerPath,
		Status:      req.Status,
	}
	if u.Name == nil && u.Level == nil && u.Threshold == nil && u.Benefits == nil && u.IconAssetID == nil && u.BannerPath == nil && u.Status == nil {
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
	if req.Name == "" {
		return RechargeProductView{}, apperr.Invalid("member: name is required")
	}
	if req.Amount <= 0 {
		return RechargeProductView{}, apperr.Invalid("member: amount must be positive")
	}
	assetType := req.AssetType
	if assetType == "" {
		assetType = "coin"
	}
	p, err := s.repo.CreateRechargeProduct(ctx, RechargeProductCreate{
		Name:         req.Name,
		Amount:       req.Amount,
		BonusAmount:  req.BonusAmount,
		GrowthAmount: req.GrowthAmount,
		AssetType:    assetType,
		SortOrder:    req.SortOrder,
		Status:       req.Status,
	})
	if err != nil {
		return RechargeProductView{}, err
	}
	return rechargeProductView(p), nil
}

// UpdateRechargeProduct applies a partial update to a recharge package.
func (s *Service) UpdateRechargeProduct(ctx context.Context, productID int64, req RechargeProductUpdateRequest) (RechargeProductView, error) {
	u := RechargeProductUpdate{
		Name:         req.Name,
		Amount:       req.Amount,
		BonusAmount:  req.BonusAmount,
		GrowthAmount: req.GrowthAmount,
		AssetType:    req.AssetType,
		SortOrder:    req.SortOrder,
		Status:       req.Status,
	}
	if u.Name == nil && u.Amount == nil && u.BonusAmount == nil && u.GrowthAmount == nil && u.AssetType == nil && u.SortOrder == nil && u.Status == nil {
		return RechargeProductView{}, apperr.Invalid("member: no recharge product fields to update")
	}
	if u.Name != nil && *u.Name == "" {
		return RechargeProductView{}, apperr.Invalid("member: name is required")
	}
	if u.Amount != nil && *u.Amount <= 0 {
		return RechargeProductView{}, apperr.Invalid("member: amount must be positive")
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
		views = append(views, RankingEntryView{
			Rank:      e.Rank,
			MemberID:  e.MemberID,
			Nickname:  e.Nickname,
			AvatarURL: s.assetURL(ctx, e.AvatarAssetID),
			Score:     e.Score,
		})
	}
	return views, nil
}

func (s *Service) memberView(ctx context.Context, m Member) MemberView {
	return MemberView{
		ID:         m.ID,
		Nickname:   m.Nickname,
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
	return MembershipTierView{
		ID:         t.ID,
		Name:       t.Name,
		Level:      t.Level,
		Threshold:  t.Threshold,
		Benefits:   t.Benefits,
		IconURL:    s.assetURL(ctx, t.IconAssetID),
		BannerPath: t.BannerPath,
		BannerURL:  s.bannerURL(ctx, t),
		Status:     t.Status,
	}
}

// bannerURL resolves a tier's VIP banner to a public URL. New tiers store the
// object key/path in banner_path (resolved as QINIU_PUBLIC_DOMAIN + path);
// tiers configured before the migration fall back to the legacy asset id.
func (s *Service) bannerURL(ctx context.Context, t MembershipTier) string {
	if t.BannerPath != "" {
		return s.assets.PublicURL(t.BannerPath)
	}
	return s.assetURL(ctx, t.BannerAssetID)
}

// CurrentTierView returns the VIP tier the member currently holds
// (members.current_tier_id), resolved to a view with its icon/banner URLs. It
// returns a nil pointer when the member is unranked or the held tier reference
// is dangling, so the "me" surface degrades to "no tier" rather than failing.
func (s *Service) CurrentTierView(ctx context.Context, memberID int64) (*MembershipTierView, error) {
	m, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if m.CurrentTierID == nil {
		return nil, nil
	}
	t, err := s.repo.GetMembershipTier(ctx, *m.CurrentTierID)
	if err != nil {
		if ae := apperr.From(err); ae != nil && ae.Code == apperr.CodeNotFound {
			return nil, nil // dangling reference: treat as unranked
		}
		return nil, err
	}
	view := s.membershipTierView(ctx, t)
	return &view, nil
}

func rechargeProductView(p RechargeProduct) RechargeProductView {
	return RechargeProductView{
		ID:           p.ID,
		Name:         p.Name,
		Amount:       p.Amount,
		BonusAmount:  p.BonusAmount,
		GrowthAmount: p.GrowthAmount,
		AssetType:    p.AssetType,
		SortOrder:    p.SortOrder,
		Status:       p.Status,
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
	case RankingWeek, RankingMonth, RankingAll:
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
