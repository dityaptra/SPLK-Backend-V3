package spd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/config"
	"splk-backend/internal/database"
	"splk-backend/internal/dokumen"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/platform/storage"
)

// ActivityLogger mendefinisikan antarmuka pencatatan log audit aktivitas.
type ActivityLogger interface {
	Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error
}

// Service mendefinisikan use-case untuk pengelolaan Surat Perjalanan Dinas.
type Service interface {
	CreateSPD(ctx context.Context, actor middleware.Actor, req CreateSPDRequest, ip, ua string) (*SPDResponse, error)
	UpdateSPD(ctx context.Context, actor middleware.Actor, id int, req UpdateSPDRequest, ip, ua string) (*SPDResponse, error)
	DeleteSPD(ctx context.Context, actor middleware.Actor, id int, ip, ua string) error
	GetSPDByID(ctx context.Context, actor middleware.Actor, id int) (*SPDResponse, error)
	ListSPD(ctx context.Context, actor middleware.Actor, filter SPDFilter, p httpx.Pagination) ([]SPDResponse, httpx.PaginationMeta, error)
	UploadFile(ctx context.Context, actor middleware.Actor, id int, filename string, size int64, fileReader io.ReadSeeker) (*SPDResponse, error)
	DownloadFile(ctx context.Context, actor middleware.Actor, id int) (io.ReadCloser, *SPDDetail, error)
}

type service struct {
	repo           Repository
	pool           *pgxpool.Pool
	storage        storage.Storage
	logger         ActivityLogger
	maxUploadBytes int64
}

// NewService membuat instance baru spd.Service.
func NewService(repo Repository, pool *pgxpool.Pool, st storage.Storage, logger ActivityLogger, cfg *config.Config) Service {
	maxBytes := int64(10 * 1024 * 1024)
	if cfg != nil && cfg.MaxUploadMB > 0 {
		maxBytes = cfg.MaxUploadMB * 1024 * 1024
	}
	return &service{
		repo:           repo,
		pool:           pool,
		storage:        st,
		logger:         logger,
		maxUploadBytes: maxBytes,
	}
}

func (s *service) CreateSPD(ctx context.Context, actor middleware.Actor, req CreateSPDRequest, ip, ua string) (*SPDResponse, error) {
	if !actor.HasRole("pegawai", "admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya pegawai atau admin yang dapat membuat surat perjalanan dinas")
	}

	pegawaiID, err := s.repo.FindPegawaiIDByUserID(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("cek profil pegawai pembuat: %w", err)
	}
	if pegawaiID == nil {
		return nil, httpx.Forbidden("PROFIL_PEGAWAI_TIDAK_DITEMUKAN", "pengguna harus memiliki profil pegawai untuk membuat surat tugas dinas")
	}

	tujuan := strings.TrimSpace(req.Tujuan)
	if tujuan == "" {
		return nil, httpx.BadRequest("TUJUAN_WAJIB", "tujuan perjalanan dinas wajib diisi")
	}

	tBerangkat, err := time.Parse("2006-01-02", req.TanggalBerangkat)
	if err != nil {
		return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal berangkat tidak valid, gunakan YYYY-MM-DD")
	}

	tKembali, err := time.Parse("2006-01-02", req.TanggalKembali)
	if err != nil {
		return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal kembali tidak valid, gunakan YYYY-MM-DD")
	}

	if tKembali.Before(tBerangkat) {
		return nil, httpx.Unprocessable("RENTANG_TANGGAL_TIDAK_VALID", "tanggal kembali tidak boleh sebelum tanggal berangkat")
	}

	dosenExists, err := s.repo.CheckDosenExists(ctx, s.pool, req.IDDosen)
	if err != nil {
		return nil, fmt.Errorf("cek keberadaan dosen: %w", err)
	}
	if !dosenExists {
		return nil, httpx.Unprocessable("DOSEN_TIDAK_DITEMUKAN", "dosen tidak ditemukan di sistem")
	}

	pesertaExists, err := s.repo.CheckPesertaExists(ctx, s.pool, req.IDPesertaProgram)
	if err != nil {
		return nil, fmt.Errorf("cek keberadaan peserta: %w", err)
	}
	if !pesertaExists {
		return nil, httpx.Unprocessable("PENDAFTARAN_TIDAK_DITEMUKAN", "peserta program tidak ditemukan di sistem")
	}

	spd := &SuratPerjalananDinas{
		IDDosen:          req.IDDosen,
		IDPesertaProgram: req.IDPesertaProgram,
		Tujuan:           tujuan,
		TanggalBerangkat: tBerangkat,
		TanggalKembali:   tKembali,
		DibuatOleh:       *pegawaiID,
	}

	if err := s.repo.Create(ctx, s.pool, spd); err != nil {
		return nil, database.MapError(err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "buat_spd", true, ip, ua)
	}

	detail, err := s.repo.FindByID(ctx, s.pool, spd.ID)
	if err != nil {
		return nil, fmt.Errorf("ambil detail spd setelah buat: %w", err)
	}
	resp := ToResponse(*detail)
	return &resp, nil
}

