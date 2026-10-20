// Package pendaftaran menyediakan logika use-case dan aturan bisnis pendaftaran program SPLK.
package pendaftaran

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/platform/httpx"
)

// ActivityLogger mendefinisikan antarmuka pencatatan log aktivitas audit.
type ActivityLogger interface {
	Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error
}

// Service mendefinisikan seluruh use-case untuk modul pendaftaran.
type Service interface {
	Daftar(ctx context.Context, actor middleware.Actor, req CreatePendaftaranRequest) (*PendaftaranDetail, error)
	AjukanUlang(ctx context.Context, actor middleware.Actor, idPeserta int) (*PendaftaranDetail, error)
	GetDetail(ctx context.Context, actor middleware.Actor, idPeserta int) (*PendaftaranDetail, error)
	List(ctx context.Context, actor middleware.Actor, filter PendaftaranFilter, p httpx.Pagination) ([]PendaftaranListItem, httpx.PaginationMeta, error)
	PenempatanMagang(ctx context.Context, actor middleware.Actor, idPeserta int, req PenempatanMagangRequest) (*PendaftaranDetail, error)
	AdminPenempatan(ctx context.Context, actor middleware.Actor, idPeserta int, req AdminPenempatanRequest) (*PendaftaranDetail, error)
	SetDosenPembimbing(ctx context.Context, actor middleware.Actor, idPeserta int, req SetDosenPembimbingRequest, ip, userAgent string) (*PendaftaranDetail, error)
	SetDosenPenguji(ctx context.Context, actor middleware.Actor, idPeserta int, req SetDosenPengujiRequest, ip, userAgent string) (*PendaftaranDetail, error)
}

type pendaftaranService struct {
	repo         Repository
	pool         *pgxpool.Pool
	clock        clock.Clock
	krsValidator KRSValidator
	eligChecker  EligibilityChecker
	access       AccessService
	logger       ActivityLogger
}

// NewService membuat instance Service baru untuk modul pendaftaran.
func NewService(
	repo Repository,
	pool *pgxpool.Pool,
	clk clock.Clock,
	krsVal KRSValidator,
	eligCheck EligibilityChecker,
	access AccessService,
	logger ActivityLogger,
) Service {
	return &pendaftaranService{
		repo:         repo,
		pool:         pool,
		clock:        clk,
		krsValidator: krsVal,
		eligChecker:  eligCheck,
		access:       access,
		logger:       logger,
	}
}

