package vipbenefit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	businessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)
	hoursPattern     = regexp.MustCompile(`^\s*(\d{1,2}):(\d{2})\s*[-–—~至]\s*(\d{1,2}):(\d{2})\s*$`)
)

func businessDayBounds(now time.Time) (time.Time, time.Time) {
	local := now.In(businessLocation)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, businessLocation)
	return start.UTC(), start.AddDate(0, 0, 1).UTC()
}

func periodKey(period string, now time.Time) (string, bool) {
	local := now.In(businessLocation)
	switch period {
	case "once":
		return "once", true
	case "daily":
		return local.Format("20060102"), true
	case "weekly":
		monday := local.AddDate(0, 0, -(int(local.Weekday())+6)%7)
		return monday.Format("20060102"), true
	case "monthly":
		return local.Format("200601"), true
	default:
		return "", false
	}
}

func couponExpiry(trigger string, now time.Time) time.Time {
	local := now.In(businessLocation)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, businessLocation)
	weekdayOffset := (int(local.Weekday()) + 6) % 7
	monday := dayStart.AddDate(0, 0, -weekdayOffset)
	switch trigger {
	case "weekday_event":
		return monday.AddDate(0, 0, 5).UTC() // Saturday 00:00; Monday-Friday only.
	case "weekly_event":
		return monday.AddDate(0, 0, 7).UTC() // Next Monday 00:00.
	case "monthly_event":
		return time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, businessLocation).UTC()
	default:
		return now.UTC().AddDate(0, 0, 30)
	}
}

func scheduledTrigger(trigger string) bool {
	switch trigger {
	case "period_start", "weekday_event", "weekly_event", "monthly_event":
		return true
	default:
		return false
	}
}

func scheduledTriggerActive(trigger string, now time.Time) bool {
	if trigger != "weekday_event" {
		return true
	}
	weekday := now.In(businessLocation).Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

func withinBusinessHours(value string, now time.Time) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	match := hoursPattern.FindStringSubmatch(value)
	if len(match) != 5 {
		return false
	}
	openMinute, ok := clockMinute(match[1], match[2])
	if !ok {
		return false
	}
	closeMinute, ok := clockMinute(match[3], match[4])
	if !ok {
		return false
	}
	local := now.In(businessLocation)
	minute := local.Hour()*60 + local.Minute()
	// Benefits never cross local midnight. A closing time at/before opening means
	// the store normally runs overnight, so only the opening-to-midnight segment
	// belongs to this business day.
	if closeMinute <= openMinute {
		return minute >= openMinute
	}
	return minute >= openMinute && minute < closeMinute
}

func clockMinute(hourValue, minuteValue string) (int, bool) {
	hour, err := strconv.Atoi(hourValue)
	if err != nil || hour < 0 || hour > 23 {
		return 0, false
	}
	minute, err := strconv.Atoi(minuteValue)
	if err != nil || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func benefitKey(kind string, memberID, tierID int64, period, trigger, key string, sequence int) string {
	return fmt.Sprintf("vipb:%s:%d:%d:%s:%s:%s:%d", kind, memberID, tierID, period, trigger, key, sequence)
}
