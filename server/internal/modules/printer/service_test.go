package printer

import (
	"context"
	"testing"

	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func isNotFound(err error) bool {
	return err != nil && apperr.From(err).Code == apperr.CodeNotFound
}

// memRepo is an in-memory Repository for exercising the console service.
type memRepo struct {
	devices map[int64]Device
	nextID  int64
}

func newMemRepo() *memRepo { return &memRepo{devices: map[int64]Device{}, nextID: 1} }

func (r *memRepo) List(_ context.Context, storeID *int64) ([]Device, error) {
	out := make([]Device, 0)
	for _, d := range r.devices {
		if storeID != nil && d.StoreID != *storeID {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *memRepo) Get(_ context.Context, id int64) (Device, error) {
	d, ok := r.devices[id]
	if !ok {
		return Device{}, apperr.NotFound("printer device not found")
	}
	return d, nil
}

func (r *memRepo) Create(_ context.Context, d Device) (Device, error) {
	d.ID = r.nextID
	r.nextID++
	r.devices[d.ID] = d
	return d, nil
}

func (r *memRepo) Update(_ context.Context, d Device) (Device, error) {
	if _, ok := r.devices[d.ID]; !ok {
		return Device{}, apperr.NotFound("printer device not found")
	}
	r.devices[d.ID] = d
	return d, nil
}

func (r *memRepo) Delete(_ context.Context, id int64) error {
	if _, ok := r.devices[id]; !ok {
		return apperr.NotFound("printer device not found")
	}
	delete(r.devices, id)
	return nil
}

func (r *memRepo) AdminCreate(ctx context.Context, d Device, _ string, _ audit.Entry) (Device, error) {
	return r.Create(ctx, d)
}

func (r *memRepo) AdminUpdate(ctx context.Context, id int64, patch DevicePatch, _ string, _ audit.Entry) (Device, error) {
	d, err := r.Get(ctx, id)
	if err != nil {
		return Device{}, err
	}
	applyPatch(&d, patch)
	return r.Update(ctx, d)
}

func (r *memRepo) AdminDelete(ctx context.Context, id int64, _ string, _ audit.Entry) error {
	return r.Delete(ctx, id)
}

func TestStoreCreateDefaultsAndScope(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	v, err := svc.StoreCreate(ctx, 7, DeviceInput{Name: "Front", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.StoreID != 7 {
		t.Fatalf("store id = %d, want 7", v.StoreID)
	}
	if v.Provider != ProviderXpyun {
		t.Fatalf("provider = %q, want default xpyun", v.Provider)
	}
	if v.Status != StatusActive {
		t.Fatalf("status = %q, want default active", v.Status)
	}
}

func TestStoreCreateValidation(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	if _, err := svc.StoreCreate(ctx, 1, DeviceInput{DeviceSN: "SN"}); err == nil {
		t.Fatal("expected error for missing name")
	}
	if _, err := svc.StoreCreate(ctx, 1, DeviceInput{Name: "x"}); err == nil {
		t.Fatal("expected error for missing deviceSn")
	}
}

func TestStoreScopeIsolation(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	owned, _ := svc.StoreCreate(ctx, 1, DeviceInput{Name: "A", DeviceSN: "SN-A"})

	// Another store cannot read, update or delete it.
	if _, err := svc.StoreUpdate(ctx, 2, owned.ID, DevicePatch{}); !isNotFound(err) {
		t.Fatalf("cross-store update err = %v, want NotFound", err)
	}
	if err := svc.StoreDelete(ctx, 2, owned.ID); !isNotFound(err) {
		t.Fatalf("cross-store delete err = %v, want NotFound", err)
	}

	list, _ := svc.StoreList(ctx, 2)
	if len(list) != 0 {
		t.Fatalf("store 2 sees %d devices, want 0", len(list))
	}
}

func TestStoreUpdatePartial(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	d, _ := svc.StoreCreate(ctx, 1, DeviceInput{Name: "Old", DeviceSN: "SN-1"})
	newName := "New"
	disabled := StatusDisabled
	v, err := svc.StoreUpdate(ctx, 1, d.ID, DevicePatch{Name: &newName, Status: &disabled})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v.Name != "New" || v.Status != StatusDisabled {
		t.Fatalf("update result = %+v", v)
	}
	if v.DeviceSN != "SN-1" {
		t.Fatalf("deviceSn changed unexpectedly: %q", v.DeviceSN)
	}
}

func TestAdminListCrossStore(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	svc.StoreCreate(ctx, 1, DeviceInput{Name: "A", DeviceSN: "SN-A"})
	svc.StoreCreate(ctx, 2, DeviceInput{Name: "B", DeviceSN: "SN-B"})

	all, err := svc.AdminList(ctx, nil)
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("admin sees %d devices, want 2", len(all))
	}

	one := int64(2)
	filtered, _ := svc.AdminList(ctx, &one)
	if len(filtered) != 1 || filtered[0].StoreID != 2 {
		t.Fatalf("filtered = %+v", filtered)
	}
}

func TestAdminCRUDRequiresStoreAndReason(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()
	entry := audit.Entry{}

	if _, err := svc.AdminCreate(ctx, AdminDeviceInput{Name: "Front", DeviceSN: "SN-1", Reason: "配置前台打印"}, "k1", entry); err == nil {
		t.Fatal("expected store validation error")
	}
	if _, err := svc.AdminCreate(ctx, AdminDeviceInput{StoreID: 1, Name: "Front", DeviceSN: "SN-1"}, "k2", entry); err == nil {
		t.Fatal("expected reason validation error")
	}
	created, err := svc.AdminCreate(ctx, AdminDeviceInput{StoreID: 1, Name: "Front", DeviceSN: "SN-1", Reason: "配置前台打印"}, "k3", entry)
	if err != nil {
		t.Fatalf("admin create: %v", err)
	}
	disabled := StatusDisabled
	updated, err := svc.AdminUpdate(ctx, created.ID, AdminDevicePatch{DevicePatch: DevicePatch{Status: &disabled}, Reason: "设备维护"}, "k4", entry)
	if err != nil || updated.Status != StatusDisabled {
		t.Fatalf("admin update = %+v, %v", updated, err)
	}
	if err := svc.AdminDelete(ctx, created.ID, AdminDeleteInput{Reason: "设备退役"}, "k5", entry); err != nil {
		t.Fatalf("admin delete: %v", err)
	}
	if _, err := svc.AdminList(ctx, nil); err != nil {
		t.Fatalf("admin list after delete: %v", err)
	}
}
