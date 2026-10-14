# PROMPTS: Pembangunan SPLK Backend Secara Bertahap

## Cara Pakai

1. Siapkan repo kosong. Letakkan di root: `PRD.md`, `RULES.md`, `ARCHITECTURE.md`, `PROMPTS.md`, dan file migrasi v2 milikmu dengan nama `sistem_pembelajaran_luar_kampus_migration.sql`.
2. Buka AI coding assistant (misalnya Claude Code) di folder repo itu.
3. Untuk setiap tahap: tempel **Prompt Pembuka** (sekali per sesi), lalu tempel **prompt tahap** yang bersangkutan.
4. Setelah tahap selesai: jalankan sendiri bagian **Verifikasi Manual**, baca diff-nya, commit, baru lanjut ke tahap berikutnya.
5. Jangan melompati tahap. Setiap tahap bergantung pada hasil tahap sebelumnya.

Prasyarat di laptop: Go, PostgreSQL (host), Docker, Git, `migrate`, `air`, `swag` (lihat jawaban instalasi sebelumnya). Buat database dan user terlebih dahulu:

    CREATE USER splk WITH PASSWORD 'ganti_password_ini';
    CREATE DATABASE splk_db OWNER splk;
    CREATE DATABASE splk_test OWNER splk;

## Peta Tahap

| Tahap | Judul | Hasil utama |
|---|---|---|
| 0 | Setup proyek dan Docker | `/health` jalan lokal dan di Docker |
| 1 | Migrasi database | 24 tabel terbentuk, patch 000002 |
| 2 | Fondasi platform | Response, error, tx, security, storage, clock |
| 3 | Autentikasi dan RBAC | Login, refresh, logout, middleware, seeder admin |
| 4 | Akun dan master data | Akun semua aktor, akademik, tempat, pembimbing lapangan |
| 5 | Periode | CRUD dan transisi status periode |
| 6 | Pendaftaran inti | Daftar, validasi stub, akses per relasi |
| 7 | Dokumen | Unggah, daftar, unduh terautentikasi |
| 8 | Cabang Magang | Penempatan, pembimbing, kesiapan |
| 9 | Cabang AM | Pemetaan sekolah, guru pamong |
| 10 | Cabang KKN | Kelompok, anggota, sinkronisasi |
| 11 | Jurnal, logbook, penyelesaian periode | Pelaksanaan dan transisi `selesai` |
| 12 | Penilaian dan nilai akhir | Instrumen, penilaian, kalkulator |
| 13 | Swagger, test E2E, finalisasi | Gerbang penerimaan MVP |
| 14 | Opsional | Review jurnal, verifikasi dokumen, pengumuman, SPD |

---

## Prompt Pembuka (tempel di awal setiap sesi)

```
Kamu adalah engineer Go senior yang membantu membangun backend SPLK
(Sistem Pembelajaran Luar Kampus) untuk UPA TIK Undiksha.

Sebelum melakukan apa pun:
1. Baca PRD.md, RULES.md, dan ARCHITECTURE.md sampai selesai.
2. Periksa kondisi repo saat ini (struktur folder, go.mod, migrations/, git log)
   supaya tidak menimpa pekerjaan sebelumnya.
3. Kerjakan HANYA tahap yang aku sebutkan di bawah. Jangan mengerjakan tahap lain.

Aturan kerja:
- Patuhi RULES.md, terutama bagian 10 (Aturan Khusus untuk AI Assistant)
  dan Definition of Done.
- Skema database v2 adalah sumber kebenaran. Jangan mengubah migrasi yang sudah ada.
- Bila ada hal ambigu yang tidak tercakup dokumen, ajukan satu pertanyaan ringkas
  sebelum menulis kode. Untuk hal kecil, pilih opsi paling sederhana dan catat
  sebagai asumsi.
- Beri komentar "// ASUMSI: A<n>" dan "// TODO(mentor): ..." sesuai RULES.md.
- Akhiri dengan laporan singkat: file dibuat/diubah, cara menjalankan, hasil
  verifikasi, asumsi yang dipakai, dan hal yang tertunda.
```

---

## Tahap 0: Setup Proyek dan Docker

