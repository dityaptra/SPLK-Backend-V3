package dokumen

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"splk-backend/internal/config"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/platform/storage"
)

// Service mendefinisikan antarmuka logika bisnis untuk dokumen.
type Service interface {
	UploadPesertaDokumen(ctx context.Context, actor middleware.Actor, idPeserta int, jenisDokumen string, filename string, size int64, fileReader io.ReadSeeker) (*DokumenResponse, error)
	UploadKelompokDokumen(ctx context.Context, actor middleware.Actor, idKelompok int, jenisDokumen string, filename string, size int64, fileReader io.ReadSeeker) (*DokumenResponse, error)
	ListPesertaDokumen(ctx context.Context, actor middleware.Actor, idPeserta int) ([]DokumenResponse, error)
	ListKelompokDokumen(ctx context.Context, actor middleware.Actor, idKelompok int) ([]DokumenResponse, error)
	DownloadDokumen(ctx context.Context, actor middleware.Actor, idDokumen int) (io.ReadCloser, *Dokumen, error)
	DeleteDokumen(ctx context.Context, actor middleware.Actor, idDokumen int) error
}

type service struct {
	repo           Repository
	pool           *pgxpool.Pool
	storage        storage.Storage
	accessSvc      AccessResolver
	maxUploadBytes int64
}

// NewService membuat instance baru dokumen Service.
func NewService(repo Repository, pool *pgxpool.Pool, st storage.Storage, accessSvc AccessResolver, cfg *config.Config) Service {
	maxBytes := int64(5 * 1024 * 1024)
	if cfg != nil && cfg.MaxUploadMB > 0 {
		maxBytes = cfg.MaxUploadMB * 1024 * 1024
	}
	return &service{
		repo:           repo,
		pool:           pool,
		storage:        st,
		accessSvc:      accessSvc,
		maxUploadBytes: maxBytes,
	}
}

func (s *service) UploadPesertaDokumen(ctx context.Context, actor middleware.Actor, idPeserta int, jenisDokumen string, filename string, size int64, fileReader io.ReadSeeker) (*DokumenResponse, error) {
	// 1. Resolusi hak akses pendaftaran
	rel, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}
	// Hak unggah pendaftaran: hanya mahasiswa pemilik atau admin
	if rel != pendaftaran.RelMahasiswa && rel != pendaftaran.RelAdmin {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya mahasiswa pemilik atau admin yang dapat mengunggah dokumen pendaftaran")
	}

	// 2. Ambil informasi status dan jenis program pendaftaran
	info, err := s.repo.GetPendaftaranInfo(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("ambil info pendaftaran: %w", err)
	}
	if info == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// Dokumen hanya boleh diunggah saat status diajukan atau berjalan
	if info.Status != pendaftaran.StatusDiajukan && info.Status != pendaftaran.StatusBerjalan {
		return nil, httpx.Unprocessable("STATUS_TIDAK_VALID", "dokumen hanya dapat diunggah saat pendaftaran berstatus diajukan atau berjalan")
	}

	// 3. Validasi aturan jenis dokumen terhadap jenis program
	if err := ValidateDokumenJenisForProgram(info.JenisProgram, jenisDokumen, false); err != nil {
		return nil, err
	}

	// 4. Validasi isi file dan konsistensi ekstensi
	headerBuf := make([]byte, 512)
	n, err := fileReader.Read(headerBuf)
	if err != nil && err != io.EOF {
		return nil, httpx.BadRequest("BACA_FILE_GAGAL", "gagal membaca cuplikan file")
	}
	mimeType, err := ValidateFileContent(filename, size, s.maxUploadBytes, headerBuf[:n])
	if err != nil {
		return nil, err
	}

	// Kembalikan posisi pembacaan file ke awal
	if _, err := fileReader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek file: %w", err)
	}

	// 5. Sanitasi nama asli dan tentukan path penyimpanan aman tanpa memuat nama asli
	cleanName := SanitizeFilename(filename)
	ext := filepath.Ext(cleanName)
	relPath := GeneratePesertaStoragePath(idPeserta, ext)

	// 6. Simpan fisik file ke storage
	_, err = s.storage.Save(ctx, relPath, fileReader)
	if err != nil {
		return nil, fmt.Errorf("simpan file ke storage: %w", err)
	}

	// 7. Simpan metadata ke basis data dalam transaksi
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	doc := &Dokumen{
		IDPesertaProgram: &idPeserta,
		JenisDokumen:     jenisDokumen,
		NamaFile:         cleanName,
		FileURL:          relPath,
		UkuranFile:       size,
		MimeType:         mimeType,
		DiunggahOleh:     actor.UserID,
	}

	if err := s.repo.Create(ctx, tx, doc); err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("simpan metadata dokumen: %w", err)
	}

	// 8. Coba transisi kesiapan diajukan -> berjalan jika status pendaftaran diajukan
	if info.Status == pendaftaran.StatusDiajukan {
		_, _ = pendaftaran.TryTransitionToBerjalan(ctx, tx, idPeserta)
	}

	if err := tx.Commit(ctx); err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("commit tx dokumen: %w", err)
	}

	resp := ToDokumenResponse(*doc)
	return &resp, nil
}

