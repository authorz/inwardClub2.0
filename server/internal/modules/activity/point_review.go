package activity

import (
	"fmt"
	"time"
)

const (
	defaultPointsDivisor     int64 = 5
	defaultCoinPointsDivisor int64 = 2000
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
			calc.Description = fmt.Sprintf("用户存入积分：%d，实际存入积分：%d（无基数积分）", requested, calc.AwardedPoints)
		} else {
			calc.Description = fmt.Sprintf("用户存入积分：%d，实际存入积分：%d（标准规则）", requested, calc.AwardedPoints)
		}
		return calc
	}

	calc.BasePoints = base
	if requested <= base {
		calc.AwardedPoints = requested / rule.PointsDivisor
		calc.Description = fmt.Sprintf(
			"用户存入积分：%d，基数积分：%d，实际存入积分：%d（基数积分规则：存入≤基数）",
			requested, base, calc.AwardedPoints,
		)
		return calc
	}

	calc.ExcessPoints = requested - base
	calc.AwardedPoints = base + calc.ExcessPoints/rule.PointsDivisor
	calc.CoinBasePoints = calc.ExcessPoints
	calc.AwardedCoins = calc.CoinBasePoints / rule.CoinPointsDivisor
	calc.Description = fmt.Sprintf(
		"用户存入积分：%d，基数积分：%d，超出部分：%d，实际存入积分：%d（基数积分规则：存入>基数）",
		requested, base, calc.ExcessPoints, calc.AwardedPoints,
	)
	return calc
}
