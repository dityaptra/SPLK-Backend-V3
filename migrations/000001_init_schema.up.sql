-- =====================================================================
-- Migration: Sistem Pembelajaran Luar Kampus (Magang, AM, KKN)
-- Database  : PostgreSQL
-- Versi     : 2.0 — Revisi lengkap berdasarkan analisis BPMN
-- Perubahan :
--   - Konsolidasi pembimbing lapangan (mentor/guru_pamong/kepala_desa)
--     ke satu tabel "pembimbing_lapangan" dengan kolom "jenis"
--   - pegawai berperan sebagai kejur/PIC
--   - Multi-role via user_roles (sudah ada, diperlengkap seed)
--   - kepala_sekolah dihapus (bukan aktor dalam BPMN)
--   - peserta_program: id_mentor & id_guru digabung ke id_pembimbing_lapangan
--   - periode ditambah kolom jenis_program
--   - Timestamp konsisten menggunakan DEFAULT CURRENT_TIMESTAMP
--   - KRS validation didelegasikan ke service (return true), tidak ada tabel KRS
--   - Tabel "user" diganti nama menjadi "users" (user adalah kata cadangan PostgreSQL)
-- =====================================================================

-- =====================================================================
-- 0. EXTENSIONS
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- untuk gen_random_uuid() jika dibutuhkan

