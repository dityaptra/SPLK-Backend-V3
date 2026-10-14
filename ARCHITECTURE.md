# ARCHITECTURE: Backend SPLK

## 1. Gambaran Umum

Backend berupa **monolit modular** dengan arsitektur berlapis (handler, service, repository). Setiap domain bisnis menjadi satu paket di `internal/`. PostgreSQL berjalan di host, sedangkan backend Go berjalan di Docker.

```mermaid
flowchart LR
    Client[Frontend / Postman] -->|HTTP JSON| API
    subgraph Docker
        API[Backend Go<br/>Gin]
        Vol[(Volume uploads/)]
        API --- Vol
    end
    API -->|host.docker.internal:5432| PG[(PostgreSQL di host)]
```

## 2. Tech Stack dan Dependency yang Diizinkan

| Kebutuhan | Pilihan |
|---|---|
| Bahasa | Go (versi yang terpasang di mesin, samakan dengan `go.mod` dan Dockerfile) |
| HTTP | `github.com/gin-gonic/gin` |
| Database | PostgreSQL, driver `github.com/jackc/pgx/v5` (`pgxpool`), SQL ditulis manual di repository |
| Migrasi | `golang-migrate` (CLI di host) |
| JWT | `github.com/golang-jwt/jwt/v5` (HS256) |
| Hash password | `golang.org/x/crypto/bcrypt` |
| Konfigurasi | environment variable, `github.com/joho/godotenv` untuk development |
| ID file | `github.com/google/uuid` |
| Logging | `log/slog` (bawaan Go) |
| Dokumentasi API | `github.com/swaggo/swag`, `github.com/swaggo/gin-swagger`, `github.com/swaggo/files` |
| Test | `testing` bawaan dan `github.com/stretchr/testify` |
| Dev tool | `air` (hot reload), `migrate`, `swag` |

Dependency di luar tabel ini tidak boleh ditambahkan tanpa persetujuan.

## 3. Struktur Folder

```
splk-backend/
├── cmd/
│   ├── api/main.go                # titik masuk server (tipis)
│   └── seed/main.go               # membuat akun admin pertama
├── internal/
│   ├── app/                       # wiring dependency (DI manual) dan konstruksi router
│   ├── config/                    # baca dan validasi env
│   ├── database/                  # pool, WithTx, DBTX, pemetaan error pg
│   ├── platform/
│   │   ├── httpx/                 # format response, AppError, binding, pagination
│   │   ├── security/              # bcrypt, JWT, generator dan hash refresh token
│   │   ├── storage/               # penyimpanan file lokal (interface + implementasi)
│   │   └── clock/                 # abstraksi waktu, hari ini zona Asia/Makassar
│   ├── middleware/                # request id, logger, recovery, cors, auth, role
│   ├── auth/                      # login, refresh, logout, me, Actor
│   ├── user/                      # akun dan profil (mahasiswa, dosen, pegawai, admin, pembimbing_lapangan)
│   ├── akademik/                  # fakultas, jurusan, program_studi
│   ├── tempat/                    # tempat_program dan pembimbing_lapangan
│   ├── periode/
│   ├── pendaftaran/               # peserta_program, akses, kesiapan, port KRS dan eligibilitas
│   ├── kkn/                       # kelompok_kkn dan sinkronisasi anggota
│   ├── dokumen/
│   ├── jurnal/
│   ├── logbook/
│   ├── penilaian/                 # instrumen, penilaian, kalkulator nilai
│   ├── pengumuman/                # opsional (Tahap 14)
│   └── spd/                       # surat perjalanan dinas, opsional (Tahap 14)
├── migrations/                    # 000001_init_schema (v2), 000002_patch_..., dan seterusnya
├── docs/                          # hasil generate swagger
├── uploads/                       # file unggahan (di-ignore git, volume Docker)
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env.example
├── PRD.md  RULES.md  ARCHITECTURE.md  PROMPTS.md
└── go.mod
```

Setiap paket modul (misalnya `periode`) berisi file dengan tanggung jawab seragam:

| File | Isi |
|---|---|
| `model.go` | Struct entitas dan konstanta enum modul |
| `dto.go` | Struct request dan response (tag `json` dan `binding`) |
| `repository.go` | Interface `Repository` dan implementasi pgx |
| `service.go` | Aturan bisnis, otorisasi relasi, transaksi |
| `handler.go` | Handler Gin dan anotasi Swagger |
| `routes.go` | `RegisterRoutes(rg *gin.RouterGroup, ...)` |
| `*_test.go` | Unit test service (fake repo) |

