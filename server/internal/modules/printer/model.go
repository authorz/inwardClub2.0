package printer

import "time"

// Device statuses and providers. Phase-1 only manages device records; the
// FakePrinter is the execution end, so no real provider credentials are used.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	ProviderXpyun = "xpyun"
)

// Device is a cloud printer registered to a store (row of printer_devices).
type Device struct {
	ID        int64
	StoreID   int64
	Name      string
	Provider  string
	DeviceSN  string
	DeviceKey string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeviceInput is the create payload for a printer device.
type DeviceInput struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	DeviceSN  string `json:"deviceSn"`
	DeviceKey string `json:"deviceKey"`
	Status    string `json:"status"`
}

// AdminDeviceInput is the headquarters create payload. Unlike store-console
// creation, the headquarters operator must explicitly select the owning store
// and provide an audit reason for the cross-store write.
type AdminDeviceInput struct {
	StoreID   int64  `json:"storeId"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	DeviceSN  string `json:"deviceSn"`
	DeviceKey string `json:"deviceKey"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

// DevicePatch is the partial-update payload; nil fields are left unchanged.
type DevicePatch struct {
	Name      *string `json:"name"`
	DeviceSN  *string `json:"deviceSn"`
	DeviceKey *string `json:"deviceKey"`
	Status    *string `json:"status"`
}

// AdminDevicePatch is the headquarters update payload. Store ownership is
// immutable after creation; changing it requires deleting and recreating the
// device so the audit trail remains unambiguous.
type AdminDevicePatch struct {
	DevicePatch
	Reason string `json:"reason"`
}

// AdminDeleteInput carries the mandatory audit reason for a headquarters
// deletion.
type AdminDeleteInput struct {
	Reason string `json:"reason"`
}

// DeviceView is the JSON shape returned by the console endpoints. The device
// key is intentionally omitted from reads.
type DeviceView struct {
	ID        int64     `json:"id"`
	StoreID   int64     `json:"storeId"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	DeviceSN  string    `json:"deviceSn"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (d Device) view() DeviceView {
	return DeviceView{
		ID:        d.ID,
		StoreID:   d.StoreID,
		Name:      d.Name,
		Provider:  d.Provider,
		DeviceSN:  d.DeviceSN,
		Status:    d.Status,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