```
Kerjakan TAHAP 0: Setup proyek dan Docker.

Tujuan: kerangka proyek Go yang bisa berjalan lokal maupun di Docker, dengan
endpoint /health yang mengecek koneksi database.

Tugas:
1. Inisialisasi modul Go dengan nama "splk-backend". Buat struktur folder sesuai
   ARCHITECTURE.md bagian 3 (folder modul yang belum dikerjakan boleh dibuat
   nanti, cukup yang dibutuhkan tahap ini).
2. internal/config: Load() membaca env sesuai ARCHITECTURE.md bagian 10, memakai
   godotenv hanya bila file .env ada. Validasi: DATABASE_URL wajib, JWT_SECRET
   minimal 32 karakter, APP_PORT valid, MAX_UPLOAD_MB > 0, APP_TIMEZONE bisa
   di-load (time.LoadLocation). Gagal start dengan pesan jelas bila tidak valid.
3. internal/database: NewPool(ctx, cfg) memakai pgxpool; set
   ConnConfig.RuntimeParams["timezone"] = "UTC"; lakukan Ping saat start.
4. cmd/api/main.go (tipis): load config, setup slog (JSON untuk production, teks
   untuk development), buat pool, jalankan Gin, route GET /health yang
   mengembalikan {"status":"ok","database":"up"} (200) atau database "down" (503).
   Implementasikan graceful shutdown (SIGINT/SIGTERM, timeout 10 detik).
5. Dockerfile multi-stage: build dua binary (api dan seed; untuk sementara
   cmd/seed boleh berupa main kosong yang mencetak "belum diimplementasikan"),
   image akhir alpine + ca-certificates + tzdata, TZ=Asia/Makassar, user
   non-root, EXPOSE 8080. Versi Go di Dockerfile harus sama dengan go.mod.
6. docker-compose.yml: hanya service "api" (JANGAN tambahkan database), env_file
   .env, extra_hosts host.docker.internal:host-gateway, volume ./uploads:/app/uploads,
   restart unless-stopped.
7. .env.example (semua variabel dengan nilai contoh aman), .gitignore (.env,
   uploads/, bin/, *.log, tmp/), .dockerignore.
8. Makefile dengan target: run, dev (air), build, test, fmt, vet, tidy,
   migrate-up, migrate-down, migrate-create name=..., seed, docker-up
   (docker compose up --build -d), docker-down, docker-logs.
9. README.md singkat: prasyarat, cara setup database, cara menjalankan lokal dan
   di Docker, catatan konfigurasi PostgreSQL host untuk Linux (listen_addresses
   dan pg_hba.conf) sesuai ARCHITECTURE.md bagian 11.

Batasan: jangan menulis fitur bisnis apa pun. Jangan menambah dependency di luar
daftar ARCHITECTURE.md bagian 2.

Kriteria penerimaan:
- go build ./... dan go vet ./... lulus.
- Dengan .env valid, "make run" menjalankan server dan GET /health mengembalikan 200.
- Dengan database dimatikan, /health mengembalikan 503 tanpa crash.
- "docker compose up --build" berjalan dan GET http://localhost:8080/health
  mengembalikan database "up" (menjangkau PostgreSQL di host).
- Menghapus JWT_SECRET dari env membuat aplikasi menolak start dengan pesan jelas.
```

**Verifikasi manual**: `curl localhost:8080/health`; matikan PostgreSQL sebentar dan ulangi; `docker compose up --build`.

---

## Tahap 1: Migrasi Database

```
Kerjakan TAHAP 1: Migrasi database.

Tujuan: skema v2 terpasang lewat golang-migrate, ditambah satu patch wajib.

Tugas:
1. Buat migrations/000001_init_schema.up.sql dengan isi file
   sistem_pembelajaran_luar_kampus_migration.sql TANPA MENGUBAH SATU BARIS PUN
   (salin apa adanya; file ini sudah memakai nama tabel users).
2. Buat migrations/000001_init_schema.down.sql: DROP TABLE IF EXISTS ... CASCADE
   untuk seluruh 24 tabel aplikasi dalam urutan aman (anak sebelum induk). Jangan
   men-drop extension pgcrypto.
3. Buat migrations/000002_patch_peserta_tempat_nullable.up.sql berisi tepat:
   ALTER TABLE "peserta_program" ALTER COLUMN "id_tempat_program" DROP NOT NULL;
   beserta komentar alasan (PRD bagian 9 nomor 1: penempatan AM dan KKN baru
   ditentukan setelah mendaftar). File down-nya mengembalikan SET NOT NULL.
   Jangan menambahkan perubahan lain.
4. Pastikan target Makefile migrate-up / migrate-down / migrate-create memakai
   DATABASE_URL versi host (localhost) dan folder migrations. Dokumentasikan di
   README cara menjalankan, termasuk untuk database splk_test.
5. Tulis test integrasi kecil (dilewati bila TEST_DATABASE_URL kosong) yang
   memeriksa: 24 tabel aplikasi ada, tabel roles berisi 5 baris seed, constraint
   unik "uq_peserta_program_mahasiswa_periode_jenis" ada, dan kolom
   peserta_program.id_tempat_program nullable.

Batasan: jangan menyunting file 000001 sesudah dibuat. Jangan memperbaiki
masalah v2 lain (CHECK, composite FK, dan sebagainya); itu bukan lingkup tahap ini.

Kriteria penerimaan:
- migrate up pada splk_db berhasil; migrate down lalu up lagi berhasil.
- Diff antara 000001 up dan file asli kosong.
- Test integrasi lulus pada splk_test.
```

**Verifikasi manual**: `make migrate-up`, lalu di `psql` jalankan `\dt` (harus ada 24 tabel ditambah `schema_migrations`) dan `\d peserta_program`.

---

## Tahap 2: Fondasi Platform

