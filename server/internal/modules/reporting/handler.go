package reporting

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// Handler exposes the console analytics read endpoints. Router wiring lives
// outside this module.
type Handler struct {
	svc *Service
}

// NewHandler builds the reporting handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Overview handles GET /admin/reports/overview.
//
// Under the admin audience the metrics aggregate across every store. If the
// request carries a pinned store scope (store console), the scope narrows the
// aggregation; the scope is taken from the JWT, never a query parameter.
func (h *Handler) Overview(c *gin.Context) {
	var f OverviewFilter
	if scope, ok := storescope.FromContext(c); ok {
		f.StoreID = &scope
	}
	view, err := h.svc.GetOverview(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// reportFilter builds a ReportFilter from the request: page from ?page/?pageSize,
// optional ?from/?to (RFC3339 date or datetime, ignored if unparseable), and the
// pinned store scope from the JWT (never a query parameter).
func reportFilter(c *gin.Context) ReportFilter {
	f := ReportFilter{Page: httpx.ParsePage(c)}
	if scope, ok := storescope.FromContext(c); ok {
		f.StoreID = &scope
	}
	if from := parseDateParam(c.Query("from")); from != nil {
		f.From = from
	}
	if to := parseDateParam(c.Query("to")); to != nil {
		f.To = to
	}
	return f
}

func parseDateParam(v string) *time.Time {
	if v == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return &t
	}
	return nil
}

// Revenue handles GET /admin/reports/revenue.
func (h *Handler) Revenue(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetRevenue(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// CatalogItems handles GET /admin/reports/catalog-items.
func (h *Handler) CatalogItems(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetCatalogItems(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Activities handles GET /admin/reports/activities.
func (h *Handler) Activities(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetActivities(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Coupons handles GET /admin/reports/coupons.
func (h *Handler) Coupons(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetCoupons(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Records handles GET /admin/reports/records.
func (h *Handler) Records(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetRecords(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Members handles GET /admin/reports/members.
func (h *Handler) Members(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetMembers(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Reservations handles GET /admin/reports/reservations.
func (h *Handler) Reservations(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetReservations(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Stores handles GET /admin/reports/stores.
func (h *Handler) Stores(c *gin.Context) {
	f := reportFilter(c)
	views, total, err := h.svc.GetStores(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}
