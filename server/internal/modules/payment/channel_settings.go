package payment

import (
	"sort"
	"sync"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// ChannelSetting is an admin-configurable toggle for a payment channel. There
// is no persistent schema for this yet (see delivery notes): settings live in
// process memory and reset to the defaults below on restart.
type ChannelSetting struct {
	Channel     string `json:"channel"`
	DisplayName string `json:"displayName"`
	Enabled     bool   `json:"enabled"`
}

// UpdateChannelSettingsRequest is the admin console PUT payload: the full set
// of channel toggles to apply.
type UpdateChannelSettingsRequest struct {
	Channels []ChannelSetting `json:"channels"`
}

// channelSettingsStore is the in-memory settings holder, keyed by channel.
// Only the channels this server actually dispatches through (wechat, offline)
// are recognised; unknown channels are rejected.
type channelSettingsStore struct {
	mu       sync.Mutex
	settings map[string]ChannelSetting
}

func newChannelSettingsStore() *channelSettingsStore {
	return &channelSettingsStore{
		settings: map[string]ChannelSetting{
			"wechat":  {Channel: "wechat", DisplayName: "微信支付", Enabled: true},
			"offline": {Channel: "offline", DisplayName: "线下聚合收款", Enabled: true},
		},
	}
}

// List returns every channel setting, ordered by channel name.
func (s *channelSettingsStore) List() []ChannelSetting {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ChannelSetting, 0, len(s.settings))
	for _, v := range s.settings {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Channel < out[j].Channel })
	return out
}

// Update applies enabled/disabled toggles for known channels and returns the
// resulting full list. An unknown channel is rejected without applying any of
// the requested updates.
func (s *channelSettingsStore) Update(updates []ChannelSetting) ([]ChannelSetting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range updates {
		if _, ok := s.settings[u.Channel]; !ok {
			return nil, apperr.Invalid("unknown payment channel: " + u.Channel)
		}
	}
	for _, u := range updates {
		existing := s.settings[u.Channel]
		existing.Enabled = u.Enabled
		s.settings[u.Channel] = existing
	}
	out := make([]ChannelSetting, 0, len(s.settings))
	for _, v := range s.settings {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Channel < out[j].Channel })
	return out, nil
}