```
Kerjakan TAHAP 2: Fondasi platform.

Tujuan: paket-paket pendukung yang dipakai semua modul.

Tugas (semua di bawah internal/):
1. database: tipe DBTX, WithTx(ctx, pool, fn) (commit/rollback/panic-safe), dan
   fungsi pemetaan error pgconn.PgError ke AppError (23505 -> 409, 23503 dan
   23514 -> 422) sesuai RULES.md bagian 4 dan 5.
2. platform/httpx: tipe AppError (HTTPStatus, Code, Message, Details) beserta
   konstruktor (BadRequest, Unauthorized, Forbidden, NotFound, Conflict,
   Unprocessable, Internal); fungsi OK/Created/NoContent/List (format {"data","meta"});
   WriteError(c, err) yang menerjemahkan AppError dan error pg, dan untuk error
   tak dikenal membalas 500 generik sambil mencatat detail ke slog; helper bind
   JSON yang mengubah kesalahan validator menjadi 400 dengan details per field;
   struct Pagination (page, limit; default 20; maksimal 100; hitung offset) dengan
   parser dari query string.
3. platform/security: HashPassword/CheckPassword (bcrypt cost 12, panjang minimal
   8 divalidasi di service), JWT (Generate/Parse HS256 dengan klaim sub, roles,
   exp; tolak algoritma selain HS256), GenerateRefreshToken (32 byte acak,
   base64url) dan HashToken (SHA-256 hex).
4. platform/clock: interface Clock {Now() time.Time; Today() time.Time} dengan
   implementasi nyata (Today dihitung pada zona APP_TIMEZONE, mengembalikan
   tanggal tanpa jam) dan implementasi tiruan untuk test.
5. platform/storage: interface Storage dan LocalStorage sesuai ARCHITECTURE.md
   bagian 9 (tolak path traversal setelah filepath.Clean, buat subfolder otomatis).
6. middleware: RequestID (baca/buat X-Request-ID, kembalikan di header),
   Logger (slog terstruktur: request_id, method, path, status, durasi),
   Recovery (panic menjadi 500 terstandar), CORS (dari CORS_ALLOWED_ORIGINS).
7. Hubungkan middleware di cmd/api (atau internal/app) dan pindahkan pembuatan
   router ke internal/app.

Batasan: belum ada fitur bisnis dan belum ada autentikasi.

Kriteria penerimaan:
- Unit test lulus untuk: Pagination (batas default/maksimal), JWT (valid,
  kedaluwarsa, tanda tangan salah, algoritma none), HashToken deterministik,
  LocalStorage (simpan, buka, hapus, tolak "../"), pemetaan error pg, Clock.Today
  pada zona Asia/Makassar di sekitar pergantian hari UTC.
- Panic buatan di sebuah route menghasilkan JSON error 500 terstandar dan server
  tetap hidup.
- Header X-Request-ID muncul di setiap response.
```

---

## Tahap 3: Autentikasi dan RBAC

```
Kerjakan TAHAP 3: Autentikasi dan RBAC (paket auth, middleware auth/role, seeder).

Tujuan: login dengan akun sendiri, token, pembatasan berdasarkan role, dan akun
admin pertama.

Tugas:
1. Paket auth (model, dto, repository, service, handler, routes) dengan endpoint
   PRD bagian 5.1:
   - POST /auth/login {username, password}. Cari user by username; tolak bila
     tidak ada, password salah, atau is_active=false dengan pesan IDENTIK
     ("username atau password salah"). Sukses mengembalikan access token,
     refresh token, expires_in, dan ringkasan user + roles.
   - POST /auth/refresh {refresh_token}: verifikasi hash dan expired_at, rotasi
     token (hapus lama, buat baru) dalam satu transaksi.
   - POST /auth/logout {refresh_token}: hapus baris token.
   - GET /auth/me: user_id, username, email, roles, serta ID profil yang
     relevan (mahasiswa/dosen/pegawai/admin/pembimbing_lapangan) bila ada.
   - PUT /auth/password {password_lama, password_baru}: minimal 8 karakter;
     setelah berhasil, hapus semua refresh token milik user.
2. Role diambil dari user_roles JOIN roles. Klaim JWT memuat daftar nama role.
3. Middleware Auth (verifikasi Bearer token, muat Actor sesuai ARCHITECTURE.md
   bagian 6, 401 terstandar bila gagal) dan RequireRoles(...string) (403 bila
   tidak punya salah satu role). Sediakan helper auth.ActorFrom(c).
4. Paket user: hanya ProfileResolver dulu, dengan method GetMahasiswaByUserID,
   GetDosenByUserID, GetPegawaiByUserID, GetAdminByUserID,
   GetPembimbingLapanganByUserID (mengembalikan nil bila tidak ada).
5. Pencatatan log_aktivitas: login berhasil, login gagal (HANYA bila user
   dikenal, sesuai RULES.md bagian 7 nomor 3), logout. Isi nama_user, ip_user,
   agent_browser. Login gagal untuk username tak dikenal cukup dicatat ke slog
   tanpa menyertakan password.
6. cmd/seed: membuat akun admin pertama dari SEED_ADMIN_USERNAME, SEED_ADMIN_EMAIL,
   SEED_ADMIN_PASSWORD (baris users, baris admin, dan user_roles role admin)
   dalam satu transaksi; idempoten (aman dijalankan ulang). Aktifkan target
   "make seed".
7. Pasang route: /health dan /auth/login, /auth/refresh publik; sisanya di
   belakang middleware Auth.

Batasan: belum ada pembuatan akun lewat API (itu Tahap 4). Jangan menyimpan
refresh token mentah. Jangan mencatat password atau token ke log.

Kriteria penerimaan:
- Unit test service: login sukses, password salah, user nonaktif, pesan error
  identik, refresh merotasi token, refresh dengan token lama ditolak, token
  kedaluwarsa ditolak.
- Integrasi: seed admin, login, GET /auth/me berisi role admin, akses endpoint
  terproteksi tanpa token menghasilkan 401, dengan role salah menghasilkan 403.
- Menjalankan seed dua kali tidak membuat data ganda.
- Tabel tokens hanya berisi hash (bukan token mentah).
```

