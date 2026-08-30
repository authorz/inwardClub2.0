package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inwardclub/server/internal/platform/audit"
	"github.com/inwardclub/server/internal/platform/businesshours"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// ConsoleService provides store-console (own-store scoped) and admin-side
// store profile operations, layered on top of the same Repository used by the
// mini-program read paths.
type ConsoleService struct {
	repo      Repository
	assets    AssetResolver
	passwords AccountPasswordVerifier
	now       func() time.Time
}

// AccountPasswordVerifier re-authenticates the current headquarters account
// before a store is deleted.
type AccountPasswordVerifier interface {
	VerifyAccountPassword(ctx context.Context, accountID int64, password string) error
}

// NewConsoleService builds the store console service. The optional password
// verifier is required only by the headquarters store-deletion operation.
func NewConsoleService(repo Repository, assets AssetResolver, passwords ...AccountPasswordVerifier) *ConsoleService {
	svc := &ConsoleService{repo: repo, assets: assets, now: time.Now}
	if len(passwords) > 0 {
		svc.passwords = passwords[0]
	}
	return svc
}

// GetProfile returns the console profile view for the caller's own store.
func (s *ConsoleService) GetProfile(ctx context.Context, storeID int64) (ConsoleProfileView, error) {
	st, err := s.repo.GetStore(ctx, storeID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	settings, err := s.repo.GetStoreSettings(ctx, storeID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st, settings), nil
}

// UpdateProfile applies a full-replace update to the caller's own store and
// returns the refreshed profile view.
func (s *ConsoleService) UpdateProfile(ctx context.Context, storeID int64, req UpdateProfileRequest) (ConsoleProfileView, error) {
	if storeID <= 0 {
		return ConsoleProfileView{}, apperr.Invalid("invalid storeID")
	}
	if _, err := businesshours.Parse(req.BusinessHours); err != nil {
		return ConsoleProfileView{}, apperr.Invalid(err.Error())
	}
	st, err := s.repo.UpdateStoreProfile(ctx, storeID, req)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	settings, err := s.repo.GetStoreSettings(ctx, storeID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st, settings), nil
}

// UpdateStatus applies a manual open/closed override until the next configured
// business boundary, or clears it when status is auto.
func (s *ConsoleService) UpdateStatus(ctx context.Context, storeID int64, status string) (ConsoleProfileView, error) {
	if storeID <= 0 {
		return ConsoleProfileView{}, apperr.Invalid("invalid storeID")
	}
	if status != BusinessStatusOpen && status != BusinessStatusClosed && status != BusinessStatusAuto {
		return ConsoleProfileView{}, apperr.Invalid("invalid status")
	}
	st, err := s.repo.GetStore(ctx, storeID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	settings, err := s.repo.GetStoreSettings(ctx, storeID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	values := settingsMap(settings)
	if status == BusinessStatusAuto {
		delete(values, businessStatusOverrideKey)
	} else {
		schedule, err := businesshours.Parse(st.BusinessHours)
		if err != nil {
			return ConsoleProfileView{}, apperr.Invalid("请先设置有效的门店营业时间")
		}
		values[businessStatusOverrideKey] = businessStatusOverride{
			Status: status,
			Until:  schedule.NextBoundary(s.now(), businesshours.ShanghaiLocation()),
		}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return ConsoleProfileView{}, apperr.Internal(err)
	}
	settings, err = s.repo.UpsertStoreSettings(ctx, storeID, raw)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st, settings), nil
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
	delete(payload, businessStatusOverrideKey)
	current, err := s.repo.GetStoreSettings(ctx, storeID)
	if err != nil {
		return StoreSettingsView{}, err
	}
	if override, ok := settingsMap(current)[businessStatusOverrideKey]; ok {
		payload[businessStatusOverrideKey] = override
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
	view := StoreSettingsView{Settings: settingsMap(settings)}
	delete(view.Settings, businessStatusOverrideKey)
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

// DeleteStore verifies the acting headquarters administrator and then
// permanently removes the target store row.
func (s *ConsoleService) DeleteStore(ctx context.Context, id, accountID int64, password string, auditEntry audit.Entry) error {
	if id <= 0 {
		return apperr.Invalid("invalid storeID")
	}
	if s.passwords == nil {
		return apperr.Internal(fmt.Errorf("admin password verifier is not configured"))
	}
	if err := s.passwords.VerifyAccountPassword(ctx, accountID, password); err != nil {
		return err
	}
	return s.repo.DeleteStore(ctx, id, auditEntry)
}

func validateStoreInput(input StoreInput) error {
	if input.Name == "" {
		return apperr.Invalid("name is required")
	}
	if input.Address == "" {
		return apperr.Invalid("address is required")
	}
	if input.BusinessHours != "" {
		if _, err := businesshours.Parse(input.BusinessHours); err != nil {
			return apperr.Invalid(err.Error())
		}
	}
	return nil
}

// ProfileView maps a store to its console profile view (resolving the logo
// URL). Handlers use this to render create/update responses.
func (s *ConsoleService) ProfileView(ctx context.Context, st Store) (ConsoleProfileView, error) {
	settings, err := s.repo.GetStoreSettings(ctx, st.ID)
	if err != nil {
		return ConsoleProfileView{}, err
	}
	return s.consoleProfileView(ctx, st, settings), nil
}

func (s *ConsoleService) consoleProfileView(ctx context.Context, st Store, settings StoreSettings) ConsoleProfileView {
	status := evaluateBusinessStatus(st, settings, s.now())
	view := ConsoleProfileView{
		ID:                       st.ID,
		Name:                     st.Name,
		Phone:                    st.Phone,
		CustomerServiceQRAssetID: st.CustomerServiceQRAssetID,
		Address:                  st.Address,
		Latitude:                 st.Latitude,
		Longitude:                st.Longitude,
		BusinessHours:            st.BusinessHours,
		Status:                   status.Status,
		StatusMode:               status.Mode,
		ScheduledOpen:            status.ScheduledOpen,
		StatusOverrideUntil:      status.OverrideUntil,
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
