// Package periode menyediakan logika bisnis dan use-case untuk pengelolaan periode program SPLK.
package periode

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

// Service mendefinisikan fungsi logika bisnis untuk periode program.
type Service interface {
	CreatePeriode(ctx context.Context, req CreatePeriodeRequest, creatorUserID int) (*PeriodeDetail, error)
	GetPeriodeByID(ctx context.Context, id int, actor middleware.Actor) (*PeriodeDetail, error)
	ListPeriode(ctx context.Context, filter PeriodeFilter, p httpx.Pagination, actor middleware.Actor) ([]PeriodeDetail, httpx.PaginationMeta, error)
	UpdatePeriode(ctx context.Context, id int, req UpdatePeriodeRequest) (*PeriodeDetail, error)
	AktifkanPeriode(ctx context.Context, id int, actorID int, ip, userAgent string) (*PeriodeDetail, error)
	SelesaikanPeriode(ctx context.Context, id int, actorID int, ip, userAgent string) (*PeriodeDetail, error)
}

type periodeService struct {
	repo   Repository
	pool   *pgxpool.Pool
	clock  clock.Clock
	logger ActivityLogger
}

// NewService membuat instance Service baru untuk manajemen periode.
func NewService(repo Repository, pool *pgxpool.Pool, clk clock.Clock, logger ActivityLogger) Service {
	return &periodeService{
		repo:   repo,
		pool:   pool,
		clock:  clk,
		logger: logger,
	}
}

// CreatePeriode membuat periode baru dengan status awal 'draft'.
func (s *periodeService) CreatePeriode(ctx context.Context, req CreatePeriodeRequest, creatorUserID int) (*PeriodeDetail, error) {
	// Validasi jenis_program sesuai enum (RULES.md bagian 7 nomor 4)
	if req.JenisProgram != "magang" && req.JenisProgram != "asistensi_mengajar" && req.JenisProgram != "kkn" {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_VALID", "jenis program harus salah satu dari: magang, asistensi_mengajar, kkn")
	}

	mulai, selesai, buka, tutup, err := parseAndValidateDates(req.TanggalMulai, req.TanggalSelesai, req.TanggalBukaDaftar, req.TanggalTutupDaftar)
	if err != nil {
		return nil, err
	}

	// RULES.md bagian 7 nomor 2: periode.dibuat_oleh hanya diisi bila pembuat memiliki profil pegawai.
	pegawaiID, err := s.repo.FindPegawaiIDByUserID(ctx, s.pool, creatorUserID)
	if err != nil {
		return nil, fmt.Errorf("cek profil pegawai pembuat: %w", err)
	}

	p := &Periode{
		Nama:               req.Nama,
		JenisProgram:       req.JenisProgram,
		TahunSemester:      req.TahunSemester,
		TanggalMulai:       mulai,
		TanggalSelesai:     selesai,
		TanggalBukaDaftar:  buka,
		TanggalTutupDaftar: tutup,
		Status:             StatusDraft,
		DibuatOleh:         pegawaiID,
	}

	if err := s.repo.CreatePeriode(ctx, s.pool, p); err != nil {
		return nil, database.MapError(err)
	}

	detail := MapToPeriodeDetail(*p)
	return &detail, nil
}

// GetPeriodeByID mengambil detail periode berdasarkan ID.
func (s *periodeService) GetPeriodeByID(ctx context.Context, id int, actor middleware.Actor) (*PeriodeDetail, error) {
	p, err := s.repo.FindPeriodeByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("cari periode by id: %w", err)
	}
	if p == nil {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}

	// PRD 5.3: Mahasiswa hanya melihat yang status aktif
	if isMahasiswaOnly(actor) && p.Status != StatusAktif {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}

	detail := MapToPeriodeDetail(*p)
	return &detail, nil
}

