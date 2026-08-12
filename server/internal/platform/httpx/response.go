// Package httpx holds the shared HTTP conventions: the fixed response envelope,
// pagination, request context keys and the base middleware. Every handler in
// every module renders through these helpers so responses are uniform.
package httpx

import (
	"net/http"
	"strings"
	"unicode"

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
			Message: clientMessage(appErr.Code, appErr.Message),
			Details: appErr.Details,
		},
	})
}

var clientMessageTranslations = map[string]string{
	"invalid request body":                               "请求内容格式不正确",
	"table code already exists in this store":            "当前门店已存在相同的桌子编号",
	"capacity cannot be lower than existing seat count":  "座位数量不能小于该桌子已有的座位数",
	"table seat capacity has been reached":               "该桌子的座位数量已达到上限",
	"table with seats or reservations cannot be deleted": "桌子下仍有座位或预约，无法删除",
	"store not found":                                    "门店不存在",
	"table not found":                                    "桌子不存在",
	"seat not found":                                     "座位不存在",
	"seat is already reserved":                           "该座位已被预约，请选择其他座位",
	"seat is not available":                              "该座位当前不可预约，请选择其他座位",
	"reservation cannot be cancelled":                    "该预约已取消或不存在",
	"member not found":                                   "会员不存在",
	"food order not found":                               "点餐订单不存在",
	"payment order not found":                            "支付订单不存在",
	"insufficient balance":                               "账户余额不足",
	"insufficient coin balance":                          "金币余额不足",
	"insufficient ticket stock":                          "票档库存不足",
}

func clientMessage(code apperr.Code, message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return fallbackClientMessage(code)
	}
	for _, r := range message {
		if unicode.Is(unicode.Han, r) {
			return message
		}
	}
	lower := strings.ToLower(message)
	if translated, ok := clientMessageTranslations[lower]; ok {
		return translated
	}
	switch {
	case strings.HasSuffix(lower, " not found"):
		return "请求的数据不存在"
	case strings.Contains(lower, "already exists"):
		return "数据已存在，请勿重复添加"
	case strings.Contains(lower, "already "):
		return "请勿重复操作"
	case strings.Contains(lower, "insufficient"):
		return "可用余额或库存不足"
	case strings.Contains(lower, "not payable"):
		return "当前订单状态无法支付"
	case strings.Contains(lower, "not available") || strings.Contains(lower, "not active"):
		return "当前资源状态不可用"
	case strings.HasPrefix(lower, "invalid ") || strings.Contains(lower, ": invalid "):
		return "请求参数格式不正确"
	case strings.Contains(lower, " is required") || strings.Contains(lower, " must "):
		return "缺少必填参数或参数不符合要求"
	case strings.Contains(lower, "cannot ") || strings.Contains(lower, " is not "):
		return "当前状态不允许此操作"
	default:
		return fallbackClientMessage(code)
	}
}

func fallbackClientMessage(code apperr.Code) string {
	switch code {
	case apperr.CodeInvalidArgument:
		return "请求参数不正确"
	case apperr.CodeUnauthenticated:
		return "登录状态已失效，请重新登录"
	case apperr.CodePermissionDenied, apperr.CodeStoreScopeRequired:
		return "没有权限执行此操作"
	case apperr.CodeNotFound, apperr.CodeMemberNotFound:
		return "请求的数据不存在"
	case apperr.CodeConflict, apperr.CodeIdempotencyConflict:
		return "当前状态不允许此操作"
	case apperr.CodeIdempotencyRequired:
		return "请求缺少防重复标识，请重试"
	case apperr.CodeInsufficientBalance:
		return "账户余额不足"
	case apperr.CodeRuleDisabled:
		return "当前功能暂不可用"
	case apperr.CodeRateLimited:
		return "操作过于频繁，请稍后再试"
	case apperr.CodeUnavailable:
		return "服务暂不可用，请稍后再试"
	case apperr.CodeNotImplemented:
		return "该功能暂未开放"
	default:
		return "服务器内部错误，请稍后再试"
	}
}
