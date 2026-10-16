package jurnal

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

// Service mendefinisikan seluruh operasi use-case untuk jurnal harian.
type Service interface {
	CreateJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, req CreateJurnalRequest, ip, ua string) (*JurnalResponse, error)
	UpdateJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, idJurnal int, req UpdateJurnalRequest, ip, ua string) (*JurnalResponse, error)
	GetJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, idJurnal int) (*JurnalResponse, error)
	ListJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, filter JurnalFilter, p httpx.Pagination) ([]JurnalResponse, httpx.PaginationMeta, error)
}

type service struct {
	repo      Repository
	pool      *pgxpool.Pool
	clock     clock.Clock
	accessSvc pendaftaran.AccessService
	logger    ActivityLogger
}

// NewService membuat instance baru jurnal Service.
func NewService(repo Repository, pool *pgxpool.Pool, clk clock.Clock, accessSvc pendaftaran.AccessService, logger ActivityLogger) Service {
	return &service{
		repo:      repo,
		pool:      pool,
		clock:     clk,
		accessSvc: accessSvc,
		logger:    logger,
	}
}

func (s *service) CreateJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, req CreateJurnalRequest, ip, ua string) (*JurnalResponse, error) {
	// 1. Validasi status jurnal
	if req.Status != StatusDraft && req.Status != StatusDisubmit {
		return nil, httpx.BadRequest("STATUS_JURNAL_TIDAK_VALID", "status jurnal hanya boleh 'draft' atau 'disubmit'")
	}

	// 2. Otorisasi aktor: hanya mahasiswa pemilik pendaftaran
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != pendaftaran.RelMahasiswa {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya mahasiswa pemilik pendaftaran yang dapat membuat jurnal")
	}

	// 3. Parse dan validasi format tanggal YYYY-MM-DD
	targetDate, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal tidak valid, gunakan YYYY-MM-DD")
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
		return nil, httpx.Unprocessable("STATUS_TIDAK_SESUAI", "jurnal hanya boleh dibuat saat status pendaftaran berjalan")
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

	// 5. Validasi BR-09: satu jurnal per peserta per tanggal
	exists, err := s.repo.ExistsByDate(ctx, s.pool, idPeserta, targetDate, 0)
	if err != nil {
		return nil, fmt.Errorf("cek tanggal jurnal: %w", err)
	}
	if exists {
		return nil, httpx.Conflict("JURNAL_DUPLIKAT_TANGGAL", "jurnal harian untuk tanggal tersebut sudah pernah dibuat")
	}

	// 6. Simpan ke basis data
	jurnal := &JurnalHarian{
		IDPesertaProgram: idPeserta,
		Tanggal:          targetDate,
		Judul:            strings.TrimSpace(req.Judul),
		Isi:              strings.TrimSpace(req.Isi),
		Kendala:          req.Kendala,
		Solusi:           req.Solusi,
		Status:           req.Status,
	}

	if err := s.repo.Create(ctx, s.pool, jurnal); err != nil {
		if strings.Contains(err.Error(), "jurnal_harian_id_peserta_program_tanggal_key") {
			return nil, httpx.Conflict("JURNAL_DUPLIKAT_TANGGAL", "jurnal harian untuk tanggal tersebut sudah pernah dibuat")
		}
		return nil, fmt.Errorf("simpan jurnal harian: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "buat_jurnal_harian", true, ip, ua)
	}

	resp := ToJurnalResponse(*jurnal)
	return &resp, nil
}

func (s *service) UpdateJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, idJurnal int, req UpdateJurnalRequest, ip, ua string) (*JurnalResponse, error) {
	// 1. Validasi status baru
	if req.Status != StatusDraft && req.Status != StatusDisubmit {
		return nil, httpx.BadRequest("STATUS_JURNAL_TIDAK_VALID", "status jurnal hanya boleh 'draft' atau 'disubmit'")
	}

	// 2. Otorisasi aktor: hanya mahasiswa pemilik pendaftaran
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != pendaftaran.RelMahasiswa {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya mahasiswa pemilik pendaftaran yang dapat mengubah jurnal")
	}

	// 3. Ambil jurnal eksisting
	jurnal, err := s.repo.GetByID(ctx, s.pool, idJurnal)
	if err != nil {
		return nil, fmt.Errorf("ambil jurnal: %w", err)
	}
	if jurnal == nil || jurnal.IDPesertaProgram != idPeserta {
		return nil, httpx.NotFound("JURNAL_TIDAK_DITEMUKAN", "jurnal tidak ditemukan")
	}

	// 4. Aturan bisnis: hanya saat status draft
	if jurnal.Status != StatusDraft {
		return nil, httpx.Unprocessable("JURNAL_BUKAN_DRAFT", "jurnal hanya dapat diubah saat berstatus draft")
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
			return nil, httpx.Unprocessable("STATUS_TIDAK_SESUAI", "jurnal hanya boleh diubah saat status pendaftaran berjalan")
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

		// Cek duplikat tanggal (kecuali jurnal ini sendiri)
		exists, err := s.repo.ExistsByDate(ctx, s.pool, idPeserta, targetDate, idJurnal)
		if err != nil {
			return nil, fmt.Errorf("cek tanggal jurnal: %w", err)
		}
		if exists {
			return nil, httpx.Conflict("JURNAL_DUPLIKAT_TANGGAL", "jurnal harian untuk tanggal tersebut sudah pernah dibuat")
		}

		jurnal.Tanggal = targetDate
	}

	jurnal.Judul = strings.TrimSpace(req.Judul)
	jurnal.Isi = strings.TrimSpace(req.Isi)
	jurnal.Kendala = req.Kendala
	jurnal.Solusi = req.Solusi
	jurnal.Status = req.Status

	if err := s.repo.Update(ctx, s.pool, jurnal); err != nil {
		if strings.Contains(err.Error(), "jurnal_harian_id_peserta_program_tanggal_key") {
			return nil, httpx.Conflict("JURNAL_DUPLIKAT_TANGGAL", "jurnal harian untuk tanggal tersebut sudah pernah dibuat")
		}
		return nil, fmt.Errorf("update jurnal: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "ubah_jurnal_harian", true, ip, ua)
	}

	resp := ToJurnalResponse(*jurnal)
	return &resp, nil
}

func (s *service) GetJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, idJurnal int) (*JurnalResponse, error) {
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	if rel != pendaftaran.RelMahasiswa && rel != pendaftaran.RelPembimbing && rel != pendaftaran.RelLapangan &&
		rel != pendaftaran.RelAdmin && rel != pendaftaran.RelPegawai {
		return nil, httpx.NotFound("JURNAL_TIDAK_DITEMUKAN", "jurnal tidak ditemukan atau akses tidak diizinkan")
	}

	jurnal, err := s.repo.GetByID(ctx, s.pool, idJurnal)
	if err != nil {
		return nil, fmt.Errorf("ambil jurnal by id: %w", err)
	}
	if jurnal == nil || jurnal.IDPesertaProgram != idPeserta {
		return nil, httpx.NotFound("JURNAL_TIDAK_DITEMUKAN", "jurnal tidak ditemukan")
	}

	resp := ToJurnalResponse(*jurnal)
	return &resp, nil
}

func (s *service) ListJurnal(ctx context.Context, actor middleware.Actor, idPeserta int, filter JurnalFilter, p httpx.Pagination) ([]JurnalResponse, httpx.PaginationMeta, error) {
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	if rel != pendaftaran.RelMahasiswa && rel != pendaftaran.RelPembimbing && rel != pendaftaran.RelLapangan &&
		rel != pendaftaran.RelAdmin && rel != pendaftaran.RelPegawai {
		return nil, httpx.PaginationMeta{}, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan atau akses tidak diizinkan")
	}

	list, meta, err := s.repo.List(ctx, s.pool, idPeserta, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("list jurnal: %w", err)
	}

	return ToJurnalResponses(list), meta, nil
}
