package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func TestDeleteStoreRequestDoesNotBindPlaintextPassword(t *testing.T) {
	var req DeleteStoreRequest
	if err := json.Unmarshal([]byte(`{"password":"secret"}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.Password != "" || req.PasswordCiphertext != "" || req.PasswordKeyID != "" {
		t.Fatalf("plaintext password must not bind: %+v", req)
	}
}

type consoleFakeRepo struct {
	store           Store
	gotUpdateID     int64
	gotUpdateFields UpdateProfileRequest
	gotCreateInput  StoreInput
	gotUpdateInput  StoreInput
	gotDeleteID     int64
	gotDeleteAudit  audit.Entry

	gotStatusID     int64
	gotStatus       string
	settings        StoreSettings
	gotSettingsID   int64
	gotSettingsJSON []byte
}

func (r *consoleFakeRepo) ListActiveStores(_ context.Context, _, _ int) ([]Store, int64, error) {
	return nil, 0, nil
}

func (r *consoleFakeRepo) GetStore(_ context.Context, id int64) (Store, error) {
	if id != r.store.ID {
		return Store{}, apperr.NotFound("store not found")
	}
	return r.store, nil
}

func (r *consoleFakeRepo) UpdateStoreProfile(_ context.Context, storeID int64, fields UpdateProfileRequest) (Store, error) {
	r.gotUpdateID = storeID
	r.gotUpdateFields = fields
	r.store.Name = fields.Name
	r.store.Phone = fields.Phone
	r.store.CustomerServiceQRAssetID = fields.CustomerServiceQRAssetID
	r.store.Address = fields.Address
	r.store.BusinessHours = fields.BusinessHours
	r.store.Latitude = fields.Latitude
	r.store.Longitude = fields.Longitude
	r.store.LogoAssetID = fields.LogoAssetID
	return r.store, nil
}

func (r *consoleFakeRepo) CreateStore(_ context.Context, input StoreInput) (Store, error) {
	r.gotCreateInput = input
	return Store{ID: 100, Name: input.Name, Phone: input.Phone, Address: input.Address,
		BusinessHours: input.BusinessHours, Latitude: input.Latitude, Longitude: input.Longitude,
		LogoAssetID: input.LogoAssetID, CustomerServiceQRAssetID: input.CustomerServiceQRAssetID, Status: StatusActive}, nil
}

func (r *consoleFakeRepo) UpdateStore(_ context.Context, id int64, input StoreInput) (Store, error) {
	r.gotUpdateID = id
	r.gotUpdateInput = input
	if id != r.store.ID {
		return Store{}, apperr.NotFound("store not found")
	}
	r.store.Name = input.Name
	r.store.Phone = input.Phone
	r.store.CustomerServiceQRAssetID = input.CustomerServiceQRAssetID
	r.store.Address = input.Address
	r.store.BusinessHours = input.BusinessHours
	r.store.Latitude = input.Latitude
	r.store.Longitude = input.Longitude
	r.store.LogoAssetID = input.LogoAssetID
	return r.store, nil
}

func (r *consoleFakeRepo) DeleteStore(_ context.Context, id int64, auditEntry audit.Entry) error {
	if id != r.store.ID {
		return apperr.NotFound("store not found")
	}
	r.gotDeleteID = id
	r.gotDeleteAudit = auditEntry
	r.store = Store{}
	return nil
}

func (r *consoleFakeRepo) UpdateStoreStatus(_ context.Context, storeID int64, status string) (Store, error) {
	r.gotStatusID = storeID
	r.gotStatus = status
	if storeID != r.store.ID {
		return Store{}, apperr.NotFound("store not found")
	}
	r.store.Status = status
	return r.store, nil
}

func (r *consoleFakeRepo) GetStoreSettings(_ context.Context, storeID int64) (StoreSettings, error) {
	if r.settings.SettingsJSON == nil {
		return StoreSettings{StoreID: storeID, SettingsJSON: []byte(`{}`)}, nil
	}
	return r.settings, nil
}

func (r *consoleFakeRepo) UpsertStoreSettings(_ context.Context, storeID int64, settingsJSON []byte) (StoreSettings, error) {
	r.gotSettingsID = storeID
	r.gotSettingsJSON = settingsJSON
	r.settings = StoreSettings{StoreID: storeID, SettingsJSON: settingsJSON, UpdatedAt: time.Now()}
	return r.settings, nil
}

func TestConsoleGetProfileMapsView(t *testing.T) {
	lat, lng := 31.0, 121.0
	logoID := int64(7)
	qrID := int64(8)
	repo := &consoleFakeRepo{store: Store{
		ID: 1, Name: "Store A", Phone: "123", Address: "Addr",
		Latitude: &lat, Longitude: &lng, BusinessHours: "15:00-02:00",
		Status: StatusActive, LogoAssetID: &logoID, CustomerServiceQRAssetID: &qrID,
	}}
	svc := NewConsoleService(repo, fakeResolver{})
	svc.now = func() time.Time {
		return time.Date(2026, 8, 30, 16, 29, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	}

	view, err := svc.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if view.ID != 1 || view.Name != "Store A" || view.Address != "Addr" || view.Status != BusinessStatusOpen || view.StatusMode != BusinessStatusModeAuto {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.LogoURL != "https://cdn.test/asset" {
		t.Fatalf("expected logo url resolved, got %q", view.LogoURL)
	}
	if view.CustomerServiceQRAssetID == nil || *view.CustomerServiceQRAssetID != qrID || view.CustomerServiceQRURL != "https://cdn.test/asset" {
		t.Fatalf("expected customer service QR resolved, got %+v", view)
	}
	if view.Latitude == nil || *view.Latitude != lat || view.Longitude == nil || *view.Longitude != lng {
		t.Fatalf("expected coordinates mapped, got latitude=%v longitude=%v", view.Latitude, view.Longitude)
	}
}

func TestConsoleUpdateProfilePassesStoreIDAndReturnsView(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 42, Name: "Old", Status: StatusActive}}
	svc := NewConsoleService(repo, fakeResolver{})

	req := UpdateProfileRequest{Name: "New Name", Phone: "999", Address: "New Addr", BusinessHours: "08:00-22:00"}
	view, err := svc.UpdateProfile(context.Background(), 42, req)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if repo.gotUpdateID != 42 {
		t.Fatalf("expected storeID 42 passed through, got %d", repo.gotUpdateID)
	}
	if repo.gotUpdateFields.Name != "New Name" {
		t.Fatalf("expected fields passed through, got %+v", repo.gotUpdateFields)
	}
	if view.Name != "New Name" || view.Address != "New Addr" {
		t.Fatalf("expected mapped view reflecting update, got %+v", view)
	}
}

func TestConsoleUpdateStatusPersistsUntilNextBusinessBoundary(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 42, Name: "Old", Status: StatusActive, BusinessHours: "15:00-02:00"}}
	svc := NewConsoleService(repo, fakeResolver{})
	now := time.Date(2026, 8, 30, 16, 29, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	svc.now = func() time.Time { return now }

	view, err := svc.UpdateStatus(context.Background(), 42, BusinessStatusClosed)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if repo.gotSettingsID != 42 {
		t.Fatalf("expected store settings update for store 42, got %d", repo.gotSettingsID)
	}
	if view.Status != BusinessStatusClosed || view.StatusMode != BusinessStatusModeManual || view.StatusOverrideUntil == nil {
		t.Fatalf("expected view reflecting new status, got %+v", view)
	}
	if got := view.StatusOverrideUntil.In(now.Location()).Format("2006-01-02 15:04"); got != "2026-08-31 02:00" {
		t.Fatalf("override boundary=%s", got)
	}

	now = time.Date(2026, 8, 31, 2, 0, 0, 0, now.Location())
	view, err = svc.GetProfile(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != BusinessStatusClosed || view.StatusMode != BusinessStatusModeAuto {
		t.Fatalf("override should expire at boundary: %+v", view)
	}
}

func TestConsoleUpdateStatusRejectsInvalidStoreID(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.UpdateStatus(context.Background(), 0, BusinessStatusOpen); err == nil {
		t.Fatal("expected error for invalid storeID")
	}
}

func TestConsoleUpdateStatusRejectsInvalidStatus(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 1, Status: StatusActive}}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.UpdateStatus(context.Background(), 1, "bogus"); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestConsoleGetSettingsDefaultsToEmptyObject(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	view, err := svc.GetSettings(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if view.Settings == nil || len(view.Settings) != 0 {
		t.Fatalf("expected empty settings object, got %+v", view.Settings)
	}
	if view.UpdatedAt != nil {
		t.Fatalf("expected nil updatedAt for unset settings, got %v", view.UpdatedAt)
	}
}

func TestConsoleUpdateSettingsPassesStoreIDAndReturnsView(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	req := UpdateSettingsRequest{Settings: map[string]any{"autoAcceptOrders": true}}
	view, err := svc.UpdateSettings(context.Background(), 7, req)
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if repo.gotSettingsID != 7 {
		t.Fatalf("expected storeID 7 passed through, got %d", repo.gotSettingsID)
	}
	if string(repo.gotSettingsJSON) != `{"autoAcceptOrders":true}` {
		t.Fatalf("expected marshalled settings persisted, got %s", repo.gotSettingsJSON)
	}
	if view.Settings["autoAcceptOrders"] != true {
		t.Fatalf("expected view reflecting new settings, got %+v", view.Settings)
	}
	if view.UpdatedAt == nil {
		t.Fatalf("expected updatedAt set after write")
	}
}

func TestConsoleUpdateSettingsPreservesAndHidesBusinessStatusOverride(t *testing.T) {
	repo := &consoleFakeRepo{settings: StoreSettings{
		StoreID:      7,
		SettingsJSON: []byte(`{"_businessStatusOverride":{"status":"closed","until":"2026-08-31T02:00:00+08:00"},"old":true}`),
	}}
	svc := NewConsoleService(repo, fakeResolver{})

	view, err := svc.UpdateSettings(context.Background(), 7, UpdateSettingsRequest{Settings: map[string]any{
		"autoAcceptOrders":        true,
		businessStatusOverrideKey: map[string]any{"status": "open"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, exposed := view.Settings[businessStatusOverrideKey]; exposed {
		t.Fatalf("internal override must not be exposed: %+v", view.Settings)
	}
	var persisted map[string]any
	if err := json.Unmarshal(repo.gotSettingsJSON, &persisted); err != nil {
		t.Fatal(err)
	}
	override := persisted[businessStatusOverrideKey].(map[string]any)
	if override["status"] != BusinessStatusClosed {
		t.Fatalf("client must not overwrite internal override: %+v", override)
	}
}

func TestConsoleGetProfileNotScopedToAnyCallerStore(t *testing.T) {
	// Admin-side reads pass an arbitrary storeID with no pinned scope; the
	// service must serve any existing store, not just a "caller's own" store.
	repo := &consoleFakeRepo{store: Store{ID: 77, Name: "Any Store", Status: StatusActive}}
	svc := NewConsoleService(repo, fakeResolver{})

	view, err := svc.GetProfile(context.Background(), 77)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if view.ID != 77 || view.Name != "Any Store" {
		t.Fatalf("unexpected view: %+v", view)
	}

	if _, err := svc.GetProfile(context.Background(), 9999); err == nil {
		t.Fatal("expected not found for unknown storeID")
	}
}

func TestConsoleGetSettingsAndUpdateSettingsForArbitraryStoreID(t *testing.T) {
	// Admin-side settings endpoints take the storeID from the path, not from
	// a pinned request scope.
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.GetSettings(context.Background(), 321); err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	req := UpdateSettingsRequest{Settings: map[string]any{"tableCount": float64(12)}}
	view, err := svc.UpdateSettings(context.Background(), 321, req)
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if repo.gotSettingsID != 321 {
		t.Fatalf("expected storeID 321 passed through, got %d", repo.gotSettingsID)
	}
	if view.Settings["tableCount"] != float64(12) {
		t.Fatalf("expected updated settings reflected, got %+v", view.Settings)
	}
}

type consolePasswordVerifier struct {
	err       error
	accountID int64
	password  string
}

func (v *consolePasswordVerifier) VerifyAccountPassword(_ context.Context, accountID int64, password string) error {
	v.accountID = accountID
	v.password = password
	return v.err
}

func TestConsoleDeleteStoreVerifiesPasswordAndDeletes(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 42, Name: "Store A", Status: StatusActive}}
	verifier := &consolePasswordVerifier{}
	svc := NewConsoleService(repo, fakeResolver{}, verifier)
	auditEntry := audit.Entry{ActorID: 7, Action: "store.delete", TargetID: 42}

	if err := svc.DeleteStore(context.Background(), 42, 7, "secret", auditEntry); err != nil {
		t.Fatalf("DeleteStore: %v", err)
	}
	if verifier.accountID != 7 || verifier.password != "secret" {
		t.Fatalf("expected password verification for account 7, got account=%d password=%q", verifier.accountID, verifier.password)
	}
	if repo.gotDeleteID != 42 || repo.gotDeleteAudit.Action != "store.delete" {
		t.Fatalf("expected deletion and audit entry, got id=%d audit=%+v", repo.gotDeleteID, repo.gotDeleteAudit)
	}
	if _, err := repo.GetStore(context.Background(), 42); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("deleted store must no longer exist, got %v", err)
	}
}

func TestConsoleDeleteStoreRejectsInvalidPassword(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 42, Name: "Store A", Status: StatusActive}}
	verifier := &consolePasswordVerifier{err: apperr.Invalid("密码错误")}
	svc := NewConsoleService(repo, fakeResolver{}, verifier)

	if err := svc.DeleteStore(context.Background(), 42, 7, "wrong", audit.Entry{}); err == nil {
		t.Fatal("expected password verification error")
	}
	if repo.gotDeleteID != 0 || repo.store.Status != StatusActive {
		t.Fatalf("store should remain unchanged after failed password verification: %+v", repo.store)
	}
}

func TestConsoleUpdateSettingsRejectsInvalidStoreID(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.UpdateSettings(context.Background(), 0, UpdateSettingsRequest{}); err == nil {
		t.Fatal("expected error for invalid storeID")
	}
}

func TestConsoleUpdateProfileRejectsInvalidStoreID(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.UpdateProfile(context.Background(), 0, UpdateProfileRequest{}); err == nil {
		t.Fatal("expected error for invalid storeID")
	}
}

func TestConsoleCreateStorePersistsAndReturns(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	logoID := int64(9)
	st, err := svc.CreateStore(context.Background(), StoreInput{
		Name: "New Store", Phone: "111", Address: "Some Addr", BusinessHours: "09:00-18:00", LogoAssetID: &logoID,
	})
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	if repo.gotCreateInput.Name != "New Store" || repo.gotCreateInput.Address != "Some Addr" {
		t.Fatalf("expected input passed through, got %+v", repo.gotCreateInput)
	}
	if st.ID != 100 || st.Name != "New Store" || st.Status != StatusActive {
		t.Fatalf("unexpected created store: %+v", st)
	}
}

func TestConsoleCreateStoreValidatesRequiredFields(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.CreateStore(context.Background(), StoreInput{Address: "Y"}); err == nil {
		t.Fatal("expected error for missing name")
	}
	if _, err := svc.CreateStore(context.Background(), StoreInput{Name: "X"}); err == nil {
		t.Fatal("expected error for missing address")
	}
}

func TestConsoleUpdateStorePersistsAndReturns(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 5, Name: "Old", Status: StatusActive}}
	svc := NewConsoleService(repo, fakeResolver{})

	st, err := svc.UpdateStore(context.Background(), 5, StoreInput{Name: "Updated", Address: "Addr2"})
	if err != nil {
		t.Fatalf("UpdateStore: %v", err)
	}
	if repo.gotUpdateID != 5 || repo.gotUpdateInput.Name != "Updated" {
		t.Fatalf("expected id/input passed through, got id=%d input=%+v", repo.gotUpdateID, repo.gotUpdateInput)
	}
	if st.Name != "Updated" || st.Address != "Addr2" {
		t.Fatalf("unexpected updated store: %+v", st)
	}
}

func TestConsoleUpdateStoreRejectsInvalidID(t *testing.T) {
	repo := &consoleFakeRepo{}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.UpdateStore(context.Background(), 0, StoreInput{Name: "X", Address: "Y"}); err == nil {
		t.Fatal("expected error for invalid storeID")
	}
}

func TestConsoleUpdateStoreValidatesRequiredFields(t *testing.T) {
	repo := &consoleFakeRepo{store: Store{ID: 5}}
	svc := NewConsoleService(repo, fakeResolver{})

	if _, err := svc.UpdateStore(context.Background(), 5, StoreInput{Address: "Y"}); err == nil {
		t.Fatal("expected error for missing name")
	}
}
