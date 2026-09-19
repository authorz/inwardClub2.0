package activity

import (
	"fmt"
	"time"

	"github.com/inwardclub/server/internal/platform/businesshours"
)

const (
	defaultPointsDivisor          int64 = 5
	defaultBelowBasePointsDivisor int64 = 5
)

var pointReviewLocation = businesshours.ShanghaiLocation()

type PointReviewRule struct {
	PointsDivisor          int64
	BelowBasePointsDivisor int64
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
	// base is retained in the function signature for compatibility with the
	// historical calculation snapshots, but it is intentionally ignored. All
	// new point-saving reviews use one uniform divisor regardless of business
	// hours or the member's withdrawal base.
	_ = base
	calc := PointReviewCalculation{RequestedPoints: requested, Window: window}
	calc.AwardedPoints = requested / rule.PointsDivisor
	calc.Description = fmt.Sprintf(
		"实际获得积分 = 存入积分 %d ÷ %d（向下取整）= %d；统一按折算比例计算，不考虑基础积分",
		requested, rule.PointsDivisor, calc.AwardedPoints,
	)
	return calc
}