// Daftar memproses pendaftaran baru mahasiswa pada suatu periode program.
// Menegakkan aturan BR-01, BR-02, BR-03 dan validasi eksternal (KRS & eligibilitas).
func (s *pendaftaranService) Daftar(ctx context.Context, actor middleware.Actor, req CreatePendaftaranRequest) (*PendaftaranDetail, error) {
	// Pastikan pemanggil adalah mahasiswa
	if !actor.HasRole("mahasiswa") {
		return nil, httpx.Forbidden("AKSES_DITOLAK", "hanya akun mahasiswa yang dapat melakukan pendaftaran program")
	}

	mhs, err := s.repo.FindMahasiswaByUserID(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("ambil profil mahasiswa: %w", err)
	}
	if mhs == nil {
		return nil, httpx.Forbidden("PROFIL_TIDAK_DITEMUKAN", "profil mahasiswa untuk pengguna ini tidak ditemukan")
	}

	// Ambil data periode
	per, buka, tutup, err := s.repo.FindPeriodeByID(ctx, s.pool, req.IDPeriode)
	if err != nil {
		return nil, fmt.Errorf("ambil data periode: %w", err)
	}
	if per == nil {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}

	// Aturan BR-01: Pendaftaran hanya jika periode.status = 'aktif'
	if per.Status != "aktif" {
		return nil, httpx.Unprocessable("PERIODE_TIDAK_AKTIF", "pendaftaran hanya dapat dilakukan pada periode yang berstatus aktif")
	}

	// Aturan BR-01: Tanggal hari ini (zona waktu yang dikonfigurasi) harus berada dalam tanggal_buka_daftar sampai tanggal_tutup_daftar
	today := s.clock.Today()
	if buka != nil {
		bukaDate := time.Date(buka.Year(), buka.Month(), buka.Day(), 0, 0, 0, 0, today.Location())
		if today.Before(bukaDate) {
			return nil, httpx.Unprocessable("PENDAFTARAN_BELUM_DIBUKA", "pendaftaran untuk periode ini belum dibuka")
		}
	}
	if tutup != nil {
		tutupDate := time.Date(tutup.Year(), tutup.Month(), tutup.Day(), 0, 0, 0, 0, today.Location())
		if today.After(tutupDate) {
			return nil, httpx.Unprocessable("PENDAFTARAN_SUDAH_DITUTUP", "pendaftaran untuk periode ini sudah ditutup")
		}
	}

	// Aturan BR-02: peserta_program.jenis_program selalu sama dengan periode.jenis_program
	jenisProgram := per.JenisProgram

	// Aturan BR-03: Satu mahasiswa hanya punya satu pendaftaran per (periode, jenis program)
	existing, err := s.repo.FindPesertaByMahasiswaAndPeriode(ctx, s.pool, mhs.ID, req.IDPeriode, jenisProgram)
	if err != nil {
		return nil, fmt.Errorf("cek pendaftaran ganda: %w", err)
	}
	if existing != nil {
		return nil, httpx.Conflict("PENDAFTARAN_GANDA", "anda sudah terdaftar pada periode dan jenis program ini")
	}

	// Jalankan validasi KRS, kemudian validasi eligibilitas program
	status := StatusDiajukan
	var alasanTolak *string

	okKRS, alasanKRS, err := s.krsValidator.Validate(ctx, *mhs, *per)
	if err != nil {
		return nil, fmt.Errorf("validasi KRS gagal: %w", err)
	}
	if !okKRS {
		status = StatusDitolak
		alasan := "Validasi KRS tidak terpenuhi: " + alasanKRS
		alasanTolak = &alasan
	} else {
		okElig, alasanElig, err := s.eligChecker.Check(ctx, *mhs, *per)
		if err != nil {
			return nil, fmt.Errorf("validasi eligibilitas gagal: %w", err)
		}
		if !okElig {
			status = StatusDitolak
			alasan := "Syarat eligibilitas program tidak terpenuhi: " + alasanElig
			alasanTolak = &alasan
		}
	}

	peserta := &PesertaProgram{
		IDMahasiswa:  mhs.ID,
		IDPeriode:    req.IDPeriode,
		JenisProgram: jenisProgram,
		Status:       status,
		AlasanTolak:  alasanTolak,
	}

	if err := s.repo.CreatePeserta(ctx, s.pool, peserta); err != nil {
		return nil, database.MapError(err)
	}

	// Kembalikan detail pendaftaran yang baru dibuat
	return s.GetDetail(ctx, actor, peserta.ID)
}

