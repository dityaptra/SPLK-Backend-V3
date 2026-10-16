package logbook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/platform/httpx"
)

// ActivityLogger mendefinisikan antarmuka pencatatan log audit aktivitas.
type ActivityLogger interface {
	Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error
}

// Service mendefinisikan seluruh operasi use-case untuk logbook bimbingan.
type Service interface {
	CreateLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, req CreateLogbookRequest, ip, ua string) (*LogbookResponse, error)
	UpdateLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, idLogbook int, req UpdateLogbookRequest, ip, ua string) (*LogbookResponse, error)
	GetLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, idLogbook int) (*LogbookResponse, error)
	ListLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, p httpx.Pagination) ([]LogbookResponse, httpx.PaginationMeta, error)
}

type service struct {
	repo      Repository
	pool      *pgxpool.Pool
	clock     clock.Clock
	accessSvc pendaftaran.AccessService
	logger    ActivityLogger
}

// NewService membuat instance baru logbook Service.
func NewService(repo Repository, pool *pgxpool.Pool, clk clock.Clock, accessSvc pendaftaran.AccessService, logger ActivityLogger) Service {
	return &service{
		repo:      repo,
		pool:      pool,
		clock:     clk,
		accessSvc: accessSvc,
		logger:    logger,
	}
}

func (s *service) CreateLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, req CreateLogbookRequest, ip, ua string) (*LogbookResponse, error) {
	// 1. Otorisasi relasi: hanya dosen pembimbing pendaftaran itu (dosen lain menghasilkan 404)
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != pendaftaran.RelPembimbing {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan atau akses tidak diizinkan")
	}

	// 2. Ambil profil dosen pemanggil
	idDosen, namaDosen, err := s.repo.FindDosenIDByUserID(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("ambil profil dosen: %w", err)
	}
	if idDosen == nil {
		return nil, httpx.NotFound("DOSEN_TIDAK_DITEMUKAN", "profil dosen tidak ditemukan")
	}

	// 3. Parse dan validasi format tanggal YYYY-MM-DD
	targetDate, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal tidak valid, gunakan YYYY-MM-DD")
	}

	var pertemuanDate *time.Time
	if req.PertemuanBerikutnya != nil && *req.PertemuanBerikutnya != "" {
		t, err := time.Parse("2006-01-02", *req.PertemuanBerikutnya)
		if err != nil {
			return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal pertemuan berikutnya tidak valid, gunakan YYYY-MM-DD")
		}
		pertemuanDate = &t
	}

	// 4. Validasi BR-08: status pendaftaran harus 'berjalan', tanggal dalam rentang periode, dan tidak di masa depan
	info, err := s.repo.GetPendaftaranInfo(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil info pendaftaran: %w", err)
	}
	if info == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	if info.StatusPeserta != pendaftaran.StatusBerjalan {
		return nil, httpx.Unprocessable("STATUS_TIDAK_SESUAI", "logbook hanya boleh dibuat saat status pendaftaran berjalan")
	}

	today := s.clock.Today()
	if targetDate.After(today) {
		return nil, httpx.Unprocessable("TANGGAL_MASA_DEPAN", "tanggal tidak boleh di masa depan")
	}

	tMulai := time.Date(info.TanggalMulai.Year(), info.TanggalMulai.Month(), info.TanggalMulai.Day(), 0, 0, 0, 0, targetDate.Location())
	tSelesai := time.Date(info.TanggalSelesai.Year(), info.TanggalSelesai.Month(), info.TanggalSelesai.Day(), 23, 59, 59, 0, targetDate.Location())
	if targetDate.Before(tMulai) || targetDate.After(tSelesai) {
		return nil, httpx.Unprocessable("TANGGAL_DILUAR_PERIODE", "tanggal harus berada dalam rentang pelaksanaan periode program")
	}

	logbook := &LogbookBimbingan{
		IDPesertaProgram:    idPeserta,
		IDDosen:             *idDosen,
		Tanggal:             targetDate,
		TopikBimbingan:      strings.TrimSpace(req.TopikBimbingan),
		Catatan:             req.Catatan,
		PertemuanBerikutnya: pertemuanDate,
	}

	if err := s.repo.Create(ctx, s.pool, logbook); err != nil {
		return nil, fmt.Errorf("simpan logbook bimbingan: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "buat_logbook_bimbingan", true, ip, ua)
	}

	resp := ToLogbookResponse(*logbook, namaDosen)
	return &resp, nil
}

