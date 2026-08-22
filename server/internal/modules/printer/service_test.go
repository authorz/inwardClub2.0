package printer

import (
	"context"
	"strings"
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

type cloudStub struct {
	*FakePrinter
	addErr      error
	updateErr   error
	addedSN     string
	addedName   string
	updatedSN   string
	updatedName string
	deletedSN   string
	voiceSN     string
	voiceType   int
	volumeLevel int
	voiceErr    error
	statuses    map[string]ProviderStatus
}

func newCloudStub() *cloudStub { return &cloudStub{FakePrinter: NewFakePrinter()} }

func (c *cloudStub) AddPrinter(_ context.Context, sn, name string) error {
	c.addedSN = sn
	c.addedName = name
	return c.addErr
}

func (c *cloudStub) UpdatePrinterName(_ context.Context, sn, name string) error {
	c.updatedSN = sn
	c.updatedName = name
	return c.updateErr
}

func (c *cloudStub) DeletePrinter(_ context.Context, sn string) error {
	c.deletedSN = sn
	return nil
}

func (c *cloudStub) SetVoice(_ context.Context, sn string, voiceType int, volumeLevel *int) error {
	c.voiceSN = sn
	c.voiceType = voiceType
	if volumeLevel != nil {
		c.volumeLevel = *volumeLevel
	}
	return c.voiceErr
}

func (c *cloudStub) QueryStatuses(_ context.Context, sns []string) (map[string]ProviderStatus, error) {
	if c.statuses != nil {
		return c.statuses, nil
	}
	return c.FakePrinter.QueryStatuses(context.Background(), sns)
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

	v, err := svc.StoreCreate(ctx, 7, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
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
	if !v.SoundEnabled {
		t.Fatal("new printer should default to sound enabled")
	}
	if v.Name != "前台" {
		t.Fatalf("name = %q, want 前台", v.Name)
	}
}

func TestStoreCreateValidation(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	if _, err := svc.StoreCreate(ctx, 1, DeviceInput{Name: "前台"}); err == nil {
		t.Fatal("expected error for missing deviceSn")
	}
	if _, err := svc.StoreCreate(ctx, 1, DeviceInput{DeviceSN: "SN-1"}); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestStoreCreatePersistsOnlyAfterProviderAccepts(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	cloud.addErr = apperr.Invalid("芯烨云拒绝添加")
	svc := NewConsoleService(repo, cloud)

	if _, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"}); err == nil {
		t.Fatal("expected provider rejection")
	}
	if cloud.addedSN != "SN-1" {
		t.Fatalf("provider received %q", cloud.addedSN)
	}
	if cloud.addedName != "前台" {
		t.Fatalf("provider received name %q", cloud.addedName)
	}
	if len(repo.devices) != 0 {
		t.Fatalf("local device persisted before provider success: %#v", repo.devices)
	}
}

func TestStoreListIncludesProviderStatus(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	svc := NewConsoleService(repo, cloud)
	created, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cloud.statuses = map[string]ProviderStatus{"SN-1": ProviderStatusAbnormal}
	list, err := svc.StoreList(context.Background(), 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID || list[0].ProviderStatus != ProviderStatusAbnormal {
		t.Fatalf("list = %#v", list)
	}
}

func TestStoreScopeIsolation(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	owned, _ := svc.StoreCreate(ctx, 1, DeviceInput{Name: "前台", DeviceSN: "SN-A"})

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

	d, _ := svc.StoreCreate(ctx, 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	disabled := StatusDisabled
	v, err := svc.StoreUpdate(ctx, 1, d.ID, DevicePatch{Status: &disabled})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v.Status != StatusDisabled {
		t.Fatalf("update result = %+v", v)
	}
	if v.DeviceSN != "SN-1" {
		t.Fatalf("deviceSn changed unexpectedly: %q", v.DeviceSN)
	}
}

func TestStoreUpdateNameSyncsProviderBeforePersisting(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	svc := NewConsoleService(repo, cloud)
	d, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	name := "后厨"
	updated, err := svc.StoreUpdate(context.Background(), 1, d.ID, DevicePatch{Name: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if cloud.updatedSN != "SN-1" || cloud.updatedName != "后厨" {
		t.Fatalf("provider update = (%q, %q)", cloud.updatedSN, cloud.updatedName)
	}
	if updated.Name != "后厨" || repo.devices[d.ID].Name != "后厨" {
		t.Fatalf("updated device = %#v", updated)
	}
}

func TestStoreUpdateNameDoesNotPersistWhenProviderRejects(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	svc := NewConsoleService(repo, cloud)
	d, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cloud.updateErr = apperr.Invalid("芯烨云拒绝修改")

	name := "后厨"
	if _, err := svc.StoreUpdate(context.Background(), 1, d.ID, DevicePatch{Name: &name}); err == nil {
		t.Fatal("expected provider rejection")
	}
	if repo.devices[d.ID].Name != "前台" {
		t.Fatalf("local name changed to %q", repo.devices[d.ID].Name)
	}
}

func TestStoreUpdateSoundSyncsProviderAndPersists(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	svc := NewConsoleService(repo, cloud)
	d, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	disabled := false
	updated, err := svc.StoreUpdate(context.Background(), 1, d.ID, DevicePatch{SoundEnabled: &disabled})
	if err != nil {
		t.Fatalf("update sound: %v", err)
	}
	if updated.SoundEnabled || repo.devices[d.ID].SoundEnabled {
		t.Fatalf("sound setting not persisted: %+v", updated)
	}
	if cloud.voiceSN != "SN-1" || cloud.voiceType != 4 || cloud.volumeLevel != 3 {
		t.Fatalf("disable voice call = (%q, %d, %d)", cloud.voiceSN, cloud.voiceType, cloud.volumeLevel)
	}

	enabled := true
	if _, err := svc.StoreUpdate(context.Background(), 1, d.ID, DevicePatch{SoundEnabled: &enabled}); err != nil {
		t.Fatalf("enable sound: %v", err)
	}
	if cloud.voiceType != 0 || cloud.volumeLevel != 0 {
		t.Fatalf("enable voice call = (%d, %d)", cloud.voiceType, cloud.volumeLevel)
	}
}

func TestStoreUpdateSoundDoesNotPersistWhenProviderRejects(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	svc := NewConsoleService(repo, cloud)
	d, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cloud.voiceErr = apperr.Invalid("芯烨云拒绝设置声音")
	disabled := false
	if _, err := svc.StoreUpdate(context.Background(), 1, d.ID, DevicePatch{SoundEnabled: &disabled}); err == nil {
		t.Fatal("expected provider rejection")
	}
	if !repo.devices[d.ID].SoundEnabled {
		t.Fatal("local sound setting changed before provider success")
	}
}

func TestStoreTestPrintUsesDeviceSoundAndScope(t *testing.T) {
	repo := newMemRepo()
	cloud := newCloudStub()
	svc := NewConsoleService(repo, cloud)
	d, err := svc.StoreCreate(context.Background(), 1, DeviceInput{Name: "前台", DeviceSN: "SN-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	disabled := false
	if _, err := svc.StoreUpdate(context.Background(), 1, d.ID, DevicePatch{SoundEnabled: &disabled}); err != nil {
		t.Fatalf("disable sound: %v", err)
	}
	if err := svc.StoreTestPrint(context.Background(), 2, d.ID); !isNotFound(err) {
		t.Fatalf("cross-store test print err = %v, want NotFound", err)
	}
	if err := svc.StoreTestPrint(context.Background(), 1, d.ID); err != nil {
		t.Fatalf("test print: %v", err)
	}
	if cloud.Count() != 1 {
		t.Fatalf("printed jobs = %d, want 1", cloud.Count())
	}
	job := cloud.Jobs[0]
	if job.DeviceSN != "SN-1" || job.Template != "test-print" || !job.Silent {
		t.Fatalf("test print job = %+v", job)
	}
	if !strings.Contains(job.Content, "测试打印") || !strings.Contains(job.Content, "打印机连接正常") {
		t.Fatalf("test print content = %q", job.Content)
	}
}

func TestAdminListCrossStore(t *testing.T) {
	svc := NewConsoleService(newMemRepo())
	ctx := context.Background()

	svc.StoreCreate(ctx, 1, DeviceInput{Name: "前台", DeviceSN: "SN-A"})
	svc.StoreCreate(ctx, 2, DeviceInput{Name: "后厨", DeviceSN: "SN-B"})

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

	if _, err := svc.AdminCreate(ctx, AdminDeviceInput{Name: "前台", DeviceSN: "SN-1", Reason: "配置前台打印"}, "k1", entry); err == nil {
		t.Fatal("expected store validation error")
	}
	if _, err := svc.AdminCreate(ctx, AdminDeviceInput{StoreID: 1, Name: "前台", DeviceSN: "SN-1"}, "k2", entry); err == nil {
		t.Fatal("expected reason validation error")
	}
	created, err := svc.AdminCreate(ctx, AdminDeviceInput{StoreID: 1, Name: "前台", DeviceSN: "SN-1", Reason: "配置前台打印"}, "k3", entry)
	if err != nil {
		t.Fatalf("admin create: %v", err)
	}
	disabled := StatusDisabled
	name := "后厨"
	updated, err := svc.AdminUpdate(ctx, created.ID, AdminDevicePatch{DevicePatch: DevicePatch{Name: &name, Status: &disabled}, Reason: "设备维护"}, "k4", entry)
	if err != nil || updated.Status != StatusDisabled || updated.Name != "后厨" {
		t.Fatalf("admin update = %+v, %v", updated, err)
	}
	if err := svc.AdminDelete(ctx, created.ID, AdminDeleteInput{Reason: "设备退役"}, "k5", entry); err != nil {
		t.Fatalf("admin delete: %v", err)
	}
	if _, err := svc.AdminList(ctx, nil); err != nil {
		t.Fatalf("admin list after delete: %v", err)
	}
}
