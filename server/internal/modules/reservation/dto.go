package reservation

import "time"

// TableView is the public table representation.
type TableView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Capacity  int    `json:"capacity"`
	LayoutURL string `json:"layoutUrl,omitempty"`
	Status    string `json:"status"`
}

// SeatView is the public seat representation.
type SeatView struct {
	ID        int64  `json:"id"`
	TableID   *int64 `json:"tableId,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Gender    string `json:"gender,omitempty"`
}

// ReservationView is the member-facing reservation representation.
type ReservationView struct {
	ID              int64     `json:"id"`
	ReservationNo   string    `json:"reservationNo"`
	StoreID         int64     `json:"storeId"`
	TableID         *int64    `json:"tableId,omitempty"`
	SeatID          *int64    `json:"seatId,omitempty"`
	PartySize       int       `json:"partySize"`
	ReservedAt      time.Time `json:"reservedAt"`
	Status          string    `json:"status"`
	Remark          string    `json:"remark,omitempty"`
	MemberNickname  string    `json:"memberNickname,omitempty"`
	MemberPhone     string    `json:"memberPhone,omitempty"`
	MemberAvatarURL string    `json:"memberAvatarUrl,omitempty"`
	TableNo         string    `json:"tableNo,omitempty"`
	SeatNo          string    `json:"seatNo,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// CreateReservationRequest is the mini-program booking payload.
type CreateReservationRequest struct {
	StoreID    int64     `json:"storeId"`
	TableID    *int64    `json:"tableId"`
	SeatID     *int64    `json:"seatId"`
	PartySize  int       `json:"partySize"`
	ReservedAt time.Time `json:"reservedAt"`
	Remark     string    `json:"remark"`
}

// WaitlistEntryView is the member-facing waitlist representation.
type WaitlistEntryView struct {
	ID        int64     `json:"id"`
	StoreID   int64     `json:"storeId"`
	PartySize int       `json:"partySize"`
	Status    string    `json:"status"`
	QueuedAt  time.Time `json:"queuedAt"`
}

// CreateWaitlistRequest is the mini-program walk-in queue payload.
type CreateWaitlistRequest struct {
	StoreID   int64 `json:"storeId"`
	PartySize int   `json:"partySize"`
}

func tableView(t Table) TableView {
	return TableView{ID: t.ID, Name: t.Name, Capacity: t.Capacity, Status: t.Status}
}

func seatView(s Seat) SeatView {
	return SeatView{
		ID: s.ID, TableID: s.TableID, Name: s.Name, Status: s.Status,
		Nickname: s.MemberNickname, AvatarURL: s.MemberAvatarURL, Gender: s.MemberGender,
	}
}

func reservationView(r Reservation) ReservationView {
	return ReservationView{
		ID:              r.ID,
		ReservationNo:   r.ReservationNo,
		StoreID:         r.StoreID,
		TableID:         r.TableID,
		SeatID:          r.SeatID,
		PartySize:       r.PartySize,
		ReservedAt:      r.ReservedAt,
		Status:          r.Status,
		Remark:          r.Remark,
		MemberNickname:  r.MemberNickname,
		MemberPhone:     r.MemberPhone,
		MemberAvatarURL: r.MemberAvatarURL,
		TableNo:         r.TableNo,
		SeatNo:          r.SeatNo,
		CreatedAt:       r.CreatedAt,
	}
}

func waitlistView(w WaitlistEntry) WaitlistEntryView {
	return WaitlistEntryView{
		ID:        w.ID,
		StoreID:   w.StoreID,
		PartySize: w.PartySize,
		Status:    w.Status,
		QueuedAt:  w.QueuedAt,
	}
}