**Verifikasi manual**: `make seed`, `POST /auth/login`, `GET /auth/me` dengan header `Authorization: Bearer <token>`.

---

## Tahap 4: Akun dan Master Data

```
Kerjakan TAHAP 4: Manajemen akun dan master data (hanya admin yang menulis).

Tujuan: admin dapat membuat semua jenis akun dan mengisi data referensi.

Tugas:
1. Paket user (lanjutan): endpoint PRD bagian 5.2 untuk akun:
   - POST/GET(list, detail)/PUT untuk /admin/mahasiswa, /admin/dosen,
     /admin/pegawai, /admin/pembimbing-lapangan, /admin/admins.
     Pembuatan = satu transaksi: insert ke tabel users (password di-hash), insert
     profil, insert user_roles sesuai jenis akun (mahasiswa, dosen, pegawai,
     admin, pembimbing_lapangan).
   - GET /admin/users (filter role, q untuk username/email/nama, pagination),
     GET /admin/users/:id, PATCH /admin/users/:id/status (is_active),
     POST /admin/users/:id/reset-password.
   - Pembimbing lapangan: id_user pada tabel pembimbing_lapangan WAJIB terisi
     lewat endpoint ini (asumsi A8), dan jenis harus sesuai jenis_program tempat
     (mentor di tempat magang, guru_pamong di tempat asistensi_mengajar,
     kepala_desa di tempat kkn).
   - Validasi: username, email, nim, nip unik (409 dengan pesan jelas);
     id_prodi harus ada.
2. Paket akademik: CRUD fakultas, jurusan, program_studi (admin menulis; GET
   terbuka untuk user login), validasi induk ada.
3. Paket tempat: CRUD tempat_program (DELETE = is_aktif false; GET
   /tempat-program terbuka untuk user login dengan filter jenis_program dan
   is_aktif) dan GET /tempat-program/:id/pembimbing-lapangan.
4. Konstanta enum jenis_program (magang, asistensi_mengajar, kkn) dan jenis
   pembimbing lapangan didefinisikan sekali di paket yang sesuai.
5. Semua route admin berada di belakang RequireRoles("admin").

Batasan: tidak ada hapus fisik akun. Respons tidak boleh memuat password_hash.

Kriteria penerimaan:
- Unit test: validasi kesesuaian jenis pembimbing lapangan dengan jenis tempat;
  pembuatan akun membuat role yang benar.
- Integrasi: admin membuat satu mahasiswa, dua dosen, satu pegawai, satu
  pembimbing lapangan per jenis; akun baru bisa login; mahasiswa memanggil
  endpoint admin mendapat 403; username ganda mendapat 409.
- Transaksi atomik: bila insert profil gagal, baris users tidak tertinggal.
```

---

## Tahap 5: Periode

```
Kerjakan TAHAP 5: Periode program.

Tugas:
1. Paket periode dengan endpoint PRD bagian 5.3.
2. Aturan:
   - Field: nama, jenis_program, tahun_semester, tanggal_mulai, tanggal_selesai,
     tanggal_buka_daftar, tanggal_tutup_daftar (opsional), status.
   - Validasi tanggal: tanggal_selesai >= tanggal_mulai;
     tanggal_tutup_daftar >= tanggal_buka_daftar bila keduanya ada.
   - Status satu arah: draft -> aktif -> selesai. Update field hanya saat draft.
   - dibuat_oleh hanya diisi bila pembuat punya profil pegawai; selain itu NULL
     (RULES.md bagian 7 nomor 2).
   - POST /admin/periode/:id/selesaikan: dalam satu transaksi ubah periode jadi
     selesai, peserta_program berstatus berjalan menjadi selesai, dan peserta
     berstatus diajukan menjadi ditolak dengan alasan_tolak otomatis (asumsi
     A12). Karena tabel peserta_program sudah ada namun modul pendaftaran belum
     dibuat, implementasikan query ini di repository periode dengan fungsi
     terpisah yang mudah dipindahkan nanti.
   - GET /periode: admin/pegawai melihat semua dengan filter jenis_program dan
     status; mahasiswa hanya melihat yang status aktif.
   - Catat ke log_aktivitas saat periode diaktifkan dan diselesaikan.
3. Route tulis di belakang RequireRoles("admin"); route baca untuk user login.

Kriteria penerimaan:
- Unit test: transisi status valid dan tidak valid, validasi tanggal, update saat
  bukan draft ditolak (422).
- Integrasi: alur draft, aktif, selesai; mahasiswa hanya melihat periode aktif;
  menyelesaikan periode tidak mengubah periode lain.
```

---

## Tahap 6: Pendaftaran Inti

