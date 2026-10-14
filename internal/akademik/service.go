package akademik

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/platform/httpx"
)

// Service mendefinisikan aturan bisnis untuk hierarki akademik.
type Service interface {
	// Fakultas
	CreateFakultas(ctx context.Context, req CreateFakultasRequest) (*Fakultas, error)
	GetFakultasByID(ctx context.Context, id int) (*Fakultas, error)
	ListFakultas(ctx context.Context) ([]Fakultas, error)
	UpdateFakultas(ctx context.Context, id int, req UpdateFakultasRequest) (*Fakultas, error)
	DeleteFakultas(ctx context.Context, id int) error

	// Jurusan
	CreateJurusan(ctx context.Context, req CreateJurusanRequest) (*Jurusan, error)
	GetJurusanByID(ctx context.Context, id int) (*Jurusan, error)
	ListJurusan(ctx context.Context, fakultasID *int) ([]Jurusan, error)
	UpdateJurusan(ctx context.Context, id int, req UpdateJurusanRequest) (*Jurusan, error)
	DeleteJurusan(ctx context.Context, id int) error

	// Program Studi
	CreateProdi(ctx context.Context, req CreateProdiRequest) (*ProgramStudi, error)
	GetProdiByID(ctx context.Context, id int) (*ProgramStudi, error)
	ListProdi(ctx context.Context, jurusanID *int) ([]ProgramStudi, error)
	UpdateProdi(ctx context.Context, id int, req UpdateProdiRequest) (*ProgramStudi, error)
	DeleteProdi(ctx context.Context, id int) error
}

type akademikService struct {
	repo Repository
	pool *pgxpool.Pool
}

// NewService membuat instance Service akademik baru.
func NewService(repo Repository, pool *pgxpool.Pool) Service {
	return &akademikService{
		repo: repo,
		pool: pool,
	}
}

// --- Fakultas ---

func (s *akademikService) CreateFakultas(ctx context.Context, req CreateFakultasRequest) (*Fakultas, error) {
	f := &Fakultas{Nama: req.Nama, Kode: req.Kode}
	if err := s.repo.CreateFakultas(ctx, s.pool, f); err != nil {
		return nil, fmt.Errorf("buat fakultas: %w", err)
	}
	return f, nil
}

func (s *akademikService) GetFakultasByID(ctx context.Context, id int) (*Fakultas, error) {
	f, err := s.repo.FindFakultasByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil fakultas: %w", err)
	}
	if f == nil {
		return nil, httpx.NotFound("FAKULTAS_TIDAK_DITEMUKAN", "fakultas tidak ditemukan")
	}
	return f, nil
}

func (s *akademikService) ListFakultas(ctx context.Context) ([]Fakultas, error) {
	return s.repo.ListFakultas(ctx, s.pool)
}

func (s *akademikService) UpdateFakultas(ctx context.Context, id int, req UpdateFakultasRequest) (*Fakultas, error) {
	existing, err := s.repo.FindFakultasByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil fakultas saat update: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("FAKULTAS_TIDAK_DITEMUKAN", "fakultas tidak ditemukan")
	}

	existing.Nama = req.Nama
	existing.Kode = req.Kode

	if err := s.repo.UpdateFakultas(ctx, s.pool, existing); err != nil {
		return nil, fmt.Errorf("update fakultas: %w", err)
	}
	return existing, nil
}

func (s *akademikService) DeleteFakultas(ctx context.Context, id int) error {
	existing, err := s.repo.FindFakultasByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("ambil fakultas saat delete: %w", err)
	}
	if existing == nil {
		return httpx.NotFound("FAKULTAS_TIDAK_DITEMUKAN", "fakultas tidak ditemukan")
	}

	count, err := s.repo.CountJurusanByFakultasID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("cek relasi jurusan: %w", err)
	}
	if count > 0 {
		return httpx.Unprocessable("FAKULTAS_MEMILIKI_JURUSAN", "fakultas masih memiliki jurusan terkait dan tidak dapat dihapus")
	}

	return s.repo.DeleteFakultas(ctx, s.pool, id)
}

// --- Jurusan ---

func (s *akademikService) CreateJurusan(ctx context.Context, req CreateJurusanRequest) (*Jurusan, error) {
	// Validasi induk ada
	fakultas, err := s.repo.FindFakultasByID(ctx, s.pool, req.FakultasID)
	if err != nil {
		return nil, fmt.Errorf("cek fakultas: %w", err)
	}
	if fakultas == nil {
		return nil, httpx.Unprocessable("FAKULTAS_TIDAK_DITEMUKAN", "fakultas yang dituju tidak ditemukan")
	}

	j := &Jurusan{
		FakultasID:  req.FakultasID,
		Nama:        req.Nama,
		KodeJurusan: req.KodeJurusan,
	}
	if err := s.repo.CreateJurusan(ctx, s.pool, j); err != nil {
		return nil, fmt.Errorf("buat jurusan: %w", err)
	}
	return j, nil
}

