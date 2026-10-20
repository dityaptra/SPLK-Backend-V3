-- Migrasi 000003: Tabel verifikasi kelulusan penguji untuk magang
CREATE TABLE IF NOT EXISTS "kelulusan_penguji" (
    "id"                 SERIAL      NOT NULL,
    "id_peserta_program" INTEGER     NOT NULL UNIQUE REFERENCES "peserta_program"("id") ON DELETE CASCADE,
    "id_dosen_penguji"   INTEGER     NOT NULL REFERENCES "dosen"("id"),
    "status_kelulusan"   VARCHAR(20) NOT NULL CHECK ("status_kelulusan" IN ('Lulus', 'Tidak Lulus')),
    "catatan"            TEXT,
    "created_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "idx_kelulusan_penguji_peserta" ON "kelulusan_penguji" ("id_peserta_program");
