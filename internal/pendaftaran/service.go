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

// Service mendefinisikan seluruh use-case untuk modul pendaftaran.
type Service interface {
	Daftar(ctx context.Context, actor middleware.Actor, req CreatePendaftaranRequest) (*PendaftaranDetail, error)
	AjukanUlang(ctx context.Context, actor middleware.Actor, idPeserta int) (*PendaftaranDetail, error)
	GetDetail(ctx context.Context, actor middleware.Actor, idPeserta int) (*PendaftaranDetail, error)
	List(ctx context.Context, actor middleware.Actor, filter PendaftaranFilter, p httpx.Pagination) ([]PendaftaranListItem, httpx.PaginationMeta, error)
}

type pendaftaranService struct {
	repo         Repository
	pool         *pgxpool.Pool
	clock        clock.Clock
	krsValidator KRSValidator
	eligChecker  EligibilityChecker
	access       AccessService
}

// NewService membuat instance Service baru untuk modul pendaftaran.
func NewService(
	repo Repository,
	pool *pgxpool.Pool,
	clk clock.Clock,
	krsVal KRSValidator,
	eligCheck EligibilityChecker,
	access AccessService,
) Service {
	return &pendaftaranService{
		repo:         repo,
		pool:         pool,
		clock:        clk,
		krsValidator: krsVal,
		eligChecker:  eligCheck,
		access:       access,
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
