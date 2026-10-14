// Package middleware menyediakan middleware HTTP untuk RequestID, Logging slog, Panic Recovery, dan CORS.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"splk-backend/internal/platform/httpx"
)

const HeaderRequestID = "X-Request-ID"

// RequestID membaca header X-Request-ID atau membuat UUID baru jika belum ada,
// lalu menyimpannya di context Gin dan menulisnya kembali ke header response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		c.Set("request_id", reqID)
		c.Writer.Header().Set(HeaderRequestID, reqID)
		c.Next()
	}
}

// Logger mencatat setiap request HTTP secara terstruktur menggunakan slog.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		reqID := c.GetString("request_id")
		status := c.Writer.Status()

		fullPath := path
		if query != "" {
			fullPath = path + "?" + query
		}

		logger.InfoContext(c.Request.Context(), "http request",
			"request_id", reqID,
			"method", c.Request.Method,
			"path", fullPath,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
		)
	}
}

// Recovery menangani panic yang terjadi saat pemrosesan request, mencatat stack trace ke slog,
// dan mengembalikan response error 500 terstandar kepada klien sehingga server tidak crash.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				reqID := c.GetString("request_id")
				stack := string(debug.Stack())

				logger.ErrorContext(c.Request.Context(), "panic recovered",
					"request_id", reqID,
					"panic", fmt.Sprintf("%v", r),
					"stack", stack,
				)

				c.Abort()
				httpx.WriteError(c, httpx.Internal("terjadi kesalahan internal pada server"))
			}
		}()

		c.Next()
	}
}

// CORS mengatur header Cross-Origin Resource Sharing sesuai daftar CORS_ALLOWED_ORIGINS.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowedMap := make(map[string]bool)
	allowAll := false

	for _, o := range allowedOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed == "*" {
			allowAll = true
		} else if trimmed != "" {
			allowedMap[trimmed] = true
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && (allowAll || allowedMap[origin]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
