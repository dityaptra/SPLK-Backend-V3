// Package kkn menyediakan logika bisnis use-case cabang KKN.
package kkn

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/platform/httpx"
)

// ActivityLogger mendefinisikan antarmuka pencatatan log audit aktivitas.
type ActivityLogger interface {
	Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error
}

// Service mendefinisikan seluruh operasi use-case untuk kelompok KKN.
type Service interface {
	CreateKelompok(ctx context.Context, actor middleware.Actor, req CreateKelompokRequest, ip, ua string) (*KelompokDetailResponse, error)
	GetKelompok(ctx context.Context, actor middleware.Actor, idKelompok int) (*KelompokDetailResponse, error)
	ListKelompok(ctx context.Context, actor middleware.Actor, idPeriode *int, p httpx.Pagination) ([]KelompokSummaryResponse, httpx.PaginationMeta, error)
	AddAnggota(ctx context.Context, actor middleware.Actor, idKelompok int, req AddAnggotaRequest, ip, ua string) (*KelompokDetailResponse, error)
	RemoveAnggota(ctx context.Context, actor middleware.Actor, idKelompok int, idPeserta int, ip, ua string) error
	SetDosenPembimbing(ctx context.Context, actor middleware.Actor, idKelompok int, req SetDosenPembimbingKelompokRequest, ip, ua string) (*KelompokDetailResponse, error)
	SetPembimbingLapangan(ctx context.Context, actor middleware.Actor, idKelompok int, req SetPembimbingLapanganKelompokRequest, ip, ua string) (*KelompokDetailResponse, error)
}

type service struct {
	repo   Repository
	pool   *pgxpool.Pool
	logger ActivityLogger
}

// NewService membuat instance baru KKN Service.
func NewService(repo Repository, pool *pgxpool.Pool, logger ActivityLogger) Service {
	return &service{
		repo:   repo,
		pool:   pool,
		logger: logger,
	}
}

