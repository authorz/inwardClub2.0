package reservation

import (
	"context"
	"strings"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// ConsoleService provides headquarters table and seat management.
type ConsoleService struct {
	repo   ConsoleRepository
	assets AssetResolver
}

func NewConsoleService(repo ConsoleRepository, assets AssetResolver) *ConsoleService {
	return &ConsoleService{repo: repo, assets: assets}
}

func (s *ConsoleService) ListTables(ctx context.Context, filter AdminTableFilter, page httpx.Page) ([]AdminTableView, int64, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Status != "" && !validAvailabilityStatus(filter.Status) {
		return nil, 0, apperr.Invalid("invalid status")
	}
	items, total, err := s.repo.ListAdminTables(ctx, filter, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]AdminTableView, 0, len(items))
	for _, item := range items {
		views = append(views, s.tableView(ctx, item))
	}
	return views, total, nil
}

func (s *ConsoleService) GetTable(ctx context.Context, id int64) (AdminTableView, error) {
	table, err := s.repo.GetAdminTable(ctx, id)
	if err != nil {
		return AdminTableView{}, err
	}
	return s.tableView(ctx, table), nil
}

func (s *ConsoleService) CreateTable(ctx context.Context, req TableWriteRequest) (AdminTableView, error) {
	table, err := s.validatedTable(ctx, req)
	if err != nil {
		return AdminTableView{}, err
	}
	created, err := s.repo.CreateAdminTable(ctx, table)
	if err != nil {
		return AdminTableView{}, err
	}
	return s.tableView(ctx, created), nil
}

func (s *ConsoleService) UpdateTable(ctx context.Context, id int64, req TableWriteRequest) (AdminTableView, error) {
	if id <= 0 {
		return AdminTableView{}, apperr.Invalid("invalid tableID")
	}
	table, err := s.validatedTable(ctx, req)
	if err != nil {
		return AdminTableView{}, err
	}
	current, err := s.repo.GetAdminTable(ctx, id)
	if err != nil {
		return AdminTableView{}, err
	}
	if req.Capacity < current.SeatCount {
		return AdminTableView{}, apperr.Conflict("capacity cannot be lower than existing seat count")
	}
	updated, err := s.repo.UpdateAdminTable(ctx, id, table)
	if err != nil {
		return AdminTableView{}, err
	}
	return s.tableView(ctx, updated), nil
}

func (s *ConsoleService) DeleteTable(ctx context.Context, id int64) error {
	if id <= 0 {
		return apperr.Invalid("invalid tableID")
	}
	return s.repo.DeleteAdminTable(ctx, id)
}

func (s *ConsoleService) StoreListTables(ctx context.Context, storeID int64, filter AdminTableFilter, page httpx.Page) ([]AdminTableView, int64, error) {
	filter.StoreID = &storeID
	return s.ListTables(ctx, filter, page)
}

func (s *ConsoleService) StoreGetTable(ctx context.Context, storeID, id int64) (AdminTableView, error) {
	view, err := s.GetTable(ctx, id)
	if err != nil {
		return AdminTableView{}, err
	}
	if view.StoreID != storeID {
		return AdminTableView{}, apperr.NotFound("table not found")
	}
	return view, nil
}

func (s *ConsoleService) StoreCreateTable(ctx context.Context, storeID int64, req TableWriteRequest) (AdminTableView, error) {
	req.StoreID = storeID
	return s.CreateTable(ctx, req)
}

func (s *ConsoleService) StoreUpdateTable(ctx context.Context, storeID, id int64, req TableWriteRequest) (AdminTableView, error) {
	if _, err := s.StoreGetTable(ctx, storeID, id); err != nil {
		return AdminTableView{}, err
	}
	req.StoreID = storeID
	return s.UpdateTable(ctx, id, req)
}

func (s *ConsoleService) StoreDeleteTable(ctx context.Context, storeID, id int64) error {
	if _, err := s.StoreGetTable(ctx, storeID, id); err != nil {
		return err
	}
	return s.DeleteTable(ctx, id)
}

func (s *ConsoleService) validatedTable(ctx context.Context, req TableWriteRequest) (Table, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	if req.StoreID <= 0 {
		return Table{}, apperr.Invalid("storeId is required")
	}
	if req.Name == "" {
		return Table{}, apperr.Invalid("name is required")
	}
	if req.Code == "" {
		return Table{}, apperr.Invalid("code is required")
	}
	if req.Capacity <= 0 {
		return Table{}, apperr.Invalid("capacity must be positive")
	}
	if req.BasePoints < 0 {
		return Table{}, apperr.Invalid("basePoints cannot be negative")
	}
	if req.LayoutAssetID != nil && *req.LayoutAssetID <= 0 {
		return Table{}, apperr.Invalid("layoutAssetId must be positive")
	}
	if !validAvailabilityStatus(req.Status) {
		return Table{}, apperr.Invalid("invalid status")
	}
	exists, err := s.repo.StoreExists(ctx, req.StoreID)
	if err != nil {
		return Table{}, err
	}
	if !exists {
		return Table{}, apperr.NotFound("store not found")
	}
	return Table{
		StoreID: req.StoreID, Name: req.Name, Code: req.Code,
		Capacity: req.Capacity, BasePoints: req.BasePoints,
		LayoutAssetID: req.LayoutAssetID, Status: req.Status,
	}, nil
}

