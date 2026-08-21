package printer

import (
	"context"
	"strings"

	"github.com/inwardclub/server/internal/platform/audit"
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

// AdminCreate registers a printer for an explicitly selected non-deleted store and
// persists the cross-store audit record atomically with the device row.
func (s *ConsoleService) AdminCreate(ctx context.Context, in AdminDeviceInput, idemKey string, entry audit.Entry) (DeviceView, error) {
	if in.StoreID <= 0 {
		return DeviceView{}, apperr.Invalid("storeId is required")
	}
	if err := validateAdminReason(in.Reason); err != nil {
		return DeviceView{}, err
	}
	d := Device{
		StoreID:   in.StoreID,
		Name:      in.Name,
		Provider:  orDefault(in.Provider, ProviderXpyun),
		DeviceSN:  in.DeviceSN,
		DeviceKey: in.DeviceKey,
		Status:    orDefault(in.Status, StatusActive),
	}
	if err := validateAdminDevice(d); err != nil {
		return DeviceView{}, err
	}
	entry.Reason = strings.TrimSpace(in.Reason)
	created, err := s.repo.AdminCreate(ctx, d, idemKey, entry)
	if err != nil {
		return DeviceView{}, err
	}
	return created.view(), nil
}

// AdminUpdate updates a device without allowing its owning store to change.
func (s *ConsoleService) AdminUpdate(ctx context.Context, id int64, in AdminDevicePatch, idemKey string, entry audit.Entry) (DeviceView, error) {
	if id <= 0 {
		return DeviceView{}, apperr.Invalid("invalid id")
	}
	if err := validateAdminReason(in.Reason); err != nil {
		return DeviceView{}, err
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return DeviceView{}, apperr.Invalid("name is required")
	}
	if in.DeviceSN != nil && strings.TrimSpace(*in.DeviceSN) == "" {
		return DeviceView{}, apperr.Invalid("deviceSn is required")
	}
	if in.Status != nil && *in.Status != StatusActive && *in.Status != StatusDisabled {
		return DeviceView{}, apperr.Invalid("invalid status")
	}
	entry.Reason = strings.TrimSpace(in.Reason)
	updated, err := s.repo.AdminUpdate(ctx, id, in.DevicePatch, idemKey, entry)
	if err != nil {
		return DeviceView{}, err
	}
	return updated.view(), nil
}

// AdminDelete permanently removes a printer with an atomic audit record.
func (s *ConsoleService) AdminDelete(ctx context.Context, id int64, in AdminDeleteInput, idemKey string, entry audit.Entry) error {
	if id <= 0 {
		return apperr.Invalid("invalid id")
	}
	if err := validateAdminReason(in.Reason); err != nil {
		return err
	}
	entry.Reason = strings.TrimSpace(in.Reason)
	return s.repo.AdminDelete(ctx, id, idemKey, entry)
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

func validateAdminReason(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return apperr.Invalid("reason is required")
	}
	if len([]rune(reason)) > 200 {
		return apperr.Invalid("reason is too long")
	}
	return nil
}

func validateAdminDevice(d Device) error {
	if strings.TrimSpace(d.Name) == "" {
		return apperr.Invalid("name is required")
	}
	if strings.TrimSpace(d.DeviceSN) == "" {
		return apperr.Invalid("deviceSn is required")
	}
	if d.Provider != ProviderXpyun {
		return apperr.Invalid("unsupported provider")
	}
	if d.Status != StatusActive && d.Status != StatusDisabled {
		return apperr.Invalid("invalid status")
	}
	return nil
}
