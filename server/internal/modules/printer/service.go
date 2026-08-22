package printer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// ConsoleService provides admin (cross-store read) and store console
// (own-store scoped CRUD) printer device management.
type ConsoleService struct {
	repo  Repository
	cloud CloudPrinter
}

// NewConsoleService builds the printer console service.
func NewConsoleService(repo Repository, clouds ...CloudPrinter) *ConsoleService {
	cloud := CloudPrinter(NewFakePrinter())
	if len(clouds) > 0 && clouds[0] != nil {
		cloud = clouds[0]
	}
	return &ConsoleService{repo: repo, cloud: cloud}
}

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
		StoreID:      in.StoreID,
		Name:         strings.TrimSpace(in.Name),
		Provider:     ProviderXpyun,
		DeviceSN:     strings.TrimSpace(in.DeviceSN),
		Status:       StatusActive,
		SoundEnabled: true,
	}
	if err := validateAdminDevice(d); err != nil {
		return DeviceView{}, err
	}
	entry.Reason = strings.TrimSpace(in.Reason)
	if err := s.ensureSNAvailable(ctx, d.DeviceSN); err != nil {
		return DeviceView{}, err
	}
	if err := s.cloud.AddPrinter(ctx, d.DeviceSN, d.Name); err != nil {
		return DeviceView{}, err
	}
	created, err := s.repo.AdminCreate(ctx, d, idemKey, entry)
	if err != nil {
		_ = s.cloud.DeletePrinter(ctx, d.DeviceSN)
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
	if err := validatePatch(in.DevicePatch); err != nil {
		return DeviceView{}, err
	}
	before, err := s.repo.Get(ctx, id)
	if err != nil {
		return DeviceView{}, err
	}
	if err := s.updateProviderSettings(ctx, before, in.DevicePatch); err != nil {
		return DeviceView{}, err
	}
	entry.Reason = strings.TrimSpace(in.Reason)
	updated, err := s.repo.AdminUpdate(ctx, id, in.DevicePatch, idemKey, entry)
	if err != nil {
		if apperr.From(err).Code != apperr.CodeIdempotencyConflict {
			s.rollbackProviderSettings(ctx, before, in.DevicePatch)
		}
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
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.cloud.DeletePrinter(ctx, d.DeviceSN); err != nil {
		return err
	}
	return s.repo.AdminDelete(ctx, id, idemKey, entry)
}

// StoreList returns only the caller's own store devices.
func (s *ConsoleService) StoreList(ctx context.Context, storeID int64) ([]DeviceView, error) {
	return s.list(ctx, &storeID)
}

// StoreCreate registers a device pinned to the caller's own store scope.
func (s *ConsoleService) StoreCreate(ctx context.Context, storeID int64, in DeviceInput) (DeviceView, error) {
	name := strings.TrimSpace(in.Name)
	deviceSN := strings.TrimSpace(in.DeviceSN)
	if name == "" {
		return DeviceView{}, apperr.Invalid("name is required")
	}
	if len([]rune(name)) > 64 {
		return DeviceView{}, apperr.Invalid("name is too long")
	}
	if deviceSN == "" {
		return DeviceView{}, apperr.Invalid("deviceSn is required")
	}
	d := Device{
		StoreID:      storeID,
		Name:         name,
		Provider:     ProviderXpyun,
		DeviceSN:     deviceSN,
		Status:       StatusActive,
		SoundEnabled: true,
	}
	if err := s.ensureSNAvailable(ctx, d.DeviceSN); err != nil {
		return DeviceView{}, err
	}
	if err := s.cloud.AddPrinter(ctx, d.DeviceSN, d.Name); err != nil {
		return DeviceView{}, err
	}
	created, err := s.repo.Create(ctx, d)
	if err != nil {
		_ = s.cloud.DeletePrinter(ctx, d.DeviceSN)
		return DeviceView{}, err
	}
	return created.view(), nil
}

// StoreUpdate applies a partial update to one of the caller's own devices.
func (s *ConsoleService) StoreUpdate(ctx context.Context, storeID, id int64, patch DevicePatch) (DeviceView, error) {
	if err := validatePatch(patch); err != nil {
		return DeviceView{}, err
	}
	d, err := s.own(ctx, storeID, id)
	if err != nil {
		return DeviceView{}, err
	}
	before := d
	if err := s.updateProviderSettings(ctx, before, patch); err != nil {
		return DeviceView{}, err
	}
	applyPatch(&d, patch)
	updated, err := s.repo.Update(ctx, d)
	if err != nil {
		s.rollbackProviderSettings(ctx, before, patch)
		return DeviceView{}, err
	}
	return updated.view(), nil
}

// StoreDelete removes one of the caller's own devices.
func (s *ConsoleService) StoreDelete(ctx context.Context, storeID, id int64) error {
	d, err := s.own(ctx, storeID, id)
	if err != nil {
		return err
	}
	if err := s.cloud.DeletePrinter(ctx, d.DeviceSN); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// AdminTestPrint sends a fixed diagnostic slip to any configured device.
func (s *ConsoleService) AdminTestPrint(ctx context.Context, id int64) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.cloud.Print(ctx, buildTestPrintJob(d, time.Now()))
}

// StoreTestPrint sends a diagnostic slip only to a device owned by storeID.
func (s *ConsoleService) StoreTestPrint(ctx context.Context, storeID, id int64) error {
	d, err := s.own(ctx, storeID, id)
	if err != nil {
		return err
	}
	return s.cloud.Print(ctx, buildTestPrintJob(d, time.Now()))
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
	if len(devices) == 0 {
		return out, nil
	}
	sns := make([]string, 0, len(devices))
	for _, d := range devices {
		sns = append(sns, d.DeviceSN)
	}
	statuses, statusErr := s.cloud.QueryStatuses(ctx, sns)
	if statusErr != nil {
		status := ProviderStatusUnknown
		message := apperr.From(statusErr).Message
		if strings.Contains(message, "配置") {
			status = ProviderStatusUnconfigured
		}
		for i := range out {
			out[i].ProviderStatus = status
			out[i].ProviderStatusMessage = message
		}
		return out, nil
	}
	for i := range out {
		out[i].ProviderStatus = statuses[out[i].DeviceSN]
		if out[i].ProviderStatus == "" {
			out[i].ProviderStatus = ProviderStatusUnknown
		}
	}
	return out, nil
}

func applyPatch(d *Device, p DevicePatch) {
	if p.Name != nil {
		d.Name = strings.TrimSpace(*p.Name)
	}
	if p.Status != nil {
		d.Status = *p.Status
	}
	if p.SoundEnabled != nil {
		d.SoundEnabled = *p.SoundEnabled
	}
}

func validatePatch(p DevicePatch) error {
	if p.Name != nil && strings.TrimSpace(*p.Name) == "" {
		return apperr.Invalid("name is required")
	}
	if p.Name != nil && len([]rune(strings.TrimSpace(*p.Name))) > 64 {
		return apperr.Invalid("name is too long")
	}
	if p.Status != nil && *p.Status != StatusActive && *p.Status != StatusDisabled {
		return apperr.Invalid("invalid status")
	}
	return nil
}

func (s *ConsoleService) updateProviderSettings(ctx context.Context, before Device, patch DevicePatch) error {
	nameChanged := patch.Name != nil && strings.TrimSpace(*patch.Name) != before.Name
	if nameChanged {
		if err := s.cloud.UpdatePrinterName(ctx, before.DeviceSN, strings.TrimSpace(*patch.Name)); err != nil {
			return err
		}
	}
	if patch.SoundEnabled != nil && *patch.SoundEnabled != before.SoundEnabled {
		if err := s.setProviderSound(ctx, before.DeviceSN, *patch.SoundEnabled); err != nil {
			if nameChanged {
				_ = s.cloud.UpdatePrinterName(ctx, before.DeviceSN, before.Name)
			}
			return err
		}
	}
	return nil
}

func (s *ConsoleService) rollbackProviderSettings(ctx context.Context, before Device, patch DevicePatch) {
	if patch.SoundEnabled != nil && *patch.SoundEnabled != before.SoundEnabled {
		_ = s.setProviderSound(ctx, before.DeviceSN, before.SoundEnabled)
	}
	if patch.Name != nil && strings.TrimSpace(*patch.Name) != before.Name {
		_ = s.cloud.UpdatePrinterName(ctx, before.DeviceSN, before.Name)
	}
}

func (s *ConsoleService) setProviderSound(ctx context.Context, sn string, enabled bool) error {
	voiceType, volumeLevel := 4, 3
	if enabled {
		voiceType, volumeLevel = 0, 0
	}
	return s.cloud.SetVoice(ctx, sn, voiceType, &volumeLevel)
}

func (s *ConsoleService) ensureSNAvailable(ctx context.Context, deviceSN string) error {
	devices, err := s.repo.List(ctx, nil)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if strings.EqualFold(strings.TrimSpace(device.DeviceSN), deviceSN) {
			return apperr.Invalid("device with this provider and SN already exists")
		}
	}
	return nil
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
	if len([]rune(strings.TrimSpace(d.Name))) > 64 {
		return apperr.Invalid("name is too long")
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

func buildTestPrintJob(d Device, now time.Time) Job {
	content := fmt.Sprintf("<CB>InwardClub</CB>\n<CB>测试打印</CB>\n--------------------------------\n打印机连接正常\n%s\n--------------------------------\n<CUT>",
		now.In(receiptLocation).Format("2006-01-02 15:04:05"))
	return Job{
		DeviceSN: d.DeviceSN,
		Template: "test-print",
		Content:  content,
		Silent:   !d.SoundEnabled,
	}
}
