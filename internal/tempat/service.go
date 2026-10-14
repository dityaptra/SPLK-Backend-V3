package tempat

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/platform/httpx"
)

// Service mendefinisikan aturan bisnis untuk modul tempat.
type Service interface {
	Create(ctx context.Context, req CreateTempatRequest) (*TempatProgram, error)
	GetByID(ctx context.Context, id int) (*TempatProgram, error)
	List(ctx context.Context, filter TempatFilter, p httpx.Pagination) ([]TempatProgram, httpx.PaginationMeta, error)
	Update(ctx context.Context, id int, req UpdateTempatRequest) (*TempatProgram, error)
	Delete(ctx context.Context, id int) error
	ListPembimbing(ctx context.Context, tempatID int) ([]PembimbingLapangan, error)
}

type tempatService struct {
	repo Repository
	pool *pgxpool.Pool
}

// NewService membuat instance Service tempat baru.
func NewService(repo Repository, pool *pgxpool.Pool) Service {
	return &tempatService{
		repo: repo,
		pool: pool,
	}
}

func (s *tempatService) Create(ctx context.Context, req CreateTempatRequest) (*TempatProgram, error) {
	if !IsValidProgram(req.JenisProgram) {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_VALID", "jenis program harus salah satu dari: magang, asistensi_mengajar, kkn")
	}

	t := &TempatProgram{
		JenisProgram: req.JenisProgram,
		Nama:         req.Nama,
		Alamat:       req.Alamat,
		Kota:         req.Kota,
		Kontak:       req.Kontak,
	}

	if err := s.repo.Create(ctx, s.pool, t); err != nil {
		return nil, fmt.Errorf("buat tempat program: %w", err)
	}

	return t, nil
}

func (s *tempatService) GetByID(ctx context.Context, id int) (*TempatProgram, error) {
	t, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil tempat program: %w", err)
	}
	if t == nil {
		return nil, httpx.NotFound("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}
	return t, nil
}

func (s *tempatService) List(ctx context.Context, filter TempatFilter, p httpx.Pagination) ([]TempatProgram, httpx.PaginationMeta, error) {
	items, total, err := s.repo.List(ctx, s.pool, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("ambil daftar tempat program: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}

	return items, meta, nil
}

func (s *tempatService) Update(ctx context.Context, id int, req UpdateTempatRequest) (*TempatProgram, error) {
	existing, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil tempat program saat update: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}

	if !IsValidProgram(req.JenisProgram) {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_VALID", "jenis program harus salah satu dari: magang, asistensi_mengajar, kkn")
	}

	isAktif := existing.IsAktif
	if req.IsAktif != nil {
		isAktif = *req.IsAktif
	}

	existing.JenisProgram = req.JenisProgram
	existing.Nama = req.Nama
	existing.Alamat = req.Alamat
	existing.Kota = req.Kota
	existing.Kontak = req.Kontak
	existing.IsAktif = isAktif

	if err := s.repo.Update(ctx, s.pool, existing); err != nil {
		return nil, fmt.Errorf("update tempat program: %w", err)
	}

	return existing, nil
}

// Delete menonaktifkan tempat program secara soft-delete (is_aktif = false) sesuai RULES.md bagian 4 nomor 8.
func (s *tempatService) Delete(ctx context.Context, id int) error {
	existing, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("ambil tempat program saat delete: %w", err)
	}
	if existing == nil {
		return httpx.NotFound("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}

	if err := s.repo.SetAktif(ctx, s.pool, id, false); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NotFound("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
		}
		return fmt.Errorf("nonaktifkan tempat program: %w", err)
	}

	return nil
}

func (s *tempatService) ListPembimbing(ctx context.Context, tempatID int) ([]PembimbingLapangan, error) {
	// Pastikan tempat ada
	existing, err := s.repo.FindByID(ctx, s.pool, tempatID)
	if err != nil {
		return nil, fmt.Errorf("ambil tempat program: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}

	return s.repo.ListPembimbingByTempatID(ctx, s.pool, tempatID)
}
