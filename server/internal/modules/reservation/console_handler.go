package reservation

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// ConsoleHandler exposes headquarters table and seat CRUD.
type ConsoleHandler struct {
	svc *ConsoleService
}

func NewConsoleHandler(svc *ConsoleService) *ConsoleHandler {
	return &ConsoleHandler{svc: svc}
}

func (h *ConsoleHandler) ListTables(c *gin.Context) {
	filter, err := parseTableFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	items, total, err := h.svc.ListTables(c.Request.Context(), filter, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, items, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) GetTable(c *gin.Context) {
	id, err := pathID(c, "tableID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	item, err := h.svc.GetTable(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, item)
}

func (h *ConsoleHandler) CreateTable(c *gin.Context) {
	var req TableWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	item, err := h.svc.CreateTable(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, item)
}

func (h *ConsoleHandler) UpdateTable(c *gin.Context) {
	id, err := pathID(c, "tableID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req TableWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	item, err := h.svc.UpdateTable(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, item)
}

func (h *ConsoleHandler) DeleteTable(c *gin.Context) {
	id, err := pathID(c, "tableID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteTable(c.Request.Context(), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

func (h *ConsoleHandler) ListSeats(c *gin.Context) {
	filter, err := parseSeatFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	items, total, err := h.svc.ListSeats(c.Request.Context(), filter, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, items, httpx.MetaFor(page, total))
}

func (h *ConsoleHandler) GetSeat(c *gin.Context) {
	id, err := pathID(c, "seatID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	item, err := h.svc.GetSeat(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, item)
}

func (h *ConsoleHandler) CreateSeat(c *gin.Context) {
	var req SeatWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	item, err := h.svc.CreateSeat(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, item)
}

func (h *ConsoleHandler) UpdateSeat(c *gin.Context) {
	id, err := pathID(c, "seatID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req SeatWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	item, err := h.svc.UpdateSeat(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, item)
}

func (h *ConsoleHandler) DeleteSeat(c *gin.Context) {
	id, err := pathID(c, "seatID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.DeleteSeat(c.Request.Context(), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

func parseTableFilter(c *gin.Context) (AdminTableFilter, error) {
	storeID, err := optionalPositiveID(c.Query("storeId"), "storeId")
	if err != nil {
		return AdminTableFilter{}, err
	}
	return AdminTableFilter{
		StoreID: storeID,
		Status:  strings.TrimSpace(c.Query("status")),
		Keyword: strings.TrimSpace(c.Query("keyword")),
	}, nil
}

func parseSeatFilter(c *gin.Context) (AdminSeatFilter, error) {
	storeID, err := optionalPositiveID(c.Query("storeId"), "storeId")
	if err != nil {
		return AdminSeatFilter{}, err
	}
	tableID, err := optionalPositiveID(c.Query("tableId"), "tableId")
	if err != nil {
		return AdminSeatFilter{}, err
	}
	return AdminSeatFilter{
		StoreID: storeID,
		TableID: tableID,
		Status:  strings.TrimSpace(c.Query("status")),
		Keyword: strings.TrimSpace(c.Query("keyword")),
	}, nil
}

func optionalPositiveID(raw, name string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, apperr.Invalid("invalid " + name)
	}
	return &id, nil
}
