package app_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/app"
	"splk-backend/internal/config"
	"splk-backend/internal/middleware"
)

func TestAppRouter(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		AppEnv:             "development",
		AppPort:            "8080",
		CORSAllowedOrigins: []string{"http://localhost:3000"},
	}

	t.Run("GET /health memiliki header X-Request-ID", func(t *testing.T) {
		// Buat router tanpa database pool (pool = nil menghasilkan database: down, tapi router dan middleware tetap bekerja)
		router := app.NewRouter(cfg, nil, discardLogger)

		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verifikasi keberadaan header X-Request-ID
		reqID := w.Header().Get(middleware.HeaderRequestID)
		assert.NotEmpty(t, reqID, "GET /health harus mengembalikan header X-Request-ID")
		_, err := uuid.Parse(reqID)
		assert.NoError(t, err, "X-Request-ID harus berupa format UUID yang valid")

		// Status 503 karena pool nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var body map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, "error", body["status"])
		assert.Equal(t, "down", body["database"])
	})
}
