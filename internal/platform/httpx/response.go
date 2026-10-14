package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OK mengirimkan response HTTP 200 dengan format {"data": data}.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

// Created mengirimkan response HTTP 201 dengan format {"data": data}.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{
		"data": data,
	})
}

// NoContent mengirimkan response HTTP 204 tanpa body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// List mengirimkan response HTTP 200 untuk daftar data dengan format {"data": data, "meta": meta}.
func List(c *gin.Context, data any, meta PaginationMeta) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": meta,
	})
}