func (s *ConsoleService) tableView(ctx context.Context, table Table) AdminTableView {
	view := adminTableView(table)
	if table.LayoutAssetID != nil && s.assets != nil {
		view.LayoutURL, _ = s.assets.PublicURLByID(ctx, *table.LayoutAssetID)
	}
	return view
}

func (s *ConsoleService) ListSeats(ctx context.Context, filter AdminSeatFilter, page httpx.Page) ([]AdminSeatView, int64, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Status != "" && !validAvailabilityStatus(filter.Status) {
		return nil, 0, apperr.Invalid("invalid status")
	}
	items, total, err := s.repo.ListAdminSeats(ctx, filter, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]AdminSeatView, 0, len(items))
	for _, item := range items {
		views = append(views, adminSeatView(item))
	}
	return views, total, nil
}

func (s *ConsoleService) GetSeat(ctx context.Context, id int64) (AdminSeatView, error) {
	seat, err := s.repo.GetAdminSeat(ctx, id)
	if err != nil {
		return AdminSeatView{}, err
	}
	return adminSeatView(seat), nil
}

func (s *ConsoleService) CreateSeat(ctx context.Context, req SeatWriteRequest) (AdminSeatView, error) {
	seat, err := validatedSeat(req)
	if err != nil {
		return AdminSeatView{}, err
	}
	created, err := s.repo.CreateAdminSeat(ctx, seat)
	if err != nil {
		return AdminSeatView{}, err
	}
	return adminSeatView(created), nil
}

func (s *ConsoleService) UpdateSeat(ctx context.Context, id int64, req SeatWriteRequest) (AdminSeatView, error) {
	if id <= 0 {
		return AdminSeatView{}, apperr.Invalid("invalid seatID")
	}
	seat, err := validatedSeat(req)
	if err != nil {
		return AdminSeatView{}, err
	}
	updated, err := s.repo.UpdateAdminSeat(ctx, id, seat)
	if err != nil {
		return AdminSeatView{}, err
	}
	return adminSeatView(updated), nil
}

func (s *ConsoleService) DeleteSeat(ctx context.Context, id int64) error {
	if id <= 0 {
		return apperr.Invalid("invalid seatID")
	}
	return s.repo.DeleteAdminSeat(ctx, id)
}

func (s *ConsoleService) StoreListSeats(ctx context.Context, storeID int64, filter AdminSeatFilter, page httpx.Page) ([]AdminSeatView, int64, error) {
	filter.StoreID = &storeID
	return s.ListSeats(ctx, filter, page)
}

func (s *ConsoleService) StoreGetSeat(ctx context.Context, storeID, id int64) (AdminSeatView, error) {
	view, err := s.GetSeat(ctx, id)
	if err != nil {
		return AdminSeatView{}, err
	}
	if view.StoreID != storeID {
		return AdminSeatView{}, apperr.NotFound("seat not found")
	}
	return view, nil
}

func (s *ConsoleService) storeTableForSeat(ctx context.Context, storeID, tableID int64) error {
	_, err := s.StoreGetTable(ctx, storeID, tableID)
	return err
}

func (s *ConsoleService) StoreCreateSeat(ctx context.Context, storeID int64, req SeatWriteRequest) (AdminSeatView, error) {
	if err := s.storeTableForSeat(ctx, storeID, req.TableID); err != nil {
		return AdminSeatView{}, err
	}
	return s.CreateSeat(ctx, req)
}

func (s *ConsoleService) StoreUpdateSeat(ctx context.Context, storeID, id int64, req SeatWriteRequest) (AdminSeatView, error) {
	if _, err := s.StoreGetSeat(ctx, storeID, id); err != nil {
		return AdminSeatView{}, err
	}
	if err := s.storeTableForSeat(ctx, storeID, req.TableID); err != nil {
		return AdminSeatView{}, err
	}
	return s.UpdateSeat(ctx, id, req)
}

func (s *ConsoleService) StoreDeleteSeat(ctx context.Context, storeID, id int64) error {
	if _, err := s.StoreGetSeat(ctx, storeID, id); err != nil {
		return err
	}
	return s.DeleteSeat(ctx, id)
}

func validatedSeat(req SeatWriteRequest) (Seat, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.TableID <= 0 {
		return Seat{}, apperr.Invalid("tableId is required")
	}
	if req.Name == "" {
		return Seat{}, apperr.Invalid("name is required")
	}
	if !validAvailabilityStatus(req.Status) {
		return Seat{}, apperr.Invalid("invalid status")
	}
	tableID := req.TableID
	return Seat{TableID: &tableID, Name: req.Name, Status: req.Status}, nil
}

func validAvailabilityStatus(status string) bool {
	switch status {
	case AvailabilityAvailable, AvailabilityPaused, AvailabilityFull, AvailabilityMaintenance:
		return true
	default:
		return false
	}
}