// AjukanUlang memproses pengajuan ulang pendaftaran yang sebelumnya ditolak (PRD 5.4, BR-13).
func (s *pendaftaranService) AjukanUlang(ctx context.Context, actor middleware.Actor, idPeserta int) (*PendaftaranDetail, error) {
	if !actor.HasRole("mahasiswa") {
		return nil, httpx.Forbidden("AKSES_DITOLAK", "hanya mahasiswa yang dapat mengajukan ulang pendaftaran")
	}

	mhs, err := s.repo.FindMahasiswaByUserID(ctx, s.pool, actor.UserID)
	if err != nil || mhs == nil {
		return nil, httpx.Forbidden("PROFIL_TIDAK_DITEMUKAN", "profil mahasiswa tidak ditemukan")
	}

	peserta, err := s.repo.FindPesertaByID(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil peserta: %w", err)
	}
	// Otorisasi: hanya pemilik pendaftaran yang berhak
	if peserta == nil || peserta.IDMahasiswa != mhs.ID {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// Hanya boleh dari status ditolak
	if peserta.Status != StatusDitolak {
		return nil, httpx.Unprocessable("STATUS_TIDAK_VALID", "hanya pendaftaran berstatus 'ditolak' yang dapat diajukan ulang")
	}

	// Periksa apakah periode masih buka
	per, buka, tutup, err := s.repo.FindPeriodeByID(ctx, s.pool, peserta.IDPeriode)
	if err != nil {
		return nil, fmt.Errorf("ambil data periode: %w", err)
	}
	if per == nil || per.Status != "aktif" {
		return nil, httpx.Unprocessable("PERIODE_PENDAFTARAN_TUTUP", "periode pendaftaran sudah tidak aktif")
	}

	today := s.clock.Today()
	if buka != nil {
		bukaDate := time.Date(buka.Year(), buka.Month(), buka.Day(), 0, 0, 0, 0, today.Location())
		if today.Before(bukaDate) {
			return nil, httpx.Unprocessable("PENDAFTARAN_BELUM_DIBUKA", "periode pendaftaran belum dibuka")
		}
	}
	if tutup != nil {
		tutupDate := time.Date(tutup.Year(), tutup.Month(), tutup.Day(), 0, 0, 0, 0, today.Location())
		if today.After(tutupDate) {
			return nil, httpx.Unprocessable("PERIODE_PENDAFTARAN_TUTUP", "periode pendaftaran sudah ditutup")
		}
	}

	// Jalankan ulang validasi KRS dan eligibilitas
	statusBaru := StatusDiajukan
	var alasanTolakBaru *string

	okKRS, alasanKRS, err := s.krsValidator.Validate(ctx, *mhs, *per)
	if err != nil {
		return nil, fmt.Errorf("validasi KRS ulang: %w", err)
	}
	if !okKRS {
		statusBaru = StatusDitolak
		alasan := "Validasi KRS tidak terpenuhi: " + alasanKRS
		alasanTolakBaru = &alasan
	} else {
		okElig, alasanElig, err := s.eligChecker.Check(ctx, *mhs, *per)
		if err != nil {
			return nil, fmt.Errorf("validasi eligibilitas ulang: %w", err)
		}
		if !okElig {
			statusBaru = StatusDitolak
			alasan := "Syarat eligibilitas program tidak terpenuhi: " + alasanElig
			alasanTolakBaru = &alasan
		}
	}

	if err := s.repo.UpdateStatusAndAlasan(ctx, s.pool, idPeserta, statusBaru, alasanTolakBaru); err != nil {
		return nil, database.MapError(err)
	}

	return s.GetDetail(ctx, actor, idPeserta)
}

// GetDetail mengambil detail lengkap pendaftaran setelah memeriksa otorisasi akses per relasi.
func (s *pendaftaranService) GetDetail(ctx context.Context, actor middleware.Actor, idPeserta int) (*PendaftaranDetail, error) {
	// 1. Periksa relasi aktor terhadap pendaftaran (tanpa relasi -> 404)
	if _, err := s.access.Resolve(ctx, actor, idPeserta); err != nil {
		return nil, err
	}

	// 2. Ambil detail lengkap dari basis data
	detail, err := s.repo.GetPendaftaranDetail(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil detail pendaftaran: %w", err)
	}
	if detail == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// 3. Hitung kesiapan (readiness)
	var idKelompok *int
	if detail.KelompokKKN != nil {
		idKelompok = &detail.KelompokKKN.ID
	}
	docTypes, err := s.repo.GetDokumenTypesByPeserta(ctx, s.pool, idPeserta, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil dokumen kesiapan: %w", err)
	}

	var tpID, plID, dpID *int
	if detail.TempatProgram != nil {
		tpID = &detail.TempatProgram.ID
	}
	if detail.PembimbingLapangan != nil {
		plID = &detail.PembimbingLapangan.ID
	}
	if detail.DosenPembimbing != nil {
		dpID = &detail.DosenPembimbing.ID
	}

	readinessInput := ReadinessInput{
		JenisProgram:         detail.JenisProgram,
		PeriodeStatus:        detail.Periode.Status,
		IDTempatProgram:      tpID,
		IDPembimbingLapangan: plID,
		IDDosenPembimbing:    dpID,
		IDKelompokKKN:        idKelompok,
		DokumenTypes:         docTypes,
	}

	detail.Kesiapan = CheckReadiness(readinessInput)
	return detail, nil
}

// List mengambil daftar pendaftaran yang dibatasi oleh cakupan role pemanggil (BR-14).
func (s *pendaftaranService) List(ctx context.Context, actor middleware.Actor, filter PendaftaranFilter, p httpx.Pagination) ([]PendaftaranListItem, httpx.PaginationMeta, error) {
	roleScope := ""
	profileID := 0

	// Otorisasi hierarki cakupan (BR-14)
	if actor.HasRole("admin") {
		roleScope = "admin"
	} else if actor.HasRole("pegawai") {
		roleScope = "pegawai"
	} else if actor.HasRole("mahasiswa") {
		roleScope = "mahasiswa"
		mhs, err := s.repo.FindMahasiswaByUserID(ctx, s.pool, actor.UserID)
		if err != nil || mhs == nil {
			return nil, httpx.PaginationMeta{}, httpx.Forbidden("PROFIL_TIDAK_DITEMUKAN", "profil mahasiswa tidak ditemukan")
		}
		profileID = mhs.ID
	} else if actor.HasRole("dosen") {
		roleScope = "dosen"
		err := s.pool.QueryRow(ctx, "SELECT id FROM dosen WHERE id_user = $1", actor.UserID).Scan(&profileID)
		if err != nil {
			return nil, httpx.PaginationMeta{}, httpx.Forbidden("PROFIL_TIDAK_DITEMUKAN", "profil dosen tidak ditemukan")
		}
	} else if actor.HasRole("pembimbing_lapangan") {
		roleScope = "pembimbing_lapangan"
		err := s.pool.QueryRow(ctx, "SELECT id FROM pembimbing_lapangan WHERE id_user = $1", actor.UserID).Scan(&profileID)
		if err != nil {
			return nil, httpx.PaginationMeta{}, httpx.Forbidden("PROFIL_TIDAK_DITEMUKAN", "profil pembimbing lapangan tidak ditemukan")
		}
	} else {
		return nil, httpx.PaginationMeta{}, httpx.Forbidden("AKSES_DITOLAK", "role pengguna tidak memiliki izin akses pendaftaran")
	}

	items, total, err := s.repo.ListPendaftaran(ctx, s.pool, roleScope, profileID, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("ambil daftar pendaftaran: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}

	return items, meta, nil
}

// PenempatanMagang menangani penetapan tempat magang dan mentor oleh mahasiswa pemilik.
// Menegakkan aturan BR-04 dan menjalankan evaluasi transisi kesiapan diajukan -> berjalan.
func (s *pendaftaranService) PenempatanMagang(ctx context.Context, actor middleware.Actor, idPeserta int, req PenempatanMagangRequest) (*PendaftaranDetail, error) {
	// 1. Otorisasi akses: hanya mahasiswa pemilik pendaftaran yang dapat memilih penempatan
	rel, err := s.access.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != RelMahasiswa {
		return nil, httpx.Forbidden("AKSES_DITOLAK", "hanya mahasiswa pemilik pendaftaran yang dapat memilih penempatan magang")
	}

	// 2. Ambil pendaftaran
	peserta, err := s.repo.FindPesertaByID(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil peserta: %w", err)
	}
	if peserta == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	if peserta.JenisProgram != "magang" {
		return nil, httpx.Unprocessable("BUKAN_PROGRAM_MAGANG", "penempatan magang hanya berlaku untuk program magang")
	}

	if peserta.Status == StatusBerjalan {
		return nil, httpx.Unprocessable("STATUS_TIDAK_VALID", "perubahan penempatan ditolak karena pendaftaran sudah berjalan")
	}

	if peserta.Status != StatusDiajukan {
		return nil, httpx.Unprocessable("STATUS_TIDAK_VALID", "perubahan penempatan hanya dapat dilakukan saat status diajukan")
	}

	// 3. Validasi BR-04
	// a. Tempat program harus berjenis magang dan is_aktif
	tempat, err := s.repo.FindTempatProgram(ctx, s.pool, req.IDTempatProgram)
	if err != nil {
		return nil, fmt.Errorf("cek tempat program: %w", err)
	}
	if tempat == nil || tempat.JenisProgram != "magang" || !tempat.IsAktif {
		return nil, httpx.Unprocessable("TEMPAT_TIDAK_VALID", "tempat program harus berjenis magang dan aktif")
	}

	// b. Pembimbing lapangan harus berjenis mentor dan terdaftar pada tempat tersebut
	mentor, err := s.repo.FindPembimbingLapangan(ctx, s.pool, req.IDPembimbingLapangan)
	if err != nil {
		return nil, fmt.Errorf("cek pembimbing lapangan: %w", err)
	}
	if mentor == nil || mentor.Jenis != "mentor" || mentor.IDTempatProgram != req.IDTempatProgram {
		return nil, httpx.Unprocessable("PEMBIMBING_TIDAK_VALID", "pembimbing lapangan harus berjenis mentor dan terdaftar pada tempat program yang dipilih")
	}

	// 4. Update dalam transaksi dan jalankan percobaan transisi kesiapan
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		if err := s.repo.UpdatePenempatanMagang(ctx, tx, idPeserta, req.IDTempatProgram, req.IDPembimbingLapangan); err != nil {
			return nil, fmt.Errorf("update penempatan magang: %w", err)
		}

		_, _ = TryTransitionToBerjalan(ctx, tx, idPeserta)

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit tx penempatan magang: %w", err)
		}
	} else {
		if err := s.repo.UpdatePenempatanMagang(ctx, nil, idPeserta, req.IDTempatProgram, req.IDPembimbingLapangan); err != nil {
			return nil, fmt.Errorf("update penempatan magang: %w", err)
		}
	}

	return s.GetDetail(ctx, actor, idPeserta)
}

// SetDosenPembimbing menangani penetapan dosen pembimbing oleh pegawai (magang) atau admin (semua jenis).
// Menegakkan aturan BR-07 dan mencatat audit ke log_aktivitas.
func (s *pendaftaranService) SetDosenPembimbing(ctx context.Context, actor middleware.Actor, idPeserta int, req SetDosenPembimbingRequest, ip, userAgent string) (*PendaftaranDetail, error) {
	// 1. Ambil pendaftaran
	peserta, err := s.repo.FindPesertaByID(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil peserta: %w", err)
	}
	if peserta == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// 2. Hak akses aktor:
	// Pegawai hanya untuk program magang; Admin untuk semua jenis
	if actor.HasRole("admin") {
		// Admin diizinkan untuk seluruh jenis program
	} else if actor.HasRole("pegawai") {
		if peserta.JenisProgram != "magang" {
			return nil, httpx.Forbidden("AKSES_DITOLAK", "pegawai hanya berwenang menetapkan dosen pembimbing untuk program magang")
		}
	} else {
		return nil, httpx.Forbidden("AKSES_DITOLAK", "hanya admin atau pegawai yang dapat menetapkan dosen pembimbing")
	}

	// 3. Validasi keberadaan dosen
	dosen, err := s.repo.FindDosen(ctx, s.pool, req.IDDosen)
	if err != nil {
		return nil, fmt.Errorf("cek dosen: %w", err)
	}
	if dosen == nil {
		return nil, httpx.Unprocessable("DOSEN_TIDAK_DITEMUKAN", "dosen pembimbing tidak ditemukan")
	}

	// 4. Validasi BR-07: id_dosen <> id_dosen_penguji (khusus non-magang; untuk magang diizinkan merangkap)
	if peserta.JenisProgram != "magang" && peserta.IDDosenPenguji != nil && *peserta.IDDosenPenguji == req.IDDosen {
		return nil, httpx.Unprocessable("DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", "dosen pembimbing tidak boleh sama dengan dosen penguji")
	}

	// 5. Update dalam transaksi dan jalankan percobaan transisi kesiapan
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		if err := s.repo.UpdateDosenPembimbing(ctx, tx, idPeserta, req.IDDosen); err != nil {
			return nil, fmt.Errorf("update dosen pembimbing: %w", err)
		}

		_, _ = TryTransitionToBerjalan(ctx, tx, idPeserta)

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit tx dosen pembimbing: %w", err)
		}
	} else {
		if err := s.repo.UpdateDosenPembimbing(ctx, nil, idPeserta, req.IDDosen); err != nil {
			return nil, fmt.Errorf("update dosen pembimbing: %w", err)
		}
	}

	// 6. Catat ke log_aktivitas
	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "tetapkan_dosen_pembimbing", true, ip, userAgent)
	}

	return s.GetDetail(ctx, actor, idPeserta)
}