func (s *service) UpdateSPD(ctx context.Context, actor middleware.Actor, id int, req UpdateSPDRequest, ip, ua string) (*SPDResponse, error) {
	if !actor.HasRole("pegawai", "admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya pegawai atau admin yang dapat memperbarui surat perjalanan dinas")
	}

	existing, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil spd: %w", err)
	}
	if existing == nil {
		return nil, httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	tujuan := strings.TrimSpace(req.Tujuan)
	if tujuan == "" {
		return nil, httpx.BadRequest("TUJUAN_WAJIB", "tujuan perjalanan dinas wajib diisi")
	}

	tBerangkat, err := time.Parse("2006-01-02", req.TanggalBerangkat)
	if err != nil {
		return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal berangkat tidak valid, gunakan YYYY-MM-DD")
	}

	tKembali, err := time.Parse("2006-01-02", req.TanggalKembali)
	if err != nil {
		return nil, httpx.Unprocessable("FORMAT_TANGGAL_TIDAK_VALID", "format tanggal kembali tidak valid, gunakan YYYY-MM-DD")
	}

	if tKembali.Before(tBerangkat) {
		return nil, httpx.Unprocessable("RENTANG_TANGGAL_TIDAK_VALID", "tanggal kembali tidak boleh sebelum tanggal berangkat")
	}

	dosenExists, err := s.repo.CheckDosenExists(ctx, s.pool, req.IDDosen)
	if err != nil {
		return nil, fmt.Errorf("cek dosen: %w", err)
	}
	if !dosenExists {
		return nil, httpx.Unprocessable("DOSEN_TIDAK_DITEMUKAN", "dosen tidak ditemukan di sistem")
	}

	pesertaExists, err := s.repo.CheckPesertaExists(ctx, s.pool, req.IDPesertaProgram)
	if err != nil {
		return nil, fmt.Errorf("cek peserta: %w", err)
	}
	if !pesertaExists {
		return nil, httpx.Unprocessable("PENDAFTARAN_TIDAK_DITEMUKAN", "peserta program tidak ditemukan di sistem")
	}

	spd := &SuratPerjalananDinas{
		ID:               id,
		IDDosen:          req.IDDosen,
		IDPesertaProgram: req.IDPesertaProgram,
		Tujuan:           tujuan,
		TanggalBerangkat: tBerangkat,
		TanggalKembali:   tKembali,
	}

	if err := s.repo.Update(ctx, s.pool, spd); err != nil {
		return nil, database.MapError(err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "update_spd", true, ip, ua)
	}

	updated, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil detail spd setelah update: %w", err)
	}
	resp := ToResponse(*updated)
	return &resp, nil
}

func (s *service) DeleteSPD(ctx context.Context, actor middleware.Actor, id int, ip, ua string) error {
	if !actor.HasRole("pegawai", "admin") {
		return httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya pegawai atau admin yang dapat menghapus surat perjalanan dinas")
	}

	existing, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("ambil spd: %w", err)
	}
	if existing == nil {
		return httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	if existing.FileURL != nil && *existing.FileURL != "" {
		_ = s.storage.Delete(ctx, *existing.FileURL)
	}

	if err := s.repo.Delete(ctx, s.pool, id); err != nil {
		return database.MapError(err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "hapus_spd", true, ip, ua)
	}

	return nil
}