```
Kerjakan TAHAP 6: Pendaftaran inti (paket pendaftaran).

Tujuan: mahasiswa dapat mendaftar, dan akses data per relasi berfungsi.

Tugas:
1. Port KRSValidator dan EligibilityChecker (ARCHITECTURE.md bagian 7) beserta
   implementasi stub yang selalu true, diberi komentar "// ASUMSI: A1" dan
   "// ASUMSI: A2" serta "// TODO(mentor)". Suntikkan lewat internal/app.
2. Endpoint:
   - POST /pendaftaran {id_periode} (mahasiswa). Aturan BR-01, BR-02, BR-03
     (data ganda menghasilkan 409). Jalankan KRSValidator lalu
     EligibilityChecker; jika gagal, simpan baris dengan status ditolak dan
     alasan_tolak, dan balas 201 dengan statusnya. Jika lolos, status diajukan.
   - POST /pendaftaran/:id/ajukan-ulang (mahasiswa pemilik; hanya dari status
     ditolak; periode masih buka; validasi dijalankan ulang).
   - GET /pendaftaran dengan scope per role: mahasiswa (miliknya), dosen (yang
     ia bimbing atau uji), pembimbing lapangan (yang ia dampingi), admin dan
     pegawai (semua, dengan filter id_periode, jenis_program, status, q).
     Pagination wajib.
   - GET /pendaftaran/:id dengan detail lengkap (periode, mahasiswa, tempat,
     dosen, pembimbing lapangan, status, kesiapan: daftar kekurangan).
3. pendaftaran.AccessService.Resolve sesuai ARCHITECTURE.md bagian 6 (relasi
   admin, pegawai, mahasiswa, dosen_pembimbing, dosen_penguji,
   pembimbing_lapangan, anggota_kelompok). Tanpa relasi menghasilkan 404.
4. pendaftaran.ReadinessChecker (fungsi murni) sesuai PRD bagian 4.3 dan
   fungsi pembantu yang mencoba transisi diajukan menjadi berjalan dalam
   transaksi (dengan SELECT ... FOR UPDATE). Belum dipanggil dari tempat lain
   selain GET detail (untuk menampilkan kekurangan); tahap berikutnya yang
   memanggilnya setelah perubahan.
5. Konstanta status (diajukan, ditolak, berjalan, selesai) dan helper
   "sudahDinilai" (status selesai dan nilai_akhir tidak NULL).

Kriteria penerimaan:
- Unit test: BR-01 (periode draft/selesai, di luar tanggal daftar), BR-02,
  stub KRS yang dibuat mengembalikan false menghasilkan status ditolak,
  ReadinessChecker untuk tiga jenis program (tabel kasus lengkap),
  AccessService untuk setiap relasi dan kasus tanpa relasi.
- Integrasi: mahasiswa A mendaftar; mahasiswa B tidak bisa melihat pendaftaran A
  (404); pendaftaran ganda menghasilkan 409; admin melihat semua.
```

---

## Tahap 7: Dokumen

```
Kerjakan TAHAP 7: Dokumen (penyimpanan lokal).

Tugas:
1. Paket dokumen dengan endpoint PRD bagian 5.5.
2. Unggah (multipart: jenis_dokumen, file):
   - jenis_dokumen salah satu: mou, proposal, proker, laporan_akhir, lainnya.
   - Validasi: ukuran <= MAX_UPLOAD_MB (http.MaxBytesReader), ekstensi
     (pdf, docx, png, jpg, jpeg) dan isi file via http.DetectContentType harus
     konsisten; tolak selain itu dengan 422.
   - Simpan lewat Storage dengan path pendaftaran/<id_peserta>/<uuid>.<ext>
     (atau kkn/<id_kelompok>/<uuid>.<ext>). dokumen.file_url diisi path RELATIF.
     nama_file = nama asli (disanitasi), ukuran_file, mime_type (hasil deteksi),
     diunggah_oleh = user id.
   - Jika insert gagal, hapus file yang sudah tersimpan.
   - Hak unggah: mahasiswa pemilik (dokumen peserta); anggota kelompok
     (dokumen kelompok, memakai Resolve relasi anggota_kelompok); admin boleh
     keduanya. Hanya saat status pendaftaran diajukan atau berjalan.
   - Aturan jenis: mou dan proposal hanya untuk peserta berjenis magang;
     proker boleh milik peserta (AM) atau kelompok (KKN); jenis lain bebas.
3. Daftar dan unduh: GET /pendaftaran/:id/dokumen, GET /kkn/kelompok/:id/dokumen
   (anggota kelompok, dosen pembimbing, pembimbing lapangan terkait, admin,
   pegawai), GET /dokumen/:id/download (periksa akses lewat pendaftaran atau
   kelompok pemilik dokumen; stream file; Content-Disposition dengan nama
   disanitasi). DELETE /dokumen/:id: pengunggah saja, hanya saat status
   pendaftaran diajukan; hapus baris lalu file.
4. Dokumen kelompok KKN: karena modul kkn belum ada, endpoint kelompok boleh
   dibuat sekarang tetapi hanya diuji penuh pada Tahap 10. Gunakan interface
   kecil untuk memeriksa keanggotaan kelompok.
5. Setelah unggah/hapus dokumen pada pendaftaran, panggil percobaan transisi
   kesiapan (dari Tahap 6) bila status diajukan.

Batasan: jangan menyajikan folder uploads sebagai static file. Jangan
mengembalikan path file ke klien.

Kriteria penerimaan:
- Unit test: validasi tipe/ukuran, penolakan ekstensi tidak cocok dengan isi,
  aturan jenis dokumen per program, path penyimpanan tidak memuat nama asli.
- Integrasi: unggah PDF valid berhasil; file 6 MB ditolak; .exe yang
  di-rename .pdf ditolak; mahasiswa lain tidak bisa mengunduh (404); admin bisa.
- File muncul di folder uploads dengan nama UUID; setelah container dibuat
  ulang, file tetap ada (volume).
```

---

## Tahap 8: Cabang Magang

