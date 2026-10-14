-- =====================================================================
-- Migration Rollback: 000001_init_schema
-- Drop seluruh 24 tabel aplikasi dalam urutan aman (anak sebelum induk)
-- Jangan men-drop extension pgcrypto
-- =====================================================================

DROP TABLE IF EXISTS "surat_perjalanan_dinas" CASCADE;
DROP TABLE IF EXISTS "dokumen" CASCADE;
DROP TABLE IF EXISTS "penilaian" CASCADE;
DROP TABLE IF EXISTS "instrumen_penilaian" CASCADE;
DROP TABLE IF EXISTS "logbook_bimbingan" CASCADE;
DROP TABLE IF EXISTS "jurnal_harian" CASCADE;
DROP TABLE IF EXISTS "peserta_program" CASCADE;
DROP TABLE IF EXISTS "kelompok_kkn" CASCADE;
DROP TABLE IF EXISTS "periode" CASCADE;
DROP TABLE IF EXISTS "pembimbing_lapangan" CASCADE;
DROP TABLE IF EXISTS "tempat_program" CASCADE;
DROP TABLE IF EXISTS "pengumuman" CASCADE;
DROP TABLE IF EXISTS "log_aktivitas" CASCADE;
DROP TABLE IF EXISTS "tokens" CASCADE;
DROP TABLE IF EXISTS "user_roles" CASCADE;
DROP TABLE IF EXISTS "admin" CASCADE;
DROP TABLE IF EXISTS "pegawai" CASCADE;
DROP TABLE IF EXISTS "dosen" CASCADE;
DROP TABLE IF EXISTS "mahasiswa" CASCADE;
DROP TABLE IF EXISTS "program_studi" CASCADE;
DROP TABLE IF EXISTS "jurusan" CASCADE;
DROP TABLE IF EXISTS "fakultas" CASCADE;
DROP TABLE IF EXISTS "roles" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;