func (s *service) CreateKelompok(ctx context.Context, actor middleware.Actor, req CreateKelompokRequest, ip, ua string) (*KelompokDetailResponse, error) {
	if strings.TrimSpace(req.NamaKelompok) == "" {
		return nil, httpx.BadRequest("NAMA_KELOMPOK_KOSONG", "nama kelompok tidak boleh kosong")
	}

	// 1. Validasi periode: harus ada dan berjenis kkn
	jenisPeriode, _, err := s.repo.FindPeriodeByID(ctx, s.pool, req.IDPeriode)
	if err != nil {
		return nil, httpx.NotFound("PERIODE_TIDAK_DITEMUKAN", "periode program tidak ditemukan")
	}
	if jenisPeriode != "kkn" {
		return nil, httpx.Unprocessable("PERIODE_BUKAN_KKN", "periode harus berjenis kkn")
	}

	// 2. Validasi tempat: harus berjenis kkn dan aktif (BR-06)
	jenisTempat, isAktif, err := s.repo.FindTempatByID(ctx, s.pool, req.IDTempatProgram)
	if err != nil {
		return nil, httpx.NotFound("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}
	if jenisTempat != "kkn" {
		return nil, httpx.Unprocessable("TEMPAT_BUKAN_KKN", "tempat program harus berjenis kkn")
	}
	if !isAktif {
		return nil, httpx.Unprocessable("TEMPAT_TIDAK_AKTIF", "tempat program KKN tidak aktif")
	}

	kelompok := &KelompokKKN{
		IDPeriode:       req.IDPeriode,
		IDTempatProgram: req.IDTempatProgram,
		NamaKelompok:    strings.TrimSpace(req.NamaKelompok),
	}

	if err := s.repo.CreateKelompok(ctx, s.pool, kelompok); err != nil {
		return nil, fmt.Errorf("simpan kelompok kkn: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "buat_kelompok_kkn", true, ip, ua)
	}

	return s.repo.FindKelompokDetailByID(ctx, s.pool, kelompok.ID)
}

func (s *service) GetKelompok(ctx context.Context, actor middleware.Actor, idKelompok int) (*KelompokDetailResponse, error) {
	canAccess, err := s.repo.CheckAksesKelompok(ctx, s.pool, idKelompok, actor.UserID, actor.Roles)
	if err != nil {
		return nil, fmt.Errorf("cek akses kelompok: %w", err)
	}
	if !canAccess {
		return nil, httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan atau akses tidak diizinkan")
	}

	detail, err := s.repo.FindKelompokDetailByID(ctx, s.pool, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil detail kelompok: %w", err)
	}
	if detail == nil {
		return nil, httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan")
	}

	return detail, nil
}

func (s *service) ListKelompok(ctx context.Context, actor middleware.Actor, idPeriode *int, p httpx.Pagination) ([]KelompokSummaryResponse, httpx.PaginationMeta, error) {
	return s.repo.ListKelompok(ctx, s.pool, idPeriode, p)
}

func (s *service) AddAnggota(ctx context.Context, actor middleware.Actor, idKelompok int, req AddAnggotaRequest, ip, ua string) (*KelompokDetailResponse, error) {
	if len(req.IDPesertaProgram) == 0 {
		return nil, httpx.BadRequest("ANGGOTA_KOSONG", "daftar id_peserta_program tidak boleh kosong")
	}

	// 1. Ambil kelompok
	kelompok, err := s.repo.FindKelompokByID(ctx, s.pool, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil data kelompok: %w", err)
	}
	if kelompok == nil {
		return nil, httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan")
	}

	// 2. Validasi setiap calon anggota
	for _, pid := range req.IDPesertaProgram {
		peserta, err := s.repo.FindPesertaForKKN(ctx, s.pool, pid)
		if err != nil {
			return nil, fmt.Errorf("cek peserta %d: %w", pid, err)
		}
		if peserta == nil {
			return nil, httpx.NotFound("PESERTA_TIDAK_DITEMUKAN", fmt.Sprintf("peserta program ID %d tidak ditemukan", pid))
		}
		if peserta.JenisProgram != "kkn" {
			return nil, httpx.Unprocessable("PESERTA_BUKAN_KKN", fmt.Sprintf("peserta program ID %d bukan pendaftaran KKN", pid))
		}
		if peserta.IDPeriode != kelompok.IDPeriode {
			return nil, httpx.Unprocessable("PERIODE_TIDAK_SESUAI", fmt.Sprintf("peserta program ID %d memiliki periode berbeda dengan kelompok", pid))
		}
		if peserta.Status != "diajukan" && peserta.Status != "berjalan" {
			return nil, httpx.Unprocessable("STATUS_PESERTA_TIDAK_SESUAI", fmt.Sprintf("peserta program ID %d memiliki status %s yang tidak dapat dimasukkan ke kelompok", pid, peserta.Status))
		}
		if peserta.IDKelompokKKN != nil && *peserta.IDKelompokKKN != idKelompok {
			return nil, httpx.Unprocessable("PESERTA_SUDAH_MEMILIKI_KELOMPOK", fmt.Sprintf("peserta program ID %d sudah masuk dalam kelompok lain", pid))
		}

		// Validasi BR-07 bila kelompok sudah memiliki dosen pembimbing
		if kelompok.IDDosenPembimbing != nil && peserta.IDDosenPenguji != nil {
			if *peserta.IDDosenPenguji == *kelompok.IDDosenPembimbing {
				return nil, httpx.Unprocessable("DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", fmt.Sprintf("peserta program ID %d memiliki dosen penguji yang sama dengan dosen pembimbing kelompok", pid))
			}
		}
	}

	// 3. Validasi aturan ketua kelompok
	if req.IDKetua != nil && *req.IDKetua > 0 {
		ketuaID := *req.IDKetua
		existingKetua, err := s.repo.GetKetuaKelompok(ctx, s.pool, idKelompok)
		if err != nil {
			return nil, fmt.Errorf("cek ketua eksisting: %w", err)
		}
		if existingKetua != nil && *existingKetua != ketuaID {
			return nil, httpx.Conflict("KETUA_SUDAH_ADA", "kelompok KKN sudah memiliki ketua")
		}

		// Pastikan id_ketua ada dalam list penambahan atau anggota yang sudah ada
		isCalon := false
		for _, pid := range req.IDPesertaProgram {
			if pid == ketuaID {
				isCalon = true
				break
			}
		}
		if !isCalon {
			// Cek apakah sudah merupakan anggota di kelompok ini
			pKetua, err := s.repo.FindPesertaForKKN(ctx, s.pool, ketuaID)
			if err != nil || pKetua == nil || pKetua.IDKelompokKKN == nil || *pKetua.IDKelompokKKN != idKelompok {
				return nil, httpx.Unprocessable("KETUA_BUKAN_ANGGOTA", "ketua harus merupakan anggota kelompok KKN ini")
			}
		}
	}

	// 4. Eksekusi dalam transaksi
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, pid := range req.IDPesertaProgram {
		isKetua := req.IDKetua != nil && *req.IDKetua == pid
		err := s.repo.AssignAnggota(
			ctx, tx, pid, kelompok.ID, kelompok.IDTempatProgram,
			kelompok.IDDosenPembimbing, kelompok.IDPembimbingLapangan, isKetua,
		)
		if err != nil {
			if strings.Contains(err.Error(), "uq_satu_ketua_per_kelompok") {
				return nil, httpx.Conflict("KETUA_SUDAH_ADA", "kelompok KKN sudah memiliki ketua")
			}
			return nil, fmt.Errorf("assign anggota %d: %w", pid, err)
		}
	}

	// Jika ketua yang ditunjuk adalah anggota lama
	if req.IDKetua != nil && *req.IDKetua > 0 {
		ketuaID := *req.IDKetua
		isNewMember := false
		for _, pid := range req.IDPesertaProgram {
			if pid == ketuaID {
				isNewMember = true
				break
			}
		}
		if !isNewMember {
			err := s.repo.AssignAnggota(
				ctx, tx, ketuaID, kelompok.ID, kelompok.IDTempatProgram,
				kelompok.IDDosenPembimbing, kelompok.IDPembimbingLapangan, true,
			)
			if err != nil {
				if strings.Contains(err.Error(), "uq_satu_ketua_per_kelompok") {
					return nil, httpx.Conflict("KETUA_SUDAH_ADA", "kelompok KKN sudah memiliki ketua")
				}
				return nil, fmt.Errorf("set ketua anggota lama %d: %w", ketuaID, err)
			}
		}
	}

	// 5. Coba transisi kesiapan diajukan -> berjalan untuk SELURUH anggota kelompok
	anggotaList, err := s.repo.FindAnggotaKelompok(ctx, tx, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil anggota untuk transisi: %w", err)
	}
	for _, a := range anggotaList {
		if a.Status == pendaftaran.StatusDiajukan {
			_, _ = pendaftaran.TryTransitionToBerjalan(ctx, tx, a.IDPesertaProgram)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx tambah anggota: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "tambah_anggota_kkn", true, ip, ua)
	}

	return s.repo.FindKelompokDetailByID(ctx, s.pool, idKelompok)
}

func (s *service) RemoveAnggota(ctx context.Context, actor middleware.Actor, idKelompok int, idPeserta int, ip, ua string) error {
	kelompok, err := s.repo.FindKelompokByID(ctx, s.pool, idKelompok)
	if err != nil {
		return fmt.Errorf("ambil kelompok: %w", err)
	}
	if kelompok == nil {
		return httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan")
	}

	peserta, err := s.repo.FindPesertaForKKN(ctx, s.pool, idPeserta)
	if err != nil {
		return fmt.Errorf("cek peserta: %w", err)
	}
	if peserta == nil || peserta.IDKelompokKKN == nil || *peserta.IDKelompokKKN != idKelompok {
		return httpx.NotFound("ANGGOTA_TIDAK_DITEMUKAN", "peserta bukan anggota kelompok KKN ini")
	}

	// Hanya boleh dikeluarkan saat status diajukan
	if peserta.Status != pendaftaran.StatusDiajukan {
		return httpx.Unprocessable("STATUS_TIDAK_SESUAI", "anggota hanya dapat dikeluarkan saat status pendaftaran masih diajukan")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.RemoveAnggota(ctx, tx, idPeserta); err != nil {
		return fmt.Errorf("remove anggota: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx remove anggota: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "keluarkan_anggota_kkn", true, ip, ua)
	}

	return nil
}

func (s *service) SetDosenPembimbing(ctx context.Context, actor middleware.Actor, idKelompok int, req SetDosenPembimbingKelompokRequest, ip, ua string) (*KelompokDetailResponse, error) {
	kelompok, err := s.repo.FindKelompokByID(ctx, s.pool, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil kelompok: %w", err)
	}
	if kelompok == nil {
		return nil, httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan")
	}

	dosenAda, err := s.repo.FindDosenByID(ctx, s.pool, req.IDDosen)
	if err != nil || !dosenAda {
		return nil, httpx.NotFound("DOSEN_TIDAK_DITEMUKAN", "dosen tidak ditemukan")
	}

	// Validasi BR-07 untuk seluruh anggota kelompok yang sudah memiliki penguji
	anggotaList, err := s.repo.FindAnggotaKelompok(ctx, s.pool, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil anggota kelompok: %w", err)
	}
	for _, a := range anggotaList {
		if a.IDDosenPenguji != nil && *a.IDDosenPenguji == req.IDDosen {
			return nil, httpx.Unprocessable("DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", fmt.Sprintf("dosen pembimbing tidak boleh sama dengan dosen penguji pada pendaftaran peserta %s", a.NamaMahasiswa))
		}
	}

	// Satu transaksi: update kelompok_kkn dan peserta_program seluruh anggota
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpdateDosenPembimbingKelompok(ctx, tx, idKelompok, req.IDDosen); err != nil {
		return nil, fmt.Errorf("update dosen pembimbing: %w", err)
	}

	// Coba transisi kesiapan untuk seluruh anggota
	for _, a := range anggotaList {
		if a.Status == pendaftaran.StatusDiajukan {
			_, _ = pendaftaran.TryTransitionToBerjalan(ctx, tx, a.IDPesertaProgram)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx update dosen pembimbing: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "tetapkan_dosen_pembimbing_kkn", true, ip, ua)
	}

	return s.repo.FindKelompokDetailByID(ctx, s.pool, idKelompok)
}

func (s *service) SetPembimbingLapangan(ctx context.Context, actor middleware.Actor, idKelompok int, req SetPembimbingLapanganKelompokRequest, ip, ua string) (*KelompokDetailResponse, error) {
	kelompok, err := s.repo.FindKelompokByID(ctx, s.pool, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil kelompok: %w", err)
	}
	if kelompok == nil {
		return nil, httpx.NotFound("KELOMPOK_TIDAK_DITEMUKAN", "kelompok KKN tidak ditemukan")
	}

	// Validasi BR-06: pembimbing lapangan harus berjenis kepala_desa dan milik desa/tempat kelompok
	jenisPL, idTempatPL, err := s.repo.FindPembimbingLapanganByID(ctx, s.pool, req.IDPembimbingLapangan)
	if err != nil {
		return nil, httpx.NotFound("PEMBIMBING_LAPANGAN_TIDAK_DITEMUKAN", "pembimbing lapangan tidak ditemukan")
	}
	if jenisPL != "kepala_desa" {
		return nil, httpx.Unprocessable("PEMBIMBING_LAPANGAN_BUKAN_KEPALA_DESA", "pembimbing lapangan KKN harus berjenis kepala_desa")
	}
	if idTempatPL != kelompok.IDTempatProgram {
		return nil, httpx.Unprocessable("PEMBIMBING_LAPANGAN_TEMPAT_BERBEDA", "kepala desa harus berasal dari tempat/desa kelompok KKN tersebut")
	}

	// Satu transaksi: update kelompok_kkn dan peserta_program seluruh anggota
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpdatePembimbingLapanganKelompok(ctx, tx, idKelompok, req.IDPembimbingLapangan); err != nil {
		return nil, fmt.Errorf("update pembimbing lapangan: %w", err)
	}

	// Coba transisi kesiapan untuk seluruh anggota
	anggotaList, err := s.repo.FindAnggotaKelompok(ctx, tx, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil anggota kelompok: %w", err)
	}
	for _, a := range anggotaList {
		if a.Status == pendaftaran.StatusDiajukan {
			_, _ = pendaftaran.TryTransitionToBerjalan(ctx, tx, a.IDPesertaProgram)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx update pembimbing lapangan: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "tetapkan_pembimbing_lapangan_kkn", true, ip, ua)
	}

	return s.repo.FindKelompokDetailByID(ctx, s.pool, idKelompok)
}
