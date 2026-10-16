package kkn_test

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
			VALUES ('kepala_desa', $1, $2, $3, $4, '081299998888')
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

func makeMultipartRequest(url, method, token, fieldName, filename string, fileContent []byte, formFields map[string]string) *http.Request {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for k, v := range formFields {
		_ = writer.WriteField(k, v)
	}

	part, _ := writer.CreateFormFile(fieldName, filename)
	_, _ = part.Write(fileContent)
	_ = writer.Close()

	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// TestIntegration_CabangKKN menguji skenario KKN sampai berjalan dengan 2 mahasiswa:
// 1. Setup periode KKN, tempat KKN, kepala desa, dan dosen pembimbing.
// 2. Mahasiswa A dan B mendaftar KKN.
// 3. Admin membuat kelompok KKN.
// 4. Admin menambahkan Mahasiswa A dan B ke kelompok, menetapkan A sebagai ketua.
// 5. Coba tetapkan ketua kedua -> ditolak 409 Conflict.
// 6. Admin menetapkan dosen pembimbing kelompok -> kedua anggota ikut terupdate.
// 7. Admin menetapkan kepala desa kelompok -> kedua anggota ikut terupdate.
// 8. Anggota baru (Mahasiswa C) ditambahkan -> mewarisi dosen pembimbing dan kepala desa.
// 9. Mahasiswa C dikeluarkan -> field kelompok dikosongkan.
// 10. Unggah proker kelompok -> SELURUH anggota kelompok (A dan B) berstatus berjalan.
// 11. Rollback atomik: bila update salah satu anggota gagal, tidak ada perubahan yang tertinggal.
func TestIntegration_CabangKKN(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()

	// Bersihkan data relevan
	_, _ = pool.Exec(ctx, `
		DELETE FROM dokumen;
		DELETE FROM peserta_program;
		DELETE FROM kelompok_kkn;
		DELETE FROM pembimbing_lapangan;
		DELETE FROM tempat_program;
		DELETE FROM periode;
		DELETE FROM log_aktivitas;
	`)

	cfg := &config.Config{
		AppEnv:        "test",
		JWTSecret:     "kkn-integration-test-secret-key-very-long",
		JWTAccessTTL:  1 * time.Hour,
		JWTRefreshTTL: 24 * time.Hour,
		UploadDir:     t.TempDir(),
		MaxUploadMB:   5,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	prodiID := ensureAkademik(t, pool)

	// Setup users
	uidAdmin := setupUser(t, pool, "admin_kkn", "admin_kkn@undiksha.ac.id", "Secret123!", "admin")
	_ = uidAdmin
	uidMhsA := setupUser(t, pool, "mhs_kkn_a", "mhs_kkn_a@undiksha.ac.id", "Secret123!", "mahasiswa")
	uidMhsB := setupUser(t, pool, "mhs_kkn_b", "mhs_kkn_b@undiksha.ac.id", "Secret123!", "mahasiswa")
	uidMhsC := setupUser(t, pool, "mhs_kkn_c", "mhs_kkn_c@undiksha.ac.id", "Secret123!", "mahasiswa")
	uidDosen := setupUser(t, pool, "dosen_kkn", "dosen_kkn@undiksha.ac.id", "Secret123!", "dosen")
	uidKepalaDesa := setupUser(t, pool, "kades_tejakula", "kades@tejakula.desa.id", "Secret123!", "pembimbing_lapangan")

	ensureMahasiswaProfile(t, pool, uidMhsA, prodiID, "230101001", "Mahasiswa KKN A")
	ensureMahasiswaProfile(t, pool, uidMhsB, prodiID, "230101002", "Mahasiswa KKN B")
	ensureMahasiswaProfile(t, pool, uidMhsC, prodiID, "230101003", "Mahasiswa KKN C")

	idDosen := ensureDosenProfile(t, pool, uidDosen, prodiID, "198701012015011001", "Dr. Dosen KKN, M.Kom.")

	// Periode KKN
	var idPeriode int
	err := pool.QueryRow(ctx, `
		INSERT INTO periode (nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai,
		                     tanggal_buka_daftar, tanggal_tutup_daftar, status)
		VALUES ('KKN Tematik 2026', 'kkn', '2026/2027 Gasal', CURRENT_DATE, CURRENT_DATE + 60,
		        CURRENT_DATE - 1, CURRENT_DATE + 30, 'aktif')
		RETURNING id
	`).Scan(&idPeriode)
	require.NoError(t, err)

	// Tempat KKN (Desa Tejakula)
	var idTempatKKN int
	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, kontak, is_aktif)
		VALUES ('kkn', 'Desa Tejakula', 'Jl. Raya Singaraja-Amlapura', 'Buleleng', '08123456789', true)
		RETURNING id
	`).Scan(&idTempatKKN)
	require.NoError(t, err)

	// Kepala Desa (Pembimbing Lapangan KKN)
	idKepalaDesa := ensurePembimbingLapangan(t, pool, idTempatKKN, uidKepalaDesa, "I Made Teja", "Perbekel Tejakula")

	// Login tokens
	tokenAdmin := getAuthToken(t, router, "admin_kkn", "Secret123!")
	tokenMhsA := getAuthToken(t, router, "mhs_kkn_a", "Secret123!")
	tokenMhsB := getAuthToken(t, router, "mhs_kkn_b", "Secret123!")
	tokenMhsC := getAuthToken(t, router, "mhs_kkn_c", "Secret123!")
	tokenDosen := getAuthToken(t, router, "dosen_kkn", "Secret123!")

	var pendaftaranIDA, pendaftaranIDB, pendaftaranIDC int

	t.Run("1. Mahasiswa A, B, dan C mendaftar KKN -> 201 Created", func(t *testing.T) {
		payloadDaftar := map[string]any{"id_periode": idPeriode}
		raw, _ := json.Marshal(payloadDaftar)

		// Mhs A
		reqA := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		reqA.Header.Set("Authorization", "Bearer "+tokenMhsA)
		reqA.Header.Set("Content-Type", "application/json")
		wA := httptest.NewRecorder()
		router.ServeHTTP(wA, reqA)
		require.Equal(t, http.StatusCreated, wA.Code)

		var respA struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wA.Body.Bytes(), &respA)
		pendaftaranIDA = respA.Data.ID
		require.Greater(t, pendaftaranIDA, 0)

		// Mhs B
		reqB := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		reqB.Header.Set("Authorization", "Bearer "+tokenMhsB)
		reqB.Header.Set("Content-Type", "application/json")
		wB := httptest.NewRecorder()
		router.ServeHTTP(wB, reqB)
		require.Equal(t, http.StatusCreated, wB.Code)

		var respB struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wB.Body.Bytes(), &respB)
		pendaftaranIDB = respB.Data.ID
		require.Greater(t, pendaftaranIDB, 0)

		// Mhs C
		reqC := httptest.NewRequest(http.MethodPost, "/pendaftaran", bytes.NewReader(raw))
		reqC.Header.Set("Authorization", "Bearer "+tokenMhsC)
		reqC.Header.Set("Content-Type", "application/json")
		wC := httptest.NewRecorder()
		router.ServeHTTP(wC, reqC)
		require.Equal(t, http.StatusCreated, wC.Code)

		var respC struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wC.Body.Bytes(), &respC)
		pendaftaranIDC = respC.Data.ID
		require.Greater(t, pendaftaranIDC, 0)
	})

	var idKelompok int

	t.Run("2. Admin membuat kelompok KKN -> 201 Created", func(t *testing.T) {
		body := map[string]any{
			"id_periode":        idPeriode,
			"id_tempat_program": idTempatKKN,
			"nama_kelompok":     "Kelompok KKN 01 Desa Tejakula",
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/admin/kkn/kelompok", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "buat kelompok kkn gagal: %s", w.Body.String())

		var resp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		idKelompok = resp.Data.ID
		require.Greater(t, idKelompok, 0)
	})

	t.Run("3. Admin menambahkan Mahasiswa A dan B ke kelompok, menetapkan A sebagai ketua -> 200 OK", func(t *testing.T) {
		body := map[string]any{
			"id_peserta_program": []int{pendaftaranIDA, pendaftaranIDB},
			"id_ketua":           pendaftaranIDA,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/kkn/kelompok/%d/anggota", idKelompok), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "tambah anggota gagal: %s", w.Body.String())

		// Verifikasi di DB:
		// A: is_ketua = true, id_tempat_program = idTempatKKN, id_kelompok_kkn = idKelompok
		var isKetuaA, isKetuaB bool
		var tptA, tptB int
		err := pool.QueryRow(ctx, "SELECT is_ketua, id_tempat_program FROM peserta_program WHERE id = $1", pendaftaranIDA).Scan(&isKetuaA, &tptA)
		require.NoError(t, err)
		assert.True(t, isKetuaA)
		assert.Equal(t, idTempatKKN, tptA)

		err = pool.QueryRow(ctx, "SELECT is_ketua, id_tempat_program FROM peserta_program WHERE id = $1", pendaftaranIDB).Scan(&isKetuaB, &tptB)
		require.NoError(t, err)
		assert.False(t, isKetuaB)
		assert.Equal(t, idTempatKKN, tptB)
	})

	t.Run("4. Menetapkan ketua kedua ditolak (409 Conflict KETUA_SUDAH_ADA)", func(t *testing.T) {
		body := map[string]any{
			"id_peserta_program": []int{pendaftaranIDB},
			"id_ketua":           pendaftaranIDB,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/kkn/kelompok/%d/anggota", idKelompok), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "KETUA_SUDAH_ADA")
	})

	t.Run("5. Admin menetapkan dosen pembimbing kelompok -> disinkronkan ke kedua anggota dalam satu transaksi", func(t *testing.T) {
		body := map[string]any{
			"id_dosen": idDosen,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/kkn/kelompok/%d/dosen-pembimbing", idKelompok), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "set dosen pembimbing kelompok gagal: %s", w.Body.String())

		// Verifikasi sinkronisasi
		var dosenKelompok *int
		err := pool.QueryRow(ctx, "SELECT id_dosen_pembimbing FROM kelompok_kkn WHERE id = $1", idKelompok).Scan(&dosenKelompok)
		require.NoError(t, err)
		require.NotNil(t, dosenKelompok)
		assert.Equal(t, idDosen, *dosenKelompok)

		var dosenA, dosenB *int
		err = pool.QueryRow(ctx, "SELECT id_dosen_pembimbing FROM peserta_program WHERE id = $1", pendaftaranIDA).Scan(&dosenA)
		require.NoError(t, err)
		require.NotNil(t, dosenA)
		assert.Equal(t, idDosen, *dosenA)

		err = pool.QueryRow(ctx, "SELECT id_dosen_pembimbing FROM peserta_program WHERE id = $1", pendaftaranIDB).Scan(&dosenB)
		require.NoError(t, err)
		require.NotNil(t, dosenB)
		assert.Equal(t, idDosen, *dosenB)
	})

	t.Run("6. Admin menetapkan kepala desa / pembimbing lapangan kelompok -> disinkronkan ke kedua anggota", func(t *testing.T) {
		body := map[string]any{
			"id_pembimbing_lapangan": idKepalaDesa,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/kkn/kelompok/%d/pembimbing-lapangan", idKelompok), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "set pembimbing lapangan kelompok gagal: %s", w.Body.String())

		// Verifikasi sinkronisasi
		var plKelompok *int
		err := pool.QueryRow(ctx, "SELECT id_pembimbing_lapangan FROM kelompok_kkn WHERE id = $1", idKelompok).Scan(&plKelompok)
		require.NoError(t, err)
		require.NotNil(t, plKelompok)
		assert.Equal(t, idKepalaDesa, *plKelompok)

		var plA, plB *int
		err = pool.QueryRow(ctx, "SELECT id_pembimbing_lapangan FROM peserta_program WHERE id = $1", pendaftaranIDA).Scan(&plA)
		require.NoError(t, err)
		require.NotNil(t, plA)
		assert.Equal(t, idKepalaDesa, *plA)

		err = pool.QueryRow(ctx, "SELECT id_pembimbing_lapangan FROM peserta_program WHERE id = $1", pendaftaranIDB).Scan(&plB)
		require.NoError(t, err)
		require.NotNil(t, plB)
		assert.Equal(t, idKepalaDesa, *plB)
	})

	t.Run("7. Anggota baru (Mahasiswa C) ditambahkan sesudah penetapan -> otomatis mewarisi dosen & pembimbing lapangan kelompok", func(t *testing.T) {
		body := map[string]any{
			"id_peserta_program": []int{pendaftaranIDC},
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/kkn/kelompok/%d/anggota", idKelompok), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var dosenC, plC *int
		var tptC int
		err := pool.QueryRow(ctx, "SELECT id_dosen_pembimbing, id_pembimbing_lapangan, id_tempat_program FROM peserta_program WHERE id = $1", pendaftaranIDC).Scan(&dosenC, &plC, &tptC)
		require.NoError(t, err)
		require.NotNil(t, dosenC)
		require.NotNil(t, plC)
		assert.Equal(t, idDosen, *dosenC)
		assert.Equal(t, idKepalaDesa, *plC)
		assert.Equal(t, idTempatKKN, tptC)
	})

	t.Run("8. Keluarkan Mahasiswa C -> berhasil, kolom kelompok, tempat, dosen, pl dikosongkan", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/kkn/kelompok/%d/anggota/%d", idKelompok, pendaftaranIDC), nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var kelC, tptC, dsnC, plC *int
		var isKetuaC bool
		err := pool.QueryRow(ctx, "SELECT id_kelompok_kkn, id_tempat_program, id_dosen_pembimbing, id_pembimbing_lapangan, is_ketua FROM peserta_program WHERE id = $1", pendaftaranIDC).Scan(&kelC, &tptC, &dsnC, &plC, &isKetuaC)
		require.NoError(t, err)
		assert.Nil(t, kelC)
		assert.Nil(t, tptC)
		assert.Nil(t, dsnC)
		assert.Nil(t, plC)
		assert.False(t, isKetuaC)
	})

	t.Run("9. Dosen pembimbing dapat melihat detail kelompok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/kkn/kelompok/%d", idKelompok), nil)
		req.Header.Set("Authorization", "Bearer "+tokenDosen)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data struct {
				NamaKelompok string `json:"nama_kelompok"`
				Ketua        struct {
					Nama string `json:"nama"`
				} `json:"ketua"`
				Anggota []struct {
					IDPesertaProgram int `json:"id_peserta_program"`
				} `json:"anggota"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Kelompok KKN 01 Desa Tejakula", resp.Data.NamaKelompok)
		assert.Equal(t, "Mahasiswa KKN A", resp.Data.Ketua.Nama)
		assert.Equal(t, 2, len(resp.Data.Anggota)) // A dan B
	})

	t.Run("10. Anggota kelompok mengunggah proker kelompok -> SELURUH anggota kelompok bertransisi ke status berjalan", func(t *testing.T) {
		// Pastikan status sebelum unggah adalah 'diajukan'
		var statusA, statusB string
		_ = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranIDA).Scan(&statusA)
		_ = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranIDB).Scan(&statusB)
		assert.Equal(t, "diajukan", statusA)
		assert.Equal(t, "diajukan", statusB)

		pdfDummy := []byte("%PDF-1.4 dummy proker kkn desa tejakula")
		reqProker := makeMultipartRequest(
			fmt.Sprintf("/kkn/kelompok/%d/dokumen", idKelompok),
			http.MethodPost,
			tokenMhsA, // Diunggah oleh Mahasiswa A
			"file",
			"proker_kkn_tejakula.pdf",
			pdfDummy,
			map[string]string{"jenis_dokumen": "proker"},
		)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, reqProker)
		require.Equal(t, http.StatusCreated, w.Code, "unggah proker kelompok gagal: %s", w.Body.String())

		// Verifikasi: KEDUA anggota kelompok (A dan B) harus berstatus 'berjalan'
		err = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranIDA).Scan(&statusA)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", statusA)

		err = pool.QueryRow(ctx, "SELECT status FROM peserta_program WHERE id = $1", pendaftaranIDB).Scan(&statusB)
		require.NoError(t, err)
		assert.Equal(t, "berjalan", statusB)
	})

	t.Run("11. Atomisitas rollback: bila penetapan dosen pembimbing melanggar BR-07 pada salah satu anggota, seluruh update dibatalkan", func(t *testing.T) {
		uidDosenPenguji := setupUser(t, pool, "penguji_b", "penguji_b@undiksha.ac.id", "Secret123!", "dosen")
		idDosenPengujiB := ensureDosenProfile(t, pool, uidDosenPenguji, prodiID, "198901012018011002", "Dosen Penguji B, M.T.")

		// Set idDosenPengujiB pada pendaftaran Mahasiswa B
		_, err = pool.Exec(ctx, "UPDATE peserta_program SET id_dosen_penguji = $1 WHERE id = $2", idDosenPengujiB, pendaftaranIDB)
		require.NoError(t, err)

		// Sekarang admin mencoba mengganti dosen pembimbing kelompok menjadi idDosenPengujiB
		// Harus ditolak dengan 422 DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI
		body := map[string]any{"id_dosen": idDosenPengujiB}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/kkn/kelompok/%d/dosen-pembimbing", idKelompok), bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI")

		// Verifikasi atomisitas: dosen pembimbing di kelompok dan kedua anggota TIDAK berubah
		var currDosenKel int
		err = pool.QueryRow(ctx, "SELECT id_dosen_pembimbing FROM kelompok_kkn WHERE id = $1", idKelompok).Scan(&currDosenKel)
		require.NoError(t, err)
		assert.Equal(t, idDosen, currDosenKel)

		var currDosenA, currDosenB int
		err = pool.QueryRow(ctx, "SELECT id_dosen_pembimbing FROM peserta_program WHERE id = $1", pendaftaranIDA).Scan(&currDosenA)
		require.NoError(t, err)
		assert.Equal(t, idDosen, currDosenA)

		err = pool.QueryRow(ctx, "SELECT id_dosen_pembimbing FROM peserta_program WHERE id = $1", pendaftaranIDB).Scan(&currDosenB)
		require.NoError(t, err)
		assert.Equal(t, idDosen, currDosenB)
	})
}
