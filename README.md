# SPLK Backend (Sistem Pembelajaran Luar Kampus)

Backend RESTful API untuk Sistem Pembelajaran Luar Kampus (Magang MBKM, Asistensi Mengajar, dan KKN) Universitas Pendidikan Ganesha (Undiksha).

Dibangun menggunakan Go, Gin Web Framework, PostgreSQL (pgx v5), arsitektur modular berlapis (*Clean Architecture / Ports & Adapters*), dokumentasi Swagger/OpenAPI interaktif, serta pengujian end-to-end terintegrasi.

---

## Daftar Isi
- [Fitur Utama](#fitur-utama)
- [Teknologi & Prasyarat](#teknologi--prasyarat)
- [Instalasi & Konfigurasi](#instalasi--konfigurasi)
- [Migrasi & Seeding Database](#migrasi--seeding-database)
- [Menjalankan Aplikasi](#menjalankan-aplikasi)
- [Dokumentasi API (Swagger)](#dokumentasi-api-swagger)
- [Pengujian (Testing & E2E)](#pengujian-testing--e2e)
- [Docker & Containerisasi](#docker--containerisasi)
- [Struktur Direktori](#struktur-direktori)
- [Aktor & Kontrol Akses](#aktor--kontrol-akses)
- [Asumsi Arsitektur (A1–A13)](#asumsi-arsitektur-a1a13)
- [Pertanyaan Terbuka untuk Mentor & Tim](#pertanyaan-terbuka-untuk-mentor--tim)

---

## Fitur Utama

1. **Autentikasi & Otorisasi:**
   - JWT Access Token (15m - 24h) & Refresh Token dengan sliding-window rate limiter dan hash SHA-256.
   - Kontrol akses berbasis Role (RBAC) dan Relasi Kepemilikan Data (ABAC: Mahasiswa pemilik, Dosen Pembimbing, Dosen Penguji, Mentor/Guru Pamong/Kades Lapangan).
2. **Manajemen Periode & Program:**
   - Multi-program: Magang, Asistensi Mengajar (AM), dan KKN.
   - Aturan pembukaan pendaftaran tanggal kalender (zona Asia/Makassar) dan pencegahan periode aktif ganda.
3. **Alur Kesiapan Otomatis (State Machine):**
   - Transisi pendaftaran dari `diajukan` -> `berjalan` secara otomatis tanpa approval manual ketika prasyarat terpenuhi:
     - **Magang:** Penempatan mitra + mentor + dosen pembimbing + MoU + proposal.
     - **Asistensi Mengajar:** Sekolah penempatan + guru pamong + dosen pembimbing + dosen penguji + dokumen proker.
     - **KKN:** Kelompok KKN (minimal 1 mhs & 1 ketua) + desa + kades + dosen pembimbing + proker kelompok.
4. **Pelaksanaan Kegiatan:**
   - Jurnal Harian Mahasiswa (validasi rentang periode, status pendaftaran berjalan, pencegahan tanggal masa depan).
   - Logbook Bimbingan Dosen Pembimbing (catatan progres, kendala, dan rekomendasi).
   - Pengunggahan & Unduh Dokumen (dukungan PDF, DOCX, gambar dengan proteksi download otorisasi).
5. **Penilaian Terintegrasi 3 Penilai:**
   - Form penilaian dinamis per jenis program & peranan penilai.
   - Penilaian independen dan paralel oleh Dosen Pembimbing, Dosen Penguji, dan Pembimbing Lapangan.
   - Kalkulasi otomatis nilai akhir (bobot tertimbang) dan grade huruf (A, B, C, D, E) saat instrumen lengkap, dilengkapi proteksi konkurensi database (`SELECT ... FOR UPDATE`).
6. **Hardening & Keamanan:**
   - Security Headers (HSTS, CSP, X-Frame-Options, X-XSS-Protection, MIME Sniffing protection).
   - Body Size Limit (10MB default) untuk menangkal Denial of Service upload payload besar.
   - In-memory Sliding Window Login Rate Limiter.

---

## Teknologi & Prasyarat

- **Go:** Versi 1.27+ (atau 1.24+)
- **PostgreSQL:** Versi 15+
- **Docker & Docker Compose** (opsional untuk container runtime)
- **Swag CLI:** `go install github.com/swaggo/swag/cmd/swag@latest`
- **Migrate CLI:** `golang-migrate`

---

## Instalasi & Konfigurasi

### 1. Clone Repositori
```bash
git clone https://github.com/splk-undiksha/splk-backend.git
cd splk-backend
```

### 2. Setup Database PostgreSQL
Jalankan di PostgreSQL client / pgAdmin / psql:
```sql
CREATE USER splk WITH PASSWORD 'ganti_password_ini';
CREATE DATABASE splk_db OWNER splk;
CREATE DATABASE splk_test OWNER splk;
```

### 3. Konfigurasi Environment (`.env`)
Salin file template `.env.example` menjadi `.env`:
```bash
cp .env.example .env
```

Contoh konfigurasi `.env`:
```env
APP_ENV=development
APP_PORT=8080
APP_TIMEZONE=Asia/Makassar

# PostgreSQL Connection String
DATABASE_URL=postgres://splk:ganti_password_ini@localhost:5432/splk_db?sslmode=disable
# Atau format parameter terpisah:
DB_HOST=localhost
DB_PORT=5432
DB_USER=splk
DB_PASSWORD=ganti_password_ini
DB_NAME=splk_db
DB_SSLMODE=disable

# Keamanan JWT (Minimal 32 karakter rahasia)
JWT_SECRET=super-secret-key-splk-mbkm-dev-minimum-32-chars-long!
JWT_ACCESS_HOURS=24
JWT_REFRESH_DAYS=7

# Upload Dokumen
UPLOAD_PATH=./uploads
MAX_UPLOAD_MB=10
```

---

## Migrasi & Seeding Database

### Menjalankan Migrasi
Migrasi terorganisir di direktori `migrations/`:
```bash
# Menggunakan Makefile
make migrate-up

# Rollback 1 langkah migrasi
make migrate-down
```
Atau menggunakan `golang-migrate` CLI langsung:
```bash
migrate -path migrations -database "postgres://splk:ganti_password_ini@localhost:5432/splk_db?sslmode=disable" up
```

### Menjalankan Database Seeder
Seeder akan membuat akun admin awal jika belum ada:
- **Username:** `admin`
- **Password:** `AdminPass123!`
- **Role:** `admin`

```bash
# Menjalankan seeder
make seed
# Atau langsung:
go run cmd/seed/main.go
```

---

## Menjalankan Aplikasi

### Mode Pengembangan (Lokal)
```bash
# Instal dependensi
go mod tidy

# Menjalankan server API
make run
# Atau:
go run cmd/api/main.go
```
Server akan aktif di `http://localhost:8080`.

Cek endpoint kesehatan:
```bash
curl http://localhost:8080/health
# Response: {"status":"ok","database":"connected","time":"2026-10-16T..."}
```

---

## Dokumentasi API (Swagger)

Swagger UI tersedia secara interaktif saat `APP_ENV=development`.

1. **Akses Swagger UI di Browser:**
   ```
   http://localhost:8080/swagger/index.html
   ```
2. **Autentikasi di Swagger:**
   - Lakukan login via endpoint `POST /auth/login`.
   - Salin token akses JWT dari respon (`data.token`).
   - Klik tombol **Authorize** di pojok kanan atas Swagger UI, lalu masukkan:
     ```
     Bearer <TOKEN_JWT_ANDA>
     ```

3. **Regenerasi Dokumentasi Swagger:**
   Jika Anda mengubah handler atau anotasi endpoint:
   ```bash
   make swag
   # Atau via binary swag:
   swag init -g cmd/api/main.go -o docs
   ```

---

## Pengujian (Testing & E2E)

Sistem memiliki pengujian komprehensif mulai dari unit test modul, isolasi aturan bisnis, hingga integrasi End-to-End (E2E).

### 1. Menjalankan Seluruh Rangkaian Test
Pastikan database `splk_test` telah dimigrasi, lalu jalankan:

**PowerShell (Windows):**
```powershell
$env:TEST_DATABASE_URL="postgres://splk:ganti_password_ini@localhost:5432/splk_test?sslmode=disable"
go test ./... -count=1 -p 1
```

**Bash / Linux / macOS:**
```bash
TEST_DATABASE_URL="postgres://splk:ganti_password_ini@localhost:5432/splk_test?sslmode=disable" go test ./... -count=1 -p 1
```

### 2. Menjalankan Pengujian End-to-End (E2E) Saja
Pengujian E2E berada di paket `test/e2e/e2e_test.go` yang merefleksikan PRD Bagian 11:
- `TestE2E_Magang`: Periode -> Pendaftaran -> Pilih Tempat & Mentor -> Dosen Pembimbing -> Upload MoU & Proposal -> Transisi Berjalan -> Jurnal & Logbook -> Selesaikan Periode -> 3 Penilaian -> Nilai Akhir Terhitung Otomatis.
- `TestE2E_AsistensiMengajar`: Periode -> Pendaftaran -> Penempatan Sekolah & Guru Pamong -> Dosen Pembimbing & Penguji -> Upload Proker -> Transisi Berjalan -> Jurnal & Logbook -> Selesaikan Periode -> 3 Penilaian -> Nilai Akhir Terhitung Otomatis.
- `TestE2E_KKN`: Periode -> Pendaftaran 2 Mhs -> Pembentukan Kelompok KKN & Ketua -> Tambah Anggota -> Dosen Pembimbing & Kades -> Proker Kelompok -> Kesiapan Kedua Mhs -> Jurnal & Logbook -> Selesaikan Periode -> Penilaian -> Nilai Akhir Terhitung Otomatis.

Jalankan test E2E:
```powershell
$env:TEST_DATABASE_URL="postgres://splk:ganti_password_ini@localhost:5432/splk_test?sslmode=disable"
go test -v ./test/e2e -run TestE2E_ -count=1
```

---

## Docker & Containerisasi

Backend SPLK dapat dijalankan dalam kontainer Docker multi-stage build yang ringan berbasis Alpine Linux.

> **Catatan Host Database:**
> Secara arsitektur, PostgreSQL berjalan di mesin host. Container backend mengakses host menggunakan `host.docker.internal`.

### Build & Jalankan Kontainer:
```bash
# Build dan jalankan di background
docker compose up -d --build

# Periksa status kontainer
docker compose ps

# Melihat log aplikasi
docker compose logs -f api

# Menghentikan kontainer
docker compose down
```

---

## Struktur Direktori

```
SPLK-backend-v3/
├── cmd/
│   ├── api/main.go            # Entrypoint server REST API Gin
│   └── seed/main.go           # Entrypoint seeder akun administrator
├── docs/                      # Swagger 2.0 / OpenAPI spec (docs.go, swagger.json/yaml)
├── internal/
│   ├── akademik/              # Master data Fakultas, Jurusan, Program Studi
│   ├── app/                   # Router assembly, dependency injection, middleware wiring
│   ├── auth/                  # Otentikasi login, refresh, logout, password hash, JWT
│   ├── config/                # Parser & validator environment configuration
│   ├── database/              # PostgreSQL connection pool & transaction helper WithTx
│   ├── dokumen/               # Upload, verifikasi tipe/ukuran, download berotorisasi
│   ├── jurnal/                # Jurnal harian mahasiswa
│   ├── kkn/                   # Kelompok KKN, anggota, dan kepengurusan
│   ├── logbook/               # Logbook bimbingan dosen pembimbing
│   ├── middleware/            # Auth JWT, RBAC, Recovery, CORS, Security Headers, Rate Limiter
│   ├── pendaftaran/           # Pendaftaran program, penempatan, kesiapan & transisi status
│   ├── penilaian/             # Instrumen penilaian, submit nilai, bobot & kalkulator nilai akhir
│   ├── periode/               # Manajemen periode program
│   ├── platform/              # Library independen (clock, httpx, security, storage)
│   ├── tempat/                # Master mitra magang, sekolah AM, dan desa KKN
│   └── user/                  # Manajemen profil Mahasiswa, Dosen, Pegawai, Admin, Pembimbing Lapangan
├── migrations/                # SQL migration files (golang-migrate)
├── test/
│   └── e2e/                   # Test integrasi end-to-end menyeluruh (Magang, AM, KKN)
├── Dockerfile                 # Multi-stage Docker build
├── docker-compose.yml         # Konfigurasi container runtime & volume mount
├── Makefile                   # Shortcut command line tugas harian
└── README.md                  # Dokumentasi teknis proyek
```

---

## Aktor & Kontrol Akses

| Role Sistem | Profil Database | Hak Akses Utama |
|---|---|---|
| `admin` | `admin` | Full control: master data akademik, tempat, periode, penempatan AM/KKN, instrumen nilai, manajemen user. |
| `pegawai` (Kejur/PIC) | `pegawai` | Verifikasi kesiapan, penetapan dosen pembimbing Magang, monitoring pendaftaran. |
| `mahasiswa` | `mahasiswa` | Pendaftaran program, input penempatan magang mandiri, unggah MoU/proposal/proker, input jurnal harian, melihat nilai akhir pribadi. |
| `dosen` | `dosen` | Membimbing peserta, mengisi logbook bimbingan, melakukan penilaian (sebagai Pembimbing dan/atau Penguji). |
| `pembimbing_lapangan` | `pembimbing_lapangan` | Mentor magang, Guru Pamong AM, Kepala Desa KKN; melakukan penilaian kinerja lapangan mahasiswa bimbingannya. |

---

## Asumsi Arsitektur (A1–A13)

Sesuai kesepakatan spesifikasi MVP pada PRD Bagian 8, asumsi-asumsi berikut telah diimplementasikan dalam kode:

- **A1 — Validasi KRS:** Validasi KRS ke sistem SIAKAD kampus di-stub (`KRSValidator` selalu mengembalikan `true`).
- **A2 — Eligibilitas AM & KKN:** Validasi prasyarat SKS/IPK di-stub (`EligibilityChecker` selalu mengembalikan `true`).
- **A3 — Autentikasi Mandiri:** Menggunakan autentikasi berbasis database internal (`username` dan `password_hash`), belum terhubung ke SSO Kampus/LDAP.
- **A4 — Rumus Nilai Akhir:** Nilai akhir dihitung dengan rumus:
  $$\text{Nilai Akhir} = \frac{\sum (\text{Nilai} \times \text{Bobot})}{\sum \text{Bobot}}$$
- **A5 — Konversi Grade Huruf:**
  - $\ge 85.00$ : **A**
  - $\ge 70.00$ : **B**
  - $\ge 55.00$ : **C**
  - $\ge 40.00$ : **D**
  - $< 40.00$ : **E**
- **A6 — SKS Diakui:** Kolom `sks_diakui` pada `peserta_program` dibiarkan `NULL` hingga ada pedoman resmi konversi kurikulum dari akademik.
- **A7 — Transisi Berjalan Otomatis:** Perubahan status pendaftaran ke `berjalan` dievaluasi otomatis oleh mesin kesiapan sistem segera setelah seluruh syarat penempatan dan dokumen terpenuhi, tanpa menunggu tombol *approval* manual.
- **A8 — Akun Mitra Lapangan:** Pembimbing lapangan (mentor/guru pamong/kepala desa) wajib memiliki akun user di sistem agar dapat menginput nilai secara resmi.
- **A9 — Input Data AM & KKN:** Tidak dibuat tabel terpisah untuk "input data AM/KKN"; seluruh data peserta terpusat pada entitas `peserta_program`, dokumen proker pada `dokumen`, dan penempatan desa/sekolah pada `tempat_program`.
- **A10 — Kewenangan Kejur/Pegawai:** Kejur direpresentasikan oleh role `pegawai` yang dapat mengelola penugasan pembimbing magang lintas jurusan.
- **A11 — Multi-Pendaftaran Semester:** Mahasiswa diperkenankan mendaftar pada program yang berbeda jika periode pendaftaran aktif (dijaga oleh constraint unik per periode dan mahasiswa).
- **A12 — Penyelesaian Periode:** Ketika admin menyelesaikan periode (`selesai`), seluruh pendaftaran berstatus `berjalan` otomatis menjadi `selesai`. Pendaftaran yang masih `diajukan` dan belum siap otomatis dinyatakan `ditolak`.
- **A13 — Pengajuan Ulang:** Mahasiswa yang pendaftarannya berstatus `ditolak` berhak melakukan pengajuan ulang (`POST /pendaftaran/:id/ajukan-ulang`) selama periode pendaftaran masih dalam masa aktif.

---

## Pertanyaan Terbuka untuk Mentor & Tim

Untuk pengembangan rilis berikutnya pasca-MVP, beberapa poin berikut perlu dikonfirmasikan lebih lanjut:

1. **Aturan Eligibilitas Riil:** Berapa batas minimal SKS lulus, IPK minimal, dan mata kuliah prasyarat wajib untuk masing-masing cabang program (AM dan KKN)?
2. **Skema Pembobotan Nilai:** Apakah total bobot 100 berlaku per program secara kumulatif antar-penilai, atau masing-masing penilai memiliki skala 100 yang kemudian dikalikan persentase peran (misal: Mitra 40%, Pembimbing 35%, Penguji 25%)?
3. **Penyediaan Akun Mitra Lapangan:** Apakah seluruh Kepala Desa, Guru Pamong, dan Mentor Industri akan dibuatkan akun langsung oleh UPA TIK, atau disediakan tautan penilaian temporer dengan token aman (magic link/invitation)?
4. **Verifikasi Jurnal Harian:** Apakah entri jurnal harian mahasiswa memerlukan persetujuan (*approval/paraf digital*) berkala oleh dosen pembimbing atau mentor industri?
5. **Restriksi Jurusan bagi PIC/Kejur:** Apakah akun Pegawai/Kejur nantinya wajib dibatasi hak aksesnya agar hanya bisa melihat dan menetapkan dosen pada mahasiswa yang berasal dari jurusannya sendiri?
6. **Formulir Tambahan Spesifik:** Apakah ada atribut formulir khusus yang perlu diinputkan mahasiswa pada saat penempatan Asistensi Mengajar (misal: mata pelajaran yang diajar) atau KKN (misal: fokus bidang pengabdian desa)?
7. **Surat Tugas & Perjalanan Dinas:** Apakah pencetakan Surat Tugas (SPPD) dan pengumuman penempatan akan diintegrasikan langsung di backend SPLK pada fase berikutnya?
