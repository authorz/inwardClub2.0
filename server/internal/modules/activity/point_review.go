package activity

import (
	"fmt"
	"time"
)

const (
	defaultPointsDivisor     int64 = 5
	defaultCoinPointsDivisor int64 = 2000
	belowBasePointsDivisor   int64 = 2
)

var pointReviewLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type PointReviewRule struct {
	PointsDivisor     int64
	CoinPointsDivisor int64
	Version           int64
}

type pointReviewWindow struct {
	InBusiness bool
	Date       string
	Start      time.Time
	End        time.Time
}

type PointReviewCalculation struct {
	RequestedPoints int64
	BasePoints      int64
	ExcessPoints    int64
	AwardedPoints   int64
	CoinBasePoints  int64
	AwardedCoins    int64
	Window          pointReviewWindow
	Description     string
}

func businessWindow(now time.Time) pointReviewWindow {
	local := now.In(pointReviewLocation)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, pointReviewLocation)
	switch {
	case local.Hour() < 10:
		start := dayStart.Add(-24*time.Hour + 17*time.Hour)
		return pointReviewWindow{true, start.Format("2006-01-02"), start, dayStart.Add(10 * time.Hour)}
	case local.Hour() >= 17:
		start := dayStart.Add(17 * time.Hour)
		return pointReviewWindow{true, start.Format("2006-01-02"), start, dayStart.Add(34 * time.Hour)}
	default:
		return pointReviewWindow{Date: dayStart.Format("2006-01-02")}
	}
}

func calculatePointReview(now time.Time, requested, base int64, rule PointReviewRule) PointReviewCalculation {
	if rule.PointsDivisor <= 0 {
		rule.PointsDivisor = defaultPointsDivisor
	}
	if rule.CoinPointsDivisor <= 0 {
		rule.CoinPointsDivisor = defaultCoinPointsDivisor
	}
	calc := PointReviewCalculation{RequestedPoints: requested, Window: businessWindow(now)}
	if !calc.Window.InBusiness || base <= 0 {
		calc.AwardedPoints = requested / rule.PointsDivisor
		calc.CoinBasePoints = requested
		calc.AwardedCoins = requested / rule.CoinPointsDivisor
		if calc.Window.InBusiness {
			calc.Description = fmt.Sprintf(
				"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；当前营业时段内无基数积分，按标准规则计算",
				requested, rule.PointsDivisor, calc.AwardedPoints,
			)
		} else {
			calc.Description = fmt.Sprintf(
				"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；非营业积分时段按标准规则计算",
				requested, rule.PointsDivisor, calc.AwardedPoints,
			)
		}
		return calc
	}

	calc.BasePoints = base
	if requested < base {
		calc.AwardedPoints = requested / belowBasePointsDivisor
		calc.Description = fmt.Sprintf(
			"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；存入积分低于基数积分 %d，按 2:1 计算",
			requested, belowBasePointsDivisor, calc.AwardedPoints, base,
		)
		return calc
	}
	if requested == base {
		calc.AwardedPoints = requested
		calc.Description = fmt.Sprintf(
			"实际获得积分 = 存入积分 %d = %d；存入积分等于基数积分，按 1:1 计算",
			requested, calc.AwardedPoints,
		)
		return calc
	}

	calc.ExcessPoints = requested - base
	calc.AwardedPoints = base + calc.ExcessPoints/rule.PointsDivisor
	calc.CoinBasePoints = calc.ExcessPoints
	calc.AwardedCoins = calc.CoinBasePoints / rule.CoinPointsDivisor
	calc.Description = fmt.Sprintf(
		"实际获得积分 = 基数积分 %d +（存入积分 %d - 基数积分 %d）÷ %d（超出部分向下取整）= %d",
		base, requested, base, rule.PointsDivisor, calc.AwardedPoints,
	)
	return calc
}