```
Kerjakan TAHAP 8: Cabang Magang.

Tugas:
1. PUT /pendaftaran/:id/penempatan-magang {id_tempat_program,
   id_pembimbing_lapangan} (mahasiswa pemilik, hanya peserta magang, status
   diajukan). Validasi BR-04: tempat berjenis magang dan is_aktif; mentor
   berjenis mentor dan milik tempat tersebut. Pengubahan ulang diperbolehkan
   selama status masih diajukan.
2. PUT /pendaftaran/:id/dosen-pembimbing {id_dosen} (pegawai untuk magang; admin
   untuk semua jenis). Validasi: dosen ada dan id_dosen <> id_dosen_penguji
   (BR-07). Catat ke log_aktivitas.
3. Setelah setiap perubahan di atas, jalankan percobaan transisi kesiapan
   (diajukan menjadi berjalan) dalam transaksi yang sama.
4. Bila pendaftaran berstatus berjalan, perubahan penempatan ditolak (422).

Kriteria penerimaan:
- Unit test: BR-04 (tempat jenis lain, tempat nonaktif, mentor milik tempat
  lain), BR-07, pegawai tidak boleh menetapkan dosen pada pendaftaran AM,
  mahasiswa tidak boleh memanggil endpoint dosen-pembimbing.
- Integrasi skenario Magang sampai berjalan: daftar, pilih tempat dan mentor,
  tetapkan dosen, unggah MoU saja (status tetap diajukan, detail menampilkan
  kekurangan "proposal"), unggah proposal (status menjadi berjalan).
```

---

## Tahap 9: Cabang Asistensi Mengajar

```
Kerjakan TAHAP 9: Cabang Asistensi Mengajar (AM).

Tugas:
1. PUT /admin/pendaftaran/:id/penempatan {id_tempat_program,
   id_pembimbing_lapangan} (admin). Berlaku untuk jenis asistensi_mengajar:
   validasi BR-05 (tempat berjenis asistensi_mengajar dan aktif; guru_pamong
   milik sekolah tersebut). Untuk jenis lain, balas 422 dengan pesan jelas
   (magang dipilih mahasiswa, KKN lewat kelompok).
2. Penetapan dosen pembimbing AM memakai endpoint Tahap 8 (admin). Penetapan
   dosen penguji: PUT /admin/pendaftaran/:id/dosen-penguji {id_dosen} untuk
   semua jenis program, validasi BR-07.
3. Proker AM diunggah mahasiswa sebagai dokumen jenis proker milik peserta
   (Tahap 7). Tidak ada tabel baru untuk "input data AM" (asumsi A9).
4. Setelah setiap perubahan, jalankan percobaan transisi kesiapan.

Kriteria penerimaan:
- Unit test: BR-05, penguji sama dengan pembimbing ditolak, endpoint
  penempatan untuk peserta magang ditolak.
- Integrasi skenario AM sampai berjalan: daftar, admin memetakan sekolah dan
  guru pamong, admin tetapkan dosen pembimbing, mahasiswa unggah proker,
  status menjadi berjalan.
```

---

## Tahap 10: Cabang KKN

```
Kerjakan TAHAP 10: Cabang KKN (paket kkn).

Tugas:
1. Endpoint PRD bagian 5.4 untuk kelompok:
   - POST /admin/kkn/kelompok {id_periode, id_tempat_program, nama_kelompok}
     (periode harus berjenis kkn; tempat berjenis kkn dan aktif).
   - GET /admin/kkn/kelompok (filter id_periode), GET /admin/kkn/kelompok/:id
     (beserta anggota dan ketua).
   - POST /admin/kkn/kelompok/:id/anggota {id_peserta_program: [..], id_ketua}:
     peserta harus berjenis kkn, periode sama, status diajukan atau berjalan,
     belum masuk kelompok lain. Dalam satu transaksi: set id_kelompok_kkn dan
     id_tempat_program peserta mengikuti kelompok; tetapkan is_ketua (hanya satu
     per kelompok, unique index v2 sebagai pengaman, tangani 409 dengan baik).
   - DELETE /admin/kkn/kelompok/:id/anggota/:pid: kosongkan id_kelompok_kkn dan
     is_ketua peserta (hanya saat status diajukan).
2. Sinkronisasi (RULES.md bagian 7 nomor 5):
   - PUT /admin/kkn/kelompok/:id/dosen-pembimbing {id_dosen}: dalam satu
     transaksi update kelompok_kkn.id_dosen_pembimbing DAN
     peserta_program.id_dosen_pembimbing seluruh anggota. Validasi BR-07 untuk
     setiap anggota yang sudah punya penguji.
   - PUT /admin/kkn/kelompok/:id/pembimbing-lapangan {id_pembimbing_lapangan}:
     sama, validasi BR-06 (berjenis kepala_desa dan milik tempat kelompok).
   - Anggota yang ditambahkan SESUDAH penetapan mewarisi dosen pembimbing dan
     pembimbing lapangan kelompok.
   - Kolom kelompok_kkn.proker_url tidak dipakai.
3. Dokumen kelompok: lengkapi dan uji endpoint dokumen kelompok dari Tahap 7
   (anggota kelompok boleh mengunggah proker). Kesiapan KKN terpenuhi bila
   proker milik kelompok atau milik peserta ada; setelah unggah atau perubahan
   anggota, jalankan percobaan transisi kesiapan untuk SELURUH anggota kelompok.
4. Resolve relasi anggota_kelompok sudah ada; pastikan dosen pembimbing dan
   pembimbing lapangan kelompok bisa melihat kelompok.

Kriteria penerimaan:
- Unit test: BR-06 (tempat bukan KKN, peserta jenis lain, periode berbeda,
  peserta sudah di kelompok lain), ketua kedua ditolak, pewarisan anggota baru.
- Integrasi skenario KKN sampai berjalan dengan dua mahasiswa: kelompok dan
  anggota, tetapkan ketua, tetapkan dosen pembimbing (kedua anggota ikut
  ter-update), unggah proker kelompok, kedua anggota berstatus berjalan.
- Atomik: bila update salah satu anggota gagal, tidak ada perubahan yang
  tertinggal.
```

