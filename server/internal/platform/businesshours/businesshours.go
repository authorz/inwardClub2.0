// Package businesshours parses a store's configured daily business interval
// and evaluates it in the store's business timezone.
package businesshours

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var intervalPattern = regexp.MustCompile(`^\s*(\d{1,2}):(\d{2})\s*[-–—]\s*(\d{1,2}):(\d{2})\s*$`)

var shanghaiLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// ShanghaiLocation is the business timezone used by all stores.
func ShanghaiLocation() *time.Location { return shanghaiLocation }

// Schedule is one recurring daily opening interval. When CloseMinute is less
// than OpenMinute, the interval crosses midnight into the following day.
type Schedule struct {
	OpenMinute  int
	CloseMinute int
}

// Parse validates a single HH:MM-HH:MM interval.
func Parse(value string) (Schedule, error) {
	matches := intervalPattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return Schedule{}, fmt.Errorf("营业时间格式应为 HH:MM-HH:MM")
	}
	values := make([]int, 4)
	for i := range values {
		parsed, err := strconv.Atoi(matches[i+1])
		if err != nil {
			return Schedule{}, fmt.Errorf("营业时间格式无效")
		}
		values[i] = parsed
	}
	if values[0] > 23 || values[2] > 23 || values[1] > 59 || values[3] > 59 {
		return Schedule{}, fmt.Errorf("营业时间超出有效范围")
	}
	openMinute := values[0]*60 + values[1]
	closeMinute := values[2]*60 + values[3]
	if openMinute == closeMinute {
		return Schedule{}, fmt.Errorf("营业开始时间与结束时间不能相同")
	}
	return Schedule{OpenMinute: openMinute, CloseMinute: closeMinute}, nil
}

// CurrentWindow returns the active business interval containing now.
func (s Schedule) CurrentWindow(now time.Time, location *time.Location) (time.Time, time.Time, bool) {
	local := now.In(location)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	minute := local.Hour()*60 + local.Minute()
	if s.OpenMinute < s.CloseMinute {
		if minute < s.OpenMinute || minute >= s.CloseMinute {
			return time.Time{}, time.Time{}, false
		}
		return day.Add(time.Duration(s.OpenMinute) * time.Minute), day.Add(time.Duration(s.CloseMinute) * time.Minute), true
	}
	if minute >= s.OpenMinute {
		return day.Add(time.Duration(s.OpenMinute) * time.Minute), day.Add(24*time.Hour + time.Duration(s.CloseMinute)*time.Minute), true
	}
	if minute < s.CloseMinute {
		return day.Add(-24*time.Hour + time.Duration(s.OpenMinute)*time.Minute), day.Add(time.Duration(s.CloseMinute) * time.Minute), true
	}
	return time.Time{}, time.Time{}, false
}

// IsOpen reports whether now falls inside the configured interval.
func (s Schedule) IsOpen(now time.Time, location *time.Location) bool {
	_, _, open := s.CurrentWindow(now, location)
	return open
}

// NextBoundary returns the next opening or closing boundary strictly after now.
func (s Schedule) NextBoundary(now time.Time, location *time.Location) time.Time {
	local := now.In(location)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	candidates := []time.Time{
		day.Add(time.Duration(s.OpenMinute) * time.Minute),
		day.Add(time.Duration(s.CloseMinute) * time.Minute),
		day.Add(24*time.Hour + time.Duration(s.OpenMinute)*time.Minute),
		day.Add(24*time.Hour + time.Duration(s.CloseMinute)*time.Minute),
	}
	var next time.Time
	for _, candidate := range candidates {
		if candidate.After(local) && (next.IsZero() || candidate.Before(next)) {
			next = candidate
		}
	}
	return next
}
