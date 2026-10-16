package e2e_test

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
	"splk-backend/internal/penilaian"
	"splk-backend/internal/platform/security"
)

// Helper koneksi test pool
func getTestPool(t *testing.T) *pgxpool.Pool {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diset; lewati E2E integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDBURL)
	require.NoError(t, err, "koneksi database test gagal")

	err = pool.Ping(ctx)
	require.NoError(t, err, "ping database test gagal")

	return pool
}

func setupTestApp(t *testing.T, pool *pgxpool.Pool) (*gin.Engine, *config.Config) {
	cfg := &config.Config{
		AppEnv:        "development",
		AppPort:       "8080",
		DatabaseURL:   os.Getenv("TEST_DATABASE_URL"),
		JWTSecret:     "e2e-super-secret-jwt-key-32bytes-long!",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 24 * time.Hour,
		UploadDir:     t.TempDir(),
		MaxUploadMB:   5,
		LogLevel:      "warn",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)
	return router, cfg
}

func executeJSON(router *gin.Engine, method, url, token string, body interface{}) *httptest.ResponseRecorder {
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

func executeMultipart(router *gin.Engine, url, token, fieldName, fileName, contentType string, content []byte, extraFields map[string]string) *httptest.ResponseRecorder {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	for k, v := range extraFields {
		_ = writer.WriteField(k, v)
	}

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName)}
	h["Content-Type"] = []string{contentType}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write(content)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, url, buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
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

func createUserActor(t *testing.T, pool *pgxpool.Pool, cfg *config.Config, username, email, roleName string) (int, string) {
	ctx := context.Background()
	var userID int
	err := database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		var roleID int
		err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE nama = $1", roleName).Scan(&roleID)
		if err != nil {
			return fmt.Errorf("role %s: %w", roleName, err)
		}

		err = tx.QueryRow(ctx, "SELECT id FROM users WHERE username = $1 OR email = $2", username, email).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			passHash, err := security.HashPassword("pass12345")
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

	token, err := security.GenerateToken(userID, []string{roleName}, cfg.JWTAccessTTL, cfg.JWTSecret)
	require.NoError(t, err)
	return userID, token
}

func createMahasiswaProfile(t *testing.T, pool *pgxpool.Pool, userID, prodiID int, nim, nama string) int {
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
	}
	return mhsID
}

func createDosenProfile(t *testing.T, pool *pgxpool.Pool, userID, prodiID int, nip, nama string) int {
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
	}
	return dosenID
}

func createPembimbingLapanganProfile(t *testing.T, pool *pgxpool.Pool, idTempat, userID int, jenis, nama string) int {
	ctx := context.Background()
	var plID int
	err := pool.QueryRow(ctx, "SELECT id FROM pembimbing_lapangan WHERE id_user = $1", userID).Scan(&plID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx, `
			INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, id_user, nama, jabatan, kontak)
			VALUES ($1, $2, $3, $4, 'Jabatan Sah', '08123456789')
			RETURNING id
		`, jenis, idTempat, userID, nama).Scan(&plID)
		require.NoError(t, err)
	} else {
		_, err = pool.Exec(ctx, "UPDATE pembimbing_lapangan SET id_tempat_program = $1, jenis = $2 WHERE id = $3", idTempat, jenis, plID)
		require.NoError(t, err)
	}
	return plID
}

