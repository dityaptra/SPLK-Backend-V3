package middleware_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMiddleware(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("RequestID membaca atau membuat UUID baru", func(t *testing.T) {
		r := gin.New()
		r.Use(middleware.RequestID())
		r.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})

		// Kasus 1: Klien tidak mengirim X-Request-ID
		req1, _ := http.NewRequest(http.MethodGet, "/ping", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		reqID1 := w1.Header().Get(middleware.HeaderRequestID)
		assert.NotEmpty(t, reqID1)
		_, err := uuid.Parse(reqID1)
		assert.NoError(t, err, "harus berupa UUID valid")

		// Kasus 2: Klien mengirim X-Request-ID
		customID := "client-custom-req-id-12345"
		req2, _ := http.NewRequest(http.MethodGet, "/ping", nil)
		req2.Header.Set(middleware.HeaderRequestID, customID)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		reqID2 := w2.Header().Get(middleware.HeaderRequestID)
		assert.Equal(t, customID, reqID2, "harus mempertahankan X-Request-ID dari klien")
	})

	t.Run("Recovery menangkap panic dan mengembalikan 500 terstandar", func(t *testing.T) {
		r := gin.New()
		r.Use(middleware.RequestID())
		r.Use(middleware.Recovery(discardLogger))

		r.GET("/panic", func(c *gin.Context) {
			panic("simulasi kepanikan tak terduga!")
		})

		req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()

		// Server tidak boleh crash
		assert.NotPanics(t, func() {
			r.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INTERNAL_SERVER_ERROR", resp.Error.Code)
		assert.Equal(t, "terjadi kesalahan internal pada server", resp.Error.Message)
		assert.NotEmpty(t, w.Header().Get(middleware.HeaderRequestID))
	})

	t.Run("CORS mengatur origin dan menangani preflight OPTIONS", func(t *testing.T) {
		r := gin.New()
		allowed := []string{"http://localhost:3000", "https://splk.undiksha.ac.id"}
		r.Use(middleware.CORS(allowed))
		r.GET("/api/data", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// Preflight OPTIONS dari origin yang diizinkan
		reqPreflight, _ := http.NewRequest(http.MethodOptions, "/api/data", nil)
		reqPreflight.Header.Set("Origin", "http://localhost:3000")
		wPreflight := httptest.NewRecorder()
		r.ServeHTTP(wPreflight, reqPreflight)

		assert.Equal(t, http.StatusNoContent, wPreflight.Code)
		assert.Equal(t, "http://localhost:3000", wPreflight.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, wPreflight.Header().Get("Access-Control-Allow-Methods"), "GET")

		// GET dari origin yang tidak diizinkan
		reqForbiddenOrigin, _ := http.NewRequest(http.MethodGet, "/api/data", nil)
		reqForbiddenOrigin.Header.Set("Origin", "http://untrusted-site.com")
		wForbidden := httptest.NewRecorder()
		r.ServeHTTP(wForbidden, reqForbiddenOrigin)

		assert.Equal(t, http.StatusOK, wForbidden.Code)
		assert.Empty(t, wForbidden.Header().Get("Access-Control-Allow-Origin"), "origin tak dikenal tidak mendapat CORS header")
	})
}
