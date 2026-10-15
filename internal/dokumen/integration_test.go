package dokumen_test

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
	"path/filepath"
	"strings"
	"testing"
	"time"

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

			_, err = tx.Exec(ctx, "INSERT INTO user_roles (id_user, id_role) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, roleID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)
	return userID
}

func setupMahasiswa(t *testing.T, pool *pgxpool.Pool, userID, prodiID int, nim, nama string) int {
	ctx := context.Background()
	var mhsID int
	err := database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", userID).Scan(&mhsID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
				INSERT INTO mahasiswa (id_user, id_prodi, nim, nama, angkatan)
				VALUES ($1, $2, $3, $4, 2024)
				RETURNING id
			`, userID, prodiID, nim, nama).Scan(&mhsID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	return mhsID
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

func TestIntegration_Dokumen(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	// Gunakan temporary directory untuk upload lokal
	uploadDir := t.TempDir()

	cfg := &config.Config{
		AppEnv:        "development",
		AppPort:       "8080",
		DatabaseURL:   os.Getenv("TEST_DATABASE_URL"),
		JWTSecret:     "super-secret-jwt-key-for-test-32bytes!",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 24 * time.Hour,
		UploadDir:     uploadDir,
		MaxUploadMB:   5, // 5 MB batas maksimal
		LogLevel:      "warn",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := app.NewRouter(cfg, pool, logger)

	prodiID := ensureAkademik(t, pool)

	// Setup users
	uidAdmin := setupUser(t, pool, "admin_dok", "admin_dok@undiksha.ac.id", "pass12345", "admin")
	tokenAdmin, _ := security.GenerateToken(uidAdmin, []string{"admin"}, cfg.JWTAccessTTL, cfg.JWTSecret)

	uidMhsA := setupUser(t, pool, "mhs_dok_a", "mhs_dok_a@undiksha.ac.id", "pass12345", "mahasiswa")
	mhsAID := setupMahasiswa(t, pool, uidMhsA, prodiID, "NIM_DOK_A", "Mahasiswa Dokumen A")
	tokenMhsA, _ := security.GenerateToken(uidMhsA, []string{"mahasiswa"}, cfg.JWTAccessTTL, cfg.JWTSecret)

	uidMhsB := setupUser(t, pool, "mhs_dok_b", "mhs_dok_b@undiksha.ac.id", "pass12345", "mahasiswa")
	_ = setupMahasiswa(t, pool, uidMhsB, prodiID, "NIM_DOK_B", "Mahasiswa Dokumen B")
	tokenMhsB, _ := security.GenerateToken(uidMhsB, []string{"mahasiswa"}, cfg.JWTAccessTTL, cfg.JWTSecret)

	// Buat Periode aktif Magang
	ctx := context.Background()
	var periodeID int
	now := time.Now()
	err := pool.QueryRow(ctx, `
		INSERT INTO periode (
			nama, jenis_program, tahun_semester,
			tanggal_mulai, tanggal_selesai,
			tanggal_buka_daftar, tanggal_tutup_daftar,
			status
		) VALUES (
			'Magang Dokumen 2026', 'magang', '2026/2027 Gasal',
			$1, $2, $3, $4, 'aktif'
		) RETURNING id
	`, now.AddDate(0, 0, -2), now.AddDate(0, 6, 0), now.AddDate(0, 0, -5), now.AddDate(0, 0, 10)).Scan(&periodeID)
	require.NoError(t, err)

	// Buat tempat program
	var tempatID int
	err = pool.QueryRow(ctx, `
		INSERT INTO tempat_program (jenis_program, nama, alamat, is_aktif)
		VALUES ('magang', 'Tempat Dokumen Test', 'Jl. Test', true)
		RETURNING id
	`).Scan(&tempatID)
	require.NoError(t, err)

	// Buat pendaftaran Mahasiswa A
	var idPesertaA int
	err = pool.QueryRow(ctx, `
		INSERT INTO peserta_program (
			id_mahasiswa, id_periode, jenis_program,
			id_tempat_program, status
		) VALUES (
			$1, $2, 'magang', $3, 'diajukan'
		) RETURNING id
	`, mhsAID, periodeID, tempatID).Scan(&idPesertaA)
	require.NoError(t, err)

	var docID int
	pdfValidContent := []byte("%PDF-1.4 sample pdf content for splk integration test")

	t.Run("1. Unggah PDF valid berhasil (201 Created) dan file tersimpan di storage", func(t *testing.T) {
		req := makeMultipartRequest(
			fmt.Sprintf("/api/v1/pendaftaran/%d/dokumen", idPesertaA),
			http.MethodPost,
			tokenMhsA,
			"file",
			"MoU_Perusahaan.pdf",
			pdfValidContent,
			map[string]string{"jenis_dokumen": "mou"},
		)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data struct {
				ID           int    `json:"id"`
				NamaFile     string `json:"nama_file"`
				JenisDokumen string `json:"jenis_dokumen"`
				MimeType     string `json:"mime_type"`
				UkuranFile   int64  `json:"ukuran_file"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Data.ID > 0)
		assert.Equal(t, "MoU_Perusahaan.pdf", resp.Data.NamaFile)
		assert.Equal(t, "mou", resp.Data.JenisDokumen)
		assert.Equal(t, "application/pdf", resp.Data.MimeType)
		assert.Equal(t, int64(len(pdfValidContent)), resp.Data.UkuranFile)

		docID = resp.Data.ID

		// Verifikasi file fisik tersimpan di disk dalam uploadDir
		// Format: uploadDir/pendaftaran/<idPeserta>/<uuid>.pdf
		pendaftaranFolder := filepath.Join(uploadDir, "pendaftaran", fmt.Sprintf("%d", idPesertaA))
		files, err := os.ReadDir(pendaftaranFolder)
		require.NoError(t, err)
		assert.Equal(t, 1, len(files))
		assert.True(t, strings.HasSuffix(files[0].Name(), ".pdf"))
		// Pastikan nama file di disk bukan nama asli
		assert.False(t, strings.Contains(files[0].Name(), "MoU"))
	})

	t.Run("2. Unggah file 6 MB ditolak (422 Unprocessable Entity)", func(t *testing.T) {
		largeContent := make([]byte, 6*1024*1024)
		copy(largeContent, []byte("%PDF-1.4 large file"))

		req := makeMultipartRequest(
			fmt.Sprintf("/api/v1/pendaftaran/%d/dokumen", idPesertaA),
			http.MethodPost,
			tokenMhsA,
			"file",
			"Dokumen_Besar.pdf",
			largeContent,
			map[string]string{"jenis_dokumen": "proposal"},
		)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "FILE_TERLALU_BESAR")
	})

	t.Run("3. .exe yang di-rename .pdf ditolak (422 Unprocessable Entity)", func(t *testing.T) {
		fakePDFContent := []byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xFF\xFF\x00\x00")

		req := makeMultipartRequest(
			fmt.Sprintf("/api/v1/pendaftaran/%d/dokumen", idPesertaA),
			http.MethodPost,
			tokenMhsA,
			"file",
			"virus_samaran.pdf",
			fakePDFContent,
			map[string]string{"jenis_dokumen": "proposal"},
		)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "KONTEN_TIDAK_SESUAI")
	})

	t.Run("4. Mahasiswa lain (B) tidak bisa mengunduh dokumen pendaftaran A (404 Not Found)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/dokumen/%d/download", docID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("5. Mahasiswa pemilik (A) bisa mengunduh dokumen (200 OK)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/dokumen/%d/download", docID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsA)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Disposition"), "MoU_Perusahaan.pdf")
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		assert.Equal(t, pdfValidContent, w.Body.Bytes())
	})

	t.Run("6. Admin bisa mengunduh dokumen pendaftaran A (200 OK)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/dokumen/%d/download", docID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenAdmin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, pdfValidContent, w.Body.Bytes())
	})

	t.Run("7. Mahasiswa pemilik melihat daftar dokumen (200 OK)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pendaftaran/%d/dokumen", idPesertaA), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsA)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var listResp struct {
			Data []map[string]any `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &listResp)
		require.NoError(t, err)
		assert.Equal(t, 1, len(listResp.Data))
		assert.Equal(t, "MoU_Perusahaan.pdf", listResp.Data[0]["nama_file"])
		// Verifikasi path internal server tidak dibocorkan
		assert.Nil(t, listResp.Data[0]["file_url"])
	})

	t.Run("8. Mahasiswa lain (B) melihat daftar dokumen pendaftaran A mendapat 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pendaftaran/%d/dokumen", idPesertaA), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("9. Mahasiswa lain (B) mencoba menghapus dokumen A ditolak (403 Forbidden)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/dokumen/%d", docID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsB)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("10. Mahasiswa pemilik (A) menghapus dokumen saat diajukan berhasil (200 OK)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/dokumen/%d", docID), nil)
		req.Header.Set("Authorization", "Bearer "+tokenMhsA)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verifikasi dari DB
		var count int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM dokumen WHERE id = $1", docID).Scan(&count)
		assert.Equal(t, 0, count)

		// Verifikasi file fisik di disk juga sudah terhapus
		pendaftaranFolder := filepath.Join(uploadDir, "pendaftaran", fmt.Sprintf("%d", idPesertaA))
		files, err := os.ReadDir(pendaftaranFolder)
		require.NoError(t, err)
		assert.Equal(t, 0, len(files))
	})
}
