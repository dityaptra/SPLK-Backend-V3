// Package pendaftaran menyediakan layanan resolusi relasi akses (AccessService).
package pendaftaran

import (
	"context"
	"fmt"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Relation mendefinisikan jenis hubungan aktor terhadap suatu pendaftaran.
type Relation string

const (
	RelAdmin      Relation = "admin"
	RelPegawai    Relation = "pegawai"
	RelMahasiswa  Relation = "mahasiswa"           // pemilik pendaftaran
	RelPembimbing Relation = "dosen_pembimbing"    // dosen pembimbing peserta
	RelPenguji    Relation = "dosen_penguji"       // dosen penguji peserta
	RelLapangan   Relation = "pembimbing_lapangan" // pembimbing lapangan peserta
	RelAnggotaKKN Relation = "anggota_kelompok"    // sesama anggota kelompok KKN
)

// AccessService mengevaluasi hubungan antara aktor dan pendaftaran tertentu.
type AccessService interface {
	Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (Relation, error)
}

type accessService struct {
	db   database.DBTX
	repo Repository
}

// NewAccessService membuat instance baru AccessService.
func NewAccessService(db database.DBTX, repo Repository) AccessService {
	return &accessService{
		db:   db,
		repo: repo,
	}
}

// Resolve menentukan relasi aktor terhadap pendaftaran tertentu.
// Sesuai ARCHITECTURE.md bagian 6: jika tidak memiliki relasi apa pun,
// mengembalikan error 404 (Not Found), bukan 403.
func (s *accessService) Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (Relation, error) {
	// 1. Role Admin dan Pegawai memiliki akses penuh ke seluruh pendaftaran
	if actor.HasRole("admin") {
		return RelAdmin, nil
	}
	if actor.HasRole("pegawai") {
		return RelPegawai, nil
	}

	// 2. Ambil pendaftaran dari basis data
	peserta, err := s.repo.FindPesertaByID(ctx, s.db, idPeserta)
	if err != nil {
		return "", fmt.Errorf("ambil peserta untuk otorisasi akses: %w", err)
	}
	if peserta == nil {
		return "", httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// 3. Relasi Mahasiswa
	if actor.HasRole("mahasiswa") {
		mhsID, err := s.repo.FindMahasiswaIDByUserID(ctx, s.db, actor.UserID)
		if err == nil && mhsID != nil {
			if peserta.IDMahasiswa == *mhsID {
				return RelMahasiswa, nil
			}
			// Cek apakah sesama anggota kelompok KKN
			if peserta.JenisProgram == "kkn" && peserta.IDKelompokKKN != nil {
				isAnggota, errCheck := s.repo.IsAnggotaKelompokKKN(ctx, s.db, *mhsID, *peserta.IDKelompokKKN)
				if errCheck == nil && isAnggota {
					return RelAnggotaKKN, nil
				}
			}
		}
	}

	// 4. Relasi Dosen (Pembimbing atau Penguji)
	if actor.HasRole("dosen") {
		dosenID, err := s.repo.FindDosenIDByUserID(ctx, s.db, actor.UserID)
		if err == nil && dosenID != nil {
			if peserta.IDDosenPembimbing != nil && *peserta.IDDosenPembimbing == *dosenID {
				return RelPembimbing, nil
			}
			if peserta.IDDosenPenguji != nil && *peserta.IDDosenPenguji == *dosenID {
				return RelPenguji, nil
			}
		}
	}

	// 5. Relasi Pembimbing Lapangan
	if actor.HasRole("pembimbing_lapangan") {
		plID, err := s.repo.FindPembimbingLapanganIDByUserID(ctx, s.db, actor.UserID)
		if err == nil && plID != nil {
			if peserta.IDPembimbingLapangan != nil && *peserta.IDPembimbingLapangan == *plID {
				return RelLapangan, nil
			}
		}
	}

	// 6. Tidak ada relasi yang cocok: kembalikan 404 (bukan 403) sesuai ARCHITECTURE.md bagian 6
	return "", httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
}
