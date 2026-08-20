package activity

import (
	"encoding/json"
	"time"
)

// VerifyTicketRequest is the store-console ticket verification body.
type VerifyTicketRequest struct {
	Code string `json:"code"`
}

// TicketVerificationView is the result of verifying a ticket.
type TicketVerificationView struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticketId"`
	TicketNo   string    `json:"ticketNo,omitempty"`
	ActivityID int64     `json:"activityId,omitempty"`
	Status     string    `json:"status"`
	VerifiedBy int64     `json:"verifiedBy"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

// ReviewPointSavingRequest is the store-console point-saving review body.
type ReviewPointSavingRequest struct {
	Decision string `json:"decision"`
	Remark   string `json:"remark,omitempty"`
}

// PointSavingView is the public representation of a point-saving request. Field
// names align with the store-console client: memberName, phone, direction,
// points, storeName, status, createdAt, note.
type PointSavingView struct {
	ID                     int64           `json:"id"`
	MemberID               int64           `json:"memberId"`
	MemberName             string          `json:"memberName"`
	Phone                  string          `json:"phone"`
	MemberAvatarURL        string          `json:"memberAvatarUrl,omitempty"`
	Direction              string          `json:"direction"`
	Points                 int64           `json:"points"`
	BasePoints             int64           `json:"basePoints"`
	ExcessPoints           int64           `json:"excessPoints"`
	AwardedPoints          int64           `json:"awardedPoints"`
	CoinBasePoints         int64           `json:"coinBasePoints"`
	AwardedCoins           int64           `json:"awardedCoins"`
	RuleVersion            int64           `json:"ruleVersion"`
	PointsDivisor          int64           `json:"pointsDivisor"`
	BelowBasePointsDivisor int64           `json:"belowBasePointsDivisor"`
	CoinPointsDivisor      int64           `json:"coinPointsDivisor"`
	BusinessDate           *time.Time      `json:"businessDate,omitempty"`
	BusinessStartAt        *time.Time      `json:"businessStartAt,omitempty"`
	BusinessEndAt          *time.Time      `json:"businessEndAt,omitempty"`
	CalculationStartAt     *time.Time      `json:"calculationStartAt,omitempty"`
	CalculationEndAt       *time.Time      `json:"calculationEndAt,omitempty"`
	LastApprovedSavingID   *int64          `json:"lastApprovedSavingId,omitempty"`
	CalculationDescription string          `json:"calculationDescription,omitempty"`
	StoreName              string          `json:"storeName"`
	Status                 string          `json:"status"`
	Note                   string          `json:"note,omitempty"`
	ReviewedBy             *int64          `json:"reviewedBy,omitempty"`
	ReviewedByType         string          `json:"reviewedByType,omitempty"`
	Reviewer               json.RawMessage `json:"reviewer,omitempty"`
	ReviewedAt             *time.Time      `json:"reviewedAt,omitempty"`
	CreatedAt              time.Time       `json:"createdAt"`
}

// VerificationView is the public representation of a verification record. Every
// stored row is a completed redemption, so status/result are reported as
// "used"/"success"; at/createdAt/verifiedAt all carry the redemption time so the
// client can read whichever field it expects.
type VerificationView struct {
	ID            int64     `json:"id"`
	Kind          string    `json:"kind"`
	RefID         int64     `json:"refId"`
	Code          string    `json:"code,omitempty"`
	ActivityTitle string    `json:"activityTitle,omitempty"`
	MemberName    string    `json:"memberName,omitempty"`
	Status        string    `json:"status"`
	Result        string    `json:"result"`
	VerifiedBy    int64     `json:"verifiedBy"`
	VerifiedAt    time.Time `json:"verifiedAt"`
	At            time.Time `json:"at"`
	CreatedAt     time.Time `json:"createdAt"`
}

// TodayActivityView is the store-console "today" activity summary consumed by
// the staff mini-program: title, a display time range, the owning store name and
// the sold-vs-verified ticket counts (pendingVerify = active, verified = used).
type TodayActivityView struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	TimeText      string `json:"timeText"`
	StoreName     string `json:"storeName"`
	PendingVerify int64  `json:"pendingVerify"`
	Verified      int64  `json:"verified"`
}

// StoreRefView is a minimal store reference embedded in aggregate responses.
type StoreRefView struct {
	Name string `json:"name"`
}

// StaffHomeView is the store-console staff landing summary. TodayActivity is
// nil (JSON null) when the store has no activity running today, so the client
// can null-check it directly.
type StaffHomeView struct {
	Store              StoreRefView  `json:"store"`
	PendingReview      int64         `json:"pendingReview"`
	TodayVerifications int64         `json:"todayVerifications"`
	TodayActivity      *ActivityView `json:"todayActivity"`
}

// StaffOperationSummaryView carries integer asset quantities for the current
// business day. Coin and point values intentionally remain separate fields.
type StaffOperationSummaryView struct {
	CoinConsumptionAmount int64 `json:"coinConsumptionAmount"`
	CoinConsumptionCount  int64 `json:"coinConsumptionCount"`
	PointDepositAmount    int64 `json:"pointDepositAmount"`
	PointDepositCount     int64 `json:"pointDepositCount"`
	PointWithdrawalAmount int64 `json:"pointWithdrawalAmount"`
	PointWithdrawalCount  int64 `json:"pointWithdrawalCount"`
}

// StaffOperationView is one store-scoped coin consumption or point request.
type StaffOperationView struct {
	RecordKey       string    `json:"recordKey"`
	Type            string    `json:"type"`
	MemberID        int64     `json:"memberId"`
	MemberName      string    `json:"memberName"`
	Phone           string    `json:"phone"`
	Amount          int64     `json:"amount"`
	Status          string    `json:"status"`
	OrderType       string    `json:"orderType,omitempty"`
	BusinessOrderNo string    `json:"businessOrderNo,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// StaffTodayOperationsView powers the staff mini-program's current-day ledger.
