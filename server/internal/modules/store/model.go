// Package store owns store profiles. Phase-1 exposes the mini-program public
// read paths; console CRUD is layered on later.
package store

import "time"

// Store is a store profile.
type Store struct {
	ID                       int64
	Name                     string
	LogoAssetID              *int64
	Phone                    string
	CustomerServiceQRAssetID *int64
	Address                  string
	Latitude                 *float64
	Longitude                *float64
	BusinessHours            string
	Status                   string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// Store status values.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusDeleted  = "deleted"
)

// StoreSettings holds a store's persisted settings blob.
type StoreSettings struct {
	StoreID      int64
	SettingsJSON []byte
	UpdatedAt    time.Time
}