---

## Tahap 11: Jurnal, Logbook, dan Penyelesaian Periode

```
Kerjakan TAHAP 11: Jurnal harian, logbook bimbingan, dan penyelesaian periode.

Tugas:
1. Paket jurnal (endpoint PRD bagian 5.6):
   - POST /pendaftaran/:id/jurnal (mahasiswa pemilik atau, untuk KKN, peserta itu
     sendiri). Field: tanggal, judul, isi, kendala, solusi, status (hanya draft
     atau disubmit). Aturan BR-08 dan BR-09 (duplikat tanggal menghasilkan 409).
   - PUT /pendaftaran/:id/jurnal/:jid: hanya pemilik, hanya saat status draft.
   - GET /pendaftaran/:id/jurnal: pemilik, dosen pembimbing, pembimbing
     lapangan, admin, pegawai; filter rentang tanggal; pagination.
2. Paket logbook:
   - POST /pendaftaran/:id/logbook (hanya dosen pembimbing pendaftaran itu).
     Field: tanggal, topik_bimbingan, catatan, pertemuan_berikutnya. Aturan BR-08.
     id_dosen diambil dari profil dosen pemanggil, bukan dari body.
   - PUT /pendaftaran/:id/logbook/:lid: hanya dosen yang membuatnya.
   - GET: pemilik, dosen pembimbing, admin, pegawai.
3. Penyelesaian periode (POST /admin/periode/:id/selesaikan dari Tahap 5): pastikan
   efeknya sudah sesuai dan tambahkan test integrasi bahwa setelah periode
   selesai, jurnal dan logbook baru ditolak (422) dan status peserta menjadi
   selesai.
4. Pengujian waktu memakai clock tiruan (tanggal jurnal di masa depan ditolak,
   tanggal di luar rentang periode ditolak).

Batasan: review jurnal (disetujui atau ditolak) belum dikerjakan; ada di
Tahap 14. Kolom reviewed_by dan catatan_reviewer dibiarkan kosong.

Kriteria penerimaan:
- Unit test: BR-08 (status bukan berjalan, tanggal masa depan, di luar periode),
  BR-09, edit jurnal bukan draft ditolak, dosen lain tidak bisa mengisi logbook
  (404).
- Integrasi: dari status berjalan, isi jurnal dan logbook; selesaikan periode;
  isi jurnal lagi harus 422.
```

---

## Tahap 12: Penilaian dan Nilai Akhir

```
Kerjakan TAHAP 12: Penilaian dan nilai akhir (paket penilaian).

Tugas:
1. CRUD /admin/instrumen-penilaian (jenis_program, nama_aspek, bobot,
   role_penilai dengan nilai dosen_pembimbing, dosen_penguji,
   pembimbing_lapangan) dan GET /admin/instrumen-penilaian/ringkasan yang
   menampilkan total bobot per (jenis_program, role_penilai) dan total
   keseluruhan per jenis_program. Jangan memaksa total 100 (aturan bobot belum
   diketahui, asumsi A4); cukup tampilkan sebagai informasi.
2. ScoreCalculator (interface + implementasi default) sesuai ARCHITECTURE.md
   bagian 8: nilai_akhir = ROUND(SUM(nilai x bobot) / SUM(bobot), 2). Konversi
   huruf sementara A>=85, B>=70, C>=55, D>=40, selainnya E (asumsi A5). Beri
   "// ASUMSI: A4/A5" dan "// TODO(mentor)". sks_diakui tidak diisi (A6).
3. Endpoint:
   - GET /pendaftaran/:id/form-penilaian: instrumen yang wajib diisi pemanggil
     (berdasarkan relasi dosen_pembimbing, dosen_penguji, atau
     pembimbing_lapangan), beserta nilai yang sudah ada.
   - POST /pendaftaran/:id/penilaian {items:[{id_instrumen, nilai, catatan}]}
     mengikuti langkah di ARCHITECTURE.md bagian 8 (transaksi, FOR UPDATE,
     BR-10, BR-11, BR-15, upsert). Bila seluruh instrumen sudah terisi, hitung
     dan simpan nilai_akhir dan grade_huruf.
   - GET /pendaftaran/:id/nilai: rincian per instrumen dan nilai akhir; mahasiswa
     pemilik hanya melihat rincian setelah nilai_akhir terisi.
   - POST /admin/pendaftaran/:id/hitung-nilai: hitung ulang manual; bila
     instrumen untuk jenis program kosong atau penilaian belum lengkap, balas
     422 dengan kode error yang jelas (misalnya INSTRUMEN_BELUM_DIATUR dan
     PENILAIAN_BELUM_LENGKAP). Catat ke log_aktivitas.
4. Penilai tanpa akun tidak mungkin ada pada alur ini (asumsi A8).

Kriteria penerimaan:
- Unit test ScoreCalculator (tabel kasus: satu instrumen, bobot tak ber-total 100,
  pembulatan, batas konversi huruf), BR-10 (status bukan selesai, nilai di luar
  0 sampai 100, instrumen jenis atau role salah), penilai bukan relasi sah
  ditolak, nilai terkunci setelah terhitung.
- Integrasi: tiga penilai mengisi urutan acak; nilai_akhir terisi tepat setelah
  penilai terakhir; pengiriman paralel dua penilai terakhir tidak menghasilkan
  hitung ganda.
```