## 4. Layering dan Aturan Dependensi

```mermaid
flowchart TD
    H[Handler<br/>HTTP, DTO, Actor] --> S[Service<br/>aturan bisnis, otorisasi, transaksi]
    S --> R[Repository<br/>SQL via pgx]
    R --> DB[(PostgreSQL)]
    S -. interface kecil .-> X[Modul lain]
```

- Panah hanya menuju ke bawah. Handler tidak boleh memanggil repository langsung.
- Lintas modul **hanya lewat interface yang didefinisikan di sisi pemakai**, disuntik dari `internal/app`. Contoh: `jurnal.Service` butuh memeriksa akses peserta, sehingga mendefinisikan `type PesertaAccess interface { ... }` dan `pendaftaran.Service` memenuhinya.
- Tidak boleh ada import melingkar. Paket `platform/*` dan `database` tidak mengimpor paket modul.

### Akses database dan transaksi

```go
// DBTX dipenuhi oleh *pgxpool.Pool maupun pgx.Tx, sehingga repository
// bisa dipakai di dalam maupun di luar transaksi.
type DBTX interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx menjalankan fn dalam satu transaksi: commit jika nil, rollback jika error atau panic.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error
```

Pola pemakaian di service: repository menerima `DBTX` pada setiap method (`repo.Create(ctx, db, ...)`), sehingga service memutuskan apakah memakai pool atau transaksi.

## 5. Siklus Request

```mermaid
sequenceDiagram
    participant C as Client
    participant M as Middleware
    participant H as Handler
    participant S as Service
    participant R as Repository
    C->>M: HTTP request
    M->>M: request id, logger, recovery, cors
    M->>M: verifikasi JWT, muat Actor, cek role
    M->>H: lanjut
    H->>H: bind dan validasi DTO
    H->>S: panggil use case (ctx, actor, input)
    S->>S: aturan bisnis dan otorisasi relasi
    S->>R: query / transaksi
    R-->>S: data
    S-->>H: hasil atau AppError
    H-->>C: JSON (sukses atau error terstandar)
```

### Format response dan error

```go
// AppError adalah error yang aman ditampilkan ke klien.
type AppError struct {
    HTTPStatus int    // 400, 401, 403, 404, 409, 422, 500
    Code       string // konstanta stabil, contoh "PERIODE_TIDAK_AKTIF"
    Message    string // pesan bahasa Indonesia untuk manusia
    Details    any    // opsional, misalnya daftar field yang salah
}
```

Handler menerjemahkan error dari service lewat satu fungsi `httpx.WriteError(c, err)`. `*AppError` ditulis apa adanya, error `pgconn.PgError` dipetakan (23505 jadi 409, 23503 dan 23514 jadi 422), selain itu menjadi 500 generik dengan detail hanya di log.

## 6. Autentikasi dan Otorisasi

### Token

| Token | Bentuk | Umur | Penyimpanan |
|---|---|---|---|
| Access | JWT HS256 dengan klaim `sub` (user id), `roles`, `exp` | 15 menit | tidak disimpan di server |
| Refresh | 32 byte acak, base64url | 7 hari | hash SHA-256 di `tokens.token_hash`, `expired_at` dalam UTC |

Refresh dirotasi: token lama dihapus dan token baru dibuat dalam satu transaksi. Logout menghapus baris token. Karena `tokens` tidak punya `revoked_at`, pencabutan berarti menghapus baris.

### Actor

Middleware autentikasi menyimpan `Actor` di context Gin:

```go
type Actor struct {
    UserID int
    Roles  []string // dari klaim JWT
}
```

ID profil (`mahasiswa.id`, `dosen.id`, dan seterusnya) **tidak dimasukkan ke JWT**. Service memuatnya saat dibutuhkan lewat `user.ProfileResolver` (query by `id_user`), agar perubahan relasi langsung berlaku.

### Dua lapis otorisasi

1. **Berbasis role** (middleware): `RequireRoles("admin")`, `RequireRoles("admin", "pegawai")`, dan seterusnya. Menjawab "boleh memanggil endpoint ini?".
2. **Berbasis relasi** (service): menjawab "boleh menyentuh pendaftaran ini?". Dipusatkan di `pendaftaran.AccessService`:

```go
type Relation string

const (
    RelAdmin        Relation = "admin"
    RelPegawai      Relation = "pegawai"
    RelMahasiswa    Relation = "mahasiswa"     // pemilik pendaftaran
    RelPembimbing   Relation = "dosen_pembimbing"
    RelPenguji      Relation = "dosen_penguji"
    RelLapangan     Relation = "pembimbing_lapangan"
    RelAnggotaKKN   Relation = "anggota_kelompok"
)

// Resolve mengembalikan relasi actor terhadap pendaftaran, atau ErrTidakDitemukan
// bila actor tidak punya hubungan apa pun (menghasilkan 404, bukan 403).
Resolve(ctx, actor, idPeserta) (Relation, error)
```

Modul `dokumen`, `jurnal`, `logbook`, dan `penilaian` memakai hasil `Resolve` ini, sehingga aturan BR-14 dan BR-15 hanya ada di satu tempat.

## 7. Domain Inti: peserta_program

`peserta_program` adalah pusat. Seluruh aktivitas (jurnal, logbook, dokumen, penilaian) mengacu ke satu `id_peserta_program`. Dokumen KKN dapat memakai `id_kelompok_kkn`.

### Pengecekan kesiapan

`pendaftaran.ReadinessChecker` menerima struktur peserta beserta daftar jenis dokumen yang dimiliki, lalu mengembalikan `(siap bool, kekurangan []string)` sesuai tabel PRD 4.3. Dipanggil di akhir setiap operasi yang bisa memenuhi syarat: penempatan, penetapan dosen, unggah dokumen, penambahan anggota kelompok. Bila `siap` dan status `diajukan`, status menjadi `berjalan` dalam transaksi yang sama.

### Port yang bisa diganti

```go
// KRSValidator memeriksa apakah mahasiswa terdaftar KRS pada periode berjalan.
// ASUMSI A1: implementasi sementara selalu mengembalikan true.
type KRSValidator interface {
    Validate(ctx context.Context, m Mahasiswa, p Periode) (ok bool, alasan string, err error)
}

// EligibilityChecker memeriksa syarat khusus program (AM dan KKN).
// ASUMSI A2: implementasi sementara selalu mengembalikan true.
type EligibilityChecker interface {
    Check(ctx context.Context, m Mahasiswa, p Periode) (ok bool, alasan string, err error)
}
```

Saat aturan sebenarnya diketahui, cukup mengganti implementasi di `internal/app` tanpa menyentuh handler maupun database.

## 8. Penilaian dan Nilai Akhir

```
instrumen_program = semua baris instrumen_penilaian dengan jenis_program = peserta.jenis_program

Syarat lengkap:
  untuk setiap instrumen i di instrumen_program terdapat satu baris penilaian
  dari user penilai yang sah untuk role_penilai(i) pada peserta ini

nilai_akhir = ROUND( SUM(nilai_i x bobot_i) / SUM(bobot_i) , 2 )
```

Sumber penilai sah per role instrumen:

| `role_penilai` | User penilai |
|---|---|
| `dosen_pembimbing` | `dosen.id_user` dari `peserta_program.id_dosen_pembimbing` |
| `dosen_penguji` | `dosen.id_user` dari `peserta_program.id_dosen_penguji` |
| `pembimbing_lapangan` | `pembimbing_lapangan.id_user` dari `peserta_program.id_pembimbing_lapangan` (harus ada akun) |

Langkah `SubmitPenilaian` (satu transaksi):

1. Kunci baris peserta (`FOR UPDATE`). Pastikan status `selesai` dan `nilai_akhir IS NULL`.
2. Tentukan role penilai pemanggil lewat `AccessService.Resolve`. Tolak instrumen yang `role_penilai`-nya berbeda atau `jenis_program`-nya tidak cocok.
3. Upsert nilai (`ON CONFLICT (id_peserta_program, id_instrumen, id_penilai) DO UPDATE`).
4. Jika instrumen program kosong, jangan menghitung. Jika semua instrumen sudah terisi, hitung lewat `ScoreCalculator`, simpan `nilai_akhir`, `grade_huruf`, dan `updated_at`.

`ScoreCalculator` berupa interface (`Hitung(nilai []NilaiInstrumen) (float64, string)`) sehingga rumus dan konversi huruf (asumsi A4 dan A5) dapat diganti tanpa mengubah alur.

## 9. Penyimpanan File Lokal

```go
type Storage interface {
    Save(ctx context.Context, relPath string, r io.Reader) (size int64, err error)
    Open(ctx context.Context, relPath string) (io.ReadCloser, error)
    Delete(ctx context.Context, relPath string) error
}
```

