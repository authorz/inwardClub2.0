package store

import "time"

// StoreView is the public store representation.
type StoreView struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	LogoURL        string   `json:"logoUrl,omitempty"`
	Phone          string   `json:"phone,omitempty"`
	Address        string   `json:"address"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	BusinessHours  string   `json:"businessHours,omitempty"`
	Status         string   `json:"status"`
	DistanceMeters *int64   `json:"distanceMeters,omitempty"`
}

// ConsoleProfileView is the store console's own-store profile representation.
type ConsoleProfileView struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	LogoURL       string   `json:"logoUrl,omitempty"`
	Phone         string   `json:"phone,omitempty"`
	Address       string   `json:"address"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	BusinessHours string   `json:"businessHours,omitempty"`
	Status        string   `json:"status"`
}

// UpdateProfileRequest is a full-replace update of the editable store profile
// fields, submitted by the store console.
type UpdateProfileRequest struct {
	Name          string   `json:"name" binding:"required"`
	Phone         string   `json:"phone"`
	Address       string   `json:"address" binding:"required"`
	BusinessHours string   `json:"businessHours"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	LogoAssetID   *int64   `json:"logoAssetId"`
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
	Name          string   `json:"name" binding:"required"`
	Phone         string   `json:"phone"`
	Address       string   `json:"address" binding:"required"`
	BusinessHours string   `json:"businessHours"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	LogoAssetID   *int64   `json:"logoAssetId"`
}

// BannerView is the public banner representation.
type BannerView struct {
	ID        int64  `json:"id"`
	Title     string `json:"title,omitempty"`
	ImageURL  string `json:"imageUrl"`
	LinkURL   string `json:"linkUrl,omitempty"`
	SortOrder int    `json:"sortOrder"`
}

// BannerAdminView is the console banner representation (admin and store).
type BannerAdminView struct {
	ID        int64     `json:"id"`
	ScopeType string    `json:"scopeType"`
	StoreID   *int64    `json:"storeId,omitempty"`
	Title     string    `json:"title,omitempty"`
	AssetID   int64     `json:"assetId"`
	ImageURL  string    `json:"imageUrl,omitempty"`
	LinkURL   string    `json:"linkUrl,omitempty"`
	SortOrder int       `json:"sortOrder"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BannerInput is the console banner create contract. On the store side
// scopeType/storeId are ignored and forced to the caller's own store scope.
type BannerInput struct {
	ScopeType string `json:"scopeType"`
	StoreID   *int64 `json:"storeId"`
	Title     string `json:"title"`
	AssetID   int64  `json:"assetId" binding:"required"`
	LinkURL   string `json:"linkUrl"`
	SortOrder int    `json:"sortOrder"`
	Status    string `json:"status"`
}

// BannerPatch is a partial update of an existing banner. Only non-nil fields
// are applied. Scope changes are admin-only.
type BannerPatch struct {
	ScopeType *string `json:"scopeType"`
	StoreID   *int64  `json:"storeId"`
	Title     *string `json:"title"`
	AssetID   *int64  `json:"assetId"`
	LinkURL   *string `json:"linkUrl"`
	SortOrder *int    `json:"sortOrder"`
	Status    *string `json:"status"`
}
