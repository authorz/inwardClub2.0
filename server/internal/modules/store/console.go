package store

import (
	"context"
	"encoding/json"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// ConsoleService provides store-console (own-store scoped) and admin-side
// store profile operations, layered on top of the same Repository used by the
// mini-program read paths.
type ConsoleService struct {
	repo   Repository
	assets AssetResolver
}

// NewConsoleService builds the store console service.
func NewConsoleService(repo Repository, assets AssetResolver) *ConsoleService {
	return &ConsoleService{repo: repo, assets: assets}
}

// GetProfile returns the console profile view for the caller's own store.
func (s *ConsoleService) GetProfile(ctx context.Context, storeID int64) (ConsoleProfileView, error) {
	st, err := s.repo.GetStore(ctx, storeID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st), nil
}

// UpdateProfile applies a full-replace update to the caller's own store and
// returns the refreshed profile view.
func (s *ConsoleService) UpdateProfile(ctx context.Context, storeID int64, req UpdateProfileRequest) (ConsoleProfileView, error) {
	if storeID <= 0 {
		return ConsoleProfileView{}, apperr.Invalid("invalid storeID")
	}
	st, err := s.repo.UpdateStoreProfile(ctx, storeID, req)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st), nil
}

// UpdateStatus updates the caller's own store's active/inactive status and
// returns the refreshed profile view.
func (s *ConsoleService) UpdateStatus(ctx context.Context, storeID int64, status string) (ConsoleProfileView, error) {
	if storeID <= 0 {
		return ConsoleProfileView{}, apperr.Invalid("invalid storeID")
	}
	if status != StatusActive && status != StatusInactive {
		return ConsoleProfileView{}, apperr.Invalid("invalid status")
	}
	st, err := s.repo.UpdateStoreStatus(ctx, storeID, status)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st), nil
}

// GetSettings returns the caller's own store's settings view.
func (s *ConsoleService) GetSettings(ctx context.Context, storeID int64) (StoreSettingsView, error) {
	if storeID <= 0 {
		return StoreSettingsView{}, apperr.Invalid("invalid storeID")
	}
	settings, err := s.repo.GetStoreSettings(ctx, storeID)
	if err != nil {
		return StoreSettingsView{}, err
	}
	return settingsView(settings), nil
}

// UpdateSettings applies a full-replace update to the caller's own store's
// settings blob and returns the refreshed view.
func (s *ConsoleService) UpdateSettings(ctx context.Context, storeID int64, req UpdateSettingsRequest) (StoreSettingsView, error) {
	if storeID <= 0 {
		return StoreSettingsView{}, apperr.Invalid("invalid storeID")
	}
	payload := req.Settings
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return StoreSettingsView{}, apperr.Invalid("invalid settings")
	}
	settings, err := s.repo.UpsertStoreSettings(ctx, storeID, raw)
	if err != nil {
		return StoreSettingsView{}, err
	}
	return settingsView(settings), nil
}

func settingsView(settings StoreSettings) StoreSettingsView {
	view := StoreSettingsView{Settings: map[string]any{}}
	if len(settings.SettingsJSON) > 0 {
		_ = json.Unmarshal(settings.SettingsJSON, &view.Settings)
	}
	if !settings.UpdatedAt.IsZero() {
		u := settings.UpdatedAt
		view.UpdatedAt = &u
	}
	return view
}

// CreateStore provisions a new store from the admin side.
func (s *ConsoleService) CreateStore(ctx context.Context, input StoreInput) (Store, error) {
	if err := validateStoreInput(input); err != nil {
		return Store{}, err
	}
	return s.repo.CreateStore(ctx, input)
}

// UpdateStore applies a full-replace update to an existing store from the
// admin side.
func (s *ConsoleService) UpdateStore(ctx context.Context, id int64, input StoreInput) (Store, error) {
	if id <= 0 {
		return Store{}, apperr.Invalid("invalid storeID")
	}
	if err := validateStoreInput(input); err != nil {
		return Store{}, err
	}
	return s.repo.UpdateStore(ctx, id, input)
}

func validateStoreInput(input StoreInput) error {
	if input.Name == "" {
		return apperr.Invalid("name is required")
	}
	if input.Address == "" {
		return apperr.Invalid("address is required")
	}
	return nil
}

// ProfileView maps a store to its console profile view (resolving the logo
// URL). Handlers use this to render create/update responses.
func (s *ConsoleService) ProfileView(ctx context.Context, st Store) ConsoleProfileView {
	return s.consoleProfileView(ctx, st)
}

func (s *ConsoleService) consoleProfileView(ctx context.Context, st Store) ConsoleProfileView {
	view := ConsoleProfileView{
		ID:                       st.ID,
		Name:                     st.Name,
		Phone:                    st.Phone,
		CustomerServiceQRAssetID: st.CustomerServiceQRAssetID,
		Address:                  st.Address,
		Latitude:                 st.Latitude,
		Longitude:                st.Longitude,
		BusinessHours:            st.BusinessHours,
		Status:                   st.Status,
	}
	if st.LogoAssetID != nil {
		if url, err := s.assets.PublicURLByID(ctx, *st.LogoAssetID); err == nil {
			view.LogoURL = url
		}
	}
	if st.CustomerServiceQRAssetID != nil {
		if url, err := s.assets.PublicURLByID(ctx, *st.CustomerServiceQRAssetID); err == nil {
			view.CustomerServiceQRURL = url
		}
	}
	return view
}
