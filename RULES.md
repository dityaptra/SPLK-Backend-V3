# RULES: Aturan Pengembangan Backend SPLK

Dokumen ini wajib dipatuhi oleh pengembang maupun AI coding assistant. Bila ada konflik, urutan prioritasnya: **RULES.md > ARCHITECTURE.md > PRD.md > prompt tahap**.

## 1. Prinsip Umum

1. Kerjakan **satu tahap per sesi**, sesuai `PROMPTS.md`. Jangan mengerjakan tahap berikutnya sebelum diminta.
2. Jangan mengarang kebutuhan. Bila aturan belum diketahui (eligibilitas, bobot, dan sejenisnya), pakai asumsi yang sudah tercatat di PRD bagian 8, beri komentar `// ASUMSI: <A-nomor>`, dan jangan menambah asumsi baru tanpa mencatatnya.
3. Bagian yang menunggu keputusan mentor ditandai `// TODO(mentor): <pertanyaan>`.
4. Pilih solusi paling sederhana yang memenuhi PRD. Jangan menambah abstraksi, library, atau fitur yang tidak diminta.
5. Setiap tahap harus meninggalkan proyek dalam keadaan **bisa di-build dan semua test lulus**.

## 2. Konvensi Kode Go

- Format dengan `gofmt` (via `make fmt`). Kode harus lolos `go vet ./...`.
- **Penamaan**: istilah domain mengikuti skema database dalam bahasa Indonesia (`PesertaProgram`, `JurnalHarian`, `DosenPembimbing`). Istilah teknis memakai bahasa Inggris (`Handler`, `Service`, `Repository`, `Config`). Field Go memakai PascalCase, tag JSON memakai `snake_case` yang sama dengan nama kolom database (contoh `id_periode`).
- **Komentar** ditulis dalam bahasa Indonesia, dan menjelaskan alasan, bukan mengulang kode. Komentar dokumentasi untuk setiap identifier yang diekspor.
- Semua fungsi yang memanggil database atau I/O menerima `context.Context` sebagai parameter pertama.
- Tidak ada variabel global yang bisa diubah, tidak ada `init()` yang punya efek samping. Dependensi disuntik lewat konstruktor (`NewService(repo, ...)`).
- Error dibungkus dengan konteks: `fmt.Errorf("ambil periode %d: %w", id, err)`. Jangan membuang error dengan `_` kecuali pada `defer` yang aman.
- Dilarang `panic` di luar `main`. Handler tidak boleh memanggil `log.Fatal`.
- Konstanta untuk nilai enum string (`StatusDiajukan`, `JenisMagang`, `RoleAdmin`, dan seterusnya) didefinisikan **satu kali** di paket yang memilikinya. Jangan menulis string literal berulang.
- Fungsi yang bergantung pada waktu memakai `clock.Clock` yang disuntik, agar bisa diuji.

## 3. Aturan Layering

Mengikuti `ARCHITECTURE.md`. Ringkasnya:

- **Handler**: parse request, validasi bentuk input, ambil `Actor` dari context, panggil service, tulis response. Tidak boleh berisi aturan bisnis atau SQL.
- **Service**: seluruh aturan bisnis, otorisasi berbasis relasi, orkestrasi transaksi. Tidak boleh mengimpor `gin` atau `net/http`.
- **Repository**: hanya SQL dan pemetaan baris ke struct. Tidak boleh berisi aturan bisnis. Menerima `database.DBTX` agar bisa dijalankan di pool maupun transaksi.
- Modul tidak boleh saling mengimpor secara melingkar. Bila modul A butuh data modul B, definisikan **interface kecil di paket A** dan suntik implementasinya dari `internal/app`.

## 4. Aturan Database

