package printer

import (
	"context"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// ConsoleService provides admin (cross-store read) and store console
// (own-store scoped CRUD) printer device management.
type ConsoleService struct {
	repo Repository
}

// NewConsoleService builds the printer console service.
func NewConsoleService(repo Repository) *ConsoleService { return &ConsoleService{repo: repo} }

// AdminList returns every store's devices, optionally filtered to one store.
func (s *ConsoleService) AdminList(ctx context.Context, storeID *int64) ([]DeviceView, error) {
	return s.list(ctx, storeID)
}

// StoreList returns only the caller's own store devices.
func (s *ConsoleService) StoreList(ctx context.Context, storeID int64) ([]DeviceView, error) {
	return s.list(ctx, &storeID)
}

// StoreCreate registers a device pinned to the caller's own store scope.
func (s *ConsoleService) StoreCreate(ctx context.Context, storeID int64, in DeviceInput) (DeviceView, error) {
	if in.Name == "" {
		return DeviceView{}, apperr.Invalid("name is required")
	}
	if in.DeviceSN == "" {
		return DeviceView{}, apperr.Invalid("deviceSn is required")
	}
	d := Device{
		StoreID:   storeID,
		Name:      in.Name,
		Provider:  orDefault(in.Provider, ProviderXpyun),
		DeviceSN:  in.DeviceSN,
		DeviceKey: in.DeviceKey,
		Status:    orDefault(in.Status, StatusActive),
	}
	created, err := s.repo.Create(ctx, d)
	if err != nil {
		return DeviceView{}, err
	}
	return created.view(), nil
}

// StoreUpdate applies a partial update to one of the caller's own devices.
func (s *ConsoleService) StoreUpdate(ctx context.Context, storeID, id int64, patch DevicePatch) (DeviceView, error) {
	d, err := s.own(ctx, storeID, id)
	if err != nil {
		return DeviceView{}, err
	}
	applyPatch(&d, patch)
	updated, err := s.repo.Update(ctx, d)
	if err != nil {
		return DeviceView{}, err
	}
	return updated.view(), nil
}

// StoreDelete removes one of the caller's own devices.
func (s *ConsoleService) StoreDelete(ctx context.Context, storeID, id int64) error {
	if _, err := s.own(ctx, storeID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// own fetches a device and asserts it belongs to storeID; other stores'
// devices surface as NotFound so scope can never leak.
func (s *ConsoleService) own(ctx context.Context, storeID, id int64) (Device, error) {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return Device{}, err
	}
	if d.StoreID != storeID {
		return Device{}, apperr.NotFound("printer device not found")
	}
	return d, nil
}

func (s *ConsoleService) list(ctx context.Context, storeID *int64) ([]DeviceView, error) {
	devices, err := s.repo.List(ctx, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]DeviceView, 0, len(devices))
	for _, d := range devices {
		out = append(out, d.view())
	}
	return out, nil
}

func applyPatch(d *Device, p DevicePatch) {
	if p.Name != nil {
		d.Name = *p.Name
	}
	if p.DeviceSN != nil {
		d.DeviceSN = *p.DeviceSN
	}
	if p.DeviceKey != nil {
		d.DeviceKey = *p.DeviceKey
	}
	if p.Status != nil {
		d.Status = *p.Status
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