// AdminPenempatan menangani pemetaan sekolah penempatan dan guru pamong oleh admin untuk program asistensi mengajar (BR-05).
func (s *pendaftaranService) AdminPenempatan(ctx context.Context, actor middleware.Actor, idPeserta int, req AdminPenempatanRequest) (*PendaftaranDetail, error) {
	// 1. Hak akses: hanya admin
	if !actor.HasRole("admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya admin yang dapat memetakan penempatan program asistensi mengajar")
	}

	// 2. Ambil pendaftaran
	peserta, err := s.repo.FindPesertaByID(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil peserta: %w", err)
	}
	if peserta == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// 3. Validasi jenis program: berlaku untuk jenis asistensi_mengajar
	// Untuk jenis lain, balas 422 dengan pesan jelas (magang dipilih mahasiswa, KKN lewat kelompok).
	if peserta.JenisProgram == "magang" {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_SESUAI", "penempatan magang dipilih mandiri oleh mahasiswa melalui endpoint penempatan-magang")
	}
	if peserta.JenisProgram == "kkn" {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_SESUAI", "penempatan KKN ditentukan melalui kelompok KKN")
	}
	if peserta.JenisProgram != "asistensi_mengajar" {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_SESUAI", "endpoint ini hanya berlaku untuk program asistensi mengajar")
	}

	if peserta.Status == StatusBerjalan {
		return nil, httpx.Unprocessable("STATUS_TIDAK_VALID", "perubahan penempatan ditolak karena pendaftaran sudah berjalan")
	}
	if peserta.Status != StatusDiajukan {
		return nil, httpx.Unprocessable("STATUS_TIDAK_VALID", "perubahan penempatan hanya dapat dilakukan saat status diajukan")
	}

	// 4. Validasi BR-05:
	// a. Tempat program harus berjenis asistensi_mengajar dan is_aktif
	tempat, err := s.repo.FindTempatProgram(ctx, s.pool, req.IDTempatProgram)
	if err != nil {
		return nil, fmt.Errorf("cek tempat program: %w", err)
	}
	if tempat == nil || tempat.JenisProgram != "asistensi_mengajar" || !tempat.IsAktif {
		return nil, httpx.Unprocessable("TEMPAT_TIDAK_VALID", "tempat program harus berjenis asistensi_mengajar dan aktif")
	}

	// b. Pembimbing lapangan harus berjenis guru_pamong dan milik tempat tersebut
	pl, err := s.repo.FindPembimbingLapangan(ctx, s.pool, req.IDPembimbingLapangan)
	if err != nil {
		return nil, fmt.Errorf("cek pembimbing lapangan: %w", err)
	}
	if pl == nil || pl.Jenis != "guru_pamong" || pl.IDTempatProgram != req.IDTempatProgram {
		return nil, httpx.Unprocessable("PEMBIMBING_TIDAK_VALID", "pembimbing lapangan harus berjenis guru_pamong dan terdaftar pada sekolah penempatan terkait")
	}

	// 5. Update dalam transaksi dan jalankan percobaan transisi kesiapan
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		if err := s.repo.UpdatePenempatanAM(ctx, tx, idPeserta, req.IDTempatProgram, req.IDPembimbingLapangan); err != nil {
			return nil, fmt.Errorf("update penempatan asistensi mengajar: %w", err)
		}

		_, _ = TryTransitionToBerjalan(ctx, tx, idPeserta)

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit tx penempatan asistensi mengajar: %w", err)
		}
	} else {
		if err := s.repo.UpdatePenempatanAM(ctx, nil, idPeserta, req.IDTempatProgram, req.IDPembimbingLapangan); err != nil {
			return nil, fmt.Errorf("update penempatan asistensi mengajar: %w", err)
		}
	}

	return s.GetDetail(ctx, actor, idPeserta)
}

