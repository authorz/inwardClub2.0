package reservation

import "time"

// AdminTableView is the headquarters table-management representation.
type AdminTableView struct {
	ID            int64     `json:"id"`
	StoreID       int64     `json:"storeId"`
	StoreName     string    `json:"storeName"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Capacity      int       `json:"capacity"`
	SeatCount     int       `json:"seatCount"`
	BasePoints    int       `json:"basePoints"`
	LayoutAssetID *int64    `json:"layoutAssetId,omitempty"`
	LayoutURL     string    `json:"layoutUrl,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// AdminSeatView is the headquarters seat-management representation.
type AdminSeatView struct {
	ID        int64     `json:"id"`
	StoreID   int64     `json:"storeId"`
	StoreName string    `json:"storeName"`
	TableID   int64     `json:"tableId"`
	TableName string    `json:"tableName"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TableWriteRequest struct {
	StoreID       int64  `json:"storeId"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	Capacity      int    `json:"capacity"`
	BasePoints    int    `json:"basePoints"`
	LayoutAssetID *int64 `json:"layoutAssetId"`
	Status        string `json:"status"`
}

type SeatWriteRequest struct {
	TableID int64  `json:"tableId"`
	Name    string `json:"name"`
	Status  string `json:"status"`
}

type AdminTableFilter struct {
	StoreID *int64
	Status  string
	Keyword string
}

type AdminSeatFilter struct {
	StoreID *int64
	TableID *int64
	Status  string
	Keyword string
}

func adminTableView(t Table) AdminTableView {
	return AdminTableView{
		ID: t.ID, StoreID: t.StoreID, StoreName: t.StoreName, Name: t.Name,
		Code: t.Code, Capacity: t.Capacity, SeatCount: t.SeatCount,
		BasePoints: t.BasePoints, LayoutAssetID: t.LayoutAssetID,
		Status: t.Status, CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func adminSeatView(s Seat) AdminSeatView {
	var tableID int64
	if s.TableID != nil {
		tableID = *s.TableID
	}
	return AdminSeatView{
		ID: s.ID, StoreID: s.StoreID, StoreName: s.StoreName, TableID: tableID,
		TableName: s.TableName, Name: s.Name, Status: s.Status,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
