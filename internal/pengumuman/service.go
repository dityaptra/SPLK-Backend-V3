package pengumuman

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// ActivityLogger mendefinisikan antarmuka pencatatan log audit aktivitas.
type ActivityLogger interface {
	Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error
}

// Service mendefinisikan seluruh use-case untuk pengelolaan pengumuman.
type Service interface {
	CreatePengumuman(ctx context.Context, actor middleware.Actor, req CreatePengumumanRequest, ip, ua string) (*PengumumanResponse, error)
	UpdatePengumuman(ctx context.Context, actor middleware.Actor, id int, req UpdatePengumumanRequest, ip, ua string) (*PengumumanResponse, error)
	DeletePengumuman(ctx context.Context, actor middleware.Actor, id int, ip, ua string) error
	GetPengumumanByID(ctx context.Context, actor middleware.Actor, id int) (*PengumumanResponse, error)
	ListPengumumanAdmin(ctx context.Context, actor middleware.Actor, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanResponse, httpx.PaginationMeta, error)
	ListPengumumanUser(ctx context.Context, actor middleware.Actor, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanResponse, httpx.PaginationMeta, error)
}

type service struct {
	repo   Repository
	pool   *pgxpool.Pool
	logger ActivityLogger
}

// NewService membuat instance baru pengumuman Service.
func NewService(repo Repository, pool *pgxpool.Pool, logger ActivityLogger) Service {
	return &service{
		repo:   repo,
		pool:   pool,
		logger: logger,
	}
}

func (s *service) CreatePengumuman(ctx context.Context, actor middleware.Actor, req CreatePengumumanRequest, ip, ua string) (*PengumumanResponse, error) {
	if !actor.HasRole("admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya admin yang dapat membuat pengumuman")
	}

	judul := strings.TrimSpace(req.Judul)
	if judul == "" {
		return nil, httpx.BadRequest("JUDUL_WAJIB", "judul pengumuman wajib diisi")
	}

	isi := strings.TrimSpace(req.Isi)
	if isi == "" {
		return nil, httpx.BadRequest("ISI_WAJIB", "isi pengumuman wajib diisi")
	}

	var targetRole *string
	if req.TargetRole != nil {
		tr := strings.TrimSpace(*req.TargetRole)
		if tr != "" {
			targetRole = &tr
		}
	}

	p := &Pengumuman{
		Judul:         judul,
		Isi:           isi,
		DibuatOleh:    &actor.UserID,
		TanggalTerbit: time.Now(),
		TargetRole:    targetRole,
	}

	if err := s.repo.Create(ctx, s.pool, p); err != nil {
		return nil, database.MapError(err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "buat_pengumuman", true, ip, ua)
	}

	detail, err := s.repo.FindByID(ctx, s.pool, p.ID)
	if err != nil {
		return nil, fmt.Errorf("ambil detail pengumuman setelah buat: %w", err)
	}
	if detail == nil {
		resp := ToResponse(PengumumanDetail{
			ID:            p.ID,
			Judul:         p.Judul,
			Isi:           p.Isi,
			DibuatOleh:    p.DibuatOleh,
			TanggalTerbit: p.TanggalTerbit,
			TargetRole:    p.TargetRole,
		})
		return &resp, nil
	}

	resp := ToResponse(*detail)
	return &resp, nil
}

func (s *service) UpdatePengumuman(ctx context.Context, actor middleware.Actor, id int, req UpdatePengumumanRequest, ip, ua string) (*PengumumanResponse, error) {
	if !actor.HasRole("admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya admin yang dapat memperbarui pengumuman")
	}

	existing, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("cek keberadaan pengumuman: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("PENGUMUMAN_TIDAK_DITEMUKAN", "pengumuman tidak ditemukan")
	}

	judul := strings.TrimSpace(req.Judul)
	if judul == "" {
		return nil, httpx.BadRequest("JUDUL_WAJIB", "judul pengumuman wajib diisi")
	}

	isi := strings.TrimSpace(req.Isi)
	if isi == "" {
		return nil, httpx.BadRequest("ISI_WAJIB", "isi pengumuman wajib diisi")
	}

	var targetRole *string
	if req.TargetRole != nil {
		tr := strings.TrimSpace(*req.TargetRole)
		if tr != "" {
			targetRole = &tr
		}
	}

	p := &Pengumuman{
		ID:         id,
		Judul:      judul,
		Isi:        isi,
		TargetRole: targetRole,
	}

	if err := s.repo.Update(ctx, s.pool, p); err != nil {
		return nil, database.MapError(err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "update_pengumuman", true, ip, ua)
	}

	updated, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil detail pengumuman setelah update: %w", err)
	}
	resp := ToResponse(*updated)
	return &resp, nil
}

func (s *service) DeletePengumuman(ctx context.Context, actor middleware.Actor, id int, ip, ua string) error {
	if !actor.HasRole("admin") {
		return httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya admin yang dapat menghapus pengumuman")
	}

	existing, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("cek keberadaan pengumuman: %w", err)
	}
	if existing == nil {
		return httpx.NotFound("PENGUMUMAN_TIDAK_DITEMUKAN", "pengumuman tidak ditemukan")
	}

	if err := s.repo.Delete(ctx, s.pool, id); err != nil {
		return database.MapError(err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "hapus_pengumuman", true, ip, ua)
	}

	return nil
}

func (s *service) GetPengumumanByID(ctx context.Context, actor middleware.Actor, id int) (*PengumumanResponse, error) {
	p, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil pengumuman: %w", err)
	}
	if p == nil {
		return nil, httpx.NotFound("PENGUMUMAN_TIDAK_DITEMUKAN", "pengumuman tidak ditemukan")
	}

	// Jika bukan admin, pastikan target_role diizinkan untuk salah satu peran aktor
	if !actor.HasRole("admin") {
		if p.TargetRole != nil && *p.TargetRole != "" && !actor.HasRole(*p.TargetRole) {
			return nil, httpx.NotFound("PENGUMUMAN_TIDAK_DITEMUKAN", "pengumuman tidak ditemukan")
		}
	}

	resp := ToResponse(*p)
	return &resp, nil
}

func (s *service) ListPengumumanAdmin(ctx context.Context, actor middleware.Actor, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanResponse, httpx.PaginationMeta, error) {
	if !actor.HasRole("admin") {
		return nil, httpx.PaginationMeta{}, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya admin yang dapat mengakses rute ini")
	}

	list, total, err := s.repo.ListAdmin(ctx, s.pool, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("daftar pengumuman admin: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}
	return ToResponses(list), meta, nil
}

func (s *service) ListPengumumanUser(ctx context.Context, actor middleware.Actor, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanResponse, httpx.PaginationMeta, error) {
	// Jika admin memanggil endpoint umum, tampilkan seluruhnya
	if actor.HasRole("admin") {
		return s.ListPengumumanAdmin(ctx, actor, filter, p)
	}

	list, total, err := s.repo.ListForRoles(ctx, s.pool, actor.Roles, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("daftar pengumuman user: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}
	return ToResponses(list), meta, nil
}