-- =====================================================================
-- 1. TABLE CREATION
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1.1 Auth & User
-- ---------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS "users" (
    "id"            SERIAL      NOT NULL,
    "username"      VARCHAR(50) NOT NULL UNIQUE,
    "email"         VARCHAR(100) NOT NULL UNIQUE,
    "password_hash" VARCHAR(255) NOT NULL,
    "is_active"     BOOLEAN     NOT NULL DEFAULT true,
    "created_at"    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

-- Daftar role yang tersedia di sistem
CREATE TABLE IF NOT EXISTS "roles" (
    "id"   SERIAL      NOT NULL,
    "nama" VARCHAR(50) NOT NULL UNIQUE,
    -- Deskripsi singkat peran
    "deskripsi" VARCHAR(255),
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "roles"."nama" IS 'mahasiswa, admin, dosen, pegawai, pembimbing_lapangan';

-- Relasi many-to-many: satu user bisa punya banyak role
CREATE TABLE IF NOT EXISTS "user_roles" (
    "id"      SERIAL  NOT NULL,
    "id_user" INTEGER NOT NULL,
    "id_role" INTEGER NOT NULL,
    PRIMARY KEY ("id"),
    UNIQUE ("id_user", "id_role")
);

-- Refresh token untuk JWT
CREATE TABLE IF NOT EXISTS "tokens" (
    "id"         SERIAL      NOT NULL,
    "id_user"    INTEGER     NOT NULL,
    "token_hash" VARCHAR(255) NOT NULL UNIQUE,
    "expired_at" TIMESTAMP   NOT NULL,
    "created_at" TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

-- Audit log aktivitas login/logout/aksi penting
CREATE TABLE IF NOT EXISTS "log_aktivitas" (
    "id"           SERIAL       NOT NULL,
    "id_user"      INTEGER      NOT NULL,
    "nama_user"    VARCHAR(100) NOT NULL,
    "ip_user"      VARCHAR(45),
    "agent_browser" VARCHAR(255),
    "action"       VARCHAR(100) NOT NULL,
    -- true = berhasil, false = gagal
    "status"       BOOLEAN      NOT NULL,
    "created_at"   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "log_aktivitas"."status" IS 'true = berhasil, false = gagal';

-- ---------------------------------------------------------------------
-- 1.2 Struktur Akademik
-- ---------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS "fakultas" (
    "id"   SERIAL      NOT NULL,
    "nama" VARCHAR(100) NOT NULL,
    "kode" VARCHAR(10),
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "jurusan" (
    "id"          SERIAL      NOT NULL,
    "id_fakultas" INTEGER     NOT NULL,
    "nama"        VARCHAR(100) NOT NULL,
    "kode_jurusan" VARCHAR(10),
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "program_studi" (
    "id"         SERIAL      NOT NULL,
    "id_jurusan" INTEGER     NOT NULL,
    "nama"       VARCHAR(100) NOT NULL,
    "kode"       VARCHAR(10),
    PRIMARY KEY ("id")
);

-- ---------------------------------------------------------------------
-- 1.3 Profil per Aktor (turunan dari "users")
-- ---------------------------------------------------------------------

-- Mahasiswa peserta MBKM
CREATE TABLE IF NOT EXISTS "mahasiswa" (
    "id"               SERIAL      NOT NULL,
    "id_user"          INTEGER     NOT NULL UNIQUE,
    "nim"              VARCHAR(20) NOT NULL UNIQUE,
    "nama"             VARCHAR(100) NOT NULL,
    "id_prodi"         INTEGER     NOT NULL,
    "angkatan"         INTEGER     NOT NULL,
    "semester_sekarang" INTEGER,
    "no_hp"            VARCHAR(20),
    PRIMARY KEY ("id")
);

-- Dosen: bisa menjadi dosen_pembimbing atau dosen_penguji
CREATE TABLE IF NOT EXISTS "dosen" (
    "id"            SERIAL      NOT NULL,
    "id_user"       INTEGER     NOT NULL UNIQUE,
    "nip"           VARCHAR(30) NOT NULL UNIQUE,
    "nama"          VARCHAR(100) NOT NULL,
    "id_prodi"      INTEGER     NOT NULL,
    "gelar_depan"   VARCHAR(50),
    "gelar_belakang" VARCHAR(50),
    PRIMARY KEY ("id")
);

-- Pegawai: berperan sebagai Kejur/PIC yang mengelola program
CREATE TABLE IF NOT EXISTS "pegawai" (
    "id"             SERIAL      NOT NULL,
    "id_user"        INTEGER     NOT NULL UNIQUE,
    "nip"            VARCHAR(30) NOT NULL UNIQUE,
    "nama"           VARCHAR(100) NOT NULL,
    "gelar_depan"    VARCHAR(50),
    "gelar_belakang" VARCHAR(50),
    -- Misal: "Koordinator MBKM", "Staff Akademik", "Kejur Teknik Informatika"
    "jabatan"        VARCHAR(100) NOT NULL,
    PRIMARY KEY ("id")
);

-- Admin sistem (bisa berlapis dengan pegawai jika diperlukan)
CREATE TABLE IF NOT EXISTS "admin" (
    "id"      SERIAL      NOT NULL,
    "id_user" INTEGER     NOT NULL UNIQUE,
    "nip"     VARCHAR(30),
    "nama"    VARCHAR(100) NOT NULL,
    PRIMARY KEY ("id")
);

-- ---------------------------------------------------------------------
-- 1.4 Tempat Program & Pembimbing Lapangan
-- ---------------------------------------------------------------------

-- Tempat pelaksanaan program: perusahaan (magang), sekolah (AM), desa (KKN)
CREATE TABLE IF NOT EXISTS "tempat_program" (
    "id"           SERIAL      NOT NULL,
    -- magang | asistensi_mengajar | kkn
    "jenis_program" VARCHAR(20) NOT NULL,
    "nama"         VARCHAR(150) NOT NULL,
    "alamat"       TEXT        NOT NULL,
    "kota"         VARCHAR(100),
    "kontak"       VARCHAR(50),
    "is_aktif"     BOOLEAN     NOT NULL DEFAULT true,
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "tempat_program"."jenis_program" IS 'magang | asistensi_mengajar | kkn';

-- Pembimbing lapangan: satu tabel untuk semua jenis
--   * Magang       → jenis = 'mentor'       (dari perusahaan/industri)
--   * AM           → jenis = 'guru_pamong'  (guru di sekolah)
--   * KKN          → jenis = 'kepala_desa'  (kepala desa/lurah)
-- id_user bersifat NULLABLE karena pembimbing lapangan
-- tidak selalu punya akun di sistem
CREATE TABLE IF NOT EXISTS "pembimbing_lapangan" (
    "id"                SERIAL      NOT NULL,
    -- mentor | guru_pamong | kepala_desa
    "jenis"             VARCHAR(20) NOT NULL,
    "id_tempat_program" INTEGER     NOT NULL,
    -- Nullable: pembimbing lapangan belum tentu punya akun sistem
    "id_user"           INTEGER     UNIQUE,
    "nama"              VARCHAR(100) NOT NULL,
    "jabatan"           VARCHAR(100),
    "kontak"            VARCHAR(50),
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "pembimbing_lapangan"."jenis" IS 'mentor | guru_pamong | kepala_desa';
COMMENT ON COLUMN "pembimbing_lapangan"."id_user" IS 'Nullable — pembimbing lapangan belum tentu punya akun sistem';

-- ---------------------------------------------------------------------
-- 1.5 Periode Program
-- ---------------------------------------------------------------------

-- Satu periode = satu jenis program pada satu semester tertentu
-- (Admin membuat periode, kemudian mahasiswa mendaftar)
CREATE TABLE IF NOT EXISTS "periode" (
    "id"              SERIAL      NOT NULL,
    "nama"            VARCHAR(100) NOT NULL,
    -- magang | asistensi_mengajar | kkn
    "jenis_program"   VARCHAR(20) NOT NULL,
    -- Contoh: "2026/2027 Gasal"
    "tahun_semester"  VARCHAR(20) NOT NULL,
    "tanggal_mulai"   DATE        NOT NULL,
    "tanggal_selesai" DATE        NOT NULL,
    -- Tanggal buka & tutup pendaftaran
    "tanggal_buka_daftar"  DATE,
    "tanggal_tutup_daftar" DATE,
    -- draft | aktif | selesai
    "status"          VARCHAR(20) NOT NULL DEFAULT 'draft',
    -- Pegawai (Kejur/PIC) yang membuat periode
    "dibuat_oleh"     INTEGER,
    "created_at"      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "periode"."jenis_program" IS 'magang | asistensi_mengajar | kkn';
COMMENT ON COLUMN "periode"."status"        IS 'draft | aktif | selesai';

-- ---------------------------------------------------------------------
-- 1.6 Kelompok KKN
-- ---------------------------------------------------------------------

-- Kelompok KKN dibuat per periode, terikat ke satu tempat (desa)
CREATE TABLE IF NOT EXISTS "kelompok_kkn" (
    "id"                  SERIAL      NOT NULL,
    "id_periode"          INTEGER     NOT NULL,
    "id_tempat_program"   INTEGER     NOT NULL,
    -- Dosen pembimbing kelompok (nullable: belum tentu langsung diassign)
    "id_dosen_pembimbing" INTEGER,
    -- Kepala desa / pembimbing lapangan KKN
    "id_pembimbing_lapangan" INTEGER,
    "nama_kelompok"       VARCHAR(50) NOT NULL,
    "proker_url"          TEXT,        -- link dokumen program kerja kelompok
    "created_at"          TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"          TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

-- ---------------------------------------------------------------------
-- 1.7 Peserta Program (Inti Pendaftaran)
-- ---------------------------------------------------------------------

-- Satu baris = satu mahasiswa yang mendaftar pada satu periode
-- Validasi KRS di-skip pada level DB, ditangani service layer (return true)
CREATE TABLE IF NOT EXISTS "peserta_program" (
    "id"                    SERIAL      NOT NULL,
    "id_mahasiswa"          INTEGER     NOT NULL,
    "id_periode"            INTEGER     NOT NULL,
    -- magang | asistensi_mengajar | kkn
    "jenis_program"         VARCHAR(20) NOT NULL,
    "id_tempat_program"     INTEGER     NOT NULL,
    -- Dosen pembimbing individual (magang & AM); untuk KKN ikut kelompok
    "id_dosen_pembimbing"   INTEGER,
    -- Dosen penguji (saat presentasi/seminar)
    "id_dosen_penguji"      INTEGER,
    -- Pembimbing lapangan (mentor / guru_pamong / kepala_desa)
    "id_pembimbing_lapangan" INTEGER,
    -- Khusus KKN: link ke kelompok
    "id_kelompok_kkn"       INTEGER,
    -- Apakah mahasiswa ini ketua kelompok KKN
    "is_ketua"              BOOLEAN     NOT NULL DEFAULT false,
    -- diajukan | ditolak | berjalan | selesai
    "status"                VARCHAR(20) NOT NULL DEFAULT 'diajukan',
    "alasan_tolak"          TEXT,
    "tanggal_daftar"        TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tanggal_mulai_aktif"   DATE,
    "tanggal_selesai_aktif" DATE,
    -- Nilai akhir setelah semua komponen dinilai
    "nilai_akhir"           DECIMAL(5,2),
    "grade_huruf"           VARCHAR(5),
    "sks_diakui"            INTEGER,
    "created_at"            TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"            TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "peserta_program"."jenis_program" IS 'magang | asistensi_mengajar | kkn';
COMMENT ON COLUMN "peserta_program"."status"        IS 'diajukan | ditolak | berjalan | selesai';

-- ---------------------------------------------------------------------
-- 1.8 Jurnal Harian & Logbook Bimbingan
-- ---------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS "jurnal_harian" (
    "id"                 SERIAL      NOT NULL,
    "id_peserta_program" INTEGER     NOT NULL,
    "tanggal"            DATE        NOT NULL,
    "judul"              VARCHAR(255),
    "isi"                TEXT        NOT NULL,
    "kendala"            TEXT,
    "solusi"             TEXT,
    -- draft | disubmit | disetujui | ditolak
    "status"             VARCHAR(20) NOT NULL DEFAULT 'draft',
    "catatan_reviewer"   TEXT,
    "reviewed_by"        INTEGER,    -- id_user yang mereview
    "reviewed_at"        TIMESTAMP,
    "created_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    -- Satu jurnal per peserta per tanggal
    UNIQUE ("id_peserta_program", "tanggal")
);

COMMENT ON COLUMN "jurnal_harian"."status" IS 'draft | disubmit | disetujui | ditolak';

CREATE TABLE IF NOT EXISTS "logbook_bimbingan" (
    "id"                 SERIAL      NOT NULL,
    "id_peserta_program" INTEGER     NOT NULL,
    "id_dosen"           INTEGER     NOT NULL,
    "tanggal"            DATE        NOT NULL,
    "topik_bimbingan"    TEXT        NOT NULL,
    "catatan"            TEXT,
    "pertemuan_berikutnya" DATE,
    "created_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

-- ---------------------------------------------------------------------
-- 1.9 Penilaian
-- ---------------------------------------------------------------------

-- Template instrumen penilaian per jenis program & role penilai
CREATE TABLE IF NOT EXISTS "instrumen_penilaian" (
    "id"           SERIAL      NOT NULL,
    -- magang | asistensi_mengajar | kkn
    "jenis_program" VARCHAR(20) NOT NULL,
    -- Misal: "Disiplin", "Kompetensi Teknis", "Laporan"
    "nama_aspek"   VARCHAR(100) NOT NULL,
    -- Bobot dalam persen; total per jenis_program & role_penilai = 100
    "bobot"        DECIMAL(5,2) NOT NULL,
    -- dosen_pembimbing | dosen_penguji | pembimbing_lapangan
    "role_penilai" VARCHAR(30)  NOT NULL,
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "instrumen_penilaian"."nama_aspek"   IS 'Contoh: "Disiplin", "Kompetensi Teknis"';
COMMENT ON COLUMN "instrumen_penilaian"."role_penilai" IS 'dosen_pembimbing | dosen_penguji | pembimbing_lapangan';

-- Nilai aktual per peserta per instrumen
CREATE TABLE IF NOT EXISTS "penilaian" (
    "id"                 SERIAL      NOT NULL,
    "id_peserta_program" INTEGER     NOT NULL,
    "id_instrumen"       INTEGER     NOT NULL,
    -- id_user penilai (dosen atau pembimbing_lapangan yang punya akun)
    "id_penilai"         INTEGER     NOT NULL,
    "nilai"              DECIMAL(5,2) NOT NULL CHECK ("nilai" >= 0 AND "nilai" <= 100),
    "catatan"            TEXT,
    "tanggal_nilai"      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    -- Satu penilai hanya boleh isi satu nilai per instrumen per peserta
    UNIQUE ("id_peserta_program", "id_instrumen", "id_penilai")
);

-- ---------------------------------------------------------------------
-- 1.10 Dokumen
-- ---------------------------------------------------------------------

-- Dokumen yang diunggah: MoU, Proposal, Proker, Laporan Akhir, dll.
-- Bisa milik peserta individual ATAU kelompok KKN
CREATE TABLE IF NOT EXISTS "dokumen" (
    "id"                 SERIAL      NOT NULL,
    -- Nullable: dokumen bisa milik kelompok KKN, bukan peserta individual
    "id_peserta_program" INTEGER,
    -- mou | proposal | laporan_akhir | proker | lainnya
    "jenis_dokumen"      VARCHAR(30) NOT NULL,
    "nama_file"          VARCHAR(255) NOT NULL,
    "file_url"           TEXT        NOT NULL,
    "ukuran_file"        INTEGER,    -- dalam bytes
    "mime_type"          VARCHAR(100),
    "diunggah_oleh"      INTEGER     NOT NULL,
    "uploaded_at"        TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Nullable: dokumen bisa milik kelompok KKN
    "id_kelompok_kkn"    INTEGER,
    "is_verified"        BOOLEAN     NOT NULL DEFAULT false,
    "verified_by"        INTEGER,
    "verified_at"        TIMESTAMP,
    PRIMARY KEY ("id"),
    -- Minimal salah satu harus diisi: peserta atau kelompok
    CONSTRAINT "chk_dokumen_pemilik"
        CHECK ("id_peserta_program" IS NOT NULL OR "id_kelompok_kkn" IS NOT NULL)
);

COMMENT ON COLUMN "dokumen"."jenis_dokumen" IS 'mou | proposal | laporan_akhir | proker | lainnya';

-- ---------------------------------------------------------------------
-- 1.11 Surat Perjalanan Dinas
-- ---------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS "surat_perjalanan_dinas" (
    "id"                 SERIAL      NOT NULL,
    "id_dosen"           INTEGER     NOT NULL,
    "id_peserta_program" INTEGER     NOT NULL,
    "tujuan"             VARCHAR(150) NOT NULL,
    "tanggal_berangkat"  DATE        NOT NULL,
    "tanggal_kembali"    DATE        NOT NULL,
    "file_url"           TEXT,
    -- Dibuat oleh pegawai (Kejur/PIC)
    "dibuat_oleh"        INTEGER     NOT NULL,
    "created_at"         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

-- ---------------------------------------------------------------------
-- 1.12 Pengumuman
-- ---------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS "pengumuman" (
    "id"             SERIAL      NOT NULL,
    "judul"          VARCHAR(150) NOT NULL,
    "isi"            TEXT        NOT NULL,
    "dibuat_oleh"    INTEGER,    -- id_user
    "tanggal_terbit" TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Jika NULL berarti untuk semua, jika diisi targetkan role tertentu
    "target_role"    VARCHAR(50),
    PRIMARY KEY ("id")
);

-- =====================================================================
-- 2. FOREIGN KEYS
-- =====================================================================

-- Hierarki akademik
ALTER TABLE "jurusan"
    ADD FOREIGN KEY ("id_fakultas") REFERENCES "fakultas"("id");
ALTER TABLE "program_studi"
    ADD FOREIGN KEY ("id_jurusan") REFERENCES "jurusan"("id");
ALTER TABLE "mahasiswa"
    ADD FOREIGN KEY ("id_prodi") REFERENCES "program_studi"("id");
ALTER TABLE "dosen"
    ADD FOREIGN KEY ("id_prodi") REFERENCES "program_studi"("id");

-- Turunan dari "users"
ALTER TABLE "mahasiswa"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id");
ALTER TABLE "dosen"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id");
ALTER TABLE "pegawai"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id");
ALTER TABLE "admin"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id");

-- Pembimbing lapangan -> user (nullable)
ALTER TABLE "pembimbing_lapangan"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id");
ALTER TABLE "pembimbing_lapangan"
    ADD FOREIGN KEY ("id_tempat_program") REFERENCES "tempat_program"("id");

-- Multi-role
ALTER TABLE "user_roles"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id") ON DELETE CASCADE;
ALTER TABLE "user_roles"
    ADD FOREIGN KEY ("id_role") REFERENCES "roles"("id");

-- Token & log aktivitas
ALTER TABLE "tokens"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id") ON DELETE CASCADE;
ALTER TABLE "log_aktivitas"
    ADD FOREIGN KEY ("id_user") REFERENCES "users"("id");

-- Periode
ALTER TABLE "periode"
    ADD FOREIGN KEY ("dibuat_oleh") REFERENCES "pegawai"("id");

-- Kelompok KKN
ALTER TABLE "kelompok_kkn"
    ADD FOREIGN KEY ("id_periode") REFERENCES "periode"("id");
ALTER TABLE "kelompok_kkn"
    ADD FOREIGN KEY ("id_tempat_program") REFERENCES "tempat_program"("id");
ALTER TABLE "kelompok_kkn"
    ADD FOREIGN KEY ("id_dosen_pembimbing") REFERENCES "dosen"("id");
ALTER TABLE "kelompok_kkn"
    ADD FOREIGN KEY ("id_pembimbing_lapangan") REFERENCES "pembimbing_lapangan"("id");

-- Peserta program (inti)
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_mahasiswa") REFERENCES "mahasiswa"("id");
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_periode") REFERENCES "periode"("id");
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_tempat_program") REFERENCES "tempat_program"("id");
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_dosen_pembimbing") REFERENCES "dosen"("id");
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_dosen_penguji") REFERENCES "dosen"("id");
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_pembimbing_lapangan") REFERENCES "pembimbing_lapangan"("id");
ALTER TABLE "peserta_program"
    ADD FOREIGN KEY ("id_kelompok_kkn") REFERENCES "kelompok_kkn"("id");

-- Jurnal harian
ALTER TABLE "jurnal_harian"
    ADD FOREIGN KEY ("id_peserta_program") REFERENCES "peserta_program"("id");
ALTER TABLE "jurnal_harian"
    ADD FOREIGN KEY ("reviewed_by") REFERENCES "users"("id");

-- Logbook bimbingan
ALTER TABLE "logbook_bimbingan"
    ADD FOREIGN KEY ("id_peserta_program") REFERENCES "peserta_program"("id");
ALTER TABLE "logbook_bimbingan"
    ADD FOREIGN KEY ("id_dosen") REFERENCES "dosen"("id");

-- Penilaian
ALTER TABLE "penilaian"
    ADD FOREIGN KEY ("id_peserta_program") REFERENCES "peserta_program"("id");
ALTER TABLE "penilaian"
    ADD FOREIGN KEY ("id_instrumen") REFERENCES "instrumen_penilaian"("id");
ALTER TABLE "penilaian"
    ADD FOREIGN KEY ("id_penilai") REFERENCES "users"("id");

-- Dokumen
ALTER TABLE "dokumen"
    ADD FOREIGN KEY ("id_peserta_program") REFERENCES "peserta_program"("id");
ALTER TABLE "dokumen"
    ADD FOREIGN KEY ("id_kelompok_kkn") REFERENCES "kelompok_kkn"("id");
ALTER TABLE "dokumen"
    ADD FOREIGN KEY ("diunggah_oleh") REFERENCES "users"("id");
ALTER TABLE "dokumen"
    ADD FOREIGN KEY ("verified_by") REFERENCES "users"("id");

-- Surat perjalanan dinas
ALTER TABLE "surat_perjalanan_dinas"
    ADD FOREIGN KEY ("id_dosen") REFERENCES "dosen"("id");
ALTER TABLE "surat_perjalanan_dinas"
    ADD FOREIGN KEY ("id_peserta_program") REFERENCES "peserta_program"("id");
ALTER TABLE "surat_perjalanan_dinas"
    ADD FOREIGN KEY ("dibuat_oleh") REFERENCES "pegawai"("id");

-- Pengumuman
ALTER TABLE "pengumuman"
    ADD FOREIGN KEY ("dibuat_oleh") REFERENCES "users"("id");

-- =====================================================================
-- 3. CONSTRAINT TAMBAHAN
-- =====================================================================

-- Satu mahasiswa tidak boleh mendaftar dua kali pada jenis program & periode yang sama
ALTER TABLE "peserta_program"
    ADD CONSTRAINT "uq_peserta_program_mahasiswa_periode_jenis"
    UNIQUE ("id_mahasiswa", "id_periode", "jenis_program");

-- Satu kelompok KKN hanya boleh punya satu ketua
CREATE UNIQUE INDEX "uq_satu_ketua_per_kelompok"
    ON "peserta_program" ("id_kelompok_kkn")
    WHERE "is_ketua" = true;

-- Periode tidak boleh memiliki tanggal tutup sebelum tanggal buka
ALTER TABLE "periode"
    ADD CONSTRAINT "chk_periode_tanggal"
    CHECK ("tanggal_selesai" >= "tanggal_mulai");

-- Bobot penilaian antara 0–100
ALTER TABLE "instrumen_penilaian"
    ADD CONSTRAINT "chk_bobot_penilaian"
    CHECK ("bobot" > 0 AND "bobot" <= 100);

-- =====================================================================
-- 4. INDEKS PERFORMA
-- =====================================================================

CREATE INDEX IF NOT EXISTS "idx_peserta_mahasiswa"   ON "peserta_program" ("id_mahasiswa");
CREATE INDEX IF NOT EXISTS "idx_peserta_periode"     ON "peserta_program" ("id_periode");
CREATE INDEX IF NOT EXISTS "idx_peserta_status"      ON "peserta_program" ("status");
CREATE INDEX IF NOT EXISTS "idx_jurnal_peserta"      ON "jurnal_harian"   ("id_peserta_program");
CREATE INDEX IF NOT EXISTS "idx_jurnal_tanggal"      ON "jurnal_harian"   ("tanggal");
CREATE INDEX IF NOT EXISTS "idx_logbook_peserta"     ON "logbook_bimbingan" ("id_peserta_program");
CREATE INDEX IF NOT EXISTS "idx_logbook_dosen"       ON "logbook_bimbingan" ("id_dosen");
CREATE INDEX IF NOT EXISTS "idx_penilaian_peserta"   ON "penilaian"       ("id_peserta_program");
CREATE INDEX IF NOT EXISTS "idx_dokumen_peserta"     ON "dokumen"         ("id_peserta_program");
CREATE INDEX IF NOT EXISTS "idx_dokumen_kelompok"    ON "dokumen"         ("id_kelompok_kkn");
CREATE INDEX IF NOT EXISTS "idx_tokens_user"         ON "tokens"          ("id_user");
CREATE INDEX IF NOT EXISTS "idx_log_user"            ON "log_aktivitas"   ("id_user");
CREATE INDEX IF NOT EXISTS "idx_pembimbing_tempat"   ON "pembimbing_lapangan" ("id_tempat_program");
CREATE INDEX IF NOT EXISTS "idx_pembimbing_jenis"    ON "pembimbing_lapangan" ("jenis");
CREATE INDEX IF NOT EXISTS "idx_periode_jenis"       ON "periode"         ("jenis_program", "status");

-- =====================================================================
-- 5. SEED DATA
-- =====================================================================

-- Roles sistem (multi-role: satu user bisa punya lebih dari satu role)
-- Catatan: pembimbing_lapangan adalah aktor eksternal (mentor/guru_pamong/kepala_desa)
INSERT INTO "roles" ("nama", "deskripsi") VALUES
    ('mahasiswa',            'Mahasiswa peserta program MBKM'),
    ('admin',                'Administrator sistem'),
    ('dosen',                'Dosen — bisa menjadi dosen_pembimbing atau dosen_penguji'),
    ('pegawai',              'Pegawai / Kejur / PIC pengelola program'),
    ('pembimbing_lapangan',  'Pembimbing di lapangan: mentor industri, guru pamong, atau kepala desa')
ON CONFLICT ("nama") DO UPDATE
    SET "deskripsi" = EXCLUDED."deskripsi";
