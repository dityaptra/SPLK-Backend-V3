// Package httpx menyediakan format response terstandar, penanganan AppError, binding JSON, dan pagination.
package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

// AppError adalah error terstandar yang aman untuk dikembalikan ke klien HTTP.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
}

// Error memenuhi interface error bawaan Go.
func (e *AppError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Konstruktor AppError

// BadRequest mengembalikan AppError dengan status HTTP 400.
func BadRequest(code, message string, details ...any) *AppError {
	var d any
	if len(details) > 0 {
		d = details[0]
	}
	return &AppError{
		HTTPStatus: http.StatusBadRequest,
		Code:       code,
		Message:    message,
		Details:    d,
	}
}

// Unauthorized mengembalikan AppError dengan status HTTP 401.
func Unauthorized(code, message string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       code,
		Message:    message,
	}
}

// Forbidden mengembalikan AppError dengan status HTTP 403.
func Forbidden(code, message string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusForbidden,
		Code:       code,
		Message:    message,
	}
}

// NotFound mengembalikan AppError dengan status HTTP 404.
func NotFound(code, message string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       code,
		Message:    message,
	}
}

// Conflict mengembalikan AppError dengan status HTTP 409.
func Conflict(code, message string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusConflict,
		Code:       code,
		Message:    message,
	}
}

// Unprocessable mengembalikan AppError dengan status HTTP 422.
func Unprocessable(code, message string, details ...any) *AppError {
	var d any
	if len(details) > 0 {
		d = details[0]
	}
	return &AppError{
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       code,
		Message:    message,
		Details:    d,
	}
}

// Internal mengembalikan AppError dengan status HTTP 500.
func Internal(message string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       "INTERNAL_SERVER_ERROR",
		Message:    message,
	}
}

// WriteError menerjemahkan error menjadi format JSON standar:
// {"error": {"code": "...", "message": "...", "details": ...}}
// Memetakan AppError, pgconn.PgError (23505 -> 409, 23503 & 23514 -> 422),
// dan jika error tidak dikenal, merespons dengan 500 generik sambil mencatat detail ke slog.
func WriteError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		renderError(c, appErr)
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			renderError(c, Conflict("DATA_DUPLIKAT", "data yang dimasukkan sudah ada atau melanggar keunikan"))
			return
		case "23503":
			renderError(c, Unprocessable("RELASI_TIDAK_VALID", "referensi data terkait tidak ditemukan atau tidak valid"))
			return
		case "23514":
			renderError(c, Unprocessable("BATASAN_TIDAK_VALID", "data yang dimasukkan melanggar batasan sistem"))
			return
		}
	}

	// Error tidak dikenal: catat ke slog dan kirim response 500 generik tanpa membocorkan pesan mentah
	slog.ErrorContext(c.Request.Context(), "unhandled server error",
		"error", err.Error(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)

	renderError(c, Internal("terjadi kesalahan internal pada server"))
}

func renderError(c *gin.Context, appErr *AppError) {
	resp := gin.H{
		"code":    appErr.Code,
		"message": appErr.Message,
	}
	if appErr.Details != nil {
		resp["details"] = appErr.Details
	}

	c.JSON(appErr.HTTPStatus, gin.H{"error": resp})
}