func ensureInstrumenPenilaian(t *testing.T, pool *pgxpool.Pool, jenisProgram string) (int, int, int) {
	ctx := context.Background()
	insertOrGet := func(role, aspek string, bobot int) int {
		var id int
		err := pool.QueryRow(ctx, `
			SELECT id FROM instrumen_penilaian 
			WHERE jenis_program = $1 AND role_penilai = $2 AND nama_aspek = $3
		`, jenisProgram, role, aspek).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			err = pool.QueryRow(ctx, `
				INSERT INTO instrumen_penilaian (jenis_program, role_penilai, nama_aspek, bobot)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, jenisProgram, role, aspek, bobot).Scan(&id)
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
		return id
	}

	idPembimbing := insertOrGet(penilaian.RoleDosenPembimbing, "Laporan & Bimbingan "+jenisProgram, 35)
	idLapangan := insertOrGet(penilaian.RolePembimbingLapangan, "Kinerja Lapangan "+jenisProgram, 35)
	idPenguji := insertOrGet(penilaian.RoleDosenPenguji, "Ujian Akhir "+jenisProgram, 30)

	return idPembimbing, idLapangan, idPenguji
}

// -------------------------------------------------------------------------------------------------
// 1. E2E MAGANG
// -------------------------------------------------------------------------------------------------
func TestE2E_Magang(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	router, cfg := setupTestApp(t, pool)
	ctx := context.Background()
	prodiID := ensureAkademik(t, pool)

	// Aktor
	_, tokenAdmin := createUserActor(t, pool, cfg, "admin_e2e_mg", "admin_mg@undiksha.ac.id", "admin")
	uidMhs, tokenMhs := createUserActor(t, pool, cfg, "mhs_e2e_mg", "mhs_mg@undiksha.ac.id", "mahasiswa")
	uidPembimbing, tokenPembimbing := createUserActor(t, pool, cfg, "dos_e2e_mg_pemb", "pemb_mg@undiksha.ac.id", "dosen")
	uidPenguji, tokenPenguji := createUserActor(t, pool, cfg, "dos_e2e_mg_peng", "peng_mg@undiksha.ac.id", "dosen")
	uidMentor, tokenMentor := createUserActor(t, pool, cfg, "mentor_e2e_mg", "mentor_mg@undiksha.ac.id", "pembimbing_lapangan")

	_ = createMahasiswaProfile(t, pool, uidMhs, prodiID, "2315101001", "Mahasiswa Magang E2E")
	dosPembimbingID := createDosenProfile(t, pool, uidPembimbing, prodiID, "19850101001", "Dosen Pembimbing Magang")
	dosPengujiID := createDosenProfile(t, pool, uidPenguji, prodiID, "19850101002", "Dosen Penguji Magang")

	// 1. Buat Tempat Magang & Mentor
	var idTempat int
	err := pool.QueryRow(ctx, `
		INSERT INTO tempat_program (nama, jenis_program, alamat, kota, kontak, is_aktif)
		VALUES ('PT Teknologi Nusantara Magang', 'magang', 'Jl. Merdeka 10', 'Denpasar', 'HRD', true)
		RETURNING id
	`).Scan(&idTempat)
	require.NoError(t, err)
	mentorID := createPembimbingLapanganProfile(t, pool, idTempat, uidMentor, "mentor", "Mentor Magang Industri")

	// 2. Buat Periode Magang Aktif
	tNow := time.Now()
	var idPeriode int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai, status)
		VALUES ('Magang Semester Ganjil E2E', 'magang', '2026/2027 Ganjil', $1, $2, 'aktif')
		RETURNING id
	`, tNow.AddDate(0, -1, 0), tNow.AddDate(0, 5, 0)).Scan(&idPeriode)
	require.NoError(t, err)

	// Pastikan Instrumen Penilaian Magang Tersedia
	instPembimbingID, instMentorID, instPengujiID := ensureInstrumenPenilaian(t, pool, "magang")

	// 3. Mahasiswa Mendaftar
	respDaftar := executeJSON(router, http.MethodPost, "/pendaftaran", tokenMhs, map[string]interface{}{
		"id_periode": idPeriode,
	})
	require.Equal(t, http.StatusCreated, respDaftar.Code, "daftar gagal: %s", respDaftar.Body.String())
	var resBody struct {
		Data struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDaftar.Body.Bytes(), &resBody)
	idPendaftaran := resBody.Data.ID
	require.Equal(t, "diajukan", resBody.Data.Status)

	// 4. Mahasiswa Memilih Tempat Magang dan Mentor (status -> 'diajukan')
	respPenempatan := executeJSON(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/penempatan-magang", idPendaftaran), tokenMhs, map[string]interface{}{
		"id_tempat_program":      idTempat,
		"id_pembimbing_lapangan": mentorID,
	})
	require.Equal(t, http.StatusOK, respPenempatan.Code, "penempatan gagal: %s", respPenempatan.Body.String())

	// 5. Penetapan Dosen Pembimbing oleh Admin
	respDosen := executeJSON(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/dosen-pembimbing", idPendaftaran), tokenAdmin, map[string]interface{}{
		"id_dosen": dosPembimbingID,
	})
	require.Equal(t, http.StatusOK, respDosen.Code, "set dosen pembimbing gagal: %s", respDosen.Body.String())

	// 6. Mahasiswa Mengunggah Dokumen MoU & Dokumen Proposal
	pdfDummy := []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n>>\nendobj\ntrailer\n<<\n>>\n%%EOF")
	respMou := executeMultipart(router, fmt.Sprintf("/pendaftaran/%d/dokumen", idPendaftaran), tokenMhs, "file", "mou_magang.pdf", "application/pdf", pdfDummy, map[string]string{
		"jenis_dokumen": "mou",
	})
	require.Equal(t, http.StatusCreated, respMou.Code, "upload mou gagal: %s", respMou.Body.String())

	respProp := executeMultipart(router, fmt.Sprintf("/pendaftaran/%d/dokumen", idPendaftaran), tokenMhs, "file", "proposal_magang.pdf", "application/pdf", pdfDummy, map[string]string{
		"jenis_dokumen": "proposal",
	})
	require.Equal(t, http.StatusCreated, respProp.Code, "upload proposal gagal: %s", respProp.Body.String())

	// 7. Verifikasi Status Pendaftaran Otomatis Bertransisi ke 'berjalan'
	respDetail := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPendaftaran), tokenMhs, nil)
	require.Equal(t, http.StatusOK, respDetail.Code)
	var detailBody struct {
		Data struct {
			Status   string `json:"status"`
			Kesiapan struct {
				Siap bool `json:"siap"`
			} `json:"kesiapan"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDetail.Body.Bytes(), &detailBody)
	assert.Equal(t, "berjalan", detailBody.Data.Status, "status pendaftaran harus otomatis berjalan setelah dokumen dan syarat lengkap")
	assert.True(t, detailBody.Data.Kesiapan.Siap)

	// 8. Mahasiswa Mengisi Jurnal Harian & Dosen Mengisi Logbook Bimbingan
	respJurnal := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs, map[string]interface{}{
		"tanggal": time.Now().Format("2006-01-02"),
		"judul":   "Onboarding dan Konfigurasi",
		"isi":     "Mengikuti sesi onboarding dan konfigurasi environment backend Go.",
		"status":  "disubmit",
	})
	require.Equal(t, http.StatusCreated, respJurnal.Code, "buat jurnal gagal: %s", respJurnal.Body.String())

	respLogbook := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenPembimbing, map[string]interface{}{
		"tanggal":         time.Now().Format("2006-01-02"),
		"topik_bimbingan": "Koordinasi awal dan penetapan target magang",
		"catatan":         "Mahasiswa siap menjalankan program magang.",
	})
	require.Equal(t, http.StatusCreated, respLogbook.Code, "buat logbook gagal: %s", respLogbook.Body.String())

	// 9. Admin Menetapkan Dosen Penguji
	respPenguji := executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/pendaftaran/%d/dosen-penguji", idPendaftaran), tokenAdmin, map[string]interface{}{
		"id_dosen": dosPengujiID,
	})
	require.Equal(t, http.StatusOK, respPenguji.Code, "set penguji gagal: %s", respPenguji.Body.String())

	// 10. Periode Selesai (Admin Menyelesaikan Periode -> Status Pendaftaran -> 'selesai')
	respSelesaiPeriode := executeJSON(router, http.MethodPost, fmt.Sprintf("/admin/periode/%d/selesaikan", idPeriode), tokenAdmin, nil)
	require.Equal(t, http.StatusOK, respSelesaiPeriode.Code, "selesaikan periode gagal: %s", respSelesaiPeriode.Body.String())

	respDetail2 := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPendaftaran), tokenMhs, nil)
	_ = json.Unmarshal(respDetail2.Body.Bytes(), &detailBody)
	assert.Equal(t, "selesai", detailBody.Data.Status, "status pendaftaran harus menjadi selesai setelah periode selesai")

	// 11. Tiga Penilai Memberikan Penilaian
	// A. Dosen Pembimbing (Nilai 85)
	respNilaiPemb := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPembimbing, map[string]interface{}{
		"catatan": "Kinerja bimbingan dan pelaporan sangat baik",
		"items": []map[string]interface{}{
			{"id_instrumen": instPembimbingID, "nilai": 85.0},
		},
	})
	require.Equal(t, http.StatusOK, respNilaiPemb.Code, "penilaian pembimbing gagal: %s", respNilaiPemb.Body.String())

	// B. Mentor Lapangan (Nilai 90)
	respNilaiMentor := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenMentor, map[string]interface{}{
		"catatan": "Adaptasi cepat dan kontribusi signifikan di proyek",
		"items": []map[string]interface{}{
			{"id_instrumen": instMentorID, "nilai": 90.0},
		},
	})
	require.Equal(t, http.StatusOK, respNilaiMentor.Code, "penilaian mentor gagal: %s", respNilaiMentor.Body.String())

	// C. Dosen Penguji (Nilai 80)
	respNilaiPenguji := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPenguji, map[string]interface{}{
		"catatan": "Penguasaan materi ujian magang komprehensif",
		"items": []map[string]interface{}{
			{"id_instrumen": instPengujiID, "nilai": 80.0},
		},
	})
	require.Equal(t, http.StatusOK, respNilaiPenguji.Code, "penilaian penguji gagal: %s", respNilaiPenguji.Body.String())

	// 12. Verifikasi Nilai Akhir Otomatis Terhitung dan Grade Huruf Terisi
	respFinal := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPendaftaran), tokenMhs, nil)
	require.Equal(t, http.StatusOK, respFinal.Code)
	var finalBody struct {
		Data struct {
			Status     string   `json:"status"`
			NilaiAkhir *float64 `json:"nilai_akhir"`
			GradeHuruf *string  `json:"grade_huruf"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respFinal.Body.Bytes(), &finalBody)

	require.NotNil(t, finalBody.Data.NilaiAkhir, "nilai akhir harus terhitung")
	require.NotNil(t, finalBody.Data.GradeHuruf, "grade huruf harus terisi")
	assert.Greater(t, *finalBody.Data.NilaiAkhir, 0.0)
	assert.Equal(t, "A", *finalBody.Data.GradeHuruf)
}

