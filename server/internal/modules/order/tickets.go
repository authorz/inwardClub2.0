package order

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// MemberTicket is one issued ticket joined with its activity, ticket type and
// store — the row backing the mini-program "my tickets" screen.
type MemberTicket struct {
	ID            int64
	ActivityID    int64
	Title         string
	ScopeType     string
	AssetID       *int64
	StartAt       *time.Time
	EndAt         *time.Time
	StoreName     string
	TicketName    string
	Status        string
	PaymentStatus string
	Code          string
}

// MyTicketView is the mini-program ticket representation consumed directly by
// pages/tickets/tickets.js.
type MyTicketView struct {
	ID         int64  `json:"id"`
	ActivityID int64  `json:"activityId"`
	Title      string `json:"title"`
	Tone       string `json:"tone"`
	ImageURL   string `json:"imageUrl"`
	TimeText   string `json:"timeText"`
	StoreName  string `json:"storeName"`
	TicketName string `json:"ticketName"`
	Qty        int    `json:"qty"`
	Status     string `json:"status"`
	Code       string `json:"code"`
}

func (r *sqlRepository) ListMemberTickets(ctx context.Context, memberID int64) ([]MemberTicket, error) {
	const q = `SELECT t.id, t.activity_id, a.title, a.scope_type, a.asset_id,
		a.start_at, a.end_at, COALESCE(s.name, ''), tt.name, t.status, bo.payment_status, t.verification_code
		FROM tickets t
		JOIN activity_orders ao ON ao.id = t.activity_order_id
		JOIN business_orders bo ON bo.id = ao.business_order_id
		JOIN activities a ON a.id = t.activity_id
		JOIN activity_ticket_types tt ON tt.id = t.ticket_type_id
		LEFT JOIN stores s ON s.id = t.store_id
		WHERE t.member_id = ?
		  AND bo.payment_status IN ('paid', 'partially_refunded', 'refunded')
		  AND t.status <> 'pending'
		ORDER BY t.id DESC`
	rows, err := r.db.QueryContext(ctx, q, memberID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var out []MemberTicket
	for rows.Next() {
		var t MemberTicket
		if err := rows.Scan(&t.ID, &t.ActivityID, &t.Title, &t.ScopeType, &t.AssetID,
			&t.StartAt, &t.EndAt, &t.StoreName, &t.TicketName, &t.Status, &t.PaymentStatus, &t.Code); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTickets returns the member's issued tickets as display views. The image
// URL is resolved from the activity asset; time text and tone are derived for
// the client.
func (s *Service) ListTickets(ctx context.Context, memberID int64) ([]MyTicketView, error) {
	tickets, err := s.repo.ListMemberTickets(ctx, memberID)
	if err != nil {
		return nil, err
	}
	views := make([]MyTicketView, 0, len(tickets))
	for _, t := range tickets {
		if t.Status == "pending" || (t.PaymentStatus != "paid" && t.PaymentStatus != "partially_refunded" && t.PaymentStatus != "refunded") {
			continue
		}
		v := MyTicketView{
			ID:         t.ID,
			ActivityID: t.ActivityID,
			Title:      t.Title,
			Tone:       ticketTone(t.ScopeType),
			TimeText:   ticketTimeText(t.StartAt),
			StoreName:  t.StoreName,
			TicketName: t.TicketName,
			Qty:        1,
			Status:     ticketStatus(t.Status),
			Code:       t.Code,
		}
		if t.AssetID != nil {
			v.ImageURL, _ = s.assets.PublicURLByID(ctx, *t.AssetID)
		}
		views = append(views, v)
	}
	return views, nil
}

// ticketTone maps the activity scope to a poster placeholder tone. Only the
// "member" tone is specially styled by the client; everything else falls back to
// the default dark poster.
func ticketTone(scopeType string) string {
	if scopeType == "global" {
		return "member"
	}
	return ""
}

// ticketTimeText renders the activity start time for display, empty when unset.
func ticketTimeText(startAt *time.Time) string {
	if startAt == nil {
		return ""
	}
	return startAt.Format("2006.01.02 15:04")
}

// ticketStatus maps a persisted ticket lifecycle status to the vocabulary the
// mini-program's ticket screen understands (unused/used/expired/refunded).
func ticketStatus(status string) string {
	switch status {
	case "active":
		return "unused"
	case "used", "expired", "refunded":
		return status
	default:
		return "expired"
	}
}

// ListTickets handles GET /mini/tickets.
func (h *Handler) ListTickets(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	views, err := h.svc.ListTickets(c.Request.Context(), memberID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}
