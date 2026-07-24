package store

import (
	"context"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// BannerConsoleService provides admin (global + any store) and store console
// (own-store scoped) banner CRUD, layered on the same Repository the mini
// read path uses.
type BannerConsoleService struct {
	repo   Repository
	assets AssetResolver
}

// NewBannerConsoleService builds the banner console service.
func NewBannerConsoleService(repo Repository, assets AssetResolver) *BannerConsoleService {
	return &BannerConsoleService{repo: repo, assets: assets}
}

// AdminList returns every banner (global and store-scoped).
func (s *BannerConsoleService) AdminList(ctx context.Context) ([]BannerAdminView, error) {
	return s.list(ctx, nil)
}

// AdminGet returns a single banner by id.
func (s *BannerConsoleService) AdminGet(ctx context.Context, id int64) (BannerAdminView, error) {
	b, err := s.repo.GetBanner(ctx, id)
	if err != nil {
		return BannerAdminView{}, err
	}
	return s.view(ctx, b), nil
}

// AdminCreate creates a banner. Admins may create global or store-scoped
// banners; the scope defaults to global when unspecified.
func (s *BannerConsoleService) AdminCreate(ctx context.Context, in BannerInput) (BannerAdminView, error) {
	scope := in.ScopeType
	if scope == "" {
		scope = BannerScopeGlobal
	}
	b := Banner{
		ScopeType: scope,
		StoreID:   in.StoreID,
		Title:     in.Title,
		AssetID:   in.AssetID,
		LinkURL:   in.LinkURL,
		SortOrder: in.SortOrder,
		Status:    orDefault(in.Status, StatusActive),
	}
	if err := validateBannerScope(b); err != nil {
		return BannerAdminView{}, err
	}
	if b.AssetID <= 0 {
		return BannerAdminView{}, apperr.Invalid("assetId is required")
	}
	created, err := s.repo.CreateBanner(ctx, b)
	if err != nil {
		return BannerAdminView{}, err
	}
	return s.view(ctx, created), nil
}

// AdminUpdate applies a partial update to any banner.
func (s *BannerConsoleService) AdminUpdate(ctx context.Context, id int64, patch BannerPatch) (BannerAdminView, error) {
	b, err := s.repo.GetBanner(ctx, id)
	if err != nil {
		return BannerAdminView{}, err
	}
	applyBannerPatch(&b, patch)
	if err := validateBannerScope(b); err != nil {
		return BannerAdminView{}, err
	}
	updated, err := s.repo.UpdateBanner(ctx, b)
	if err != nil {
		return BannerAdminView{}, err
	}
	return s.view(ctx, updated), nil
}

// AdminDelete removes any banner.
func (s *BannerConsoleService) AdminDelete(ctx context.Context, id int64) error {
	return s.repo.DeleteBanner(ctx, id)
}

// StoreList returns only the caller's own store-scoped banners.
func (s *BannerConsoleService) StoreList(ctx context.Context, storeID int64) ([]BannerAdminView, error) {
	return s.list(ctx, &storeID)
}

// StoreGet returns one of the caller's own banners; other stores' or global
// banners surface as NotFound.
func (s *BannerConsoleService) StoreGet(ctx context.Context, storeID, id int64) (BannerAdminView, error) {
	b, err := s.ownBanner(ctx, storeID, id)
	if err != nil {
		return BannerAdminView{}, err
	}
	return s.view(ctx, b), nil
}

// StoreCreate creates a banner forced to the caller's own store scope. Any
// scopeType/storeId in the request is ignored.
func (s *BannerConsoleService) StoreCreate(ctx context.Context, storeID int64, in BannerInput) (BannerAdminView, error) {
	if in.AssetID <= 0 {
		return BannerAdminView{}, apperr.Invalid("assetId is required")
	}
	b := Banner{
		ScopeType: BannerScopeStore,
		StoreID:   &storeID,
		Title:     in.Title,
		AssetID:   in.AssetID,
		LinkURL:   in.LinkURL,
		SortOrder: in.SortOrder,
		Status:    orDefault(in.Status, StatusActive),
	}
	created, err := s.repo.CreateBanner(ctx, b)
	if err != nil {
		return BannerAdminView{}, err
	}
	return s.view(ctx, created), nil
}

// StoreUpdate applies a partial update to one of the caller's own banners. The
// scope cannot be changed from the store console.
func (s *BannerConsoleService) StoreUpdate(ctx context.Context, storeID, id int64, patch BannerPatch) (BannerAdminView, error) {
	b, err := s.ownBanner(ctx, storeID, id)
	if err != nil {
		return BannerAdminView{}, err
	}
	// Scope is pinned to the caller's store; ignore any scope changes.
	patch.ScopeType = nil
	patch.StoreID = nil
	applyBannerPatch(&b, patch)
	updated, err := s.repo.UpdateBanner(ctx, b)
	if err != nil {
		return BannerAdminView{}, err
	}
	return s.view(ctx, updated), nil
}

// StoreDelete removes one of the caller's own banners.
func (s *BannerConsoleService) StoreDelete(ctx context.Context, storeID, id int64) error {
	if _, err := s.ownBanner(ctx, storeID, id); err != nil {
		return err
	}
	return s.repo.DeleteBanner(ctx, id)
}

// ownBanner fetches a banner and asserts it is store-scoped to storeID.
func (s *BannerConsoleService) ownBanner(ctx context.Context, storeID, id int64) (Banner, error) {
	b, err := s.repo.GetBanner(ctx, id)
	if err != nil {
		return Banner{}, err
	}
	if b.StoreID == nil || *b.StoreID != storeID {
		return Banner{}, apperr.NotFound("banner not found")
	}
	return b, nil
}

func (s *BannerConsoleService) list(ctx context.Context, storeID *int64) ([]BannerAdminView, error) {
	banners, err := s.repo.ListBanners(ctx, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]BannerAdminView, 0, len(banners))
	for _, b := range banners {
		out = append(out, s.view(ctx, b))
	}
	return out, nil
}

