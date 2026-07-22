// Package store owns store profiles, banners, tables and seats. Phase-1 exposes
// the mini-program public read paths; console CRUD is layered on later.
package store

import "time"

// Store is a store profile.
type Store struct {
	ID            int64
	Name          string
	LogoAssetID   *int64
	Phone         string
	Address       string
	Latitude      *float64
	Longitude     *float64
	BusinessHours string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Banner is a promotional banner scoped globally or to a store.
type Banner struct {
	ID        int64
	ScopeType string
	StoreID   *int64
	Title     string
	AssetID   int64
	LinkURL   string
	SortOrder int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store status values.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// StoreSettings holds a store's persisted settings blob.
type StoreSettings struct {
	StoreID      int64
	SettingsJSON []byte
	UpdatedAt    time.Time
}

// Banner scope types.
const (
	BannerScopeGlobal = "global"
	BannerScopeStore  = "store"
)
