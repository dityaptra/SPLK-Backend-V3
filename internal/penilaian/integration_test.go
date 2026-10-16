package penilaian_test

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
	"sync"
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
	"splk-backend/internal/penilaian"
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
			VALUES ('mentor', $1, $2, $3, $4, '081299998888')
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

func TestIntegration_PenilaianTigaPenilai(t *testing.T) {
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
	_ = setupUser(t, pool, "admin_nilai", "admin_nilai@undiksha.ac.id", "pass12345", "admin")
	uidMhs := setupUser(t, pool, "mhs_nilai", "mhs_nilai@undiksha.ac.id", "pass12345", "mahasiswa")
	uidPembimbing := setupUser(t, pool, "dos_pembimbing_nilai", "dos_pembimbing_nilai@undiksha.ac.id", "pass12345", "dosen")
	uidPenguji := setupUser(t, pool, "dos_penguji_nilai", "dos_penguji_nilai@undiksha.ac.id", "pass12345", "dosen")
	uidLapangan := setupUser(t, pool, "pl_nilai", "pl_nilai@undiksha.ac.id", "pass12345", "pembimbing_lapangan")

	mhsID := ensureMahasiswaProfile(t, pool, uidMhs, prodiID, "2300000099", "Mahasiswa Dinilai")
	dosPembimbingID := ensureDosenProfile(t, pool, uidPembimbing, prodiID, "1980000091", "Dosen Pembimbing Sah")
	dosPengujiID := ensureDosenProfile(t, pool, uidPenguji, prodiID, "1980000092", "Dosen Penguji Sah")

	tokenAdmin := getAuthToken(t, router, "admin_nilai", "pass12345")
	tokenMhs := getAuthToken(t, router, "mhs_nilai", "pass12345")
	tokenPembimbing := getAuthToken(t, router, "dos_pembimbing_nilai", "pass12345")
	tokenPenguji := getAuthToken(t, router, "dos_penguji_nilai", "pass12345")
	tokenLapangan := getAuthToken(t, router, "pl_nilai", "pass12345")

	// Setup Tempat Program Magang
	var idTempat int
	err := pool.QueryRow(ctx, `
		INSERT INTO tempat_program (nama, jenis_program, alamat, kota, kontak, is_aktif)
		VALUES ('PT Teknologi Integrasi Penilaian', 'magang', 'Kuningan', 'Jakarta', 'HRD', true)
		RETURNING id
	`).Scan(&idTempat)
	require.NoError(t, err)

	plID := ensurePembimbingLapangan(t, pool, idTempat, uidLapangan, "Mentor Industri Sah", "Senior Engineer")

	// Setup Periode Magang
	tNow := time.Now()
	var idPeriode int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai, status)
		VALUES ('Periode Magang Penilaian Test', 'magang', '2026/2027 Genap', $1, $2, 'selesai')
		RETURNING id
	`, tNow.AddDate(0, -6, 0), tNow.AddDate(0, -1, 0)).Scan(&idPeriode)
	require.NoError(t, err)

	// Setup Peserta Program Magang dengan status 'selesai'
	var idPendaftaran int
	err = pool.QueryRow(ctx, `
		INSERT INTO peserta_program (id_periode, jenis_program, id_mahasiswa, id_tempat_program, id_dosen_pembimbing, id_dosen_penguji, id_pembimbing_lapangan, status)
		VALUES ($1, 'magang', $2, $3, $4, $5, $6, 'selesai')
		RETURNING id
	`, idPeriode, mhsID, idTempat, dosPembimbingID, dosPengujiID, plID).Scan(&idPendaftaran)
	require.NoError(t, err)

	// Setup 3 Instrumen Penilaian untuk 'magang'
	var instPembimbingID, instPengujiID, instLapanganID int

	// 1. Instrumen Pembimbing (bobot 30)
	resp := executeRequest(router, http.MethodPost, "/admin/instrumen-penilaian", tokenAdmin, map[string]interface{}{
		"jenis_program": "magang",
		"nama_aspek":    "Bimbingan dan Sikap Kerja",
		"bobot":         30,
		"role_penilai":  penilaian.RoleDosenPembimbing,
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	var inst1Resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &inst1Resp)
	instPembimbingID = inst1Resp.Data.ID

	// 2. Instrumen Penguji (bobot 30)
	resp = executeRequest(router, http.MethodPost, "/admin/instrumen-penilaian", tokenAdmin, map[string]interface{}{
		"jenis_program": "magang",
		"nama_aspek":    "Ujian dan Penguasaan Laporan",
		"bobot":         30,
		"role_penilai":  penilaian.RoleDosenPenguji,
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	var inst2Resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &inst2Resp)
	instPengujiID = inst2Resp.Data.ID

	// 3. Instrumen Pembimbing Lapangan (bobot 40)
	resp = executeRequest(router, http.MethodPost, "/admin/instrumen-penilaian", tokenAdmin, map[string]interface{}{
		"jenis_program": "magang",
		"nama_aspek":    "Kinerja Praktik Industri",
		"bobot":         40,
		"role_penilai":  penilaian.RolePembimbingLapangan,
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	var inst3Resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &inst3Resp)
	instLapanganID = inst3Resp.Data.ID

	// Cleanup defer
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM penilaian WHERE id_peserta_program = $1", idPendaftaran)
		_, _ = pool.Exec(ctx, "DELETE FROM instrumen_penilaian WHERE id IN ($1, $2, $3)", instPembimbingID, instPengujiID, instLapanganID)
		_, _ = pool.Exec(ctx, "DELETE FROM peserta_program WHERE id = $1", idPendaftaran)
		_, _ = pool.Exec(ctx, "DELETE FROM periode WHERE id = $1", idPeriode)
		_, _ = pool.Exec(ctx, "DELETE FROM pembimbing_lapangan WHERE id = $1", plID)
		_, _ = pool.Exec(ctx, "DELETE FROM tempat_program WHERE id = $1", idTempat)
	}()

	// =========================================================================
	// 1. Pengujian Form Penilaian
	// =========================================================================
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/form-penilaian", idPendaftaran), tokenPenguji, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Mahasiswa mencoba melihat rincian nilai sebelum lengkap -> 403
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/nilai", idPendaftaran), tokenMhs, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	// =========================================================================
	// 2. Tiga penilai mengisi dalam urutan acak
	//    Penilai pertama: Dosen Penguji mengisi duluan (Nilai: 90)
	// =========================================================================
	nilaiPenguji := 90.0
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPenguji, map[string]interface{}{
		"items": []map[string]interface{}{
			{"id_instrumen": instPengujiID, "nilai": nilaiPenguji, "catatan": "Presentasi sangat baik"},
		},
	})
	assert.Equal(t, http.StatusOK, resp.Code)
	var pengujiResp struct {
		Data struct {
			SemuaInstrumenTerisi bool     `json:"semua_instrumen_terisi"`
			NilaiAkhir           *float64 `json:"nilai_akhir"`
		} `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &pengujiResp)
	assert.False(t, pengujiResp.Data.SemuaInstrumenTerisi)
	assert.Nil(t, pengujiResp.Data.NilaiAkhir)

	// Verifikasi di DB nilai_akhir masih NULL
	var checkNilaiAkhir *float64
	err = pool.QueryRow(ctx, "SELECT nilai_akhir FROM peserta_program WHERE id = $1", idPendaftaran).Scan(&checkNilaiAkhir)
	require.NoError(t, err)
	assert.Nil(t, checkNilaiAkhir)

	// =========================================================================
	// 3. Pengiriman paralel oleh dua penilai terakhir
	//    - Dosen Pembimbing (Nilai: 80)
	//    - Pembimbing Lapangan (Nilai: 85)
	//    Formula: (30*80 + 30*90 + 40*85) / (30 + 30 + 40)
	//           = (2400 + 2700 + 3400) / 100 = 8500 / 100 = 85.00 -> Grade "A"
	// =========================================================================
	var wg sync.WaitGroup
	wg.Add(2)

	var respPembimbing, respLapangan *httptest.ResponseRecorder

	go func() {
		defer wg.Done()
		nilaiPembimbing := 80.0
		respPembimbing = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPembimbing, map[string]interface{}{
			"items": []map[string]interface{}{
				{"id_instrumen": instPembimbingID, "nilai": nilaiPembimbing, "catatan": "Bimbingan aktif"},
			},
		})
	}()

	go func() {
		defer wg.Done()
		nilaiLap := 85.0
		respLapangan = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenLapangan, map[string]interface{}{
			"items": []map[string]interface{}{
				{"id_instrumen": instLapanganID, "nilai": nilaiLap, "catatan": "Kinerja industri memuaskan"},
			},
		})
	}()

	wg.Wait()

	// Kedua request harus berhasil 200 OK (konkurensi aman berkat FOR UPDATE)
	assert.Equal(t, http.StatusOK, respPembimbing.Code)
	assert.Equal(t, http.StatusOK, respLapangan.Code)

	// Tepat salah satu dari mereka (atau yang terakhir commit) men-trigger perhitungan nilai akhir
	// Dan di database nilai_akhir harus tepat 85.00 dan grade_huruf 'A'
	var finalScore float64
	var finalGrade string
	err = pool.QueryRow(ctx, "SELECT nilai_akhir, grade_huruf FROM peserta_program WHERE id = $1", idPendaftaran).Scan(&finalScore, &finalGrade)
	require.NoError(t, err)
	assert.InDelta(t, 85.00, finalScore, 0.01)
	assert.Equal(t, "A", finalGrade)

	// =========================================================================
	// 4. Mahasiswa melihat rincian nilai setelah terisi -> 200 OK
	// =========================================================================
	resp = executeRequest(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d/nilai", idPendaftaran), tokenMhs, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
	var nilaiResp struct {
		Data struct {
			NilaiAkhir *float64 `json:"nilai_akhir"`
			GradeHuruf *string  `json:"grade_huruf"`
			Items      []struct {
				NamaAspek string  `json:"nama_aspek"`
				Nilai     float64 `json:"nilai"`
			} `json:"items"`
		} `json:"data"`
	}
	err = json.Unmarshal(resp.Body.Bytes(), &nilaiResp)
	require.NoError(t, err)
	require.NotNil(t, nilaiResp.Data.NilaiAkhir)
	assert.Equal(t, 85.00, *nilaiResp.Data.NilaiAkhir)
	assert.Equal(t, "A", *nilaiResp.Data.GradeHuruf)
	assert.Equal(t, 3, len(nilaiResp.Data.Items))

	// =========================================================================
	// 5. Penilai mencoba mengubah nilai setelah terhitung -> ditolak 422 NILAI_SUDAH_TERKUNCI
	// =========================================================================
	nilaiBaru := 100.0
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPembimbing, map[string]interface{}{
		"items": []map[string]interface{}{
			{"id_instrumen": instPembimbingID, "nilai": nilaiBaru},
		},
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	// =========================================================================
	// 6. Admin hitung nilai manual -> 200 OK & log aktivitas tercatat
	// =========================================================================
	resp = executeRequest(router, http.MethodPost, fmt.Sprintf("/admin/pendaftaran/%d/hitung-nilai", idPendaftaran), tokenAdmin, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
}