func (s *BannerConsoleService) view(ctx context.Context, b Banner) BannerAdminView {
	v := BannerAdminView{
		ID:        b.ID,
		ScopeType: b.ScopeType,
		StoreID:   b.StoreID,
		Title:     b.Title,
		AssetID:   b.AssetID,
		LinkURL:   b.LinkURL,
		SortOrder: b.SortOrder,
		Status:    b.Status,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
	if b.AssetID > 0 {
		if url, err := s.assets.PublicURLByID(ctx, b.AssetID); err == nil {
			v.ImageURL = url
		}
	}
	return v
}

// validateBannerScope enforces the store_id/scope_type invariant: a store
// banner needs a store_id, a global banner must not carry one.
func validateBannerScope(b Banner) error {
	switch b.ScopeType {
	case BannerScopeGlobal:
		if b.StoreID != nil {
			return apperr.Invalid("global banner must not set storeId")
		}
	case BannerScopeStore:
		if b.StoreID == nil || *b.StoreID <= 0 {
			return apperr.Invalid("store banner requires storeId")
		}
	default:
		return apperr.Invalid("invalid scopeType")
	}
	return nil
}

func applyBannerPatch(b *Banner, p BannerPatch) {
	if p.ScopeType != nil {
		b.ScopeType = *p.ScopeType
		if b.ScopeType == BannerScopeGlobal {
			b.StoreID = nil
		}
	}
	if p.StoreID != nil {
		b.StoreID = p.StoreID
	}
	if p.Title != nil {
		b.Title = *p.Title
	}
	if p.AssetID != nil {
		b.AssetID = *p.AssetID
	}
	if p.LinkURL != nil {
		b.LinkURL = *p.LinkURL
	}
	if p.SortOrder != nil {
		b.SortOrder = *p.SortOrder
	}
	if p.Status != nil {
		b.Status = *p.Status
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
