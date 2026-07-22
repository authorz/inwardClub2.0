package reservation

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// Handler exposes the mini-program reservation endpoints and the store-console
// arrival endpoint. Router wiring is owned elsewhere.
type Handler struct {
	svc *Service
}

// NewHandler builds the reservation handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Tables handles GET /mini/stores/{storeID}/tables.
func (h *Handler) Tables(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, err := h.svc.ListTables(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// Seats handles GET /mini/stores/{storeID}/seats.
func (h *Handler) Seats(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, err := h.svc.ListSeats(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// ListReservations handles GET /mini/reservations.
func (h *Handler) ListReservations(c *gin.Context) {
	claims := authn.MustFromContext(c)
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListReservations(c.Request.Context(), claims.SubjectID(), page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// CreateReservation handles POST /mini/reservations.
func (h *Handler) CreateReservation(c *gin.Context) {
	claims := authn.MustFromContext(c)
	var req CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.CreateReservation(c.Request.Context(), claims.SubjectID(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// GetReservation handles GET /mini/reservations/{reservationID}.
func (h *Handler) GetReservation(c *gin.Context) {
	claims := authn.MustFromContext(c)
	id, err := pathID(c, "reservationID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetReservation(c.Request.Context(), claims.SubjectID(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// CancelReservation handles POST /mini/reservations/{reservationID}/cancel.
func (h *Handler) CancelReservation(c *gin.Context) {
	claims := authn.MustFromContext(c)
	id, err := pathID(c, "reservationID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.CancelReservation(c.Request.Context(), claims.SubjectID(), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

// CreateWaitlistEntry handles POST /mini/waitlist-entries.
func (h *Handler) CreateWaitlistEntry(c *gin.Context) {
	claims := authn.MustFromContext(c)
	var req CreateWaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.CreateWaitlistEntry(c.Request.Context(), claims.SubjectID(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// StoreList handles GET /store/reservations. The store scope comes from the JWT
// via storescope; results are limited to the acting store.
func (h *Handler) StoreList(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListStoreReservations(c.Request.Context(), storeID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// StoreGet handles GET /store/reservations/{reservationID}, scoped to the acting
// store; a booking outside the scope is reported as not found.
func (h *Handler) StoreGet(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "reservationID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetStoreReservation(c.Request.Context(), storeID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// Arrive handles POST /store/reservations/{reservationID}/arrive. The store
// scope comes from the JWT via storescope; the acting staff identity from claims.
func (h *Handler) Arrive(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	claims := authn.MustFromContext(c)
	id, err := pathID(c, "reservationID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.ArriveReservation(c.Request.Context(), storeID, id, string(claims.SubjectType), claims.SubjectID()); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