func (s *service) UpdateLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, idLogbook int, req UpdateLogbookRequest, ip, ua string) (*LogbookResponse, error) {
	// 1. Otorisasi relasi: harus pembimbing pendaftaran
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != pendaftaran.RelPembimbing {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan atau akses tidak diizinkan")
	}

	// 2. Ambil profil dosen pemanggil
	idDosen, namaDosen, err := s.repo.FindDosenIDByUserID(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("ambil profil dosen: %w", err)
	}
	if idDosen == nil {
		return nil, httpx.NotFound("DOSEN_TIDAK_DITEMUKAN", "profil dosen tidak ditemukan")
	}

	// 3. Ambil data logbook eksisting
	existing, existingNamaDosen, err := s.repo.GetByID(ctx, s.pool, idLogbook)
	if err != nil {
		return nil, fmt.Errorf("ambil logbook: %w", err)
	}
	if existing == nil || existing.IDPesertaProgram != idPeserta {
		return nil, httpx.NotFound("LOGBOOK_TIDAK_DITEMUKAN", "logbook bimbingan tidak ditemukan")
	}

	// 4. Aturan bisnis: hanya dosen yang membuatnya yang berhak mengubah
	if existing.IDDosen != *idDosen {
		return nil, httpx.NotFound("LOGBOOK_TIDAK_DITEMUKAN", "logbook bimbingan tidak ditemukan atau akses tidak diizinkan")
	}

	if namaDosen == "" {
		namaDosen = existingNamaDosen
	}

	// 5. Cek jika tanggal diubah
	if req.Tanggal != nil && *req.Tanggal != "" {
		targetDate, err := time.Parse("2006-01-02", *req.Tanggal)
		if err != nil {
			return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal tidak valid, gunakan YYYY-MM-DD")
		}

		info, err := s.repo.GetPendaftaranInfo(ctx, s.pool, idPeserta)
		if err != nil {
			return nil, fmt.Errorf("ambil info pendaftaran: %w", err)
		}
		if info == nil {
			return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
		}

		if info.StatusPeserta != pendaftaran.StatusBerjalan {
			return nil, httpx.Unprocessable("STATUS_TIDAK_SESUAI", "logbook hanya boleh diubah saat status pendaftaran berjalan")
		}

		today := s.clock.Today()
		if targetDate.After(today) {
			return nil, httpx.Unprocessable("TANGGAL_MASA_DEPAN", "tanggal tidak boleh di masa depan")
		}

		tMulai := time.Date(info.TanggalMulai.Year(), info.TanggalMulai.Month(), info.TanggalMulai.Day(), 0, 0, 0, 0, targetDate.Location())
		tSelesai := time.Date(info.TanggalSelesai.Year(), info.TanggalSelesai.Month(), info.TanggalSelesai.Day(), 23, 59, 59, 0, targetDate.Location())
		if targetDate.Before(tMulai) || targetDate.After(tSelesai) {
			return nil, httpx.Unprocessable("TANGGAL_DILUAR_PERIODE", "tanggal harus berada dalam rentang pelaksanaan periode program")
		}

		existing.Tanggal = targetDate
	}

	if req.PertemuanBerikutnya != nil {
		if *req.PertemuanBerikutnya != "" {
			t, err := time.Parse("2006-01-02", *req.PertemuanBerikutnya)
			if err != nil {
				return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal pertemuan berikutnya tidak valid, gunakan YYYY-MM-DD")
			}
			existing.PertemuanBerikutnya = &t
		} else {
			existing.PertemuanBerikutnya = nil
		}
	}

	existing.TopikBimbingan = strings.TrimSpace(req.TopikBimbingan)
	existing.Catatan = req.Catatan

	if err := s.repo.Update(ctx, s.pool, existing); err != nil {
		return nil, fmt.Errorf("update logbook: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "ubah_logbook_bimbingan", true, ip, ua)
	}

	resp := ToLogbookResponse(*existing, namaDosen)
	return &resp, nil
}

func (s *service) GetLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, idLogbook int) (*LogbookResponse, error) {
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != pendaftaran.RelMahasiswa && rel != pendaftaran.RelPembimbing && rel != pendaftaran.RelAdmin && rel != pendaftaran.RelPegawai {
		return nil, httpx.NotFound("LOGBOOK_TIDAK_DITEMUKAN", "logbook bimbingan tidak ditemukan atau akses tidak diizinkan")
	}

	logbook, namaDosen, err := s.repo.GetByID(ctx, s.pool, idLogbook)
	if err != nil {
		return nil, fmt.Errorf("ambil logbook by id: %w", err)
	}
	if logbook == nil || logbook.IDPesertaProgram != idPeserta {
		return nil, httpx.NotFound("LOGBOOK_TIDAK_DITEMUKAN", "logbook bimbingan tidak ditemukan")
	}

	resp := ToLogbookResponse(*logbook, namaDosen)
	return &resp, nil
}

func (s *service) ListLogbook(ctx context.Context, actor middleware.Actor, idPeserta int, p httpx.Pagination) ([]LogbookResponse, httpx.PaginationMeta, error) {
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	if rel != pendaftaran.RelMahasiswa && rel != pendaftaran.RelPembimbing && rel != pendaftaran.RelAdmin && rel != pendaftaran.RelPegawai {
		return nil, httpx.PaginationMeta{}, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan atau akses tidak diizinkan")
	}

	return s.repo.List(ctx, s.pool, idPeserta, p)
}