// SetDosenPenguji menangani penetapan dosen penguji oleh admin untuk semua jenis program.
// Menegakkan aturan BR-07 dan mencatat audit ke log_aktivitas.
func (s *pendaftaranService) SetDosenPenguji(ctx context.Context, actor middleware.Actor, idPeserta int, req SetDosenPengujiRequest, ip, userAgent string) (*PendaftaranDetail, error) {
	// 1. Hak akses aktor: hanya admin
	if !actor.HasRole("admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya admin yang dapat menetapkan dosen penguji")
	}

	// 2. Ambil pendaftaran
	peserta, err := s.repo.FindPesertaByID(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil peserta: %w", err)
	}
	if peserta == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// 3. Validasi keberadaan dosen
	dosen, err := s.repo.FindDosen(ctx, s.pool, req.IDDosen)
	if err != nil {
		return nil, fmt.Errorf("cek dosen: %w", err)
	}
	if dosen == nil {
		return nil, httpx.Unprocessable("DOSEN_TIDAK_DITEMUKAN", "dosen penguji tidak ditemukan")
	}

	// 4. Validasi BR-07: id_dosen <> id_dosen_pembimbing (khusus non-magang; untuk magang diizinkan merangkap)
	if peserta.JenisProgram != "magang" && peserta.IDDosenPembimbing != nil && *peserta.IDDosenPembimbing == req.IDDosen {
		return nil, httpx.Unprocessable("DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", "dosen penguji tidak boleh sama dengan dosen pembimbing")
	}

	// 5. Update dalam transaksi dan jalankan percobaan transisi kesiapan
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		if err := s.repo.UpdateDosenPenguji(ctx, tx, idPeserta, req.IDDosen); err != nil {
			return nil, fmt.Errorf("update dosen penguji: %w", err)
		}

		_, _ = TryTransitionToBerjalan(ctx, tx, idPeserta)

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit tx dosen penguji: %w", err)
		}
	} else {
		if err := s.repo.UpdateDosenPenguji(ctx, nil, idPeserta, req.IDDosen); err != nil {
			return nil, fmt.Errorf("update dosen penguji: %w", err)
		}
	}

	// 6. Catat ke log_aktivitas
	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "tetapkan_dosen_penguji", true, ip, userAgent)
	}

	return s.GetDetail(ctx, actor, idPeserta)
}
