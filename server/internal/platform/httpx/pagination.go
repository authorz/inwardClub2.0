package httpx

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Page holds a normalised, clamped pagination request.
type Page struct {
	Page     int
	PageSize int
}

// Offset returns the SQL offset for the page.
func (p Page) Offset() int { return (p.Page - 1) * p.PageSize }

// Limit returns the SQL limit for the page.
func (p Page) Limit() int { return p.PageSize }

// Meta is the pagination block returned in list response envelopes.
type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// ParsePage reads and clamps ?page and ?pageSize from the request.
func ParsePage(c *gin.Context) Page {
	page := atoiDefault(c.Query("page"), defaultPage)
	if page < 1 {
		page = defaultPage
	}
	size := atoiDefault(c.Query("pageSize"), defaultPageSize)
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return Page{Page: page, PageSize: size}
}

// MetaFor builds pagination meta from a page and total count.
func MetaFor(p Page, total int64) Meta {
	return Meta{Page: p.Page, PageSize: p.PageSize, Total: total}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
