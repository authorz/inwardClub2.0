package store

import (
	"encoding/json"
	"time"

	"github.com/inwardclub/server/internal/platform/businesshours"
)

const businessStatusOverrideKey = "_businessStatusOverride"

const (
	BusinessStatusOpen   = "open"
	BusinessStatusClosed = "closed"
	BusinessStatusAuto   = "auto"

	BusinessStatusModeAuto   = "auto"
	BusinessStatusModeManual = "manual"
)

type businessStatusOverride struct {
	Status string    `json:"status"`
	Until  time.Time `json:"until"`
}

type effectiveBusinessStatus struct {
	Status        string
	Mode          string
	ScheduledOpen bool
	OverrideUntil *time.Time
}

func settingsMap(settings StoreSettings) map[string]any {
	values := map[string]any{}
	if len(settings.SettingsJSON) > 0 {
		_ = json.Unmarshal(settings.SettingsJSON, &values)
	}
	if values == nil {
		values = map[string]any{}
	}
	return values
}

func statusOverrideFromSettings(settings StoreSettings) (businessStatusOverride, bool) {
	values := settingsMap(settings)
	raw, ok := values[businessStatusOverrideKey]
	if !ok {
		return businessStatusOverride{}, false
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return businessStatusOverride{}, false
	}
	var override businessStatusOverride
	if err := json.Unmarshal(encoded, &override); err != nil {
		return businessStatusOverride{}, false
	}
	if override.Status != BusinessStatusOpen && override.Status != BusinessStatusClosed {
		return businessStatusOverride{}, false
	}
	return override, !override.Until.IsZero()
}

func evaluateBusinessStatus(st Store, settings StoreSettings, now time.Time) effectiveBusinessStatus {
	result := effectiveBusinessStatus{Status: BusinessStatusClosed, Mode: BusinessStatusModeAuto}
	if st.Status != StatusActive {
		return result
	}
	if schedule, err := businesshours.Parse(st.BusinessHours); err == nil {
		result.ScheduledOpen = schedule.IsOpen(now, businesshours.ShanghaiLocation())
		if result.ScheduledOpen {
			result.Status = BusinessStatusOpen
		}
	}
	if override, ok := statusOverrideFromSettings(settings); ok && now.Before(override.Until) {
		result.Status = override.Status
		result.Mode = BusinessStatusModeManual
		until := override.Until
		result.OverrideUntil = &until
	}
	return result
}
