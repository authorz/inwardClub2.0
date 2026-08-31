package activity

import (
	"fmt"
	"time"

	"github.com/inwardclub/server/internal/platform/businesshours"
)

const (
	defaultPointsDivisor          int64 = 5
	defaultBelowBasePointsDivisor int64 = 2
	defaultCoinPointsDivisor      int64 = 2000
)

var pointReviewLocation = businesshours.ShanghaiLocation()

type PointReviewRule struct {
	PointsDivisor          int64
	BelowBasePointsDivisor int64
	CoinPointsDivisor      int64
	Version                int64
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

func businessWindow(now time.Time, configuredHours string) (pointReviewWindow, error) {
	schedule, err := businesshours.Parse(configuredHours)
	if err != nil {
		return pointReviewWindow{}, fmt.Errorf("门店营业时间配置无效：%w", err)
	}
	local := now.In(pointReviewLocation)
	start, end, open := schedule.CurrentWindow(local, pointReviewLocation)
	if !open {
		return pointReviewWindow{Date: local.Format("2006-01-02")}, nil
	}
	return pointReviewWindow{InBusiness: true, Date: start.Format("2006-01-02"), Start: start, End: end}, nil
}

func calculatePointReview(window pointReviewWindow, requested, base int64, rule PointReviewRule) PointReviewCalculation {
	if rule.PointsDivisor <= 0 {
		rule.PointsDivisor = defaultPointsDivisor
	}
	if rule.BelowBasePointsDivisor <= 0 {
		rule.BelowBasePointsDivisor = defaultBelowBasePointsDivisor
	}
	if rule.CoinPointsDivisor <= 0 {
		rule.CoinPointsDivisor = defaultCoinPointsDivisor
	}
	calc := PointReviewCalculation{RequestedPoints: requested, Window: window}
	var coinDescription string
	if base <= 0 {
		calc.CoinBasePoints = requested
		calc.AwardedCoins = calc.CoinBasePoints / rule.CoinPointsDivisor
		coinDescription = fmt.Sprintf(
			"奖励金币 = 存入积分 %d ÷ %d（向下取整）= %d",
			calc.CoinBasePoints, rule.CoinPointsDivisor, calc.AwardedCoins,
		)
	} else {
		calc.BasePoints = base
		if requested > base {
			calc.ExcessPoints = requested - base
			calc.CoinBasePoints = calc.ExcessPoints
			calc.AwardedCoins = calc.CoinBasePoints / rule.CoinPointsDivisor
			coinDescription = fmt.Sprintf(
				"奖励金币 = 盈利积分 %d ÷ %d（向下取整）= %d",
				calc.CoinBasePoints, rule.CoinPointsDivisor, calc.AwardedCoins,
			)
		} else {
			coinDescription = "奖励金币 = 0；存入积分未超过基数积分"
		}
	}
	if !calc.Window.InBusiness || base <= 0 {
		calc.AwardedPoints = requested / rule.PointsDivisor
		if calc.Window.InBusiness {
			calc.Description = fmt.Sprintf(
				"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；当前营业时段内无基数积分，按标准规则计算；%s",
				requested, rule.PointsDivisor, calc.AwardedPoints, coinDescription,
			)
		} else {
			calc.Description = fmt.Sprintf(
				"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；非营业积分时段按标准规则计算；%s",
				requested, rule.PointsDivisor, calc.AwardedPoints, coinDescription,
			)
		}
		return calc
	}

	if requested < base {
		calc.AwardedPoints = requested / rule.BelowBasePointsDivisor
		calc.Description = fmt.Sprintf(
			"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；存入积分低于基数积分 %d；%s",
			requested, rule.BelowBasePointsDivisor, calc.AwardedPoints, base, coinDescription,
		)
		return calc
	}
	if requested == base {
		calc.AwardedPoints = requested
		calc.Description = fmt.Sprintf(
			"实际获得积分 = 存入积分 %d = %d；存入积分等于基数积分，按 1:1 计算；%s",
			requested, calc.AwardedPoints, coinDescription,
		)
		return calc
	}

	calc.AwardedPoints = base + calc.ExcessPoints/rule.PointsDivisor
	calc.Description = fmt.Sprintf(
		"实际获得积分 = 基数积分 %d +（存入积分 %d - 基数积分 %d）÷ %d（超出部分向下取整）= %d；%s",
		base, requested, base, rule.PointsDivisor, calc.AwardedPoints, coinDescription,
	)
	return calc
}
