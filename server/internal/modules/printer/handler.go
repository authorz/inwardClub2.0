package printer

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// ConsoleHandler exposes printer device management for the admin (cross-store
// read) and store (own-store CRUD) consoles. Route wiring decides which group
// mounts which method.
type ConsoleHandler struct {
	svc  *ConsoleService
	jobs *JobRepository
}

// NewConsoleHandler builds the printer console handler.
func NewConsoleHandler(svc *ConsoleService, jobRepositories ...*JobRepository) *ConsoleHandler {
	h := &ConsoleHandler{svc: svc}
	if len(jobRepositories) > 0 {
		h.jobs = jobRepositories[0]
	}
	return h
}

// --- Admin ---

// AdminList handles GET /admin/printer-devices. An optional ?storeId filter
// narrows the result to a single store.
func (h *ConsoleHandler) AdminList(c *gin.Context) {
	var storeID *int64
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		storeID = &id
	}
	views, err := h.svc.AdminList(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// AdminPrintJobs handles GET /admin/print-jobs.
func (h *ConsoleHandler) AdminPrintJobs(c *gin.Context) {
	if h.jobs == nil {
		httpx.Fail(c, apperr.Internal(nil))
		return
	}
	f := PrintJobFilter{
		Page:    httpx.ParsePage(c),
		Status:  strings.TrimSpace(c.Query("status")),
		Keyword: strings.TrimSpace(c.Query("keyword")),
	}
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	if raw := c.Query("createdFrom"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.Fail(c, apperr.Invalid("invalid createdFrom"))
			return
		}
		f.CreatedFrom = &parsed
	}
	if raw := c.Query("createdTo"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.Fail(c, apperr.Invalid("invalid createdTo"))
			return
		}
		before := parsed.Add(24 * time.Hour)
		f.CreatedBefore = &before
	}
	jobs, total, err := h.jobs.List(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, jobs, httpx.MetaFor(f.Page, total))
}

// AdminCreate handles POST /admin/printer-devices.
func (h *ConsoleHandler) AdminCreate(c *gin.Context) {
	var in AdminDeviceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	entry := audit.FromContext(c, "printer.device.create", "printer_device", 0)
	view, err := h.svc.AdminCreate(c.Request.Context(), in, idempotency.Key(c), entry)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminUpdate handles PATCH /admin/printer-devices/:id.
func (h *ConsoleHandler) AdminUpdate(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in AdminDevicePatch
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	entry := audit.FromContext(c, "printer.device.update", "printer_device", id)
	view, err := h.svc.AdminUpdate(c.Request.Context(), id, in, idempotency.Key(c), entry)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminDelete handles DELETE /admin/printer-devices/:id.
func (h *ConsoleHandler) AdminDelete(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in AdminDeleteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	entry := audit.FromContext(c, "printer.device.delete", "printer_device", id)
	if err := h.svc.AdminDelete(c.Request.Context(), id, in, idempotency.Key(c), entry); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}

// --- Store ---

// StoreList handles GET /store/printer-devices.
func (h *ConsoleHandler) StoreList(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	views, err := h.svc.StoreList(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// StoreCreate handles POST /store/printer-devices. The device is pinned to the
// caller's own store scope.
func (h *ConsoleHandler) StoreCreate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var in DeviceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.StoreCreate(c.Request.Context(), storeID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreUpdate handles PATCH /store/printer-devices/:id.
func (h *ConsoleHandler) StoreUpdate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var patch DevicePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.StoreUpdate(c.Request.Context(), storeID, id, patch)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreDelete handles DELETE /store/printer-devices/:id.
func (h *ConsoleHandler) StoreDelete(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.StoreDelete(c.Request.Context(), storeID, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
