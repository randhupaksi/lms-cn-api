package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage    = 1
	defaultPerPage = 20
	maxPerPage     = 100
)

type Request struct {
	Page    int
	PerPage int
}

type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func FromContext(c *gin.Context) Request {
	page := positiveInt(c.Query("page"), defaultPage)
	perPage := positiveInt(c.Query("per_page"), defaultPerPage)
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return Request{Page: page, PerPage: perPage}
}

func (r Request) Offset() int { return (r.Page - 1) * r.PerPage }

func (r Request) Meta(total int64) Meta {
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(r.PerPage) - 1) / int64(r.PerPage))
	}
	return Meta{Page: r.Page, PerPage: r.PerPage, Total: total, TotalPages: totalPages}
}

func positiveInt(raw string, fallback int) int {
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
