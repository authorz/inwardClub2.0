package reservation

import (
	"context"
	"testing"
)

type consoleRepoStub struct {
	storeExists  bool
	table        Table
	seat         Seat
	createdTable Table
	updatedTable Table
	createdSeat  Seat
}

func (r *consoleRepoStub) StoreExists(context.Context, int64) (bool, error) {
	return r.storeExists, nil
}
func (r *consoleRepoStub) ListAdminTables(context.Context, AdminTableFilter, int, int) ([]Table, int64, error) {
	return nil, 0, nil
}
func (r *consoleRepoStub) GetAdminTable(context.Context, int64) (Table, error) {
	return r.table, nil
}
func (r *consoleRepoStub) CreateAdminTable(_ context.Context, table Table) (Table, error) {
	r.createdTable = table
	table.ID = 1
	return table, nil
}
func (r *consoleRepoStub) UpdateAdminTable(_ context.Context, _ int64, table Table) (Table, error) {
	r.updatedTable = table
	return table, nil
}
func (r *consoleRepoStub) DeleteAdminTable(context.Context, int64) error { return nil }
func (r *consoleRepoStub) ListAdminSeats(context.Context, AdminSeatFilter, int, int) ([]Seat, int64, error) {
	return nil, 0, nil
}
func (r *consoleRepoStub) GetAdminSeat(context.Context, int64) (Seat, error) {
	return r.seat, nil
}
func (r *consoleRepoStub) CreateAdminSeat(_ context.Context, seat Seat) (Seat, error) {
	r.createdSeat = seat
	seat.ID = 1
	return seat, nil
}
func (r *consoleRepoStub) UpdateAdminSeat(_ context.Context, _ int64, seat Seat) (Seat, error) {
	return seat, nil
}
func (r *consoleRepoStub) DeleteAdminSeat(context.Context, int64) error { return nil }

func TestConsoleServiceCreateTableValidatesAndNormalizes(t *testing.T) {
	repo := &consoleRepoStub{storeExists: true}
	svc := NewConsoleService(repo)

	view, err := svc.CreateTable(context.Background(), TableWriteRequest{
		StoreID: 7, Name: "  靠窗桌  ", Code: " A-01 ", Capacity: 4,
		BasePoints: 50, Status: AvailabilityAvailable,
	})
	if err != nil {
		t.Fatalf("CreateTable() error = %v", err)
	}
	if repo.createdTable.Name != "靠窗桌" || repo.createdTable.Code != "A-01" {
		t.Fatalf("table was not normalized: %+v", repo.createdTable)
	}
	if view.Capacity != 4 || view.BasePoints != 50 {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestConsoleServiceCreateTableRejectsInvalidInput(t *testing.T) {
	tests := []TableWriteRequest{
		{StoreID: 0, Name: "A", Code: "1", Status: AvailabilityAvailable},
		{StoreID: 1, Name: "", Code: "1", Status: AvailabilityAvailable},
		{StoreID: 1, Name: "A", Code: "", Status: AvailabilityAvailable},
		{StoreID: 1, Name: "A", Code: "1", Capacity: 0, Status: AvailabilityAvailable},
		{StoreID: 1, Name: "A", Code: "1", Capacity: -1, Status: AvailabilityAvailable},
		{StoreID: 1, Name: "A", Code: "1", BasePoints: -1, Status: AvailabilityAvailable},
		{StoreID: 1, Name: "A", Code: "1", Status: "unknown"},
	}
	for i, req := range tests {
		repo := &consoleRepoStub{storeExists: true}
		if _, err := NewConsoleService(repo).CreateTable(context.Background(), req); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestConsoleServiceCreateTableRequiresExistingStore(t *testing.T) {
	repo := &consoleRepoStub{storeExists: false}
	_, err := NewConsoleService(repo).CreateTable(context.Background(), TableWriteRequest{
		StoreID: 99, Name: "A", Code: "A1", Status: AvailabilityAvailable,
	})
	if err == nil {
		t.Fatal("expected missing store error")
	}
}

func TestConsoleServiceUpdateTableRejectsCapacityBelowSeatCount(t *testing.T) {
	repo := &consoleRepoStub{
		storeExists: true,
		table:       Table{ID: 1, StoreID: 7, SeatCount: 3},
	}
	_, err := NewConsoleService(repo).UpdateTable(context.Background(), 1, TableWriteRequest{
		StoreID: 7, Name: "A", Code: "A1", Capacity: 2, Status: AvailabilityAvailable,
	})
	if err == nil {
		t.Fatal("expected capacity conflict")
	}
}

func TestConsoleServiceUpdateTableAllowsChangingStoreWithSeats(t *testing.T) {
	repo := &consoleRepoStub{
		storeExists: true,
		table:       Table{ID: 1, StoreID: 7, SeatCount: 3},
	}
	view, err := NewConsoleService(repo).UpdateTable(context.Background(), 1, TableWriteRequest{
		StoreID: 8, Name: "A", Code: "A1", Capacity: 3, Status: AvailabilityAvailable,
	})
	if err != nil {
		t.Fatalf("changing table store with seats should succeed: %v", err)
	}
	if repo.updatedTable.StoreID != 8 || view.StoreID != 8 {
		t.Fatalf("expected table to move to store 8, updated=%+v view=%+v", repo.updatedTable, view)
	}
}

func TestConsoleServiceCreateSeatRequiresTable(t *testing.T) {
	repo := &consoleRepoStub{}
	svc := NewConsoleService(repo)
	if _, err := svc.CreateSeat(context.Background(), SeatWriteRequest{
		Name: "1号位", Status: AvailabilityAvailable,
	}); err == nil {
		t.Fatal("expected tableId validation error")
	}
	if _, err := svc.CreateSeat(context.Background(), SeatWriteRequest{
		TableID: 2, Name: " 1号位 ", Status: AvailabilityAvailable,
	}); err != nil {
		t.Fatalf("CreateSeat() error = %v", err)
	}
	if repo.createdSeat.Name != "1号位" || repo.createdSeat.TableID == nil || *repo.createdSeat.TableID != 2 {
		t.Fatalf("unexpected seat: %+v", repo.createdSeat)
	}
}

func TestConsoleServiceStoreScopeRejectsOtherStore(t *testing.T) {
	repo := &consoleRepoStub{
		storeExists: true,
		table:       Table{ID: 1, StoreID: 8, Name: "其他门店桌子"},
		seat:        Seat{ID: 2, StoreID: 8, Name: "其他门店座位"},
	}
	svc := NewConsoleService(repo)
	if _, err := svc.StoreGetTable(context.Background(), 7, 1); err == nil {
		t.Fatal("expected cross-store table to be hidden")
	}
	if _, err := svc.StoreGetSeat(context.Background(), 7, 2); err == nil {
		t.Fatal("expected cross-store seat to be hidden")
	}
}

func TestConsoleServiceStoreCreatePinsStore(t *testing.T) {
	repo := &consoleRepoStub{storeExists: true}
	svc := NewConsoleService(repo)
	_, err := svc.StoreCreateTable(context.Background(), 7, TableWriteRequest{
		StoreID: 99, Name: "本店桌子", Code: "T1", Capacity: 2, Status: AvailabilityAvailable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.createdTable.StoreID != 7 {
		t.Fatalf("store id = %d, want 7", repo.createdTable.StoreID)
	}
}
