# SPLK Backend

Backend Sistem Pembelajaran Luar Kampus (Magang, Asistensi Mengajar, KKN) untuk UPA TIK Undiksha.

## Prasyarat

- **Go** 1.27+
- **PostgreSQL** 15+ (berjalan di host, bukan container)
- **Docker** dan **Docker Compose** (untuk menjalankan backend di container)
- **Git**
- **migrate** CLI (`golang-migrate`)
- **air** (hot reload, opsional)
- **swag** (generator Swagger, opsional pada tahap ini)

## Setup Database

Buat database dan user di PostgreSQL:

```sql
CREATE USER splk WITH PASSWORD 'ganti_password_ini';
CREATE DATABASE splk_db OWNER splk;
CREATE DATABASE splk_test OWNER splk;
```

## Konfigurasi

Salin `.env.example` ke `.env` dan sesuaikan:

```bash
cp .env.example .env
```

Variabel penting:
- `DATABASE_URL` — connection string PostgreSQL
- `JWT_SECRET` — string acak minimal 32 karakter
- `APP_PORT` — port server (default 8080)
- `APP_TIMEZONE` — zona waktu untuk perhitungan tanggal (default Asia/Makassar)

Lihat `.env.example` untuk daftar lengkap variabel.

## Menjalankan Lokal

```bash
# Pertama kali: install dependency
go mod tidy

# Jalankan server
make run

# Atau dengan hot reload
make dev

# Cek kesehatan
curl http://localhost:8080/health
```

## Menjalankan di Docker

Pastikan `DATABASE_URL` di `.env` menggunakan `host.docker.internal` sebagai host:

```
DATABASE_URL=postgres://splk:password@host.docker.internal:5432/splk_db?sslmode=disable
```

```bash
# Build dan jalankan
make docker-up

# Lihat log
make docker-logs

# Hentikan
make docker-down
```

## Catatan Konfigurasi PostgreSQL (Linux)

Di Linux, PostgreSQL perlu dikonfigurasi agar bisa dijangkau dari container Docker:

1. Edit `postgresql.conf`, ubah `listen_addresses`:
   ```
   listen_addresses = 'localhost,172.17.0.1'
   ```
   Atau gunakan `'*'` untuk menerima dari semua interface.

2. Tambahkan baris di `pg_hba.conf` untuk jaringan Docker:
   ```
   host    splk_db    splk    172.16.0.0/12    scram-sha-256
   ```

3. Restart PostgreSQL:
   ```bash
   sudo systemctl restart postgresql
   ```

Docker Desktop di Windows dan macOS umumnya **tidak memerlukan** langkah ini karena `host.docker.internal` sudah berfungsi secara bawaan.

## Migrasi Database

Perintah migrasi menggunakan file di folder `migrations/` dan terhubung ke database host (`localhost`):

```bash
# Jalankan semua migrasi pada splk_db (menggunakan DATABASE_URL dari .env atau default localhost)
make migrate-up

# Rollback satu langkah
make migrate-down

# Buat file migrasi baru
make migrate-create name=nama_migrasi
```

Untuk database test (`splk_test`):

```bash
# Jalankan migrasi pada database test
DATABASE_URL="postgres://splk:ganti_password_ini@localhost:5432/splk_test?sslmode=disable" make migrate-up

# Rollback satu langkah pada database test
DATABASE_URL="postgres://splk:ganti_password_ini@localhost:5432/splk_test?sslmode=disable" make migrate-down
```

## Perintah Lainnya

```bash
make build    # Build binary
make test     # Jalankan test
make fmt      # Format kode
make vet      # Analisis statis
make tidy     # Rapikan go.mod
make seed     # Jalankan seeder admin (Tahap 3)
make swag     # Generate Swagger (Tahap 13)
```

Gunakan `make help` untuk melihat semua target yang tersedia.