type StaffTodayOperationsView struct {
	Date    string                    `json:"date"`
	Summary StaffOperationSummaryView `json:"summary"`
	Entries []StaffOperationView      `json:"entries"`
}

func ticketVerificationView(v TicketVerification) TicketVerificationView {
	return TicketVerificationView{
		ID: v.ID, TicketID: v.TicketID, TicketNo: v.TicketNo, ActivityID: v.ActivityID,
		Status: v.Status, VerifiedBy: v.VerifiedBy, VerifiedAt: v.VerifiedAt,
	}
}

func pointSavingView(p PointSaving) PointSavingView {
	return PointSavingView{
		ID: p.ID, MemberID: p.MemberID, MemberName: p.MemberName, Phone: p.Phone,
		MemberAvatarURL: p.MemberAvatarURL,
		Direction:       PointSavingDirection, Points: p.Points, StoreName: p.StoreName,
		BasePoints: p.BasePoints, ExcessPoints: p.ExcessPoints, AwardedPoints: p.AwardedPoints,
		CoinBasePoints: p.CoinBasePoints, AwardedCoins: p.AwardedCoins,
		RuleVersion: p.RuleVersion, PointsDivisor: p.PointsDivisor,
		BelowBasePointsDivisor: p.BelowBasePointsDivisor,
		CoinPointsDivisor:      p.CoinPointsDivisor, BusinessDate: p.BusinessDate,
		BusinessStartAt: p.BusinessStartAt, BusinessEndAt: p.BusinessEndAt,
		CalculationStartAt: p.CalculationStartAt, CalculationEndAt: p.CalculationEndAt,
		LastApprovedSavingID:   p.LastApprovedSavingID,
		CalculationDescription: p.CalculationDescription, Status: p.Status, Note: p.Remark,
		ReviewedBy: p.ReviewedBy, ReviewedByType: p.ReviewedByType,
		Reviewer: json.RawMessage(p.ReviewerSnapshotJSON), ReviewedAt: p.ReviewedAt, CreatedAt: p.CreatedAt,
	}
}

func todayActivityView(a TodayActivity) TodayActivityView {
	return TodayActivityView{
		ID: a.ID, Title: a.Title, TimeText: todayTimeText(a.StartAt, a.EndAt),
		StoreName: a.StoreName, PendingVerify: a.PendingVerify, Verified: a.Verified,
	}
}

// todayTimeText renders an activity's start/end window as a "HH:MM-HH:MM"
// display string. Missing bounds degrade gracefully; no bounds at all yields "全天".
func todayTimeText(start, end *time.Time) string {
	switch {
	case start != nil && end != nil:
		return start.Format("15:04") + "-" + end.Format("15:04")
	case start != nil:
		return start.Format("15:04") + "起"
	case end != nil:
		return "至" + end.Format("15:04")
	default:
		return "全天"
	}
}

func verificationView(v Verification) VerificationView {
	return VerificationView{
		ID: v.ID, Kind: v.Kind, RefID: v.RefID, Code: v.Code, ActivityTitle: v.ActivityTitle,
		MemberName: v.MemberName, Status: verificationStatusUsed, Result: verificationResultSuccess,
		VerifiedBy: v.VerifiedBy, VerifiedAt: v.VerifiedAt, At: v.VerifiedAt, CreatedAt: v.VerifiedAt,
	}
}

func staffOperationView(entry StaffOperation) StaffOperationView {
	return StaffOperationView{
		RecordKey: entry.RecordKey, Type: entry.Type, MemberID: entry.MemberID,
		MemberName: entry.MemberName, Phone: entry.Phone, Amount: entry.Amount,
		Status: entry.Status, OrderType: entry.OrderType,
		BusinessOrderNo: entry.BusinessOrderNo, CreatedAt: entry.CreatedAt,
	}
}
