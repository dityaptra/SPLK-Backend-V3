package pendaftaran_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"splk-backend/internal/config"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/security"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diset; lewati integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDBURL)
	require.NoError(t, err, "koneksi database test gagal")

	err = pool.Ping(ctx)
	require.NoError(t, err, "ping database test gagal")

	return pool
}

func ensureAkademik(t *testing.T, pool *pgxpool.Pool) int {
	ctx := context.Background()
	var prodiID int
	err := database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		var fakID, jurID int
		err := tx.QueryRow(ctx, "SELECT id FROM fakultas WHERE kode = 'FTK'").Scan(&fakID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, "INSERT INTO fakultas (nama, kode) VALUES ('Fakultas Teknik dan Kejuruan', 'FTK') RETURNING id").Scan(&fakID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, "SELECT id FROM jurusan WHERE kode_jurusan = 'TI'").Scan(&jurID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, "INSERT INTO jurusan (id_fakultas, nama, kode_jurusan) VALUES ($1, 'Teknik Informatika', 'TI') RETURNING id", fakID).Scan(&jurID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, "SELECT id FROM program_studi WHERE kode = 'PTI'").Scan(&prodiID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, "INSERT INTO program_studi (id_jurusan, nama, kode) VALUES ($1, 'Pendidikan Teknik Informatika', 'PTI') RETURNING id", jurID).Scan(&prodiID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	return prodiID
}

func setupUser(t *testing.T, pool *pgxpool.Pool, username, email, password, roleName string) int {
	ctx := context.Background()
	var userID int
	err := database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		var roleID int
		err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE nama = $1", roleName).Scan(&roleID)
		if err != nil {
			return fmt.Errorf("role %s: %w", roleName, err)
		}

		err = tx.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
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
		} else if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO user_roles (id_user, id_role)
			VALUES ($1, $2)
			ON CONFLICT (id_user, id_role) DO NOTHING
		`, userID, roleID)
		return err
	})
	require.NoError(t, err)
	return userID
}

func ensureMahasiswaProfile(t *testing.T, pool *pgxpool.Pool, userID, prodiID int, nim, nama string) int {
	ctx := context.Background()
	var mhsID int
	err := pool.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", userID).Scan(&mhsID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO mahasiswa (id_user, id_prodi, nim, nama, angkatan)
			VALUES ($1, $2, $3, $4, 2023)
			RETURNING id
		`, userID, prodiID, nim, nama).Scan(&mhsID)
		require.NoError(t, err)
	} else {
		require.NoError(t, err)
	}
	return mhsID
}

func getAuthToken(t *testing.T, router *gin.Engine, username, password string) string {
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "login gagal: %s", w.Body.String())

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data.Token)
	return resp.Data.Token
}

