package user_test

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
	"splk-backend/internal/user"
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

func ensureAdminUser(t *testing.T, pool *pgxpool.Pool, username, email, password string) {
	ctx := context.Background()
	err := database.WithTx(ctx, pool, func(tx pgx.Tx) error {
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
			VALUES ($1, 'Admin Tahap 4', 'ADM004')
			ON CONFLICT (id_user) DO NOTHING
		`, userID)
		return err
	})
	require.NoError(t, err)
}

func getAdminAuthToken(t *testing.T, router *gin.Engine, username, password string) string {
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "gagal login admin: %s", w.Body.String())

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp.Data.Token
}

func doHTTP(router *gin.Engine, method, url string, payload any, token string) *httptest.ResponseRecorder {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUserAndMasterDataIntegration(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cfg := &config.Config{
		AppEnv:        "testing",
		JWTSecret:     "integration_test_secret_key_minimum_32_chars!",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		AppTimezone:   time.FixedZone("WITA", 8*3600),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	adminUser := "admin_master_t4"
	adminPass := "AdminMaster123!"
	ensureAdminUser(t, pool, adminUser, "admin_master_t4@undiksha.ac.id", adminPass)
	adminToken := getAdminAuthToken(t, router, adminUser, adminPass)
	require.NotEmpty(t, adminToken)

	timestamp := time.Now().UnixNano()

	// 1. Setup Master Akademik: Fakultas -> Jurusan -> Program Studi
	var prodiID int
	{
		// Fakultas
		fakReq := map[string]any{
			"nama": fmt.Sprintf("Fakultas Teknik %d", timestamp),
			"kode": "FTK",
		}
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/fakultas", fakReq, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var fakResp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &fakResp)
		require.NoError(t, err)
		fakID := fakResp.Data.ID

		// Jurusan
		jurReq := map[string]any{
			"id_fakultas":  fakID,
			"nama":         fmt.Sprintf("Jurusan Teknik Informatika %d", timestamp),
			"kode_jurusan": "JTI",
		}
		w = doHTTP(router, http.MethodPost, "/api/v1/admin/jurusan", jurReq, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var jurResp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &jurResp)
		require.NoError(t, err)
		jurID := jurResp.Data.ID

		// Program Studi
		prodiReq := map[string]any{
			"id_jurusan": jurID,
			"nama":       fmt.Sprintf("S1 Sistem Informasi %d", timestamp),
			"kode":       "SI",
		}
		w = doHTTP(router, http.MethodPost, "/api/v1/admin/program-studi", prodiReq, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var prodiResp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &prodiResp)
		require.NoError(t, err)
		prodiID = prodiResp.Data.ID
		require.Positive(t, prodiID)
	}

	// 2. Setup Master Tempat Program (Magang, Asistensi Mengajar, KKN)
	var tempatMagangID, tempatAMID, tempatKKNID int
	{
		// Magang
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/tempat-program", map[string]any{
			"jenis_program": "magang",
			"nama":          fmt.Sprintf("PT Bali Software %d", timestamp),
			"alamat":        "Denpasar",
			"kota":          "Denpasar",
		}, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var tResp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &tResp)
		tempatMagangID = tResp.Data.ID

		// Asistensi Mengajar
		w = doHTTP(router, http.MethodPost, "/api/v1/admin/tempat-program", map[string]any{
			"jenis_program": "asistensi_mengajar",
			"nama":          fmt.Sprintf("SMA Negeri 1 Singaraja %d", timestamp),
			"alamat":        "Singaraja",
			"kota":          "Buleleng",
		}, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		_ = json.Unmarshal(w.Body.Bytes(), &tResp)
		tempatAMID = tResp.Data.ID

		// KKN
		w = doHTTP(router, http.MethodPost, "/api/v1/admin/tempat-program", map[string]any{
			"jenis_program": "kkn",
			"nama":          fmt.Sprintf("Desa Ambengan %d", timestamp),
			"alamat":        "Sukasada",
			"kota":          "Buleleng",
		}, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		_ = json.Unmarshal(w.Body.Bytes(), &tResp)
		tempatKKNID = tResp.Data.ID

		require.Positive(t, tempatMagangID)
		require.Positive(t, tempatAMID)
		require.Positive(t, tempatKKNID)
	}

	// 3. Admin Buat Akun Aktor Sesuai Spesifikasi:
	// - 1 Mahasiswa
	// - 2 Dosen
	// - 1 Pegawai
	// - 3 Pembimbing Lapangan (1 mentor -> magang, 1 guru_pamong -> AM, 1 kepala_desa -> KKN)
	mhsUser := fmt.Sprintf("mhs_%d", timestamp)
	mhsPass := "PasswordMhs123!"
	dosen1User := fmt.Sprintf("dosen1_%d", timestamp)
	dosen1Pass := "PasswordDosen123!"
	dosen2User := fmt.Sprintf("dosen2_%d", timestamp)
	dosen2Pass := "PasswordDosen123!"
	pegawaiUser := fmt.Sprintf("pegawai_%d", timestamp)
	pegawaiPass := "PasswordPeg123!"

	mentorUser := fmt.Sprintf("mentor_%d", timestamp)
	guruUser := fmt.Sprintf("guru_%d", timestamp)
	kadesUser := fmt.Sprintf("kades_%d", timestamp)
	plPass := "PasswordPL123!"

	var mhsDetail user.MahasiswaDetail

	t.Run("Admin Buat 1 Mahasiswa", func(t *testing.T) {
		req := map[string]any{
			"username":          mhsUser,
			"email":             fmt.Sprintf("%s@mhs.undiksha.ac.id", mhsUser),
			"password":          mhsPass,
			"nim":               fmt.Sprintf("NIM%d", timestamp%10000000),
			"nama":              "Mahasiswa Satu",
			"id_prodi":          prodiID,
			"angkatan":          2022,
			"semester_sekarang": 5,
		}
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/mahasiswa", req, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		var resp struct {
			Data user.MahasiswaDetail `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		mhsDetail = resp.Data
		assert.Equal(t, mhsUser, mhsDetail.Username)
		assert.True(t, mhsDetail.IsActive)
		assert.Equal(t, prodiID, mhsDetail.ProdiID)
	})

	t.Run("Admin Buat 2 Dosen", func(t *testing.T) {
		req1 := map[string]any{
			"username":       dosen1User,
			"email":          fmt.Sprintf("%s@undiksha.ac.id", dosen1User),
			"password":       dosen1Pass,
			"nip":            fmt.Sprintf("NIP1_%d", timestamp%10000000),
			"nama":           "Dr. Dosen Satu",
			"id_prodi":       prodiID,
			"gelar_depan":    "Dr.",
			"gelar_belakang": "M.Kom",
		}
		w1 := doHTTP(router, http.MethodPost, "/api/v1/admin/dosen", req1, adminToken)
		require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

		req2 := map[string]any{
			"username":       dosen2User,
			"email":          fmt.Sprintf("%s@undiksha.ac.id", dosen2User),
			"password":       dosen2Pass,
			"nip":            fmt.Sprintf("NIP2_%d", timestamp%10000000),
			"nama":           "Dosen Dua",
			"id_prodi":       prodiID,
			"gelar_belakang": "M.T",
		}
		w2 := doHTTP(router, http.MethodPost, "/api/v1/admin/dosen", req2, adminToken)
		require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())
	})

	t.Run("Admin Buat 1 Pegawai", func(t *testing.T) {
		req := map[string]any{
			"username": dosen2User + "_peg", // username unik
			"email":    fmt.Sprintf("%s_peg@undiksha.ac.id", pegawaiUser),
			"password": pegawaiPass,
			"nip":      fmt.Sprintf("NIP_PEG_%d", timestamp%10000000),
			"nama":     "Pegawai Tata Usaha",
			"jabatan":  "Staff Akademik",
		}
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/pegawai", req, adminToken)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	})

	t.Run("Admin Buat 3 Pembimbing Lapangan Sesuai Jenis dan Tempat", func(t *testing.T) {
		// 1. Mentor -> Tempat Magang
		req1 := map[string]any{
			"username":          mentorUser,
			"email":             fmt.Sprintf("%s@ptbali.com", mentorUser),
			"password":          plPass,
			"jenis":             "mentor",
			"id_tempat_program": tempatMagangID,
			"nama":              "Mentor Magang Senior",
			"jabatan":           "Tech Lead",
		}
		w1 := doHTTP(router, http.MethodPost, "/api/v1/admin/pembimbing-lapangan", req1, adminToken)
		require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

		// 2. Guru Pamong -> Tempat Asistensi Mengajar
		req2 := map[string]any{
			"username":          guruUser,
			"email":             fmt.Sprintf("%s@sman1.sch.id", guruUser),
			"password":          plPass,
			"jenis":             "guru_pamong",
			"id_tempat_program": tempatAMID,
			"nama":              "Guru Pamong Senior",
			"jabatan":           "Guru Bahasa",
		}
		w2 := doHTTP(router, http.MethodPost, "/api/v1/admin/pembimbing-lapangan", req2, adminToken)
		require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())

		// 3. Kepala Desa -> Tempat KKN
		req3 := map[string]any{
			"username":          kadesUser,
			"email":             fmt.Sprintf("%s@ambengan.desa.id", kadesUser),
			"password":          plPass,
			"jenis":             "kepala_desa",
			"id_tempat_program": tempatKKNID,
			"nama":              "Kepala Desa Ambengan",
			"jabatan":           "Perbekel",
		}
		w3 := doHTTP(router, http.MethodPost, "/api/v1/admin/pembimbing-lapangan", req3, adminToken)
		require.Equal(t, http.StatusCreated, w3.Code, w3.Body.String())
	})

	// 4. Verifikasi Akun Baru Bisa Login dan Mendapatkan Token
	var mhsToken string
	t.Run("Akun Baru Bisa Login", func(t *testing.T) {
		// Login Mahasiswa
		w := doHTTP(router, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"username": mhsUser,
			"password": mhsPass,
		}, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp struct {
			Data struct {
				Token string `json:"token"`
				User  struct {
					Roles []string `json:"roles"`
				} `json:"user"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Data.Token)
		assert.Contains(t, resp.Data.User.Roles, "mahasiswa")
		mhsToken = resp.Data.Token

		// Login Dosen
		w = doHTTP(router, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"username": dosen1User,
			"password": dosen1Pass,
		}, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		// Login Mentor (Pembimbing Lapangan)
		w = doHTTP(router, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"username": mentorUser,
			"password": plPass,
		}, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Data.User.Roles, "pembimbing_lapangan")
	})

	// 5. Verifikasi Otorisasi: Mahasiswa Mengakses Endpoint Admin Ditolak 403 Forbidden
	t.Run("Mahasiswa Mengakses Endpoint Admin Ditolak 403", func(t *testing.T) {
		require.NotEmpty(t, mhsToken)
		w := doHTTP(router, http.MethodGet, "/api/v1/admin/users", nil, mhsToken)
		assert.Equal(t, http.StatusForbidden, w.Code)

		w = doHTTP(router, http.MethodPost, "/api/v1/admin/mahasiswa", map[string]any{
			"username": "hacker",
		}, mhsToken)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// 6. Verifikasi Duplikasi Username / Email -> 409 Conflict
	t.Run("Duplikasi Username atau Email Ditolak 409 Conflict", func(t *testing.T) {
		reqDup := map[string]any{
			"username": mhsUser, // username yang sama
			"email":    fmt.Sprintf("unique_%d@mhs.undiksha.ac.id", timestamp),
			"password": "Password123!",
			"nim":      fmt.Sprintf("NIM_DIFF_%d", timestamp%10000000),
			"nama":     "Mahasiswa Duplikat",
			"id_prodi": prodiID,
			"angkatan": 2023,
		}
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/mahasiswa", reqDup, adminToken)
		assert.Equal(t, http.StatusConflict, w.Code)

		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Equal(t, "USERNAME_SUDAH_DIGUNAKAN", errResp.Error.Code)
	})

	// 7. Verifikasi Ketidakcocokan Jenis Pembimbing Lapangan dengan Tempat -> 422 Unprocessable
	t.Run("Mentor di Tempat KKN Ditolak 422 Unprocessable Entity", func(t *testing.T) {
		reqMismatch := map[string]any{
			"username":          fmt.Sprintf("mentor_kkn_%d", timestamp),
			"email":             fmt.Sprintf("mentor_kkn_%d@test.com", timestamp),
			"password":          plPass,
			"jenis":             "mentor",    // Mentor harusnya magang
			"id_tempat_program": tempatKKNID, // Tempat KKN
			"nama":              "Mentor di Desa",
		}
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/pembimbing-lapangan", reqMismatch, adminToken)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Equal(t, "JENIS_PEMBIMBING_TIDAK_COCOK", errResp.Error.Code)
	})

	// 8. Verifikasi Transaksi Atomik (Rollback Gagal Tidak Meninggalkan Orphan User)
	t.Run("Transaksi Atomik Rollback Tidak Meninggalkan Baris Orphan di users", func(t *testing.T) {
		var userCountBefore int
		err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&userCountBefore)
		require.NoError(t, err)

		// Mencoba buat mahasiswa dengan prodi yang tidak ada (999999)
		// Validasi prodi mencegah sebelum tx, mari kita uji dengan prodi_id valid tapi kita buat error di dalam transaksi
		// Jika kita panggil API dengan prodi tidak ada -> 422 (tertolak sebelum tx)
		invalidReq := map[string]any{
			"username": fmt.Sprintf("orphan_test_%d", timestamp),
			"email":    fmt.Sprintf("orphan_%d@test.com", timestamp),
			"password": "Password123!",
			"nim":      fmt.Sprintf("NIM_ORP_%d", timestamp%10000000),
			"nama":     "Orphan Tester",
			"id_prodi": 99999999, // tidak valid
			"angkatan": 2024,
		}
		w := doHTTP(router, http.MethodPost, "/api/v1/admin/mahasiswa", invalidReq, adminToken)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		// Langsung uji atomic rollback di level database.WithTx:
		ctx := context.Background()
		testRollbackUser := fmt.Sprintf("rollback_user_%d", timestamp)
		passHash, _ := security.HashPassword("Password123!")

		_ = database.WithTx(ctx, pool, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO users (username, email, password_hash, is_active)
				VALUES ($1, $2, $3, true)
			`, testRollbackUser, testRollbackUser+"@rollback.com", passHash)
			if err != nil {
				return err
			}
			// Simulasikan kegagalan pada insert profil kedua
			return errors.New("simulasi kegagalan insert profil anak")
		})

		var userFound bool
		err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", testRollbackUser).Scan(&userFound)
		require.NoError(t, err)
		assert.False(t, userFound, "user tidak boleh tersimpan karena transaksi di-rollback")
	})

	// 9. Verifikasi Soft-Delete Tempat Program
	t.Run("Soft Delete Tempat Program", func(t *testing.T) {
		w := doHTTP(router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/tempat-program/%d", tempatMagangID), nil, adminToken)
		require.Equal(t, http.StatusOK, w.Code)

		// Periksa di database bahwa is_aktif = false
		var isAktif bool
		err := pool.QueryRow(context.Background(), "SELECT is_aktif FROM tempat_program WHERE id = $1", tempatMagangID).Scan(&isAktif)
		require.NoError(t, err)
		assert.False(t, isAktif, "tempat program harus berstatus is_aktif = false setelah soft delete")

		// Data relasi pembimbing lapangan tetap ada (tidak cascade delete fisik)
		var plCount int
		err = pool.QueryRow(context.Background(), "SELECT count(*) FROM pembimbing_lapangan WHERE id_tempat_program = $1", tempatMagangID).Scan(&plCount)
		require.NoError(t, err)
		assert.Positive(t, plCount, "relasi pembimbing lapangan tidak boleh terhapus fisik")
	})

	// 10. Verifikasi Nonaktifkan User & Reset Password
	t.Run("Nonaktifkan User dan Reset Password", func(t *testing.T) {
		// Nonaktifkan Mahasiswa
		w := doHTTP(router, http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/status", mhsDetail.UserID), map[string]any{
			"is_active": false,
		}, adminToken)
		require.Equal(t, http.StatusOK, w.Code)

		// Mahasiswa mencoba login -> 401 Unauthorized (KREDENSIAL_TIDAK_VALID untuk cegah enumerasi user)
		w = doHTTP(router, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"username": mhsUser,
			"password": mhsPass,
		}, "")
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Equal(t, "KREDENSIAL_TIDAK_VALID", errResp.Error.Code)

		// Reset password mahasiswa oleh admin
		newMhsPass := "PasswordBaruMhs123!"
		w = doHTTP(router, http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/reset-password", mhsDetail.UserID), map[string]string{
			"password_baru": newMhsPass,
		}, adminToken)
		require.Equal(t, http.StatusOK, w.Code)

		// Aktifkan kembali mahasiswa
		w = doHTTP(router, http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/status", mhsDetail.UserID), map[string]any{
			"is_active": true,
		}, adminToken)
		require.Equal(t, http.StatusOK, w.Code)

		// Login dengan password baru sukses
		w = doHTTP(router, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"username": mhsUser,
			"password": newMhsPass,
		}, "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
