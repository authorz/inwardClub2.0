package reporting

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

var reportLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// Handler exposes the console analytics read endpoints. Router wiring lives
// outside this module.
type Handler struct {
	svc *Service
}

// NewHandler builds the reporting handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Overview handles GET /admin/reports/overview.
//
// Under the admin audience the metrics aggregate across every store unless a
// storeId query narrows the report. Store-console requests remain pinned to the
// JWT store scope and ignore any client-supplied storeId.
func (h *Handler) Overview(c *gin.Context) {
	storeID, err := reportStoreID(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetOverview(c.Request.Context(), OverviewFilter{StoreID: storeID})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// reportFilter builds a ReportFilter from the request: page from ?page/?pageSize,
// optional ?from/?to (RFC3339 date or datetime, ignored if unparseable), and the
// pinned store scope from the JWT, or the optional admin ?storeId filter.
func reportFilter(c *gin.Context) (ReportFilter, error) {
	storeID, err := reportStoreID(c)
	if err != nil {
		return ReportFilter{}, err
	}
	f := ReportFilter{Page: httpx.ParsePage(c), StoreID: storeID}
	if from := parseDateParam(c.Query("from")); from != nil {
		f.From = from
	}
	if to := parseDateParam(c.Query("to")); to != nil {
		// A date-only upper bound means the whole local calendar date, rather
		// than midnight at its start. RFC3339 datetimes retain their exact bound.
		if _, err := time.Parse("2006-01-02", c.Query("to")); err == nil {
			endOfDay := to.AddDate(0, 0, 1).Add(-time.Nanosecond)
			to = &endOfDay
		}
		f.To = to
	}
	return f, nil
}

func reportStoreID(c *gin.Context) (*int64, error) {
	if scope, ok := storescope.FromContext(c); ok {
		return &scope, nil
	}
	raw := strings.TrimSpace(c.Query("storeId"))
	if raw == "" {
		return nil, nil
	}
	storeID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || storeID <= 0 {
		return nil, apperr.Invalid("invalid storeId")
	}
	return &storeID, nil
}

func parseDateParam(v string) *time.Time {
	if v == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t
	}
	if t, err := time.ParseInLocation("2006-01-02", v, reportLocation); err == nil {
		return &t
	}
	return nil
}

// Revenue handles GET /admin/reports/revenue and GET /store/reports/revenue.
func (h *Handler) Revenue(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetRevenue(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// CatalogItems handles GET /admin/reports/catalog-items.
func (h *Handler) CatalogItems(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetCatalogItems(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Activities handles GET /admin/reports/activities.
func (h *Handler) Activities(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetActivities(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Coupons handles GET /admin/reports/coupons.
func (h *Handler) Coupons(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetCoupons(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Records handles GET /admin/reports/records.
func (h *Handler) Records(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetRecords(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Members handles GET /admin/reports/members.
func (h *Handler) Members(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetMembers(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Reservations handles GET /admin/reports/reservations.
func (h *Handler) Reservations(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetReservations(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// Stores handles GET /admin/reports/stores.
func (h *Handler) Stores(c *gin.Context) {
	f, err := reportFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, total, err := h.svc.GetStores(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}
