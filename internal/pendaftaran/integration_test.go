package pendaftaran_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
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
	_, _ = pool.Exec(ctx, "DELETE FROM dokumen")
	_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
	_, _ = pool.Exec(ctx, "DELETE FROM kelompok_kkn")
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

func makeMultipartRequest(url, method, token, fieldName, filename string, fileContent []byte, extraFields map[string]string) *http.Request {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if filename != "" && len(fileContent) > 0 {
		part, _ := writer.CreateFormFile(fieldName, filename)
		_, _ = part.Write(fileContent)
	}

	for k, v := range extraFields {
		_ = writer.WriteField(k, v)
	}
	_ = writer.Close()

	req, _ := http.NewRequest(method, url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestIntegration_CabangMagang(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()

	// Bersihkan data
	_, _ = pool.Exec(ctx, "DELETE FROM log_aktivitas")
	_, _ = pool.Exec(ctx, "DELETE FROM dokumen")
	_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
	_, _ = pool.Exec(ctx, "DELETE FROM pembimbing_lapangan")
	_, _ = pool.Exec(ctx, "DELETE FROM tempat_program")
	_, _ = pool.Exec(ctx, "DELETE FROM kelompok_kkn")
	_, _ = pool.Exec(ctx, "DELETE FROM periode")
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM log_aktivitas")
		_, _ = pool.Exec(ctx, "DELETE FROM dokumen")
		_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
		_, _ = pool.Exec(ctx, "DELETE FROM pembimbing_lapangan")
		_, _ = pool.Exec(ctx, "DELETE FROM tempat_program")
		_, _ = pool.Exec(ctx, "DELETE FROM kelompok_kkn")
		_, _ = pool.Exec(ctx, "DELETE FROM periode")
	}()

	prodiID := ensureAkademik(t, pool)

	// Setup users
	uidMhs := setupUser(t, pool, "mhs_mg_1", "mhs_mg_1@undiksha.ac.id", "Pass123!", "mahasiswa")
	_ = ensureMahasiswaProfile(t, pool, uidMhs, prodiID, "NIM_MG_01", "Mhs Magang Satu")

	uidMhsLain := setupUser(t, pool, "mhs_mg_2", "mhs_mg_2@undiksha.ac.id", "Pass123!", "mahasiswa")
	_ = ensureMahasiswaProfile(t, pool, uidMhsLain, prodiID, "NIM_MG_02", "Mhs Magang Dua")

	uidPegawai := setupUser(t, pool, "pegawai_mg", "pegawai_mg@undiksha.ac.id", "Pass123!", "pegawai")
	_, _ = pool.Exec(ctx, `
		INSERT INTO pegawai (id_user, nip, nama, jabatan)
		VALUES ($1, 'NIP_PEG_MG', 'Pegawai Magang', 'Koordinator')
		ON CONFLICT (id_user) DO NOTHING
	`, uidPegawai)

	uidDosen := setupUser(t, pool, "dosen_mg", "dosen_mg@undiksha.ac.id", "Pass123!", "dosen")
	var idDosenPembimbing int
	err := pool.QueryRow(ctx, `
		INSERT INTO dosen (id_user, nip, nama, id_prodi)
		VALUES ($1, 'NIP_DOS_MG_1', 'Dr. Dosen Pembimbing', $2)
		ON CONFLICT (id_user) DO UPDATE SET nama = EXCLUDED.nama
		RETURNING id
	`, uidDosen, prodiID).Scan(&idDosenPembimbing)
	require.NoError(t, err)

	uidDosenPenguji := setupUser(t, pool, "dosen_uji", "dosen_uji@undiksha.ac.id", "Pass123!", "dosen")
	var idDosenPenguji int
	err = pool.QueryRow(ctx, `
		INSERT INTO dosen (id_user, nip, nama, id_prodi)
		VALUES ($1, 'NIP_DOS_MG_2', 'Dr. Dosen Penguji', $2)
		ON CONFLICT (id_user) DO UPDATE SET nama = EXCLUDED.nama
		RETURNING id
	`, uidDosenPenguji, prodiID).Scan(&idDosenPenguji)
	require.NoError(t, err)

	// Setup Tempat Program
	var idTempatMagang, idTempatAM, idTempatNonaktif, idTempatLain int
	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('magang', 'PT Teknologi Maju', 'Jl. Sudirman 1', 'Denpasar', true)
		RETURNING id
	`).Scan(&idTempatMagang)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('asistensi_mengajar', 'SMA Negeri 1 Singaraja', 'Jl. Veteran', 'Singaraja', true)
		RETURNING id
	`).Scan(&idTempatAM)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('magang', 'PT Tutup', 'Jl. Renon', 'Denpasar', false)
		RETURNING id
	`).Scan(&idTempatNonaktif)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('magang', 'PT Lain', 'Jl. Gatot Subroto', 'Denpasar', true)
		RETURNING id
	`).Scan(&idTempatLain)
	require.NoError(t, err)

	// Setup Pembimbing Lapangan
	var idMentorValid, idGuruPamong, idMentorLain int
	err = pool.QueryRow(ctx, `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, nama, jabatan)
		VALUES ('mentor', $1, 'Kak Mentor Budi', 'Lead Developer')
		RETURNING id
	`, idTempatMagang).Scan(&idMentorValid)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, nama, jabatan)
		VALUES ('guru_pamong', $1, 'Ibu Guru Ani', 'Guru Komputer')
		RETURNING id
	`, idTempatMagang).Scan(&idGuruPamong)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, nama, jabatan)
		VALUES ('mentor', $1, 'Kak Mentor Lain', 'Senior Engineer')
		RETURNING id
	`, idTempatLain).Scan(&idMentorLain)
	require.NoError(t, err)

	// Setup Periode Magang Aktif
	var periodeMagangID int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (
			nama, jenis_program, tahun_semester,
			tanggal_mulai, tanggal_selesai,
			tanggal_buka_daftar, tanggal_tutup_daftar,
			status
		) VALUES (
			'Magang Nasional Gasal 2026/2027', 'magang', '2026/2027-1',
			'2026-08-01', '2026-12-31',
			CURRENT_DATE - INTERVAL '1 day', CURRENT_DATE + INTERVAL '10 days',
			'aktif'
		) RETURNING id
	`).Scan(&periodeMagangID)
	require.NoError(t, err)

	uploadDir := t.TempDir()
	cfg := &config.Config{
		AppEnv:             "test",
		JWTSecret:          "kunci-rahasia-jwt-untuk-testing-harus-minimal-32-karakter!",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		UploadDir:          uploadDir,
		MaxUploadMB:        10,
		CORSAllowedOrigins: []string{"*"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	tokenMhs := getAuthToken(t, router, "mhs_mg_1", "Pass123!")
	tokenMhsLain := getAuthToken(t, router, "mhs_mg_2", "Pass123!")
	tokenPegawai := getAuthToken(t, router, "pegawai_mg", "Pass123!")

	var pendaftaranID int

	t.Run("1. Mahasiswa mendaftar magang -> 201 Created", func(t *testing.T) {
		body := map[string]any{"id_periode": periodeMagangID}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data struct {
				ID           int    `json:"id"`
				Status       string `json:"status"`
				JenisProgram string `json:"jenis_program"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		pendaftaranID = resp.Data.ID
		assert.Equal(t, "diajukan", resp.Data.Status)
		assert.Equal(t, "magang", resp.Data.JenisProgram)
	})

	t.Run("2. Penempatan magang - validasi BR-04 dan otorisasi", func(t *testing.T) {
		url := fmt.Sprintf("/pendaftaran/%d/penempatan-magang", pendaftaranID)

		// 2a. Bukan pemilik pendaftaran -> 404 (ARCHITECTURE §6)
		payloadValid := map[string]any{
			"id_tempat_program":      idTempatMagang,
			"id_pembimbing_lapangan": idMentorValid,
		}
		raw, _ := json.Marshal(payloadValid)
		req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhsLain)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// 2b. BR-04: Tempat berjenis asistensi_mengajar -> 422 TEMPAT_TIDAK_VALID
		payloadAM := map[string]any{
			"id_tempat_program":      idTempatAM,
			"id_pembimbing_lapangan": idMentorValid,
		}
		raw, _ = json.Marshal(payloadAM)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "TEMPAT_TIDAK_VALID")

		// 2c. BR-04: Tempat non-aktif -> 422 TEMPAT_TIDAK_VALID
		payloadNonaktif := map[string]any{
			"id_tempat_program":      idTempatNonaktif,
			"id_pembimbing_lapangan": idMentorValid,
		}
		raw, _ = json.Marshal(payloadNonaktif)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "TEMPAT_TIDAK_VALID")

		// 2d. BR-04: Pembimbing berjenis guru_pamong -> 422 PEMBIMBING_TIDAK_VALID
		payloadGuru := map[string]any{
			"id_tempat_program":      idTempatMagang,
			"id_pembimbing_lapangan": idGuruPamong,
		}
		raw, _ = json.Marshal(payloadGuru)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "PEMBIMBING_TIDAK_VALID")

		// 2e. BR-04: Mentor terdaftar di tempat lain -> 422 PEMBIMBING_TIDAK_VALID
		payloadMentorLain := map[string]any{
			"id_tempat_program":      idTempatMagang,
			"id_pembimbing_lapangan": idMentorLain,
		}
		raw, _ = json.Marshal(payloadMentorLain)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "PEMBIMBING_TIDAK_VALID")

		// 2f. Sukses penempatan valid -> 200 OK
		raw, _ = json.Marshal(payloadValid)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// 2g. Pengubahan ulang saat masih diajukan diizinkan -> 200 OK
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("3. Penetapan dosen pembimbing - validasi BR-07, role dan audit log", func(t *testing.T) {
		url := fmt.Sprintf("/pendaftaran/%d/dosen-pembimbing", pendaftaranID)

		// 3a. Mahasiswa mencoba memanggil endpoint ini -> 403 Forbidden
		bodyDosen := map[string]any{"id_dosen": idDosenPembimbing}
		raw, _ := json.Marshal(bodyDosen)
		req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		// 3b. Dosen tidak ditemukan -> 422 DOSEN_TIDAK_DITEMUKAN
		bodyNotFound := map[string]any{"id_dosen": 999999}
		raw, _ = json.Marshal(bodyNotFound)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenPegawai)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "DOSEN_TIDAK_DITEMUKAN")

		// 3c. Pegawai menetapkan dosen pembimbing valid -> 200 OK
		raw, _ = json.Marshal(bodyDosen)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenPegawai)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// 3d. Verifikasi log_aktivitas
		var logCount int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM log_aktivitas
			WHERE action = 'tetapkan_dosen_pembimbing' AND id_user = $1
		`, uidPegawai).Scan(&logCount)
		require.NoError(t, err)
		assert.Equal(t, 1, logCount)
	})

	t.Run("4. Cek detail dan kesiapan magang belum siap karena dokumen belum diunggah", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pendaftaran/%d", pendaftaranID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				Status   string `json:"status"`
				Kesiapan struct {
					Siap       bool     `json:"siap"`
					Kekurangan []string `json:"kekurangan"`
				} `json:"kesiapan"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "diajukan", resp.Data.Status)
		assert.False(t, resp.Data.Kesiapan.Siap)
		assert.Contains(t, resp.Data.Kesiapan.Kekurangan, "dokumen mou belum diunggah")
		assert.Contains(t, resp.Data.Kesiapan.Kekurangan, "dokumen proposal belum diunggah")
	})

	t.Run("5. Unggah dokumen mou dan proposal -> pemicu transisi otomatis ke berjalan", func(t *testing.T) {
		pdfValid := []byte("%PDF-1.4 dummy pdf content for testing transition")

		// 5a. Unggah MoU
		reqMou := makeMultipartRequest(
			fmt.Sprintf("/pendaftaran/%d/dokumen", pendaftaranID),
			http.MethodPost,
			tokenMhs,
			"file",
			"mou.pdf",
			pdfValid,
			map[string]string{"jenis_dokumen": "mou"},
		)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, reqMou)
		require.Equal(t, http.StatusCreated, w.Code)

		// Periksa status: masih diajukan (proposal belum)
		var status string
		err := pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "diajukan", status)

		// 5b. Unggah Proposal -> harus memicu transisi ke berjalan!
		reqProp := makeMultipartRequest(
			fmt.Sprintf("/pendaftaran/%d/dokumen", pendaftaranID),
			http.MethodPost,
			tokenMhs,
			"file",
			"proposal.pdf",
			pdfValid,
			map[string]string{"jenis_dokumen": "proposal"},
		)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, reqProp)
		require.Equal(t, http.StatusCreated, w.Code)

		// Periksa status di database: sekarang harus 'berjalan'
		err = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", status)

		// Cek detail lewat GET endpoint
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pendaftaran/%d", pendaftaranID), nil)
		reqGet.Header.Set("Authorization", "Bearer "+tokenMhs)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, reqGet)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				Status   string `json:"status"`
				Kesiapan struct {
					Siap bool `json:"siap"`
				} `json:"kesiapan"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", resp.Data.Status)
		assert.True(t, resp.Data.Kesiapan.Siap)
	})

	t.Run("6. Mahasiswa mencoba mengubah penempatan saat berstatus berjalan -> 422 STATUS_TIDAK_VALID", func(t *testing.T) {
		payloadValid := map[string]any{
			"id_tempat_program":      idTempatMagang,
			"id_pembimbing_lapangan": idMentorValid,
		}
		raw, _ := json.Marshal(payloadValid)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/pendaftaran/%d/penempatan-magang", pendaftaranID), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "STATUS_TIDAK_VALID")
	})
}

func TestIntegration_CabangAM(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()

	// Bersihkan data
	_, _ = pool.Exec(ctx, "DELETE FROM log_aktivitas")
	_, _ = pool.Exec(ctx, "DELETE FROM dokumen")
	_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
	_, _ = pool.Exec(ctx, "DELETE FROM pembimbing_lapangan")
	_, _ = pool.Exec(ctx, "DELETE FROM tempat_program")
	_, _ = pool.Exec(ctx, "DELETE FROM kelompok_kkn")
	_, _ = pool.Exec(ctx, "DELETE FROM periode")
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM log_aktivitas")
		_, _ = pool.Exec(ctx, "DELETE FROM dokumen")
		_, _ = pool.Exec(ctx, "DELETE FROM peserta_program")
		_, _ = pool.Exec(ctx, "DELETE FROM pembimbing_lapangan")
		_, _ = pool.Exec(ctx, "DELETE FROM tempat_program")
		_, _ = pool.Exec(ctx, "DELETE FROM kelompok_kkn")
		_, _ = pool.Exec(ctx, "DELETE FROM periode")
	}()

	prodiID := ensureAkademik(t, pool)

	// Setup users
	uidMhs := setupUser(t, pool, "mhs_am_1", "mhs_am_1@undiksha.ac.id", "Pass123!", "mahasiswa")
	_ = ensureMahasiswaProfile(t, pool, uidMhs, prodiID, "NIM_AM_01", "Mhs AM Satu")

	uidPegawai := setupUser(t, pool, "pegawai_am", "pegawai_am@undiksha.ac.id", "Pass123!", "pegawai")
	_, _ = pool.Exec(ctx, `
		INSERT INTO pegawai (id_user, nip, nama, jabatan)
		VALUES ($1, 'NIP_PEG_AM', 'Pegawai AM', 'Koordinator')
		ON CONFLICT (id_user) DO NOTHING
	`, uidPegawai)

	uidAdmin := setupUser(t, pool, "admin_am", "admin_am@undiksha.ac.id", "Pass123!", "admin")
	_, _ = pool.Exec(ctx, `
		INSERT INTO admin (id_user, nip, nama)
		VALUES ($1, 'NIP_ADM_AM', 'Admin AM')
		ON CONFLICT (id_user) DO NOTHING
	`, uidAdmin)

	uidDosenPembimbing := setupUser(t, pool, "dosen_am_pembimbing", "dosen_am_pembimbing@undiksha.ac.id", "Pass123!", "dosen")
	var idDosenPembimbing int
	err := pool.QueryRow(ctx, `
		INSERT INTO dosen (id_user, nip, nama, id_prodi)
		VALUES ($1, 'NIP_DOS_AM_1', 'Dr. Dosen AM Pembimbing', $2)
		ON CONFLICT (id_user) DO UPDATE SET nama = EXCLUDED.nama
		RETURNING id
	`, uidDosenPembimbing, prodiID).Scan(&idDosenPembimbing)
	require.NoError(t, err)

	uidDosenPenguji := setupUser(t, pool, "dosen_am_penguji", "dosen_am_penguji@undiksha.ac.id", "Pass123!", "dosen")
	var idDosenPenguji int
	err = pool.QueryRow(ctx, `
		INSERT INTO dosen (id_user, nip, nama, id_prodi)
		VALUES ($1, 'NIP_DOS_AM_2', 'Dr. Dosen AM Penguji', $2)
		ON CONFLICT (id_user) DO UPDATE SET nama = EXCLUDED.nama
		RETURNING id
	`, uidDosenPenguji, prodiID).Scan(&idDosenPenguji)
	require.NoError(t, err)

	// Setup Tempat Program
	var idTempatAM, idTempatMagang, idTempatNonaktif, idTempatAMLain int
	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('asistensi_mengajar', 'SMP Negeri 1 Singaraja', 'Jl. Veteran 1', 'Singaraja', true)
		RETURNING id
	`).Scan(&idTempatAM)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('magang', 'PT Solusi AM', 'Jl. Sudirman 5', 'Denpasar', true)
		RETURNING id
	`).Scan(&idTempatMagang)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('asistensi_mengajar', 'SMP Tutup', 'Jl. Renon', 'Denpasar', false)
		RETURNING id
	`).Scan(&idTempatNonaktif)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, is_aktif)
		VALUES ('asistensi_mengajar', 'SMA Negeri 2 Singaraja', 'Jl. Pramuka', 'Singaraja', true)
		RETURNING id
	`).Scan(&idTempatAMLain)
	require.NoError(t, err)

	// Setup Pembimbing Lapangan
	var idGuruPamongValid, idMentor, idGuruPamongLain int
	err = pool.QueryRow(ctx, `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, nama, jabatan)
		VALUES ('guru_pamong', $1, 'Ibu Guru Wayan', 'Guru Pembimbing')
		RETURNING id
	`, idTempatAM).Scan(&idGuruPamongValid)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, nama, jabatan)
		VALUES ('mentor', $1, 'Pak Mentor Industri', 'Mentor')
		RETURNING id
	`, idTempatAM).Scan(&idMentor)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, nama, jabatan)
		VALUES ('guru_pamong', $1, 'Pak Guru Made', 'Guru Pamong Sekolah Lain')
		RETURNING id
	`, idTempatAMLain).Scan(&idGuruPamongLain)
	require.NoError(t, err)

	// Setup Periode Asistensi Mengajar Aktif
	var periodeAMID int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (
			nama, jenis_program, tahun_semester,
			tanggal_mulai, tanggal_selesai,
			tanggal_buka_daftar, tanggal_tutup_daftar,
			status
		) VALUES (
			'Asistensi Mengajar Semester Gasal 2026/2027', 'asistensi_mengajar', '2026/2027-1',
			'2026-08-01', '2026-12-31',
			CURRENT_DATE - INTERVAL '1 day', CURRENT_DATE + INTERVAL '10 days',
			'aktif'
		) RETURNING id
	`).Scan(&periodeAMID)
	require.NoError(t, err)

	uploadDir := t.TempDir()
	cfg := &config.Config{
		AppEnv:             "test",
		JWTSecret:          "kunci-rahasia-jwt-untuk-testing-harus-minimal-32-karakter!",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		UploadDir:          uploadDir,
		MaxUploadMB:        10,
		CORSAllowedOrigins: []string{"*"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	tokenMhs := getAuthToken(t, router, "mhs_am_1", "Pass123!")
	tokenPegawai := getAuthToken(t, router, "pegawai_am", "Pass123!")
	tokenAdmin := getAuthToken(t, router, "admin_am", "Pass123!")

	var pendaftaranID int

	t.Run("1. Mahasiswa mendaftar Asistensi Mengajar -> 201 Created", func(t *testing.T) {
		body := map[string]any{"id_periode": periodeAMID}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data struct {
				ID           int    `json:"id"`
				Status       string `json:"status"`
				JenisProgram string `json:"jenis_program"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		pendaftaranID = resp.Data.ID
		assert.Equal(t, "diajukan", resp.Data.Status)
		assert.Equal(t, "asistensi_mengajar", resp.Data.JenisProgram)
	})

	t.Run("2. Pemetaan sekolah penempatan dan guru pamong oleh Admin - validasi BR-05 dan hak akses", func(t *testing.T) {
		url := fmt.Sprintf("/admin/pendaftaran/%d/penempatan", pendaftaranID)

		// 2a. Pegawai mencoba memanggil endpoint admin penempatan -> 403 Forbidden
		payloadValid := map[string]any{
			"id_tempat_program":      idTempatAM,
			"id_pembimbing_lapangan": idGuruPamongValid,
		}
		raw, _ := json.Marshal(payloadValid)
		req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenPegawai)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		// 2b. Mahasiswa mencoba memanggil endpoint admin penempatan -> 403 Forbidden
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		// 2c. BR-05: Tempat berjenis magang -> 422 TEMPAT_TIDAK_VALID
		payloadMagang := map[string]any{
			"id_tempat_program":      idTempatMagang,
			"id_pembimbing_lapangan": idGuruPamongValid,
		}
		raw, _ = json.Marshal(payloadMagang)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "TEMPAT_TIDAK_VALID")

		// 2d. BR-05: Tempat non-aktif -> 422 TEMPAT_TIDAK_VALID
		payloadNonaktif := map[string]any{
			"id_tempat_program":      idTempatNonaktif,
			"id_pembimbing_lapangan": idGuruPamongValid,
		}
		raw, _ = json.Marshal(payloadNonaktif)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "TEMPAT_TIDAK_VALID")

		// 2e. BR-05: Pembimbing lapangan berjenis mentor -> 422 PEMBIMBING_TIDAK_VALID
		payloadMentor := map[string]any{
			"id_tempat_program":      idTempatAM,
			"id_pembimbing_lapangan": idMentor,
		}
		raw, _ = json.Marshal(payloadMentor)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "PEMBIMBING_TIDAK_VALID")

		// 2f. BR-05: Guru pamong milik sekolah penempatan lain -> 422 PEMBIMBING_TIDAK_VALID
		payloadGuruLain := map[string]any{
			"id_tempat_program":      idTempatAM,
			"id_pembimbing_lapangan": idGuruPamongLain,
		}
		raw, _ = json.Marshal(payloadGuruLain)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "PEMBIMBING_TIDAK_VALID")

		// 2g. Admin sukses memetakan sekolah dan guru pamong valid -> 200 OK
		raw, _ = json.Marshal(payloadValid)
		req = httptest.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("3. Admin menetapkan dosen pembimbing dan dosen penguji - validasi BR-07", func(t *testing.T) {
		urlPembimbing := fmt.Sprintf("/pendaftaran/%d/dosen-pembimbing", pendaftaranID)
		urlPenguji := fmt.Sprintf("/admin/pendaftaran/%d/dosen-penguji", pendaftaranID)

		// 3a. Admin tetapkan dosen pembimbing
		bodyPembimbing := map[string]any{"id_dosen": idDosenPembimbing}
		raw, _ := json.Marshal(bodyPembimbing)
		req := httptest.NewRequest(http.MethodPut, urlPembimbing, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// 3b. BR-07: Coba tetapkan dosen penguji dengan id_dosen yang sama dengan dosen pembimbing -> 422
		bodySamePenguji := map[string]any{"id_dosen": idDosenPembimbing}
		raw, _ = json.Marshal(bodySamePenguji)
		req = httptest.NewRequest(http.MethodPut, urlPenguji, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI")

		// 3c. Admin tetapkan dosen penguji valid
		bodyPenguji := map[string]any{"id_dosen": idDosenPenguji}
		raw, _ = json.Marshal(bodyPenguji)
		req = httptest.NewRequest(http.MethodPut, urlPenguji, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// 3d. Verifikasi audit log untuk kedua aksi
		var logPengujiCount int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM log_aktivitas
			WHERE action = 'tetapkan_dosen_penguji' AND id_user = $1
		`, uidAdmin).Scan(&logPengujiCount)
		require.NoError(t, err)
		assert.Equal(t, 1, logPengujiCount)
	})

	t.Run("4. Cek kesiapan AM: belum siap karena dokumen proker belum diunggah", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pendaftaran/%d", pendaftaranID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhs)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				Status   string `json:"status"`
				Kesiapan struct {
					Siap       bool     `json:"siap"`
					Kekurangan []string `json:"kekurangan"`
				} `json:"kesiapan"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "diajukan", resp.Data.Status)
		assert.False(t, resp.Data.Kesiapan.Siap)
		assert.Contains(t, resp.Data.Kesiapan.Kekurangan, "dokumen proker belum diunggah")
	})

	t.Run("5. Mahasiswa mengunggah dokumen proker -> transisi otomatis ke status berjalan", func(t *testing.T) {
		pdfValid := []byte("%PDF-1.4 dummy pdf content for proker AM testing")

		reqProker := makeMultipartRequest(
			fmt.Sprintf("/pendaftaran/%d/dokumen", pendaftaranID),
			http.MethodPost,
			tokenMhs,
			"file",
			"proker_am.pdf",
			pdfValid,
			map[string]string{"jenis_dokumen": "proker"},
		)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, reqProker)
		require.Equal(t, http.StatusCreated, w.Code)

		// Periksa status di database: sekarang harus 'berjalan'
		var status string
		err := pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", status)

		// Cek detail lewat GET endpoint
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pendaftaran/%d", pendaftaranID), nil)
		reqGet.Header.Set("Authorization", "Bearer "+tokenMhs)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, reqGet)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				Status   string `json:"status"`
				Kesiapan struct {
					Siap bool `json:"siap"`
				} `json:"kesiapan"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", resp.Data.Status)
		assert.True(t, resp.Data.Kesiapan.Siap)
	})

	t.Run("6. Admin mencoba mengubah penempatan saat berstatus berjalan -> 422 STATUS_TIDAK_VALID", func(t *testing.T) {
		payloadValid := map[string]any{
			"id_tempat_program":      idTempatAM,
			"id_pembimbing_lapangan": idGuruPamongValid,
		}
		raw, _ := json.Marshal(payloadValid)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/pendaftaran/%d/penempatan", pendaftaranID), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "STATUS_TIDAK_VALID")
	})
}
