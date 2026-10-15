package periode_test

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

func ensureProdi(t *testing.T, pool *pgxpool.Pool) int {
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

// TestIntegration_PeriodeLifecycle menguji siklus lengkap periode:
// create draft -> update draft -> aktifkan -> selesaikan (dengan peserta_program),
// serta hak akses mahasiswa vs admin.
func TestIntegration_PeriodeLifecycle(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()

	// Bersihkan data periode dan peserta sebelumnya di database test
	_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
	_, _ = pool.Exec(ctx, "DELETE FROM periode")

	prodiID := ensureProdi(t, pool)

	// 1. Setup Admin dengan profil Pegawai
	adminWithPegawaiID := setupUser(t, pool, "admin_pegawai_t5", "admin_pegawai_t5@undiksha.ac.id", "AdminPass123!", "admin")
	var pegawaiID int
	err := pool.QueryRow(ctx, "SELECT id FROM pegawai WHERE id_user = $1", adminWithPegawaiID).Scan(&pegawaiID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO pegawai (id_user, nama, nip, jabatan)
			VALUES ($1, 'Admin Pegawai T5', 'PEG_T5_001', 'Staff Akademik')
			RETURNING id
		`, adminWithPegawaiID).Scan(&pegawaiID)
		require.NoError(t, err)
	}

	// 2. Setup Admin Murni (tanpa profil pegawai)
	adminPureID := setupUser(t, pool, "admin_pure_t5", "admin_pure_t5@undiksha.ac.id", "AdminPass123!", "admin")
	_, _ = pool.Exec(ctx, "DELETE FROM pegawai WHERE id_user = $1", adminPureID)
	_, _ = pool.Exec(ctx, "INSERT INTO admin (id_user, nama, nip) VALUES ($1, 'Admin Pure', 'ADM_PURE_001') ON CONFLICT (id_user) DO NOTHING", adminPureID)

	// 3. Setup Mahasiswa
	mhsUserID := setupUser(t, pool, "mhs_t5", "mhs_t5@undiksha.ac.id", "MhsPass123!", "mahasiswa")
	var mhs1ID int
	err = pool.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", mhsUserID).Scan(&mhs1ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO mahasiswa (id_user, id_prodi, nim, nama, angkatan)
			VALUES ($1, $2, 'NIM_T5_001', 'Mahasiswa T5', 2023)
			RETURNING id
		`, mhsUserID, prodiID).Scan(&mhs1ID)
		require.NoError(t, err)
	}

	// Mahasiswa kedua untuk data peserta
	mhs2UserID := setupUser(t, pool, "mhs_t5_2", "mhs_t5_2@undiksha.ac.id", "MhsPass123!", "mahasiswa")
	var mhs2ID int
	err = pool.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", mhs2UserID).Scan(&mhs2ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO mahasiswa (id_user, id_prodi, nim, nama, angkatan)
			VALUES ($1, $2, 'NIM_T5_002', 'Mahasiswa T5 Kedua', 2023)
			RETURNING id
		`, mhs2UserID, prodiID).Scan(&mhs2ID)
		require.NoError(t, err)
	}

	// Mahasiswa ketiga untuk periode lain
	mhs3UserID := setupUser(t, pool, "mhs_t5_3", "mhs_t5_3@undiksha.ac.id", "MhsPass123!", "mahasiswa")
	var mhs3ID int
	err = pool.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", mhs3UserID).Scan(&mhs3ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO mahasiswa (id_user, id_prodi, nim, nama, angkatan)
			VALUES ($1, $2, 'NIM_T5_003', 'Mahasiswa T5 Ketiga', 2023)
			RETURNING id
		`, mhs3UserID, prodiID).Scan(&mhs3ID)
		require.NoError(t, err)
	}

	cfg := &config.Config{
		AppEnv:             "test",
		JWTSecret:          "kunci-rahasia-jwt-untuk-testing-harus-minimal-32-karakter!",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		CORSAllowedOrigins: []string{"*"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	tokenAdminPegawai := getAuthToken(t, router, "admin_pegawai_t5", "AdminPass123!")
	tokenAdminPure := getAuthToken(t, router, "admin_pure_t5", "AdminPass123!")
	tokenMahasiswa := getAuthToken(t, router, "mhs_t5", "MhsPass123!")

	var periode1ID int
	var periode2ID int

	t.Run("1. Admin dengan profil pegawai membuat periode -> dibuat_oleh terisi id_pegawai", func(t *testing.T) {
		createBody := map[string]any{
			"nama":                 "Magang Semester Ganjil 2026/2027",
			"jenis_program":        "magang",
			"tahun_semester":       "2026/2027-1",
			"tanggal_mulai":        "2026-08-01",
			"tanggal_selesai":      "2026-12-31",
			"tanggal_buka_daftar":  "2026-06-01",
			"tanggal_tutup_daftar": "2026-07-20",
		}
		raw, _ := json.Marshal(createBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/periode", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdminPegawai)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "respons: %s", w.Body.String())

		var resp struct {
			Data struct {
				ID         int    `json:"id"`
				Nama       string `json:"nama"`
				Status     string `json:"status"`
				DibuatOleh *int   `json:"dibuat_oleh"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		periode1ID = resp.Data.ID
		assert.Equal(t, "draft", resp.Data.Status)
		require.NotNil(t, resp.Data.DibuatOleh, "RULES.md §7 #2: dibuat_oleh harus terisi untuk pembuat berprofil pegawai")
		assert.Equal(t, pegawaiID, *resp.Data.DibuatOleh)
	})

	t.Run("2. Admin murni (tanpa pegawai) membuat periode -> dibuat_oleh adalah NULL", func(t *testing.T) {
		createBody := map[string]any{
			"nama":            "KKN Periode 2 2026",
			"jenis_program":   "kkn",
			"tahun_semester":  "2026/2027-1",
			"tanggal_mulai":   "2026-07-01",
			"tanggal_selesai": "2026-08-31",
		}
		raw, _ := json.Marshal(createBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/periode", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdminPure)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "respons: %s", w.Body.String())

		var resp struct {
			Data struct {
				ID         int  `json:"id"`
				DibuatOleh *int `json:"dibuat_oleh"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		periode2ID = resp.Data.ID
		assert.Nil(t, resp.Data.DibuatOleh, "RULES.md §7 #2: dibuat_oleh harus NULL untuk admin tanpa profil pegawai")
	})

	t.Run("3. Admin mengubah periode saat masih draft -> sukses 200", func(t *testing.T) {
		updateBody := map[string]any{
			"nama":                 "Magang Semester Ganjil 2026/2027 (Revisi)",
			"tahun_semester":       "2026/2027-1",
			"tanggal_mulai":        "2026-08-15",
			"tanggal_selesai":      "2026-12-31",
			"tanggal_buka_daftar":  "2026-06-01",
			"tanggal_tutup_daftar": "2026-08-01",
		}
		raw, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/periode/%d", periode1ID), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdminPegawai)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "respons: %s", w.Body.String())

		var resp struct {
			Data struct {
				Nama string `json:"nama"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Magang Semester Ganjil 2026/2027 (Revisi)", resp.Data.Nama)
	})

	t.Run("4. Mahasiswa mencoba update periode -> 403 Forbidden", func(t *testing.T) {
		updateBody := map[string]any{
			"nama":            "Hacking Periode",
			"tahun_semester":  "2026/2027-1",
			"tanggal_mulai":   "2026-08-15",
			"tanggal_selesai": "2026-12-31",
		}
		raw, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/periode/%d", periode1ID), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMahasiswa)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("5. Mahasiswa GET /periode saat semua periode masih draft -> daftar kosong", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/periode", nil)
		req.Header.Set("Authorization", "Bearer "+tokenMahasiswa)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []any `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data, "PRD 5.3: mahasiswa tidak boleh melihat periode draft")
	})

	t.Run("6. Mahasiswa GET /periode/:id untuk periode draft -> 404 Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/periode/%d", periode1ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMahasiswa)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code, "PRD 5.3: detail periode draft tersembunyi bagi mahasiswa")
	})

	t.Run("7. Admin mengaktifkan periode -> status menjadi aktif dan tercatat di log_aktivitas", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/periode/%d/aktifkan", periode1ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdminPegawai)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "respons: %s", w.Body.String())

		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "aktif", resp.Data.Status)

		// Verifikasi log_aktivitas
		var logCount int
		err = pool.QueryRow(ctx, "SELECT count(*) FROM log_aktivitas WHERE id_user = $1 AND action = 'aktifkan_periode'", adminWithPegawaiID).Scan(&logCount)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, logCount, 1)
	})

	t.Run("8. Mahasiswa GET /periode sekarang melihat periode yang berstatus aktif", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/periode", nil)
		req.Header.Set("Authorization", "Bearer "+tokenMahasiswa)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, periode1ID, resp.Data[0].ID)
		assert.Equal(t, "aktif", resp.Data[0].Status)
	})

	t.Run("9. Admin mencoba update periode setelah aktif -> ditolak 422 PERIODE_BUKAN_DRAFT", func(t *testing.T) {
		updateBody := map[string]any{
			"nama":            "Ubah Saat Aktif",
			"tahun_semester":  "2026/2027-1",
			"tanggal_mulai":   "2026-08-15",
			"tanggal_selesai": "2026-12-31",
		}
		raw, _ := json.Marshal(updateBody)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/periode/%d", periode1ID), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdminPegawai)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("10. Selesaikan periode secara atomik: peserta berjalan->selesai, diajukan->ditolak, periode lain tidak tersentuh", func(t *testing.T) {
		// Sisipkan data peserta dummy untuk periode1
		var pesertaBerjalanP1ID, pesertaDiajukanP1ID int
		err := pool.QueryRow(ctx, `
			INSERT INTO peserta_program (id_mahasiswa, id_periode, jenis_program, status)
			VALUES ($1, $2, 'magang', 'berjalan')
			RETURNING id
		`, mhs1ID, periode1ID).Scan(&pesertaBerjalanP1ID)
		require.NoError(t, err)

		err = pool.QueryRow(ctx, `
			INSERT INTO peserta_program (id_mahasiswa, id_periode, jenis_program, status)
			VALUES ($1, $2, 'magang', 'diajukan')
			RETURNING id
		`, mhs2ID, periode1ID).Scan(&pesertaDiajukanP1ID)
		require.NoError(t, err)

		// Sisipkan data peserta dummy untuk periode2 (KKN, draft)
		var pesertaP2ID int
		err = pool.QueryRow(ctx, `
			INSERT INTO peserta_program (id_mahasiswa, id_periode, jenis_program, status)
			VALUES ($1, $2, 'kkn', 'berjalan')
			RETURNING id
		`, mhs3ID, periode2ID).Scan(&pesertaP2ID)
		require.NoError(t, err)

		// Selesaikan periode 1
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/periode/%d/selesaikan", periode1ID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdminPegawai)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "respons: %s", w.Body.String())

		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "selesai", resp.Data.Status)

		// Verifikasi status peserta di database:
		// 1. Peserta p1 'berjalan' -> 'selesai'
		var statusP1Berjalan string
		err = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pesertaBerjalanP1ID).Scan(&statusP1Berjalan)
		require.NoError(t, err)
		assert.Equal(t, "selesai", statusP1Berjalan)

		// 2. Peserta p1 'diajukan' -> 'ditolak' dengan alasan_tolak (ASUMSI A12)
		var statusP1Diajukan string
		var alasanTolakP1 *string
		err = pool.QueryRow(ctx, "SELECT status, alasan_tolak FROM peserta_program WHERE id = $1", pesertaDiajukanP1ID).Scan(&statusP1Diajukan, &alasanTolakP1)
		require.NoError(t, err)
		assert.Equal(t, "ditolak", statusP1Diajukan)
		require.NotNil(t, alasanTolakP1)
		assert.NotEmpty(t, *alasanTolakP1)

		// 3. Periode 2 dan pesertanya sama sekali tidak terpengaruh
		var statusPeriode2 string
		err = pool.QueryRow(ctx, "SELECT status FROM periode WHERE id = $1", periode2ID).Scan(&statusPeriode2)
		require.NoError(t, err)
		assert.Equal(t, "draft", statusPeriode2, "periode 2 harus tetap draft")

		var statusP2 string
		err = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pesertaP2ID).Scan(&statusP2)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", statusP2, "peserta di periode 2 tidak boleh berubah")

		// 4. Verifikasi log_aktivitas selesaikan_periode
		var logCount int
		err = pool.QueryRow(ctx, "SELECT count(*) FROM log_aktivitas WHERE id_user = $1 AND action = 'selesaikan_periode'", adminWithPegawaiID).Scan(&logCount)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, logCount, 1)
	})
}