// -------------------------------------------------------------------------------------------------
// 2. E2E ASISTENSI MENGAJAR (AM)
// -------------------------------------------------------------------------------------------------
func TestE2E_AsistensiMengajar(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	router, cfg := setupTestApp(t, pool)
	ctx := context.Background()
	prodiID := ensureAkademik(t, pool)

	// Aktor
	_, tokenAdmin := createUserActor(t, pool, cfg, "admin_e2e_am", "admin_am@undiksha.ac.id", "admin")
	uidMhs, tokenMhs := createUserActor(t, pool, cfg, "mhs_e2e_am", "mhs_am@undiksha.ac.id", "mahasiswa")
	uidPembimbing, tokenPembimbing := createUserActor(t, pool, cfg, "dos_e2e_am_pemb", "pemb_am@undiksha.ac.id", "dosen")
	uidPenguji, tokenPenguji := createUserActor(t, pool, cfg, "dos_e2e_am_peng", "peng_am@undiksha.ac.id", "dosen")
	uidGuruPamong, tokenGuruPamong := createUserActor(t, pool, cfg, "guru_e2e_am", "guru_am@undiksha.ac.id", "pembimbing_lapangan")

	_ = createMahasiswaProfile(t, pool, uidMhs, prodiID, "2315102001", "Mahasiswa AM E2E")
	dosPembimbingID := createDosenProfile(t, pool, uidPembimbing, prodiID, "19850202001", "Dosen Pembimbing AM")
	dosPengujiID := createDosenProfile(t, pool, uidPenguji, prodiID, "19850202002", "Dosen Penguji AM")

	// 1. Buat Sekolah Penempatan & Guru Pamong
	var idSekolah int
	err := pool.QueryRow(ctx, `
		INSERT INTO tempat_program (nama, jenis_program, alamat, kota, kontak, is_aktif)
		VALUES ('SMA Negeri 1 Singaraja', 'asistensi_mengajar', 'Jl. Pramuka No. 1', 'Singaraja', 'Kepsek', true)
		RETURNING id
	`).Scan(&idSekolah)
	require.NoError(t, err)
	guruPamongID := createPembimbingLapanganProfile(t, pool, idSekolah, uidGuruPamong, "guru_pamong", "Guru Pamong AM")

	// 2. Buat Periode AM Aktif
	tNow := time.Now()
	var idPeriode int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai, status)
		VALUES ('Asistensi Mengajar E2E 2026', 'asistensi_mengajar', '2026/2027 Ganjil', $1, $2, 'aktif')
		RETURNING id
	`, tNow.AddDate(0, -1, 0), tNow.AddDate(0, 5, 0)).Scan(&idPeriode)
	require.NoError(t, err)

	instPembimbingID, instGuruID, instPengujiID := ensureInstrumenPenilaian(t, pool, "asistensi_mengajar")

	// 3. Mahasiswa Mendaftar AM (Status awal langsung 'diajukan')
	respDaftar := executeJSON(router, http.MethodPost, "/pendaftaran", tokenMhs, map[string]interface{}{
		"id_periode": idPeriode,
	})
	require.Equal(t, http.StatusCreated, respDaftar.Code, "daftar AM gagal: %s", respDaftar.Body.String())
	var resBody struct {
		Data struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDaftar.Body.Bytes(), &resBody)
	idPendaftaran := resBody.Data.ID
	require.Equal(t, "diajukan", resBody.Data.Status)

	// 4. Admin Melakukan Pemetaan Sekolah dan Guru Pamong
	respPenempatan := executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/pendaftaran/%d/penempatan", idPendaftaran), tokenAdmin, map[string]interface{}{
		"id_tempat_program":      idSekolah,
		"id_pembimbing_lapangan": guruPamongID,
	})
	require.Equal(t, http.StatusOK, respPenempatan.Code, "penempatan AM gagal: %s", respPenempatan.Body.String())

	// 5. Admin Menetapkan Dosen Pembimbing & Dosen Penguji
	respDosen := executeJSON(router, http.MethodPut, fmt.Sprintf("/pendaftaran/%d/dosen-pembimbing", idPendaftaran), tokenAdmin, map[string]interface{}{
		"id_dosen": dosPembimbingID,
	})
	require.Equal(t, http.StatusOK, respDosen.Code, "set dosen pembimbing AM gagal: %s", respDosen.Body.String())

	respPenguji := executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/pendaftaran/%d/dosen-penguji", idPendaftaran), tokenAdmin, map[string]interface{}{
		"id_dosen": dosPengujiID,
	})
	require.Equal(t, http.StatusOK, respPenguji.Code, "set dosen penguji AM gagal: %s", respPenguji.Body.String())

	// 6. Mahasiswa Mengunggah Dokumen Proker
	pdfDummy := []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n>>\nendobj\ntrailer\n<<\n>>\n%%EOF")
	respProker := executeMultipart(router, fmt.Sprintf("/pendaftaran/%d/dokumen", idPendaftaran), tokenMhs, "file", "proker_am.pdf", "application/pdf", pdfDummy, map[string]string{
		"jenis_dokumen": "proker",
	})
	require.Equal(t, http.StatusCreated, respProker.Code, "upload proker AM gagal: %s", respProker.Body.String())

	// 7. Status Pendaftaran Otomatis Bertransisi ke 'berjalan'
	respDetail := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPendaftaran), tokenMhs, nil)
	require.Equal(t, http.StatusOK, respDetail.Code)
	var detailBody struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDetail.Body.Bytes(), &detailBody)
	assert.Equal(t, "berjalan", detailBody.Data.Status, "status AM harus bertransisi ke berjalan")

	// 8. Mahasiswa Mengisi Jurnal & Dosen Mengisi Logbook
	respJurnal := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPendaftaran), tokenMhs, map[string]interface{}{
		"tanggal": time.Now().Format("2006-01-02"),
		"judul":   "Persiapan Perangkat Pembelajaran",
		"isi":     "Membantu guru pamong menyiapkan silabus dan media ajar interaktif.",
		"status":  "disubmit",
	})
	require.Equal(t, http.StatusCreated, respJurnal.Code, "buat jurnal AM gagal: %s", respJurnal.Body.String())

	respLogbook := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPendaftaran), tokenPembimbing, map[string]interface{}{
		"tanggal":         time.Now().Format("2006-01-02"),
		"topik_bimbingan": "Bimbingan pedagogik awal AM",
		"catatan":         "Silabus dan RPP telah diperiksa.",
	})
	require.Equal(t, http.StatusCreated, respLogbook.Code, "buat logbook AM gagal: %s", respLogbook.Body.String())

	// 9. Admin Menyelesaikan Periode AM
	respSelesai := executeJSON(router, http.MethodPost, fmt.Sprintf("/admin/periode/%d/selesaikan", idPeriode), tokenAdmin, nil)
	require.Equal(t, http.StatusOK, respSelesai.Code)

	// 10. Tiga Penilai Memberikan Nilai
	_ = executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPembimbing, map[string]interface{}{
		"catatan": "Bimbingan pedagogik sangat aktif",
		"items":   []map[string]interface{}{{"id_instrumen": instPembimbingID, "nilai": 88.0}},
	})
	_ = executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenGuruPamong, map[string]interface{}{
		"catatan": "Kinerja di kelas dan interaksi siswa luar biasa",
		"items":   []map[string]interface{}{{"id_instrumen": instGuruID, "nilai": 92.0}},
	})
	_ = executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPendaftaran), tokenPenguji, map[string]interface{}{
		"catatan": "Ujian praktik mengajar memuaskan",
		"items":   []map[string]interface{}{{"id_instrumen": instPengujiID, "nilai": 84.0}},
	})

	// 11. Verifikasi Nilai Akhir AM
	respFinal := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPendaftaran), tokenMhs, nil)
	require.Equal(t, http.StatusOK, respFinal.Code)
	var finalBody struct {
		Data struct {
			Status     string   `json:"status"`
			NilaiAkhir *float64 `json:"nilai_akhir"`
			GradeHuruf *string  `json:"grade_huruf"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respFinal.Body.Bytes(), &finalBody)
	assert.Equal(t, "selesai", finalBody.Data.Status)
	require.NotNil(t, finalBody.Data.NilaiAkhir)
	assert.Greater(t, *finalBody.Data.NilaiAkhir, 80.0)
	assert.Equal(t, "A", *finalBody.Data.GradeHuruf)
}

