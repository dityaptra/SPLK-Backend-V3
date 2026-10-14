package httpx

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// Pagination menyimpan parameter halaman dan batas jumlah data.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// PaginationMeta menyimpan metadata paginasi untuk response daftar.
type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// Offset menghitung offset query SQL berdasarkan Page dan Limit.
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// ParsePagination membaca parameter `page` dan `limit` dari query string URL.
// Menerapkan default page = 1, default limit = 20, dan batas maksimal limit = 100.
func ParsePagination(c *gin.Context) Pagination {
	page := DefaultPage
	limit := DefaultLimit

	if pageStr := c.Query("page"); pageStr != "" {
		if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
			page = val
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			if val > MaxLimit {
				limit = MaxLimit
			} else {
				limit = val
			}
		}
	}

	return Pagination{
		Page:  page,
		Limit: limit,
	}
}
