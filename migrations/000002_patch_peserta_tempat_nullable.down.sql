-- =====================================================================
-- Migration Rollback: 000002_patch_peserta_tempat_nullable
-- Mengembalikan kolom id_tempat_program menjadi NOT NULL
-- =====================================================================

ALTER TABLE "peserta_program" ALTER COLUMN "id_tempat_program" SET NOT NULL;
