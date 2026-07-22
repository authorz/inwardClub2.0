package asset

// UploadCredentialRequest is the client request for a signed upload token.
type UploadCredentialRequest struct {
	Purpose     string `json:"purpose" binding:"required"`
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	SizeBytes   int64  `json:"sizeBytes" binding:"required"`
	Visibility  string `json:"visibility"`
}

// AssetView is the public representation of an asset returned to clients.
type AssetView struct {
	AssetID int64  `json:"assetId"`
	URL     string `json:"url"`
	Width   *int   `json:"width,omitempty"`
	Height  *int   `json:"height,omitempty"`
	Status  string `json:"status"`
}