func (s *service) UploadKelompokDokumen(ctx context.Context, actor middleware.Actor, idKelompok int, jenisDokumen string, filename string, size int64, fileReader io.ReadSeeker) (*DokumenResponse, error) {
	// 1. Otorisasi akses tulis kelompok (anggota kelompok atau admin)
	canWrite, err := s.repo.CheckKelompokAccess(ctx, s.pool, idKelompok, actor.UserID, actor.Roles, true)
	if err != nil {
		return nil, fmt.Errorf("cek akses kelompok: %w", err)
	}
	if !canWrite {
		return nil, httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya anggota kelompok atau admin yang dapat mengunggah dokumen kelompok KKN")
	}

	// 2. Validasi aturan jenis dokumen untuk kelompok KKN
	if err := ValidateDokumenJenisForProgram("kkn", jenisDokumen, true); err != nil {
		return nil, err
	}

	// 3. Validasi isi file dan konsistensi ekstensi
	headerBuf := make([]byte, 512)
	n, err := fileReader.Read(headerBuf)
	if err != nil && err != io.EOF {
		return nil, httpx.BadRequest("BACA_FILE_GAGAL", "gagal membaca cuplikan file")
	}
	mimeType, err := ValidateFileContent(filename, size, s.maxUploadBytes, headerBuf[:n])
	if err != nil {
		return nil, err
	}

	if _, err := fileReader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek file: %w", err)
	}

	cleanName := SanitizeFilename(filename)
	ext := filepath.Ext(cleanName)
	relPath := GenerateKelompokStoragePath(idKelompok, ext)

	_, err = s.storage.Save(ctx, relPath, fileReader)
	if err != nil {
		return nil, fmt.Errorf("simpan file ke storage: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	doc := &Dokumen{
		IDKelompokKKN: &idKelompok,
		JenisDokumen:  jenisDokumen,
		NamaFile:      cleanName,
		FileURL:       relPath,
		UkuranFile:    size,
		MimeType:      mimeType,
		DiunggahOleh:  actor.UserID,
	}

	if err := s.repo.Create(ctx, tx, doc); err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("simpan metadata dokumen kelompok: %w", err)
	}

	// Coba transisi kesiapan diajukan -> berjalan untuk seluruh anggota kelompok
	if memberIDs, err := s.repo.GetKelompokMemberIDs(ctx, tx, idKelompok); err == nil {
		for _, mID := range memberIDs {
			_, _ = pendaftaran.TryTransitionToBerjalan(ctx, tx, mID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		_ = s.storage.Delete(ctx, relPath)
		return nil, fmt.Errorf("commit tx dokumen kelompok: %w", err)
	}

	resp := ToDokumenResponse(*doc)
	return &resp, nil
}

func (s *service) ListPesertaDokumen(ctx context.Context, actor middleware.Actor, idPeserta int) ([]DokumenResponse, error) {
	// Resolusi relasi: jika tidak memiliki hubungan dengan pendaftaran, kembalikan 404
	_, err := s.accessSvc.Resolve(ctx, actor, idPeserta)
	if err != nil {
		return nil, err
	}

	list, err := s.repo.ListByPesertaID(ctx, s.pool, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("list dokumen peserta: %w", err)
	}

	return ToDokumenResponses(list), nil
}

func (s *service) ListKelompokDokumen(ctx context.Context, actor middleware.Actor, idKelompok int) ([]DokumenResponse, error) {
	// Resolusi akses baca kelompok (anggota kelompok, dosen pembimbing, pembimbing lapangan, admin, pegawai)
	canRead, err := s.repo.CheckKelompokAccess(ctx, s.pool, idKelompok, actor.UserID, actor.Roles, false)
	if err != nil {
		return nil, fmt.Errorf("cek akses kelompok: %w", err)
	}
	if !canRead {
		return nil, httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan atau akses tidak diizinkan")
	}

	list, err := s.repo.ListByKelompokID(ctx, s.pool, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("list dokumen kelompok: %w", err)
	}

	return ToDokumenResponses(list), nil
}

func (s *service) DownloadDokumen(ctx context.Context, actor middleware.Actor, idDokumen int) (io.ReadCloser, *Dokumen, error) {
	doc, err := s.repo.GetByID(ctx, s.pool, idDokumen)
	if err != nil {
		return nil, nil, fmt.Errorf("ambil metadata dokumen: %w", err)
	}
	if doc == nil {
		return nil, nil, httpx.NotFound("DOKUMEN_TIDAK_DITEMUKAN", "dokumen tidak ditemukan")
	}

	// Periksa hak akses berdasarkan pemilik dokumen (peserta atau kelompok)
	if doc.IDPesertaProgram != nil {
		_, err := s.accessSvc.Resolve(ctx, actor, *doc.IDPesertaProgram)
		if err != nil {
			return nil, nil, err
		}
	} else if doc.IDKelompokKKN != nil {
		canRead, err := s.repo.CheckKelompokAccess(ctx, s.pool, *doc.IDKelompokKKN, actor.UserID, actor.Roles, false)
		if err != nil {
			return nil, nil, fmt.Errorf("cek akses kelompok: %w", err)
		}
		if !canRead {
			return nil, nil, httpx.NotFound("DOKUMEN_TIDAK_DITEMUKAN", "dokumen tidak ditemukan atau akses tidak diizinkan")
		}
	}

	rc, err := s.storage.Open(ctx, doc.FileURL)
	if err != nil {
		return nil, nil, httpx.NotFound("FILE_TIDAK_DITEMUKAN", "file fisik tidak ditemukan di sistem penyimpanan")
	}

	return rc, doc, nil
}

func (s *service) DeleteDokumen(ctx context.Context, actor middleware.Actor, idDokumen int) error {
	doc, err := s.repo.GetByID(ctx, s.pool, idDokumen)
	if err != nil {
		return fmt.Errorf("ambil dokumen: %w", err)
	}
	if doc == nil {
		return httpx.NotFound("DOKUMEN_TIDAK_DITEMUKAN", "dokumen tidak ditemukan")
	}

	// Hanya pengunggah asli yang boleh menghapus dokumen
	if doc.DiunggahOleh != actor.UserID {
		return httpx.Forbidden("HAK_AKSES_DITOLAK", "hanya pengunggah yang dapat menghapus dokumen ini")
	}

	var idPeserta int
	if doc.IDPesertaProgram != nil {
		idPeserta = *doc.IDPesertaProgram
		info, err := s.repo.GetPendaftaranInfo(ctx, s.pool, idPeserta)
		if err != nil {
			return fmt.Errorf("cek status pendaftaran: %w", err)
		}
		if info == nil || info.Status != pendaftaran.StatusDiajukan {
			return httpx.Unprocessable("STATUS_TIDAK_VALID", "dokumen hanya dapat dihapus saat status pendaftaran diajukan")
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Delete(ctx, tx, idDokumen); err != nil {
		return fmt.Errorf("delete dokumen dari basis data: %w", err)
	}

	if idPeserta > 0 {
		_, _ = pendaftaran.TryTransitionToBerjalan(ctx, tx, idPeserta)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx delete dokumen: %w", err)
	}

	// Hapus file fisik dari storage
	_ = s.storage.Delete(ctx, doc.FileURL)

	return nil
}
