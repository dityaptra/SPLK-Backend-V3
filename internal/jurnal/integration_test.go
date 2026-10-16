package jurnal_test

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

func ensureDosenProfile(t *testing.T, pool *pgxpool.Pool, userID, prodiID int, nip, nama string) int {
	ctx := context.Background()
	var dosenID int
	err := pool.QueryRow(ctx, "SELECT id FROM dosen WHERE id_user = $1", userID).Scan(&dosenID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO dosen (id_user, id_prodi, nip, nama)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, userID, prodiID, nip, nama).Scan(&dosenID)
		require.NoError(t, err)
	} else {
		require.NoError(t, err)
	}
	return dosenID
}

func ensurePembimbingLapangan(t *testing.T, pool *pgxpool.Pool, idTempat int, idUser int, nama, jabatan string) int {
	ctx := context.Background()
	var plID int
	err := pool.QueryRow(ctx, "SELECT id FROM pembimbing_lapangan WHERE id_user = $1", idUser).Scan(&plID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, id_user, nama, jabatan, kontak)
			VALUES ('guru_pamong', $1, $2, $3, $4, '081299998888')
			RETURNING id
		`, idTempat, idUser, nama, jabatan).Scan(&plID)
		require.NoError(t, err)
	} else {
		_, err = pool.Exec(ctx, "UPDATE pembimbing_lapangan SET id_tempat_program = $1 WHERE id = $2", idTempat, plID)
		require.NoError(t, err)
	}
	return plID
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

func executeRequest(router *gin.Engine, method, url, token string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestIntegration_JurnalLogbookSelesaikan(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()

	cfg := &config.Config{
		AppEnv:        "development",
		AppPort:       "8080",
		DatabaseURL:   os.Getenv("TEST_DATABASE_URL"),
		JWTSecret:     "super-secret-jwt-key-for-test-32bytes!",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 24 * time.Hour,
		UploadDir:     t.TempDir(),
		MaxUploadMB:   5,
		LogLevel:      "warn",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	prodiID := ensureAkademik(t, pool)

	// Setup users
	_ = setupUser(t, pool, "admin_jurnal", "admin_jurnal@undiksha.ac.id", "pass12345", "admin")
	uidMhs1 := setupUser(t, pool, "mhs_jurnal_1", "mhs1_jurnal@undiksha.ac.id", "pass12345", "mahasiswa")
	uidMhs2 := setupUser(t, pool, "mhs_jurnal_2", "mhs2_jurnal@undiksha.ac.id", "pass12345", "mahasiswa")
	uidDos1 := setupUser(t, pool, "dos_jurnal_1", "dos1_jurnal@undiksha.ac.id", "pass12345", "dosen")
	uidDos2 := setupUser(t, pool, "dos_jurnal_2", "dos2_jurnal@undiksha.ac.id", "pass12345", "dosen")
	uidPl1 := setupUser(t, pool, "pl_jurnal_1", "pl1_jurnal@undiksha.ac.id", "pass12345", "pembimbing_lapangan")

	mhs1ID := ensureMahasiswaProfile(t, pool, uidMhs1, prodiID, "2300000001", "Mahasiswa Jurnal 1")
	ensureMahasiswaProfile(t, pool, uidMhs2, prodiID, "2300000002", "Mahasiswa Jurnal 2")
	dos1ID := ensureDosenProfile(t, pool, uidDos1, prodiID, "1980000001", "Dosen Pembimbing")
	ensureDosenProfile(t, pool, uidDos2, prodiID, "1980000002", "Dosen Lain")

	tokenAdmin := getAuthToken(t, router, "admin_jurnal", "pass12345")
	tokenMhs1 := getAuthToken(t, router, "mhs_jurnal_1", "pass12345")
	tokenMhs2 := getAuthToken(t, router, "mhs_jurnal_2", "pass12345")
	tokenDos1 := getAuthToken(t, router, "dos_jurnal_1", "pass12345")
	tokenDos2 := getAuthToken(t, router, "dos_jurnal_2", "pass12345")
	tokenPl1 := getAuthToken(t, router, "pl_jurnal_1", "pass12345")

	// Setup Tempat Program
	var idTempat int
	err := pool.QueryRow(ctx, `
		INSERT INTO tempat_program (nama, jenis_program, alamat, kota, kontak, is_aktif)
		VALUES ('SMA Negeri Jurnal Test', 'asistensi_mengajar', 'Jl. Pendidikan', 'Singaraja', 'Kepsek', true)
		RETURNING id
	`).Scan(&idTempat)
	require.NoError(t, err)

	pl1ID := ensurePembimbingLapangan(t, pool, idTempat, uidPl1, "Guru Pamong Jurnal", "Guru Pamong")

	// Setup Periode Aktif (rentang 30 hari lalu sampai 30 hari kedepan)
	tNow := time.Now()
	tMulai := tNow.AddDate(0, -1, 0)
	tSelesai := tNow.AddDate(0, 1, 0)

	var idPeriode int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai, status)
		VALUES ('Periode AM Jurnal Test', 'asistensi_mengajar', '2026/2027 Ganjil', $1, $2, 'aktif')
		RETURNING id
	`, tMulai, tSelesai).Scan(&idPeriode)
	require.NoError(t, err)

	// Setup Peserta Program (status 'berjalan')
	var idPendaftaran int
	err = pool.QueryRow(ctx, `
		INSERT INTO peserta_program (id_periode, jenis_program, id_mahasiswa, id_tempat_program, id_dosen_pembimbing, id_pembimbing_lapangan, status)
		VALUES ($1, 'asistensi_mengajar', $2, $3, $4, $5, 'berjalan')
		RETURNING id
	`, idPeriode, mhs1ID, idTempat, dos1ID, pl1ID).Scan(&idPendaftaran)
	require.NoError(t, err)

	defer func() {
		// Cleanup order
		_, _ = pool.Exec(ctx, "DELETE FROM jurnal_harian WHERE id_peserta_program = $1", idPendaftaran)
		_, _ = pool.Exec(ctx, "DELETE FROM logbook_bimbingan WHERE id_peserta_program = $1", idPendaftaran)
		_, _ = pool.Exec(ctx, "DELETE FROM peserta_program WHERE id = $1", idPendaftaran)
		_, _ = pool.Exec(ctx, "DELETE FROM periode WHERE id = $1", idPeriode)
		_, _ = pool.Exec(ctx, "DELETE FROM pembimbing_lapangan WHERE id = $1", pl1ID)
		_, _ = pool.Exec(ctx, "DELETE FROM tempat_program WHERE id = $1", idTempat)
	}()

	tYesterday := tNow.AddDate(0, 0, -1).Format("2006-01-02")
	tTomorrow := tNow.AddDate(0, 0, 1).Format("2006-01-02")
	tPastYear := tNow.AddDate(-2, 0, 0).Format("2006-01-02")

	// ========================================================
	// 1. PENGUJIAN JURNAL
	// ========================================================

	// 1.1 Mahasiswa lain mencoba mengisi jurnal -> ditolak 404/403
	resp := executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs2, map[string]interface{}{
		"tanggal": tYesterday,
		"judul":   "Jurnal Ilegal",
		"isi":     "Isi jurnal",
		"status":  "draft",
	})
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, resp.Code)

	// 1.2 Tanggal masa depan -> ditolak 422 BR-08
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs1, map[string]interface{}{
		"tanggal": tTomorrow,
		"judul":   "Jurnal Masa Depan",
		"isi":     "Isi jurnal",
		"status":  "draft",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	// 1.3 Tanggal di luar periode -> ditolak 422 BR-08
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs1, map[string]interface{}{
		"tanggal": tPastYear,
		"judul":   "Jurnal Lampau",
		"isi":     "Isi jurnal",
		"status":  "draft",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	// 1.4 Pemilik membuat jurnal draft yang valid -> 201 Created
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs1, map[string]interface{}{
		"tanggal": tYesterday,
		"judul":   "Hari Pertama Mengajar",
		"isi":     "Observasi ruang kelas dan perkenalan dengan guru",
		"kendala": "Belum hafal nama siswa",
		"solusi":  "Membuat denah tempat duduk siswa",
		"status":  "draft",
	})
	assert.Equal(t, http.StatusCreated, resp.Code)
	var jCreateResp struct {
		Data struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Judul  string `json:"judul"`
		} `json:"data"`
	}
	err = json.Unmarshal(resp.Body.Bytes(), &jCreateResp)
	require.NoError(t, err)
	jurnalID := jCreateResp.Data.ID
	assert.Equal(t, "draft", jCreateResp.Data.Status)
	assert.Equal(t, "Hari Pertama Mengajar", jCreateResp.Data.Judul)

	// 1.5 BR-09: Duplikat tanggal untuk peserta yang sama -> ditolak 409 Conflict
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs1, map[string]interface{}{
		"tanggal": tYesterday,
		"judul":   "Duplikat Jurnal",
		"isi":     "Mencoba kirim jurnal kedua di tanggal yang sama",
		"status":  "draft",
	})
	assert.Equal(t, http.StatusConflict, resp.Code)

	// 1.6 Mahasiswa mengedit jurnal draft menjadi disubmit -> 200 OK
	resp = executeRequest(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/jurnal/%d", idPendaftaran, jurnalID), tokenMhs1, map[string]interface{}{
		"judul":   "Hari Pertama Mengajar (Updated)",
		"isi":     "Observasi ruang kelas dan perkenalan dengan guru pamong",
		"kendala": "Belum hafal nama siswa",
		"solusi":  "Membuat denah tempat duduk siswa",
		"status":  "disubmit",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	// 1.7 Mahasiswa mengedit jurnal yang SUDAH disubmit -> ditolak 422 JURNAL_BUKAN_DRAFT
	resp = executeRequest(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/jurnal/%d", idPendaftaran, jurnalID), tokenMhs1, map[string]interface{}{
		"judul":  "Hari Pertama Mengajar (Edit Lagi)",
		"isi":    "Isi baru",
		"status": "disubmit",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	// 1.8 Akses GET /pendaftaran/:id/jurnal oleh pihak yang berwenang
	// Dosen pembimbing dapat melihat jurnal
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/jurnal?status=disubmit", idPendaftaran), tokenDos1, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Pembimbing lapangan dapat melihat jurnal
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenPl1, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Admin dapat melihat jurnal
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenAdmin, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Detail jurnal
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/jurnal/%d", idPendaftaran, jurnalID), tokenMhs1, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// ========================================================
	// 2. PENGUJIAN LOGBOOK
	// ========================================================

	tNextWeek := tNow.AddDate(0, 0, 7).Format("2006-01-02")

	// 2.1 Dosen lain (bukan pembimbing pendaftaran itu) mencoba mengisi logbook -> ditolak 404
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenDos2, map[string]interface{}{
		"tanggal":         tYesterday,
		"topik_bimbingan": "Bimbingan Proker Ilegal",
		"catatan":         "Catatan",
	})
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// 2.2 Dosen pembimbing mengisi logbook dengan tanggal masa depan -> ditolak 422 BR-08
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenDos1, map[string]interface{}{
		"tanggal":         tTomorrow,
		"topik_bimbingan": "Bimbingan Masa Depan",
		"catatan":         "Catatan",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	// 2.3 Dosen pembimbing yang sah mengisi logbook valid -> 201 Created
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenDos1, map[string]interface{}{
		"tanggal":              tYesterday,
		"topik_bimbingan":      "Penyusunan RPP dan Modul Ajar",
		"catatan":              "Mahasiswa diarahkan untuk menyesuaikan dengan Kurikulum Merdeka",
		"pertemuan_berikutnya": tNextWeek,
	})
	assert.Equal(t, http.StatusCreated, resp.Code)
	var lbCreateResp struct {
		Data struct {
			ID             int    `json:"id"`
			TopikBimbingan string `json:"topik_bimbingan"`
		} `json:"data"`
	}
	err = json.Unmarshal(resp.Body.Bytes(), &lbCreateResp)
	require.NoError(t, err)
	logbookID := lbCreateResp.Data.ID
	assert.Equal(t, "Penyusunan RPP dan Modul Ajar", lbCreateResp.Data.TopikBimbingan)

	// 2.4 Dosen lain mencoba mengupdate logbook -> ditolak 404
	resp = executeRequest(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/logbook/%d", idPendaftaran, logbookID), tokenDos2, map[string]interface{}{
		"topik_bimbingan": "Penyusunan RPP Diubah Dosen Lain",
		"catatan":         "Catatan bajakan",
	})
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// 2.5 Dosen pembuat memperbarui logbook -> 200 OK
	resp = executeRequest(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/logbook/%d", idPendaftaran, logbookID), tokenDos1, map[string]interface{}{
		"topik_bimbingan":      "Penyusunan RPP dan Modul Ajar (Final)",
		"catatan":              "Mahasiswa telah memperbaiki draft modul",
		"pertemuan_berikutnya": tNextWeek,
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	// 2.6 Akses GET /pendaftaran/:id/logbook oleh mahasiswa pemilik
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenMhs1, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Detail logbook
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/logbook/%d", idPendaftaran, logbookID), tokenMhs1, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// ========================================================
	// 3. PENYELESAIAN PERIODE & VALIDASI STATUS SETELAH SELESAI
	// ========================================================

	// 3.1 Admin menyelesaikan periode
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/admin/periode/%d/selesaikan", idPeriode), tokenAdmin, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// 3.2 Verifikasi status periode adalah 'selesai' dan status peserta_program adalah 'selesai'
	var periodeStatus, pesertaStatus string
	err = pool.QueryRow(ctx, "SELECT status FROM periode WHERE id = $1", idPeriode).Scan(&periodeStatus)
	require.NoError(t, err)
	assert.Equal(t, "selesai", periodeStatus)

	err = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", idPendaftaran).Scan(&pesertaStatus)
	require.NoError(t, err)
	assert.Equal(t, "selesai", pesertaStatus)

	// 3.3 Mengisi jurnal baru setelah periode selesai -> Ditolak 422 (STATUS_TIDAK_SESUAI)
	tTwoDaysAgo := tNow.AddDate(0, 0, -2).Format("2006-01-02")
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs1, map[string]interface{}{
		"tanggal": tTwoDaysAgo,
		"judul":   "Jurnal Pasca Selesai",
		"isi":     "Mencoba isi setelah selesai",
		"status":  "draft",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	// 3.4 Mengisi logbook baru setelah periode selesai -> Ditolak 422 (STATUS_TIDAK_SESUAI)
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenDos1, map[string]interface{}{
		"tanggal":         tTwoDaysAgo,
		"topik_bimbingan": "Bimbingan Pasca Selesai",
		"catatan":         "Mencoba isi logbook setelah selesai",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}