// TestIntegration_PendaftaranInti menguji skenario end-to-end pendaftaran inti:
// 1. Mahasiswa A mendaftar -> 201 Created
// 2. Mahasiswa B tidak bisa melihat pendaftaran A -> 404 Not Found (ARCHITECTURE §6)
// 3. Pendaftaran ganda ditolak -> 409 Conflict
// 4. Admin melihat seluruh pendaftaran
// 5. Mahasiswa B melihat hanya miliknya setelah mendaftar
func TestIntegration_PendaftaranInti(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()

	// Bersihkan data pendaftaran dan periode di test database
	_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
	_, _ = pool.Exec(ctx, "DELETE FROM periode")

	prodiID := ensureAkademik(t, pool)

	// Setup Mahasiswa A dan Mahasiswa B
	userMhsAID := setupUser(t, pool, "mhs_a_t6", "mhs_a_t6@undiksha.ac.id", "MhsPass123!", "mahasiswa")
	ensureMahasiswaProfile(t, pool, userMhsAID, prodiID, "NIM_T6_001", "Mahasiswa A")

	userMhsBID := setupUser(t, pool, "mhs_b_t6", "mhs_b_t6@undiksha.ac.id", "MhsPass123!", "mahasiswa")
	ensureMahasiswaProfile(t, pool, userMhsBID, prodiID, "NIM_T6_002", "Mahasiswa B")

	// Setup Admin
	adminID := setupUser(t, pool, "admin_t6", "admin_t6@undiksha.ac.id", "AdminPass123!", "admin")
	_, _ = pool.Exec(ctx, "INSERT INTO admin (id_user, nama, nip) VALUES ($1, 'Admin T6', 'ADM_T6') ON CONFLICT (id_user) DO NOTHING", adminID)

	// Buat Periode Magang Aktif
	var periodeID int
	err := pool.QueryRow(ctx, `
		INSERT INTO periode (
			nama, jenis_program, tahun_semester,
			tanggal_mulai, tanggal_selesai,
			tanggal_buka_daftar, tanggal_tutup_daftar,
			status
		) VALUES (
			'Magang MBKM Semester Ganjil 2026/2027', 'magang', '2026/2027-1',
			'2026-08-01', '2026-12-31',
			CURRENT_DATE - INTERVAL '5 days', CURRENT_DATE + INTERVAL '20 days',
			'aktif'
		) RETURNING id
	`).Scan(&periodeID)
	require.NoError(t, err)

	cfg := &config.Config{
		AppEnv:             "test",
		JWTSecret:          "kunci-rahasia-jwt-untuk-testing-harus-minimal-32-karakter!",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		CORSAllowedOrigins: []string{"*"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	tokenMhsA := getAuthToken(t, router, "mhs_a_t6", "MhsPass123!")
	tokenMhsB := getAuthToken(t, router, "mhs_b_t6", "MhsPass123!")
	tokenAdmin := getAuthToken(t, router, "admin_t6", "AdminPass123!")

	var pendaftaranAID int

	t.Run("1. Mahasiswa A mendaftar periode aktif -> 201 Created dengan status diajukan", func(t *testing.T) {
		body := map[string]any{
			"id_periode": periodeID,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhsA)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "respons: %s", w.Body.String())

		var resp struct {
			Data struct {
				ID           int    `json:"id"`
				Status       string `json:"status"`
				JenisProgram string `json:"jenis_program"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		pendaftaranAID = resp.Data.ID
		assert.Equal(t, "diajukan", resp.Data.Status)
		assert.Equal(t, "magang", resp.Data.JenisProgram)
	})

	t.Run("2. Mahasiswa A dapat melihat detail pendaftaran miliknya -> 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pendaftaran/%d", pendaftaranAID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsA)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				ID       int `json:"id"`
				Kesiapan struct {
					Siap bool `json:"siap"`
				} `json:"kesiapan"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, pendaftaranAID, resp.Data.ID)
		assert.False(t, resp.Data.Kesiapan.Siap, "belum ada penempatan dan dokumen maka siap harus false")
	})

	t.Run("3. Mahasiswa B TIDAK BISA melihat pendaftaran Mahasiswa A -> 404 Not Found (ARCHITECTURE §6)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pendaftaran/%d", pendaftaranAID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code, "ARCHITECTURE.md §6: tanpa relasi harus mengembalikan 404, bukan 403")
	})

	t.Run("4. Mahasiswa A mencoba mendaftar ulang pada periode yang sama -> 409 Conflict", func(t *testing.T) {
		body := map[string]any{
			"id_periode": periodeID,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhsA)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("5. Admin dapat melihat seluruh daftar pendaftaran", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pendaftaran", nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []struct {
				ID           int    `json:"id"`
				NIMMahasiswa string `json:"nim_mahasiswa"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, "NIM_T6_001", resp.Data[0].NIMMahasiswa)
	})

	t.Run("6. Mahasiswa B melihat daftar kosong sebelum mendaftar", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pendaftaran", nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []any `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data, "Mahasiswa B tidak boleh melihat pendaftaran Mahasiswa A")
	})

	t.Run("7. Mahasiswa B mendaftar -> 201 Created", func(t *testing.T) {
		body := map[string]any{
			"id_periode": periodeID,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("8. Mahasiswa B melihat hanya miliknya di list pendaftaran", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pendaftaran", nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []struct {
				NIMMahasiswa string `json:"nim_mahasiswa"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, "NIM_T6_002", resp.Data[0].NIMMahasiswa)
	})

	t.Run("9. Admin melihat total 2 pendaftaran", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pendaftaran", nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []any `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Data, 2)
	})
}