1. **Skema v2 adalah sumber kebenaran.** `migrations/000001_*.up.sql` berisi file v2 milik pengguna **tanpa perubahan apa pun**. Jangan menyunting migrasi yang sudah dibuat. Perubahan skema selalu lewat migrasi baru bernomor berikutnya, lengkap dengan file `down`.
2. Tabel akun bernama **`users`** (jamak), bukan `user`, karena `user` adalah kata cadangan PostgreSQL. Dilarang membuat atau mengacu ke tabel bernama `user`.
3. Seluruh query memakai placeholder (`$1`, `$2`). **Dilarang** menyusun SQL dengan penggabungan string atau `fmt.Sprintf` untuk nilai dari input. Untuk kolom urutan atau filter dinamis, gunakan allowlist.
4. Operasi yang menyentuh lebih dari satu tabel wajib memakai `database.WithTx`.
5. Perhitungan nilai akhir dan transisi status memakai `SELECT ... FOR UPDATE` pada baris `peserta_program` untuk mencegah balapan data.
6. Koneksi diatur `timezone=UTC` (lewat `RuntimeParams`). Semua timestamp yang ditulis aplikasi adalah UTC. Jangan menyimpan waktu lokal.
7. Pemetaan error PostgreSQL: `23505` (unique) menjadi 409, `23503` (foreign key) menjadi 422, `23514` (check) menjadi 422. Jangan membocorkan pesan error mentah database ke klien.
8. Tidak ada `DELETE` fisik untuk data akademik. Gunakan `is_aktif` atau `is_active`. Pengecualian: refresh token, anggota kelompok (set `id_kelompok_kkn` ke NULL), dan dokumen yang dihapus pengunggah.

## 5. Aturan API

- Prefiks `/api/v1`. Resource memakai kata benda dalam bahasa Indonesia berformat kebab-case (`/tempat-program`).
- **Format sukses**: `{"data": ..., "meta": {...}}`. `meta` hanya ada untuk daftar (`page`, `limit`, `total`).
- **Format error**: `{"error": {"code": "KODE_ERROR", "message": "pesan untuk manusia", "details": [...]}}`. `code` berupa konstanta stabil (misalnya `PERIODE_TIDAK_AKTIF`) agar frontend bisa bergantung padanya.
- Kode status: `200` ok, `201` dibuat, `204` tanpa isi, `400` input tidak valid atau JSON rusak, `401` belum login atau token tidak valid, `403` tidak berwenang, `404` tidak ditemukan, `409` konflik (data ganda), `422` melanggar aturan bisnis, `500` kesalahan server.
- Validasi bentuk input dengan tag `binding` pada DTO. Aturan bisnis divalidasi di service.
- Jangan pernah mengembalikan `password_hash`, `token_hash`, atau path file mutlak.
- Mengakses data yang bukan miliknya mengembalikan `404` (bukan `403`) bila keberadaan datanya tidak boleh diketahui penanya, misalnya mahasiswa melihat pendaftaran mahasiswa lain.
- Daftar selalu berpagination (`page`, `limit`; default 20; maksimal 100) dan urutan yang deterministik.

## 6. Aturan Keamanan

- `JWT_SECRET` wajib minimal 32 karakter. Aplikasi menolak start bila kosong atau terlalu pendek.
- Password di-hash dengan `bcrypt` (cost 12). Minimal 8 karakter.
- Refresh token: 32 byte acak, dikirim ke klien sekali, disimpan sebagai hash SHA-256 di `tokens.token_hash`. Rotasi setiap refresh.
- Pesan error login selalu sama untuk username salah maupun password salah (mencegah enumerasi akun). Akun `is_active = false` ditolak.
- **Jangan** mencatat password, token, atau isi header `Authorization` di log.
- Upload file: cek ekstensi dan isi file (`http.DetectContentType`), batasi ukuran dengan `http.MaxBytesReader`, simpan dengan nama acak UUID, bentuk path dari data yang dikontrol server (bukan dari nama file pengguna), dan tolak path traversal. `nama_file` asli hanya disimpan sebagai metadata.
- File hanya bisa diunduh lewat endpoint terautentikasi yang memeriksa hak akses (BR-14).
- Seluruh endpoint selain `/health`, `/auth/login`, dan `/auth/refresh` wajib melewati middleware autentikasi.
- CORS hanya untuk origin yang terdaftar di `CORS_ALLOWED_ORIGINS`.
- Rahasia hanya dari environment variable. File `.env` masuk `.gitignore`. Hanya `.env.example` yang di-commit.

## 7. Penanganan Batasan Database v2

Aturan wajib bagi kode, merujuk PRD bagian 9:

1. Migrasi `000002` hanya berisi `ALTER TABLE peserta_program ALTER COLUMN id_tempat_program DROP NOT NULL`. Jangan menambahkan perubahan lain ke migrasi ini tanpa persetujuan.
2. `periode.dibuat_oleh` hanya diisi bila pembuat memiliki profil `pegawai`. Jangan mengisinya dengan ID admin.
3. Catatan login gagal untuk username yang tidak dikenal tidak masuk `log_aktivitas`.
4. Validasi `status`, `jenis_program`, `role_penilai`, dan `jenis` (pembimbing lapangan) dilakukan di service karena database tidak memiliki CHECK.
5. Setiap perubahan dosen pembimbing atau pembimbing lapangan pada kelompok KKN harus meng-update seluruh anggotanya dalam satu transaksi. Kolom `kelompok_kkn.proker_url` tidak dipakai.
6. Status `dinilai` tidak ada. Gunakan `nilai_akhir IS NOT NULL` pada pendaftaran berstatus `selesai`.

## 8. Aturan Pengujian

- Setiap service yang berisi aturan bisnis punya unit test table-driven dengan repository tiruan (fake). Wajib diuji: kalkulator nilai akhir, transisi status, pemeriksaan kesiapan, otorisasi per relasi, validasi periode pendaftaran.
- Integration test memakai database terpisah `splk_test` di PostgreSQL host. Migrasi dijalankan sebelum test, dan tiap test membersihkan datanya sendiri. Test integrasi diberi build tag atau dilewati bila `TEST_DATABASE_URL` kosong.
- Test tidak boleh bergantung pada urutan eksekusi atau waktu nyata (pakai `clock` tiruan).
- Bug yang ditemukan diperbaiki bersama test yang mereproduksinya.

## 9. Aturan Git dan Docker

- Commit kecil dan bermakna dengan format `tipe(lingkup): ringkasan`, contoh `feat(auth): tambah login dan refresh token`. Tipe: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.
- Satu tahap, satu cabang atau minimal satu rangkaian commit yang jelas.
- Yang masuk `.gitignore`: `.env`, `uploads/`, `bin/`, `*.log`.
- **Hanya backend Go yang di-container.** PostgreSQL berjalan di host dan dijangkau lewat `host.docker.internal`. Jangan menambahkan service database ke `docker-compose.yml`.
- Folder `uploads/` dipasang sebagai volume agar file tidak hilang saat container dibuat ulang.
- Image memakai multi-stage build, berjalan sebagai user non-root, dengan `CGO_ENABLED=0`.

## 10. Aturan Khusus untuk AI Assistant

**Sebelum menulis kode**

1. Baca `PRD.md`, `RULES.md`, `ARCHITECTURE.md` sampai selesai, lalu baca prompt tahap yang diminta.
2. Periksa kondisi repo saat ini (struktur folder, `go.mod`, migrasi) agar tidak menimpa pekerjaan sebelumnya.
3. Bila ada hal yang ambigu dan tidak tercakup di dokumen, ajukan **satu pertanyaan ringkas** sebelum mulai. Jangan menebak untuk keputusan yang sulit diubah.

**Saat bekerja**

4. Hanya menambah dependency dari daftar di ARCHITECTURE.md bagian 2. Dependency lain harus diminta persetujuan dulu.
5. Jangan mengubah file di luar lingkup tahap, kecuali perbaikan kecil yang diperlukan agar build berjalan. Sebutkan perbaikan itu dalam laporan.
6. Jangan melonggarkan aturan keamanan, melewati validasi, atau menonaktifkan test demi membuat sesuatu lulus.
7. Jangan membuat data atau referensi palsu (misalnya ID, kredensial, atau URL) di kode produksi. Data contoh hanya di seeder atau test.

**Definisi selesai (Definition of Done) setiap tahap**

- `gofmt` bersih, `go vet ./...`, `go build ./...`, dan `go test ./...` lulus.
- Seluruh kriteria penerimaan pada prompt tahap terpenuhi dan sudah dicoba (curl atau test).
- Migrasi (bila ada) bisa `up`, `down`, lalu `up` lagi tanpa error.
- Anotasi Swagger ditambahkan untuk endpoint baru.
- Laporan akhir singkat berisi: file yang dibuat atau diubah, cara menjalankan dan mencoba, asumsi yang dipakai, serta hal yang masih tertunda.

## 11. Anti-Pola yang Dilarang

- Logika bisnis di handler atau SQL di service.
- `SELECT *` pada kode produksi (sebutkan kolom).
- Menyimpan status di string literal yang tersebar.
- Mengandalkan `localhost` untuk menjangkau PostgreSQL dari dalam container.
- Menyajikan folder `uploads/` sebagai static file.
- Menangkap error lalu mengabaikannya tanpa log.
- Mengembalikan error internal (stack trace, pesan SQL) ke klien.
