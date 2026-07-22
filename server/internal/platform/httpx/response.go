// Package httpx holds the shared HTTP conventions: the fixed response envelope,
// pagination, request context keys and the base middleware. Every handler in
// every module renders through these helpers so responses are uniform.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Envelope is the success response shape: {"data": ..., "meta": ...}.
type Envelope struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

// ErrorEnvelope is the failure response shape: {"error": {code, message, details}}.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the client-facing error payload; internal causes are never included.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// OK writes a 200 with a data envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Data: data})
}

// Created writes a 201 with a data envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Data: data})
}

// List writes a 200 with data plus pagination meta.
func List(c *gin.Context, data any, meta Meta) {
	if data == nil {
		data = []any{}
	}
	c.JSON(http.StatusOK, Envelope{Data: data, Meta: meta})
}

// NoData writes a 200 with an empty object payload for actions with no body.
func NoData(c *gin.Context) {
	c.JSON(http.StatusOK, Envelope{Data: gin.H{}})
}

// Fail normalises any error to an *apperr.Error and renders the error envelope.
// It aborts the gin chain so no further handler writes to the response.
func Fail(c *gin.Context, err error) {
	appErr := apperr.From(err)
	// Attach to the gin error list so the logging middleware can record the cause.
	_ = c.Error(err)
	c.AbortWithStatusJSON(appErr.Status, ErrorEnvelope{
		Error: ErrorBody{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}
