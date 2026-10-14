-- =====================================================================
-- Migration: 000002_patch_peserta_tempat_nullable
-- Alasan: PRD bagian 9 nomor 1: penempatan AM dan KKN baru ditentukan
-- setelah mendaftar, sehingga id_tempat_program harus nullable saat pendaftaran.
-- =====================================================================

ALTER TABLE "peserta_program" ALTER COLUMN "id_tempat_program" DROP NOT NULL;
