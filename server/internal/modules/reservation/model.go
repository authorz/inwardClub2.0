// Package reservation owns table/seat reservations, the walk-in waitlist and
// member arrival records as three independent state machines. The mini program
// books/cancels reservations and joins the waitlist; the store console records
// a member's arrival against a booking. Table/seat availability is read from the
// store-owned tables/seats tables.
package reservation

import "time"

// Table is a bookable table in a store.
type Table struct {
	ID            int64
	StoreID       int64
	StoreName     string
	Name          string
	Code          string
	Capacity      int
	BasePoints    int
	LayoutAssetID *int64
	Status        string
	SeatCount     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Seat is a bookable seat, optionally attached to a table.
type Seat struct {
	ID              int64
	StoreID         int64
	StoreName       string
	TableID         *int64
	TableName       string
	Name            string
	Status          string
	MemberNickname  string
	MemberAvatarURL string
	MemberGender    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Reservation is a member's booking of a table or seat.
type Reservation struct {
	ID            int64
	ReservationNo string
	StoreID       int64
	MemberID      int64
	TableID       *int64
	SeatID        *int64
	PartySize     int
	ReservedAt    time.Time
	Status        string
	Remark        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WaitlistEntry is a walk-in party queued for a table.
type WaitlistEntry struct {
	ID        int64
	StoreID   int64
	MemberID  int64
	PartySize int
	Status    string
	QueuedAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Reservation status values (see 00009_reservation.sql).
const (
	StatusBooked    = "booked"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
	StatusArrived   = "arrived"
)

// Waitlist status values.
const (
	WaitlistWaiting = "waiting"
	WaitlistCalled  = "called"
	WaitlistSeated  = "seated"
	WaitlistLeft    = "left"
)

// Availability status shared by tables and seats.
const (
	AvailabilityAvailable   = "available"
	AvailabilityPaused      = "paused"
	AvailabilityFull        = "full"
	AvailabilityMaintenance = "maintenance"
	AvailabilityReserved    = "reserved"
)