// -------------------------------------------------------------------------------------------------
// 3. E2E KKN (Kuliah Kerja Nyata)
// -------------------------------------------------------------------------------------------------
func TestE2E_KKN(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()
	router, cfg := setupTestApp(t, pool)
	ctx := context.Background()
	prodiID := ensureAkademik(t, pool)

	// Aktor
	_, tokenAdmin := createUserActor(t, pool, cfg, "admin_e2e_kkn", "admin_kkn@undiksha.ac.id", "admin")
	uidMhs1, tokenMhs1 := createUserActor(t, pool, cfg, "mhs_kkn_1", "mhs_kkn1@undiksha.ac.id", "mahasiswa")
	uidMhs2, tokenMhs2 := createUserActor(t, pool, cfg, "mhs_kkn_2", "mhs_kkn2@undiksha.ac.id", "mahasiswa")
	uidPembimbing, tokenPembimbing := createUserActor(t, pool, cfg, "dos_e2e_kkn_pemb", "pemb_kkn@undiksha.ac.id", "dosen")
	uidKepalaDesa, tokenKepalaDesa := createUserActor(t, pool, cfg, "kades_e2e_kkn", "kades_kkn@undiksha.ac.id", "pembimbing_lapangan")
	uidPenguji, tokenPenguji := createUserActor(t, pool, cfg, "dos_e2e_kkn_peng", "peng_kkn@undiksha.ac.id", "dosen")

	_ = createMahasiswaProfile(t, pool, uidMhs1, prodiID, "2315103001", "Ketua Kelompok KKN")
	_ = createMahasiswaProfile(t, pool, uidMhs2, prodiID, "2315103002", "Anggota Kelompok KKN")
	dosPembimbingID := createDosenProfile(t, pool, uidPembimbing, prodiID, "19850303001", "Dosen Pembimbing KKN")
	dosPengujiID := createDosenProfile(t, pool, uidPenguji, prodiID, "19850303002", "Dosen Penguji KKN")

	// 1. Buat Desa Penempatan KKN & Kepala Desa
	var idDesa int
	err := pool.QueryRow(ctx, `
		INSERT INTO tempat_program (nama, jenis_program, alamat, kota, kontak, is_aktif)
		VALUES ('Desa Munduk Buleleng', 'kkn', 'Kecamatan Banjar', 'Buleleng', 'Sekdes', true)
		RETURNING id
	`).Scan(&idDesa)
	require.NoError(t, err)
	kadesID := createPembimbingLapanganProfile(t, pool, idDesa, uidKepalaDesa, "kepala_desa", "Kepala Desa Munduk")

	// 2. Buat Periode KKN Aktif
	tNow := time.Now()
	var idPeriode int
	err = pool.QueryRow(ctx, `
		INSERT INTO periode (nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai, status)
		VALUES ('KKN Tematik Desa Wisata E2E', 'kkn', '2026/2027 Ganjil', $1, $2, 'aktif')
		RETURNING id
	`, tNow.AddDate(0, -1, 0), tNow.AddDate(0, 5, 0)).Scan(&idPeriode)
	require.NoError(t, err)

	instPembimbingID, instKadesID, instPengujiID := ensureInstrumenPenilaian(t, pool, "kkn")

	// 3. Dua Mahasiswa Mendaftar KKN (Status langsung 'diajukan')
	respDaftar1 := executeJSON(router, http.MethodPost, "/pendaftaran", tokenMhs1, map[string]interface{}{
		"id_periode": idPeriode,
	})
	require.Equal(t, http.StatusCreated, respDaftar1.Code, "daftar KKN mhs1 gagal: %s", respDaftar1.Body.String())
	var resBody1 struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDaftar1.Body.Bytes(), &resBody1)
	idPeserta1 := resBody1.Data.ID

	respDaftar2 := executeJSON(router, http.MethodPost, "/pendaftaran", tokenMhs2, map[string]interface{}{
		"id_periode": idPeriode,
	})
	require.Equal(t, http.StatusCreated, respDaftar2.Code, "daftar KKN mhs2 gagal: %s", respDaftar2.Body.String())
	var resBody2 struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDaftar2.Body.Bytes(), &resBody2)
	idPeserta2 := resBody2.Data.ID

	// 4. Admin Membentuk Kelompok KKN
	respKelompok := executeJSON(router, http.MethodPost, "/admin/kkn/kelompok", tokenAdmin, map[string]interface{}{
		"nama_kelompok":     "Kelompok 10 Munduk E2E",
		"id_periode":        idPeriode,
		"id_tempat_program": idDesa,
	})
	require.Equal(t, http.StatusCreated, respKelompok.Code, "buat kelompok KKN gagal: %s", respKelompok.Body.String())
	var kelompokBody struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respKelompok.Body.Bytes(), &kelompokBody)
	idKelompok := kelompokBody.Data.ID

	// 5. Admin Menambahkan Peserta 1 (Ketua) dan Peserta 2 (Anggota) ke Kelompok KKN
	respAddAnggota := executeJSON(router, http.MethodPost, fmt.Sprintf("/admin/kkn/kelompok/%d/anggota", idKelompok), tokenAdmin, map[string]interface{}{
		"id_peserta_program": []int{idPeserta1, idPeserta2},
		"id_ketua":           idPeserta1,
	})
	require.Equal(t, http.StatusOK, respAddAnggota.Code, "tambah anggota KKN gagal: %s", respAddAnggota.Body.String())

	// 6. Admin Menetapkan Dosen Pembimbing & Kepala Desa untuk Kelompok KKN
	respDosen := executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/kkn/kelompok/%d/dosen-pembimbing", idKelompok), tokenAdmin, map[string]interface{}{
		"id_dosen": dosPembimbingID,
	})
	require.Equal(t, http.StatusOK, respDosen.Code, "set dosen pembimbing kelompok KKN gagal: %s", respDosen.Body.String())

	respKades := executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/kkn/kelompok/%d/pembimbing-lapangan", idKelompok), tokenAdmin, map[string]interface{}{
		"id_pembimbing_lapangan": kadesID,
	})
	require.Equal(t, http.StatusOK, respKades.Code, "set kades kelompok KKN gagal: %s", respKades.Body.String())

	// 7. Ketua Kelompok Mengunggah Dokumen Proker Kelompok
	pdfDummy := []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n>>\nendobj\ntrailer\n<<\n>>\n%%EOF")
	respProkerKKN := executeMultipart(router, fmt.Sprintf("/kkn/kelompok/%d/dokumen", idKelompok), tokenMhs1, "file", "proker_kkn.pdf", "application/pdf", pdfDummy, map[string]string{
		"jenis_dokumen": "proker",
	})
	require.Equal(t, http.StatusCreated, respProkerKKN.Code, "upload proker kelompok KKN gagal: %s", respProkerKKN.Body.String())

	// 8. Verifikasi Bahwa Kedua Peserta KKN Otomatis Bertransisi Menjadi 'berjalan'
	respDet1 := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPeserta1), tokenMhs1, nil)
	require.Equal(t, http.StatusOK, respDet1.Code)
	var det1 struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDet1.Body.Bytes(), &det1)
	assert.Equal(t, "berjalan", det1.Data.Status, "peserta 1 KKN harus bertransisi ke berjalan")

	respDet2 := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPeserta2), tokenMhs2, nil)
	require.Equal(t, http.StatusOK, respDet2.Code)
	var det2 struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respDet2.Body.Bytes(), &det2)
	assert.Equal(t, "berjalan", det2.Data.Status, "peserta 2 KKN harus bertransisi ke berjalan")

	// 9. Mahasiswa Mengisi Jurnal & Dosen Mengisi Logbook
	respJurnal := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/jurnal", idPeserta1), tokenMhs1, map[string]interface{}{
		"tanggal": time.Now().Format("2006-01-02"),
		"judul":   "Sosialisasi Program Kerja Desa",
		"isi":     "Sosialisasi program kerja digitalisasi desa bersama warga.",
		"status":  "disubmit",
	})
	require.Equal(t, http.StatusCreated, respJurnal.Code, "buat jurnal KKN gagal: %s", respJurnal.Body.String())

	respLogbook := executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/logbook", idPeserta1), tokenPembimbing, map[string]interface{}{
		"tanggal":         time.Now().Format("2006-01-02"),
		"topik_bimbingan": "Monitoring pelaksanaan proker KKN",
		"catatan":         "Koordinasi dengan aparat desa berjalan baik.",
	})
	require.Equal(t, http.StatusCreated, respLogbook.Code, "buat logbook KKN gagal: %s", respLogbook.Body.String())

	// 10. Admin Menetapkan Dosen Penguji untuk Kedua Peserta
	_ = executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/pendaftaran/%d/dosen-penguji", idPeserta1), tokenAdmin, map[string]interface{}{"id_dosen": dosPengujiID})
	_ = executeJSON(router, http.MethodPut, fmt.Sprintf("/admin/pendaftaran/%d/dosen-penguji", idPeserta2), tokenAdmin, map[string]interface{}{"id_dosen": dosPengujiID})

	// 11. Admin Menyelesaikan Periode KKN
	respSelesai := executeJSON(router, http.MethodPost, fmt.Sprintf("/admin/periode/%d/selesaikan", idPeriode), tokenAdmin, nil)
	require.Equal(t, http.StatusOK, respSelesai.Code)

	// 12. Penilaian Peserta 1 KKN oleh 3 Penilai
	_ = executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPeserta1), tokenPembimbing, map[string]interface{}{
		"catatan": "Kepemimpinan kelompok KKN sangat baik",
		"items":   []map[string]interface{}{{"id_instrumen": instPembimbingID, "nilai": 86.0}},
	})
	_ = executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPeserta1), tokenKepalaDesa, map[string]interface{}{
		"catatan": "Warga desa sangat terbantu dengan program kerja",
		"items":   []map[string]interface{}{{"id_instrumen": instKadesID, "nilai": 90.0}},
	})
	_ = executeJSON(router, http.MethodPost, fmt.Sprintf("/pendaftaran/%d/penilaian", idPeserta1), tokenPenguji, map[string]interface{}{
		"catatan": "Evaluasi dan presentasi laporan KKN komprehensif",
		"items":   []map[string]interface{}{{"id_instrumen": instPengujiID, "nilai": 85.0}},
	})

	// 13. Verifikasi Nilai Akhir Peserta 1 KKN
	respFinal := executeJSON(router, http.MethodGet, fmt.Sprintf("/pendaftaran/%d", idPeserta1), tokenMhs1, nil)
	require.Equal(t, http.StatusOK, respFinal.Code)
	var finalBody struct {
		Data struct {
			Status     string   `json:"status"`
			NilaiAkhir *float64 `json:"nilai_akhir"`
			GradeHuruf *string  `json:"grade_huruf"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respFinal.Body.Bytes(), &finalBody)
	assert.Equal(t, "selesai", finalBody.Data.Status)
	require.NotNil(t, finalBody.Data.NilaiAkhir)
	assert.Greater(t, *finalBody.Data.NilaiAkhir, 80.0)
	assert.Equal(t, "A", *finalBody.Data.GradeHuruf)
}