func (s *service) GetSPDByID(ctx context.Context, actor middleware.Actor, id int) (*SPDResponse, error) {
	spd, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil spd: %w", err)
	}
	if spd == nil {
		return nil, httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	if !s.canAccessSPD(ctx, actor, spd) {
		return nil, httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	resp := ToResponse(*spd)
	return &resp, nil
}

func (s *service) ListSPD(ctx context.Context, actor middleware.Actor, filter SPDFilter, p httpx.Pagination) ([]SPDResponse, httpx.PaginationMeta, error) {
	// Jika aktor adalah dosen dan bukan admin/pegawai, batasi hanya ke dosen terkait
	if actor.HasRole("dosen") && !actor.HasRole("admin", "pegawai") {
		dosenID, err := s.repo.FindDosenIDByUserID(ctx, s.pool, actor.UserID)
		if err != nil {
			return nil, httpx.PaginationMeta{}, fmt.Errorf("ambil id dosen: %w", err)
		}
		if dosenID == nil {
			return nil, httpx.PaginationMeta{}, httpx.Forbidden("DOSEN_TIDAK_DITEMUKAN", "profil dosen tidak ditemukan")
		}
		filter.IDDosen = *dosenID
	}

	// Jika aktor adalah mahasiswa dan bukan admin/pegawai, batasi hanya ke peserta miliknya
	if actor.HasRole("mahasiswa") && !actor.HasRole("admin", "pegawai") {
		mhsID, err := s.repo.FindMahasiswaIDByUserID(ctx, s.pool, actor.UserID)
		if err != nil {
			return nil, httpx.PaginationMeta{}, fmt.Errorf("ambil id mahasiswa: %w", err)
		}
		if mhsID == nil {
			return nil, httpx.PaginationMeta{}, httpx.Forbidden("MAHASISWA_TIDAK_DITEMUKAN", "profil mahasiswa tidak ditemukan")
		}
		// Cari id peserta program dari mahasiswa ini
		var pesertaID int
		err = s.pool.QueryRow(ctx, "SELECT id FROM peserta_program WHERE id_mahasiswa = $1 ORDER BY id DESC LIMIT 1", *mhsID).Scan(&pesertaID)
		if err != nil {
			return []SPDResponse{}, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: 0}, nil
		}
		filter.IDPesertaProgram = pesertaID
	}

	list, total, err := s.repo.List(ctx, s.pool, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("list spd: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: total,
	}
	return ToResponses(list), meta, nil
}

func (s *service) UploadFile(ctx context.Context, actor middleware.Actor, id int, filename string, size int64, fileReader io.ReadSeeker) (*SPDResponse, error) {
	if !actor.HasRole("pegawai", "admin") {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya pegawai atau admin yang dapat mengunggah berkas surat perjalanan dinas")
	}

	spd, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil spd: %w", err)
	}
	if spd == nil {
		return nil, httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	headerBuf := make([]byte, 512)
	n, err := fileReader.Read(headerBuf)
	if err != nil && err != io.EOF {
		return nil, httpx.BadRequest("BACA_FILE_GAGAL", "gagal membaca cuplikan file")
	}
	_, err = dokumen.ValidateFileContent(filename, size, s.maxUploadBytes, headerBuf[:n])
	if err != nil {
		return nil, err
	}

	if _, err := fileReader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek file: %w", err)
	}

	cleanName := dokumen.SanitizeFilename(filename)
	ext := filepath.Ext(cleanName)
	relPath := fmt.Sprintf("spd/%d_%s%s", id, uuid.New().String(), ext)

	_, err = s.storage.Save(ctx, relPath, fileReader)
	if err != nil {
		return nil, fmt.Errorf("simpan file spd: %w", err)
	}

	// Hapus file fisik lama jika ada
	if spd.FileURL != nil && *spd.FileURL != "" {
		_ = s.storage.Delete(ctx, *spd.FileURL)
	}

	if err := s.repo.UpdateFileURL(ctx, s.pool, id, relPath); err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("update file_url spd: %w", err)
	}

	updated, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil detail spd setelah upload: %w", err)
	}
	resp := ToResponse(*updated)
	return &resp, nil
}

func (s *service) DownloadFile(ctx context.Context, actor middleware.Actor, id int) (io.ReadCloser, *SPDDetail, error) {
	spd, err := s.repo.FindByID(ctx, s.pool, id)
	if err != nil {
		return nil, nil, fmt.Errorf("ambil spd: %w", err)
	}
	if spd == nil {
		return nil, nil, httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	if !s.canAccessSPD(ctx, actor, spd) {
		return nil, nil, httpx.NotFound("SPD_TIDAK_DITEMUKAN", "surat perjalanan dinas tidak ditemukan")
	}

	if spd.FileURL == nil || *spd.FileURL == "" {
		return nil, nil, httpx.NotFound("FILE_TIDAK_TERSEDIA", "dokumen berkas surat perjalanan dinas belum diunggah")
	}

	rc, err := s.storage.Open(ctx, *spd.FileURL)
	if err != nil {
		return nil, nil, httpx.NotFound("FILE_FISIK_TIDAK_DITEMUKAN", "file fisik tidak ditemukan di sistem penyimpanan")
	}

	return rc, spd, nil
}

func (s *service) canAccessSPD(ctx context.Context, actor middleware.Actor, spd *SPDDetail) bool {
	if actor.HasRole("admin", "pegawai") {
		return true
	}

	// Cek jika aktor adalah dosen yang tertera pada SPD
	if actor.HasRole("dosen") {
		dosenID, err := s.repo.FindDosenIDByUserID(ctx, s.pool, actor.UserID)
		if err == nil && dosenID != nil && *dosenID == spd.IDDosen {
			return true
		}
	}

	// Cek jika aktor adalah mahasiswa yang tertera pada peserta SPD
	if actor.HasRole("mahasiswa") {
		mhsID, err := s.repo.FindMahasiswaIDByUserID(ctx, s.pool, actor.UserID)
		if err == nil && mhsID != nil {
			var match bool
			_ = s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM peserta_program WHERE id = $1 AND id_mahasiswa = $2)", spd.IDPesertaProgram, *mhsID).Scan(&match)
			if match {
				return true
			}
		}
	}

	return false
}
