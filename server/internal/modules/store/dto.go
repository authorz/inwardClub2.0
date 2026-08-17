package store

import "time"

// StoreView is the public store representation.
type StoreView struct {
	ID                       int64    `json:"id"`
	Name                     string   `json:"name"`
	LogoURL                  string   `json:"logoUrl,omitempty"`
	Phone                    string   `json:"phone,omitempty"`
	CustomerServiceQRAssetID *int64   `json:"-"`
	CustomerServiceQRURL     string   `json:"customerServiceQrUrl,omitempty"`
	Address                  string   `json:"address"`
	Latitude                 *float64 `json:"latitude,omitempty"`
	Longitude                *float64 `json:"longitude,omitempty"`
	BusinessHours            string   `json:"businessHours,omitempty"`
	Status                   string   `json:"status"`
	DistanceMeters           *int64   `json:"distanceMeters,omitempty"`
}

// ConsoleProfileView is the store console's own-store profile representation.
type ConsoleProfileView struct {
	ID                       int64    `json:"id"`
	Name                     string   `json:"name"`
	LogoURL                  string   `json:"logoUrl,omitempty"`
	Phone                    string   `json:"phone,omitempty"`
	CustomerServiceQRAssetID *int64   `json:"customerServiceQrAssetId,omitempty"`
	CustomerServiceQRURL     string   `json:"customerServiceQrUrl,omitempty"`
	Address                  string   `json:"address"`
	Latitude                 *float64 `json:"latitude,omitempty"`
	Longitude                *float64 `json:"longitude,omitempty"`
	BusinessHours            string   `json:"businessHours,omitempty"`
	Status                   string   `json:"status"`
}

// UpdateProfileRequest is a full-replace update of the editable store profile
// fields, submitted by the store console.
type UpdateProfileRequest struct {
	Name                     string   `json:"name" binding:"required"`
	Phone                    string   `json:"phone"`
	CustomerServiceQRAssetID *int64   `json:"customerServiceQrAssetId"`
	Address                  string   `json:"address" binding:"required"`
	BusinessHours            string   `json:"businessHours"`
	Latitude                 *float64 `json:"latitude"`
	Longitude                *float64 `json:"longitude"`
	LogoAssetID              *int64   `json:"logoAssetId"`
}

// StoreStatusPatch updates the caller's own store's active/inactive status.
type StoreStatusPatch struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// StoreSettingsView is the store console's settings representation. Settings
// are an opaque, store-defined JSON object; the schema is not interpreted by
// the server.
type StoreSettingsView struct {
	Settings  map[string]any `json:"settings"`
	UpdatedAt *time.Time     `json:"updatedAt,omitempty"`
}

// UpdateSettingsRequest is a full-replace update of the store's settings blob.
type UpdateSettingsRequest struct {
	Settings map[string]any `json:"settings"`
}

// StoreInput is the admin-side store create/update contract.
type StoreInput struct {
	Name                     string   `json:"name" binding:"required"`
	Phone                    string   `json:"phone"`
	CustomerServiceQRAssetID *int64   `json:"customerServiceQrAssetId"`
	Address                  string   `json:"address" binding:"required"`
	BusinessHours            string   `json:"businessHours"`
	Latitude                 *float64 `json:"latitude"`
	Longitude                *float64 `json:"longitude"`
	LogoAssetID              *int64   `json:"logoAssetId"`
}

// DeleteStoreRequest re-authenticates the current headquarters administrator
// before a store is soft-deleted.
type DeleteStoreRequest struct {
	Password string `json:"password"`
}
