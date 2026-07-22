// Package apperr defines the single application error model used across every
// module. Handlers, services and repositories all return *Error so the HTTP
// layer can render a consistent JSON error envelope and pick the right status.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable, machine-readable error code returned to clients. Codes are
// centralised here so no module invents its own ad-hoc strings.
type Code string

const (
	CodeInvalidArgument     Code = "INVALID_ARGUMENT"
	CodeUnauthenticated     Code = "UNAUTHENTICATED"
	CodePermissionDenied    Code = "PERMISSION_DENIED"
	CodeNotFound            Code = "NOT_FOUND"
	CodeConflict            Code = "CONFLICT"
	CodeIdempotencyConflict Code = "IDEMPOTENCY_CONFLICT"
	CodeIdempotencyRequired Code = "IDEMPOTENCY_KEY_REQUIRED"
	CodeMemberNotFound      Code = "MEMBER_NOT_FOUND"
	CodeInsufficientBalance Code = "INSUFFICIENT_BALANCE"
	CodeRuleDisabled        Code = "RULE_DISABLED"
	CodeStoreScopeRequired  Code = "STORE_SCOPE_REQUIRED"
	CodeRateLimited         Code = "RATE_LIMITED"
	CodeUnavailable         Code = "UNAVAILABLE"
	CodeInternal            Code = "INTERNAL"
	CodeNotImplemented      Code = "NOT_IMPLEMENTED"
)

// httpStatusByCode maps stable codes to HTTP statuses. Constructors below use it
// so callers never hand-pick a status.
var httpStatusByCode = map[Code]int{
	CodeInvalidArgument:     http.StatusBadRequest,
	CodeUnauthenticated:     http.StatusUnauthorized,
	CodePermissionDenied:    http.StatusForbidden,
	CodeNotFound:            http.StatusNotFound,
	CodeConflict:            http.StatusConflict,
	CodeIdempotencyConflict: http.StatusConflict,
	CodeIdempotencyRequired: http.StatusBadRequest,
	CodeMemberNotFound:      http.StatusNotFound,
	CodeInsufficientBalance: http.StatusConflict,
	CodeRuleDisabled:        http.StatusConflict,
	CodeStoreScopeRequired:  http.StatusForbidden,
	CodeRateLimited:         http.StatusTooManyRequests,
	CodeUnavailable:         http.StatusServiceUnavailable,
	CodeInternal:            http.StatusInternalServerError,
	CodeNotImplemented:      http.StatusNotImplemented,
}

// Error is the canonical application error.
type Error struct {
	Code    Code
	Message string
	Status  int
	Details any
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// WithDetails attaches structured details (e.g. field validation errors).
func (e *Error) WithDetails(details any) *Error {
	e.Details = details
	return e
}

// WithCause wraps an underlying error for logging without leaking it to clients.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// New builds an *Error from a code, deriving the HTTP status from the code.
func New(code Code, message string) *Error {
	status, ok := httpStatusByCode[code]
	if !ok {
		status = http.StatusInternalServerError
	}
	return &Error{Code: code, Message: message, Status: status}
}

// Common constructors keep call sites terse and consistent.

func Invalid(message string) *Error         { return New(CodeInvalidArgument, message) }
func Unauthenticated(message string) *Error { return New(CodeUnauthenticated, message) }
func Forbidden(message string) *Error       { return New(CodePermissionDenied, message) }
func NotFound(message string) *Error        { return New(CodeNotFound, message) }
func Conflict(message string) *Error        { return New(CodeConflict, message) }
func NotImplemented(message string) *Error  { return New(CodeNotImplemented, message) }

// Internal wraps an unexpected error; the message is safe for clients while the
// cause is preserved for logs.
func Internal(cause error) *Error {
	return New(CodeInternal, "internal error").WithCause(cause)
}

// From normalises any error into an *Error so the HTTP layer has one code path.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