func (s *akademikService) GetJurusanByID(ctx context.Context, id int) (*Jurusan, error) {
	j, err := s.repo.FindJurusanByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil jurusan: %w", err)
	}
	if j == nil {
		return nil, httpx.NotFound("JURUSAN_TIDAK_DITEMUKAN", "jurusan tidak ditemukan")
	}
	return j, nil
}

func (s *akademikService) ListJurusan(ctx context.Context, fakultasID *int) ([]Jurusan, error) {
	return s.repo.ListJurusan(ctx, s.pool, fakultasID)
}

func (s *akademikService) UpdateJurusan(ctx context.Context, id int, req UpdateJurusanRequest) (*Jurusan, error) {
	existing, err := s.repo.FindJurusanByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil jurusan saat update: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("JURUSAN_TIDAK_DITEMUKAN", "jurusan tidak ditemukan")
	}

	// Validasi induk jika diubah
	if existing.FakultasID != req.FakultasID {
		fakultas, err := s.repo.FindFakultasByID(ctx, s.pool, req.FakultasID)
		if err != nil {
			return nil, fmt.Errorf("cek fakultas saat update: %w", err)
		}
		if fakultas == nil {
			return nil, httpx.Unprocessable("FAKULTAS_TIDAK_DITEMUKAN", "fakultas yang dituju tidak ditemukan")
		}
	}

	existing.FakultasID = req.FakultasID
	existing.Nama = req.Nama
	existing.KodeJurusan = req.KodeJurusan

	if err := s.repo.UpdateJurusan(ctx, s.pool, existing); err != nil {
		return nil, fmt.Errorf("update jurusan: %w", err)
	}
	return existing, nil
}

func (s *akademikService) DeleteJurusan(ctx context.Context, id int) error {
	existing, err := s.repo.FindJurusanByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("ambil jurusan saat delete: %w", err)
	}
	if existing == nil {
		return httpx.NotFound("JURUSAN_TIDAK_DITEMUKAN", "jurusan tidak ditemukan")
	}

	count, err := s.repo.CountProdiByJurusanID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("cek relasi prodi: %w", err)
	}
	if count > 0 {
		return httpx.Unprocessable("JURUSAN_MEMILIKI_PRODI", "jurusan masih memiliki program studi terkait dan tidak dapat dihapus")
	}

	return s.repo.DeleteJurusan(ctx, s.pool, id)
}

// --- Program Studi ---

func (s *akademikService) CreateProdi(ctx context.Context, req CreateProdiRequest) (*ProgramStudi, error) {
	// Validasi induk ada
	jurusan, err := s.repo.FindJurusanByID(ctx, s.pool, req.JurusanID)
	if err != nil {
		return nil, fmt.Errorf("cek jurusan: %w", err)
	}
	if jurusan == nil {
		return nil, httpx.Unprocessable("JURUSAN_TIDAK_DITEMUKAN", "jurusan yang dituju tidak ditemukan")
	}

	p := &ProgramStudi{
		JurusanID: req.JurusanID,
		Nama:      req.Nama,
		Kode:      req.Kode,
	}
	if err := s.repo.CreateProdi(ctx, s.pool, p); err != nil {
		return nil, fmt.Errorf("buat program studi: %w", err)
	}
	return p, nil
}

func (s *akademikService) GetProdiByID(ctx context.Context, id int) (*ProgramStudi, error) {
	p, err := s.repo.FindProdiByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil program studi: %w", err)
	}
	if p == nil {
		return nil, httpx.NotFound("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}
	return p, nil
}

func (s *akademikService) ListProdi(ctx context.Context, jurusanID *int) ([]ProgramStudi, error) {
	return s.repo.ListProdi(ctx, s.pool, jurusanID)
}

func (s *akademikService) UpdateProdi(ctx context.Context, id int, req UpdateProdiRequest) (*ProgramStudi, error) {
	existing, err := s.repo.FindProdiByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil prodi saat update: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}

	// Validasi induk jika diubah
	if existing.JurusanID != req.JurusanID {
		jurusan, err := s.repo.FindJurusanByID(ctx, s.pool, req.JurusanID)
		if err != nil {
			return nil, fmt.Errorf("cek jurusan saat update prodi: %w", err)
		}
		if jurusan == nil {
			return nil, httpx.Unprocessable("JURUSAN_TIDAK_DITEMUKAN", "jurusan yang dituju tidak ditemukan")
		}
	}

	existing.JurusanID = req.JurusanID
	existing.Nama = req.Nama
	existing.Kode = req.Kode

	if err := s.repo.UpdateProdi(ctx, s.pool, existing); err != nil {
		return nil, fmt.Errorf("update program studi: %w", err)
	}
	return existing, nil
}

func (s *akademikService) DeleteProdi(ctx context.Context, id int) error {
	existing, err := s.repo.FindProdiByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("ambil prodi saat delete: %w", err)
	}
	if existing == nil {
		return httpx.NotFound("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}

	count, err := s.repo.CountUsersByProdiID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("cek relasi user ke prodi: %w", err)
	}
	if count > 0 {
		return httpx.Unprocessable("PRODI_MEMILIKI_DATA", "program studi masih memiliki data mahasiswa atau dosen terkait dan tidak dapat dihapus")
	}

	return s.repo.DeleteProdi(ctx, s.pool, id)
}