---

## Tahap 13: Swagger, Test End-to-End, dan Finalisasi

```
Kerjakan TAHAP 13: Swagger, test end-to-end, dan finalisasi MVP.

Tugas:
1. Lengkapi anotasi Swagger seluruh endpoint (ringkasan, parameter, skema
   request/response, kode error, skema keamanan Bearer). Pasang gin-swagger di
   /swagger/*any (hanya saat APP_ENV=development). Tambahkan target "make swag".
2. Tulis tiga integration test end-to-end sesuai PRD bagian 11 (Magang, AM, KKN)
   yang memanggil router sungguhan lewat httptest pada database splk_test,
   dari pembuatan akun sampai nilai_akhir terisi. Sediakan helper pembuat data
   uji agar test tetap ringkas.
3. Tinjau konsistensi: format error di semua endpoint, semua endpoint selain
   yang publik berada di belakang Auth, pagination di semua daftar, tidak ada
   SELECT *, tidak ada password_hash atau token_hash yang bocor di response.
4. Hardening: rate limit login sederhana (in-memory, misalnya 5 percobaan per
   menit per IP; kembalikan 429), batas ukuran body JSON, header keamanan dasar.
5. Perbarui README: setup dari nol, daftar variabel environment, cara menjalankan
   migrasi/seed/test, cara mencoba alur dengan Swagger atau koleksi Postman, dan
   daftar asumsi serta pertanyaan terbuka untuk mentor (salin dari PRD bagian 8
   dan 10).
6. Pastikan Dockerfile dan docker-compose bekerja dari clone bersih.

Kriteria penerimaan:
- go test ./... lulus, termasuk tiga test E2E.
- Swagger terbuka dan seluruh endpoint tercantum.
- Clone bersih: ikuti README, "docker compose up --build", seed, login, dan
  jalankan satu alur dari Swagger tanpa error.
```

---

## Tahap 14 (Opsional): Fitur Tambahan

Kerjakan hanya setelah mentor mengonfirmasi kebutuhannya.

```
Kerjakan TAHAP 14: fitur tambahan. Aku akan menyebut sub-tugas yang diinginkan
(pilih dari daftar). Untuk tiap sub-tugas, ikuti pola modul yang sama, tulis
test, dan jangan mengubah migrasi lama (bila butuh perubahan skema, buat migrasi
baru lengkap dengan file down).

14a. Review jurnal: dosen pembimbing dan pembimbing lapangan terkait dapat
     menyetujui atau menolak jurnal berstatus disubmit (isi status, catatan_reviewer,
     reviewed_by, reviewed_at). Jurnal ditolak boleh diedit ulang oleh mahasiswa.
14b. Verifikasi dokumen: admin atau pegawai menandai dokumen terverifikasi
     (is_verified, verified_by, verified_at).
14c. Pengumuman: CRUD admin; GET untuk user login difilter target_role (NULL
     berarti untuk semua) dengan pagination.
14d. Surat perjalanan dinas: pegawai membuat surat untuk dosen dan peserta
     (dibuat_oleh mengacu pegawai.id); unggah dan unduh file memakai Storage.
14e. Perbaikan skema lanjutan (PRD bagian 9, hanya setelah disetujui mentor):
     migrasi baru untuk CHECK status dan jenis_program, composite FK jenis
     program, CHECK dosen pembimbing berbeda penguji, log_aktivitas.id_user
     nullable, tokens.revoked_at, tabel jabatan Kejur per jurusan. Sesuaikan
     kode dan test yang terdampak.
```

---

## Prompt Perbaikan (dipakai bila hasil tahap bermasalah)

```
Hasil Tahap <N> bermasalah. Berikut gejalanya:
<tempel pesan error / perilaku yang salah / output test>

Langkah:
1. Reproduksi masalah dan jelaskan akar penyebabnya sebelum mengubah kode.
2. Perbaiki dengan perubahan sekecil mungkin yang tetap mematuhi RULES.md.
3. Tambahkan test yang gagal sebelum perbaikan dan lulus sesudahnya.
4. Jalankan gofmt, go vet, go build, dan go test ./... lalu laporkan hasilnya.
Jangan mengerjakan hal di luar perbaikan ini.
```

## Prompt Tinjauan (dipakai sebelum commit tiap tahap)

```
Tinjau diff Tahap <N> terhadap RULES.md, ARCHITECTURE.md, dan PRD.md.
Periksa khususnya: pelanggaran layering, SQL tanpa parameter, rahasia yang bocor
di response atau log, aturan bisnis (BR-nn) yang terlewat, transaksi yang
seharusnya ada, error yang diabaikan, dan test yang belum mencakup kasus gagal.
Laporkan temuan berurutan dari yang paling serius, lengkap dengan file dan baris,
tanpa mengubah kode.
```
