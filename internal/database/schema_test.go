package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSchemaMigration memeriksa bahwa skema database v2 dan patch 000002 telah terpasang dengan benar.
// Test ini dilewati jika TEST_DATABASE_URL kosong.
func TestSchemaMigration(t *testing.T) {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL tidak diatur; lewati test integrasi migrasi skema")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDBURL)
	if err != nil {
		t.Fatalf("koneksi ke test database gagal: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database gagal: %v", err)
	}

	// 1. Periksa keberadaan seluruh 24 tabel aplikasi
	t.Run("24 Tabel Aplikasi Ada", func(t *testing.T) {
		expectedTables := []string{
			"users",
			"roles",
			"user_roles",
			"tokens",
			"log_aktivitas",
			"fakultas",
			"jurusan",
			"program_studi",
			"mahasiswa",
			"dosen",
			"pegawai",
			"admin",
			"tempat_program",
			"pembimbing_lapangan",
			"periode",
			"kelompok_kkn",
			"peserta_program",
			"jurnal_harian",
			"logbook_bimbingan",
			"instrumen_penilaian",
			"penilaian",
			"dokumen",
			"surat_perjalanan_dinas",
			"pengumuman",
		}

		rows, err := pool.Query(ctx, `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		`)
		if err != nil {
			t.Fatalf("gagal mengambil daftar tabel: %v", err)
		}
		defer rows.Close()

		foundTables := make(map[string]bool)
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				t.Fatalf("scan table_name: %v", err)
			}
			foundTables[tableName] = true
		}
		if rows.Err() != nil {
			t.Fatalf("iterasi tabel: %v", rows.Err())
		}

		for _, table := range expectedTables {
			if !foundTables[table] {
				t.Errorf("tabel wajib '%s' tidak ditemukan di database", table)
			}
		}
	})

	// 2. Periksa tabel roles berisi 5 baris seed
	t.Run("Roles Berisi 5 Baris Seed", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM roles").Scan(&count)
		if err != nil {
			t.Fatalf("gagal menghitung baris tabel roles: %v", err)
		}
		if count != 5 {
			t.Errorf("jumlah baris roles: didapat %d, diharapkan 5", count)
		}

		expectedRoles := []string{"mahasiswa", "admin", "dosen", "pegawai", "pembimbing_lapangan"}
		for _, role := range expectedRoles {
			var exists bool
			err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM roles WHERE nama = $1)", role).Scan(&exists)
			if err != nil {
				t.Fatalf("gagal cek role '%s': %v", role, err)
			}
			if !exists {
				t.Errorf("role seed '%s' tidak ditemukan di tabel roles", role)
			}
		}
	})

	// 3. Periksa constraint unik 'uq_peserta_program_mahasiswa_periode_jenis' ada
	t.Run("Constraint Unik Peserta Mahasiswa Periode Jenis Ada", func(t *testing.T) {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.table_constraints 
				WHERE table_name = 'peserta_program' 
				  AND constraint_name = 'uq_peserta_program_mahasiswa_periode_jenis'
				  AND constraint_type = 'UNIQUE'
			)
		`
		if err := pool.QueryRow(ctx, query).Scan(&exists); err != nil {
			t.Fatalf("gagal cek constraint unik: %v", err)
		}
		if !exists {
			t.Errorf("constraint unik 'uq_peserta_program_mahasiswa_periode_jenis' tidak ditemukan")
		}
	})

	// 4. Periksa kolom peserta_program.id_tempat_program nullable (hasil migrasi 000002)
	t.Run("Kolom peserta_program.id_tempat_program Nullable", func(t *testing.T) {
		var isNullable string
		query := `
			SELECT is_nullable 
			FROM information_schema.columns 
			WHERE table_name = 'peserta_program' 
			  AND column_name = 'id_tempat_program'
		`
		if err := pool.QueryRow(ctx, query).Scan(&isNullable); err != nil {
			t.Fatalf("gagal cek nullability kolom id_tempat_program: %v", err)
		}
		if isNullable != "YES" {
			t.Errorf("peserta_program.id_tempat_program is_nullable: didapat '%s', diharapkan 'YES'", isNullable)
		}
	})
}