// ListPeriode mengambil daftar periode dengan opsi filter dan pagination.
func (s *periodeService) ListPeriode(ctx context.Context, filter PeriodeFilter, p httpx.Pagination, actor middleware.Actor) ([]PeriodeDetail, httpx.PaginationMeta, error) {
	// PRD 5.3: Mahasiswa hanya melihat yang status aktif
	if isMahasiswaOnly(actor) {
		filter.Status = StatusAktif
	}

	list, total, err := s.repo.ListPeriode(ctx, s.pool, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("ambil daftar periode: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}

	return MapListToPeriodeDetail(list), meta, nil
}

// UpdatePeriode memperbarui data periode yang masih berstatus 'draft'.
func (s *periodeService) UpdatePeriode(ctx context.Context, id int, req UpdatePeriodeRequest) (*PeriodeDetail, error) {
	existing, err := s.repo.FindPeriodeByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("cari periode by id: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}

	// Aturan bisnis: Update field hanya saat status masih draft
	if !CanUpdate(existing.Status) {
		return nil, httpx.Unprocessable("PERIODE_BUKAN_DRAFT", "periode hanya dapat diubah saat berstatus draft")
	}

	_, _, _, _, err = parseAndValidateDates(req.TanggalMulai, req.TanggalSelesai, req.TanggalBukaDaftar, req.TanggalTutupDaftar)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdatePeriode(ctx, s.pool, id, req); err != nil {
		return nil, database.MapError(err)
	}

	updated, err := s.repo.FindPeriodeByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil periode terupdate: %w", err)
	}

	detail := MapToPeriodeDetail(*updated)
	return &detail, nil
}

// AktifkanPeriode mengubah status periode dari 'draft' ke 'aktif'.
func (s *periodeService) AktifkanPeriode(ctx context.Context, id int, actorID int, ip, userAgent string) (*PeriodeDetail, error) {
	existing, err := s.repo.FindPeriodeByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("cari periode by id: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}

	// Validasi transisi status satu arah: draft -> aktif
	if !IsTransisiValid(existing.Status, StatusAktif) {
		return nil, httpx.Unprocessable("TRANSISI_STATUS_TIDAK_VALID", fmt.Sprintf("periode dengan status '%s' tidak dapat diaktifkan", existing.Status))
	}

	if err := s.repo.UpdateStatusPeriode(ctx, s.pool, id, StatusAktif); err != nil {
		return nil, database.MapError(err)
	}

	// Catat ke log_aktivitas saat periode diaktifkan
	if s.logger != nil {
		_ = s.logger.Log(ctx, actorID, "aktifkan_periode", true, ip, userAgent)
	}

	existing.Status = StatusAktif
	detail := MapToPeriodeDetail(*existing)
	return &detail, nil
}

// SelesaikanPeriode mengubah status periode dari 'aktif' ke 'selesai' dan memproses transisi peserta secara atomik.
func (s *periodeService) SelesaikanPeriode(ctx context.Context, id int, actorID int, ip, userAgent string) (*PeriodeDetail, error) {
	existing, err := s.repo.FindPeriodeByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("cari periode by id: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}

	// Validasi transisi status satu arah: aktif -> selesai
	if !IsTransisiValid(existing.Status, StatusSelesai) {
		return nil, httpx.Unprocessable("TRANSISI_STATUS_TIDAK_VALID", fmt.Sprintf("periode dengan status '%s' tidak dapat diselesaikan", existing.Status))
	}

	// Eksekusi atomik dalam transaksi: periode -> selesai, peserta 'berjalan' -> 'selesai', peserta 'diajukan' -> 'ditolak'
	// ASUMSI: A12 - peserta dengan status diajukan saat periode diselesaikan ditolak dengan alasan otomatis.
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := s.repo.UpdateStatusPeriode(ctx, tx, id, StatusSelesai); err != nil {
			return fmt.Errorf("update status periode tx: %w", err)
		}

		if _, _, err := s.repo.SelesaikanPeriodeTx(ctx, tx, id); err != nil {
			return fmt.Errorf("selesaikan peserta program tx: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, database.MapError(err)
	}

	// Catat ke log_aktivitas saat periode diselesaikan
	if s.logger != nil {
		_ = s.logger.Log(ctx, actorID, "selesaikan_periode", true, ip, userAgent)
	}

	existing.Status = StatusSelesai
	detail := MapToPeriodeDetail(*existing)
	return &detail, nil
}

// parseAndValidateDates menguraikan string tanggal YYYY-MM-DD dan memvalidasi aturan rentang tanggal.
func parseAndValidateDates(tMulaiStr, tSelesaiStr string, tBukaStr, tTutupStr *string) (mulai, selesai time.Time, buka, tutup *time.Time, err error) {
	mulai, err = time.Parse("2006-01-02", tMulaiStr)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal mulai tidak valid, gunakan YYYY-MM-DD")
	}

	selesai, err = time.Parse("2006-01-02", tSelesaiStr)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal selesai tidak valid, gunakan YYYY-MM-DD")
	}

	// Aturan: tanggal_selesai >= tanggal_mulai
	if selesai.Before(mulai) {
		return time.Time{}, time.Time{}, nil, nil, httpx.Unprocessable("TANGGAL_TIDAK_VALID", "tanggal selesai tidak boleh lebih awal dari tanggal mulai")
	}

	if tBukaStr != nil {
		b, err := time.Parse("2006-01-02", *tBukaStr)
		if err != nil {
			return time.Time{}, time.Time{}, nil, nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal buka daftar tidak valid, gunakan YYYY-MM-DD")
		}
		buka = &b
	}

	if tTutupStr != nil {
		t, err := time.Parse("2006-01-02", *tTutupStr)
		if err != nil {
			return time.Time{}, time.Time{}, nil, nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal tutup daftar tidak valid, gunakan YYYY-MM-DD")
		}
		tutup = &t
	}

	// Aturan: tanggal_tutup_daftar >= tanggal_buka_daftar bila keduanya ada
	if buka != nil && tutup != nil {
		if tutup.Before(*buka) {
			return time.Time{}, time.Time{}, nil, nil, httpx.Unprocessable("TANGGAL_TIDAK_VALID", "tanggal tutup daftar tidak boleh lebih awal dari tanggal buka daftar")
		}
	}

	return mulai, selesai, buka, tutup, nil
}

// isMahasiswaOnly memeriksa apakah actor hanya memiliki role mahasiswa (bukan admin atau pegawai).
func isMahasiswaOnly(actor middleware.Actor) bool {
	if actor.HasRole("admin", "pegawai") {
		return false
	}
	return actor.HasRole("mahasiswa")
}