- Implementasi `LocalStorage` bekerja di bawah `UPLOAD_DIR` dan menolak path yang keluar dari direktori itu setelah `filepath.Clean`.
- Path relatif yang disimpan di `dokumen.file_url`: `pendaftaran/<id_peserta>/<uuid>.<ext>` atau `kkn/<id_kelompok>/<uuid>.<ext>`.
- Unggah: validasi ukuran dan tipe, simpan file, lalu insert baris `dokumen`. Jika insert gagal, file dihapus kembali. Hapus: hapus baris dulu, lalu file.
- Unduh melalui `GET /dokumen/:id/download` setelah pemeriksaan akses. Header `Content-Disposition` memakai `nama_file` asli yang sudah disanitasi.
- Interface memudahkan pindah ke MinIO atau object storage kelak.

## 10. Konfigurasi

| Variabel | Contoh | Keterangan |
|---|---|---|
| `APP_ENV` | `development` | `development` atau `production` |
| `APP_PORT` | `8080` | |
| `APP_TIMEZONE` | `Asia/Makassar` | Untuk perhitungan "hari ini" |
| `DATABASE_URL` | `postgres://splk:***@host.docker.internal:5432/splk_db?sslmode=disable` | Dari host (tanpa Docker) ganti host menjadi `localhost` |
| `JWT_SECRET` | string acak minimal 32 karakter | Wajib |
| `JWT_ACCESS_TTL` | `15m` | |
| `JWT_REFRESH_TTL` | `168h` | |
| `UPLOAD_DIR` | `/app/uploads` | Di host: `./uploads` |
| `MAX_UPLOAD_MB` | `5` | |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Dipisah koma |
| `LOG_LEVEL` | `info` | |
| `SEED_ADMIN_USERNAME`, `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD` | | Hanya dipakai `cmd/seed` |

`config.Load()` memvalidasi seluruh nilai dan menghentikan aplikasi dengan pesan jelas bila ada yang tidak valid.

## 11. Docker dan Deployment Lokal

- **Hanya backend** yang di-container. Dockerfile multi-stage membangun dua binary (`api` dan `seed`), image akhir berbasis Alpine, berjalan sebagai user non-root, `TZ=Asia/Makassar`.
- `docker-compose.yml` hanya memiliki service `api`, dengan `env_file: .env`, `extra_hosts: ["host.docker.internal:host-gateway"]` (diperlukan di Linux), volume `./uploads:/app/uploads`, dan `restart: unless-stopped`.
- **Migrasi dijalankan dari host** dengan CLI `migrate`, bukan di dalam container.
- **Menjangkau PostgreSQL di host dari container** (khusus Linux): set `listen_addresses` agar mencakup interface Docker, tambahkan aturan di `pg_hba.conf` untuk rentang jaringan Docker (misalnya `172.16.0.0/12`) dengan `scram-sha-256`, lalu restart PostgreSQL. Docker Desktop (Windows dan macOS) umumnya tidak memerlukan langkah ini.
- Akun admin pertama dibuat dengan `make seed` (dari host) atau `docker compose run --rm api ./seed`.

## 12. Logging, Observabilitas, dan Kesehatan

- `slog` berformat JSON di production dan teks di development. Setiap request mencatat `request_id`, method, path, status, durasi, dan `user_id` bila ada.
- Middleware `request id` membaca `X-Request-ID` atau membuat yang baru, lalu mengembalikannya di header response.
- `GET /health` mengembalikan `{"status":"ok","database":"up"}` (HTTP 200) atau `database":"down"` (HTTP 503) berdasarkan `pool.Ping`.
- `log_aktivitas` mencatat login berhasil dan gagal (untuk user yang dikenal), logout, dan aksi penting administratif (penetapan pembimbing, penyelesaian periode, penghitungan nilai).
- Shutdown halus: tangkap `SIGINT` dan `SIGTERM`, hentikan server dengan timeout 10 detik, lalu tutup pool.

## 13. Strategi Pengujian

| Level | Cakupan | Cara |
|---|---|---|
| Unit | Service (kalkulator nilai, kesiapan, transisi status, relasi akses, validasi tanggal) | Fake repository, `clock` tiruan, table-driven |
| Integrasi | Repository dan alur HTTP end-to-end tiga program | Database `splk_test` di host, migrasi dijalankan sebelum test, `httptest` memanggil router sungguhan |
| Kontrak | Dokumentasi Swagger sesuai kode | `swag init` tanpa error di CI lokal (`make swag`) |

Integration test end-to-end (Magang, AM, KKN) menjadi gerbang penerimaan MVP (PRD bagian 11).
