package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/app"
	"splk-backend/internal/auth"
	"splk-backend/internal/config"
	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/security"
)

func runSeedAdmin(ctx context.Context, pool *pgxpool.Pool, username, email, password string) error {
	return database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		var roleAdminID int
		err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE nama = 'admin'").Scan(&roleAdminID)
		if err != nil {
			return err
		}

		var userID int
		err = tx.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if err != nil { // ErrNoRows
			passHash, err := security.HashPassword(password)
			if err != nil {
				return err
			}
			err = tx.QueryRow(ctx, `
				INSERT INTO users (username, email, password_hash, is_active)
				VALUES ($1, $2, $3, true)
				RETURNING id
			`, username, email, passHash).Scan(&userID)
			if err != nil {
				return err
			}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO user_roles (id_user, id_role)
			VALUES ($1, $2)
			ON CONFLICT (id_user, id_role) DO NOTHING
		`, userID, roleAdminID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO admin (id_user, nama, nip)
			VALUES ($1, 'Admin Test', 'ADM999')
			ON CONFLICT (id_user) DO NOTHING
		`, userID)
		return err
	})
}

func TestAuthIntegration(t *testing.T) {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diatur; lewati test integrasi auth")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loc, _ := time.LoadLocation("Asia/Makassar")
	cfg := &config.Config{
		AppEnv:             "development",
		AppPort:            "8080",
		AppTimezone:        loc,
		DatabaseURL:        testDBURL,
		JWTSecret:          "kunci_rahasia_testing_32_karakter_sangat_aman_sekali",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		CORSAllowedOrigins: []string{"*"},
	}

	pool, err := database.NewPool(ctx, cfg)
	require.NoError(t, err)
	defer pool.Close()

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, discardLogger)

	// Tambahkan rute pengujian khusus untuk memverifikasi RequireRoles("dosen")
	router.GET("/test/khusus-dosen", middleware.Auth(cfg.JWTSecret), middleware.RequireRoles("dosen"), func(c *gin.Context) {
		c.String(http.StatusOK, "akses dosen diizinkan")
	})

	adminUser := "admin_integ"
	adminEmail := "admin_integ@undiksha.ac.id"
	adminPass := "AdminInteg123"

	t.Run("Seed Admin Idempoten (Dijalankan Dua Kali)", func(t *testing.T) {
		// Jalankan seed pertama
		err := runSeedAdmin(ctx, pool, adminUser, adminEmail, adminPass)
		require.NoError(t, err)

		// Jalankan seed kedua
		err = runSeedAdmin(ctx, pool, adminUser, adminEmail, adminPass)
		require.NoError(t, err)

		// Verifikasi tidak ada data ganda
		var userCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE username = $1", adminUser).Scan(&userCount)
		require.NoError(t, err)
		assert.Equal(t, 1, userCount, "user admin harus tetap tepat 1 baris")

		var adminProfileCount int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM admin a 
			JOIN users u ON a.id_user = u.id 
			WHERE u.username = $1
		`, adminUser).Scan(&adminProfileCount)
		require.NoError(t, err)
		assert.Equal(t, 1, adminProfileCount, "profil admin harus tetap tepat 1 baris")
	})

	var accessToken string
	var rawRefreshToken string

	t.Run("Login Admin Sukses dan Token Disimpan Sebagai Hash", func(t *testing.T) {
		loginBody := auth.LoginRequest{
			Username: adminUser,
			Password: adminPass,
		}
		bodyBytes, _ := json.Marshal(loginBody)

		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "login gagal: %s", w.Body.String())

		var resp struct {
			Data auth.LoginResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		accessToken = resp.Data.Token
		rawRefreshToken = resp.Data.RefreshToken
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, rawRefreshToken)
		assert.Contains(t, resp.Data.User.Roles, "admin")

		// Verifikasi tabel tokens hanya berisi hash, BUKAN token mentah
		var tokenHashInDB string
		err = pool.QueryRow(ctx, `
			SELECT t.token_hash 
			FROM tokens t 
			JOIN users u ON t.id_user = u.id 
			WHERE u.username = $1 
			ORDER BY t.created_at DESC 
			LIMIT 1
		`, adminUser).Scan(&tokenHashInDB)
		require.NoError(t, err)

		assert.NotEqual(t, rawRefreshToken, tokenHashInDB, "token di DB TIDAK boleh sama dengan raw refresh token")
		assert.Equal(t, security.HashToken(rawRefreshToken), tokenHashInDB, "token di DB harus sama dengan SHA-256 hash raw token")
	})

	t.Run("GET /auth/me Berisi Role Admin dan Profil Admin", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data auth.MeResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, adminUser, resp.Data.Username)
		assert.Contains(t, resp.Data.Roles, "admin")
		assert.NotNil(t, resp.Data.Profiles.AdminID, "profil admin harus terisi")
	})

	t.Run("Akses Endpoint Terproteksi Tanpa Token Menghasilkan 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
	})

	t.Run("Akses Endpoint dengan Role yang Salah Menghasilkan 403", func(t *testing.T) {
		// Admin mencoba mengakses route khusus dosen
		req, _ := http.NewRequest(http.MethodGet, "/test/khusus-dosen", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "AKSES_DITOLAK", resp.Error.Code)
	})

	t.Run("Refresh Merotasi Token dan Token Lama Ditolak", func(t *testing.T) {
		// 1. Refresh dengan token aktif
		refreshBody := auth.RefreshRequest{RefreshToken: rawRefreshToken}
		bodyBytes, _ := json.Marshal(refreshBody)

		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data auth.RefreshResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		newRefreshToken := resp.Data.RefreshToken
		assert.NotEmpty(t, newRefreshToken)
		assert.NotEqual(t, rawRefreshToken, newRefreshToken, "token baru harus berbeda dari token lama")

		// 2. Coba refresh lagi menggunakan TOKEN LAMA (harus ditolak karena sudah dirotasi)
		reqOld, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(bodyBytes))
		reqOld.Header.Set("Content-Type", "application/json")
		wOld := httptest.NewRecorder()
		router.ServeHTTP(wOld, reqOld)

		assert.Equal(t, http.StatusUnauthorized, wOld.Code, "token lama harus ditolak setelah dirotasi")
	})
}
