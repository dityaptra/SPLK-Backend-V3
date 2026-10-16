package kkn

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

type mockKKNRepo struct {
	createKelompokFn func(ctx context.Context, db database.DBTX, k *KelompokKKN) error
	findKelompokFn   func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error)
	findDetailFn     func(ctx context.Context, db database.DBTX, id int) (*KelompokDetailResponse, error)
	listKelompokFn   func(ctx context.Context, db database.DBTX, idPeriode *int, p httpx.Pagination) ([]KelompokSummaryResponse, httpx.PaginationMeta, error)

	findPeriodeFn func(ctx context.Context, db database.DBTX, id int) (string, string, error)
	findTempatFn  func(ctx context.Context, db database.DBTX, id int) (string, bool, error)
	findDosenFn   func(ctx context.Context, db database.DBTX, id int) (bool, error)
	findPLFn      func(ctx context.Context, db database.DBTX, id int) (string, int, error)

	findPesertaFn func(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error)
	findAnggotaFn func(ctx context.Context, db database.DBTX, idKelompok int) ([]AnggotaKelompok, error)
	getKetuaFn    func(ctx context.Context, db database.DBTX, idKelompok int) (*int, error)
	assignFn      func(ctx context.Context, db database.DBTX, idPeserta int, idKelompok int, idTempat int, idDosen *int, idPL *int, isKetua bool) error
	removeFn      func(ctx context.Context, db database.DBTX, idPeserta int) error
	updateDosenFn func(ctx context.Context, db database.DBTX, idKelompok int, idDosen int) error
	updatePLFn    func(ctx context.Context, db database.DBTX, idKelompok int, idPL int) error
	checkAksesFn  func(ctx context.Context, db database.DBTX, idKelompok int, userID int, roles []string) (bool, error)
}

func (m *mockKKNRepo) CreateKelompok(ctx context.Context, db database.DBTX, k *KelompokKKN) error {
	if m.createKelompokFn != nil {
		return m.createKelompokFn(ctx, db, k)
	}
	k.ID = 100
	k.CreatedAt = time.Now()
	k.UpdatedAt = time.Now()
	return nil
}

func (m *mockKKNRepo) FindKelompokByID(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
	if m.findKelompokFn != nil {
		return m.findKelompokFn(ctx, db, id)
	}
	return nil, nil
}

func (m *mockKKNRepo) FindKelompokDetailByID(ctx context.Context, db database.DBTX, id int) (*KelompokDetailResponse, error) {
	if m.findDetailFn != nil {
		return m.findDetailFn(ctx, db, id)
	}
	return &KelompokDetailResponse{ID: id, NamaKelompok: "Kelompok Mock"}, nil
}

func (m *mockKKNRepo) ListKelompok(ctx context.Context, db database.DBTX, idPeriode *int, p httpx.Pagination) ([]KelompokSummaryResponse, httpx.PaginationMeta, error) {
	if m.listKelompokFn != nil {
		return m.listKelompokFn(ctx, db, idPeriode, p)
	}
	return nil, httpx.PaginationMeta{}, nil
}

func (m *mockKKNRepo) FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (string, string, error) {
	if m.findPeriodeFn != nil {
		return m.findPeriodeFn(ctx, db, id)
	}
	return "kkn", "aktif", nil
}

func (m *mockKKNRepo) FindTempatByID(ctx context.Context, db database.DBTX, id int) (string, bool, error) {
	if m.findTempatFn != nil {
		return m.findTempatFn(ctx, db, id)
	}
	return "kkn", true, nil
}

func (m *mockKKNRepo) FindDosenByID(ctx context.Context, db database.DBTX, id int) (bool, error) {
	if m.findDosenFn != nil {
		return m.findDosenFn(ctx, db, id)
	}
	return true, nil
}

func (m *mockKKNRepo) FindPembimbingLapanganByID(ctx context.Context, db database.DBTX, id int) (string, int, error) {
	if m.findPLFn != nil {
		return m.findPLFn(ctx, db, id)
	}
	return "kepala_desa", 10, nil
}

func (m *mockKKNRepo) FindPesertaForKKN(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
	if m.findPesertaFn != nil {
		return m.findPesertaFn(ctx, db, idPeserta)
	}
	return nil, nil
}

func (m *mockKKNRepo) FindAnggotaKelompok(ctx context.Context, db database.DBTX, idKelompok int) ([]AnggotaKelompok, error) {
	if m.findAnggotaFn != nil {
		return m.findAnggotaFn(ctx, db, idKelompok)
	}
	return nil, nil
}

func (m *mockKKNRepo) GetKetuaKelompok(ctx context.Context, db database.DBTX, idKelompok int) (*int, error) {
	if m.getKetuaFn != nil {
		return m.getKetuaFn(ctx, db, idKelompok)
	}
	return nil, nil
}

func (m *mockKKNRepo) AssignAnggota(ctx context.Context, db database.DBTX, idPeserta int, idKelompok int, idTempat int, idDosen *int, idPL *int, isKetua bool) error {
	if m.assignFn != nil {
		return m.assignFn(ctx, db, idPeserta, idKelompok, idTempat, idDosen, idPL, isKetua)
	}
	return nil
}

func (m *mockKKNRepo) RemoveAnggota(ctx context.Context, db database.DBTX, idPeserta int) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, db, idPeserta)
	}
	return nil
}

func (m *mockKKNRepo) UpdateDosenPembimbingKelompok(ctx context.Context, db database.DBTX, idKelompok int, idDosen int) error {
	if m.updateDosenFn != nil {
		return m.updateDosenFn(ctx, db, idKelompok, idDosen)
	}
	return nil
}

func (m *mockKKNRepo) UpdatePembimbingLapanganKelompok(ctx context.Context, db database.DBTX, idKelompok int, idPL int) error {
	if m.updatePLFn != nil {
		return m.updatePLFn(ctx, db, idKelompok, idPL)
	}
	return nil
}

func (m *mockKKNRepo) CheckAksesKelompok(ctx context.Context, db database.DBTX, idKelompok int, userID int, roles []string) (bool, error) {
	if m.checkAksesFn != nil {
		return m.checkAksesFn(ctx, db, idKelompok, userID, roles)
	}
	return true, nil
}

func TestKKN_CreateKelompok_BR06(t *testing.T) {
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}

	t.Run("gagal: nama kelompok kosong", func(t *testing.T) {
		repo := &mockKKNRepo{}
		svc := NewService(repo, nil, nil)
		_, err := svc.CreateKelompok(context.Background(), actorAdmin, CreateKelompokRequest{
			IDPeriode:       1,
			IDTempatProgram: 1,
			NamaKelompok:    "   ",
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 400, appErr.HTTPStatus)
		assert.Equal(t, "NAMA_KELOMPOK_KOSONG", appErr.Code)
	})

	t.Run("gagal: periode bukan KKN (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findPeriodeFn: func(ctx context.Context, db database.DBTX, id int) (string, string, error) {
				return "magang", "aktif", nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.CreateKelompok(context.Background(), actorAdmin, CreateKelompokRequest{
			IDPeriode:       1,
			IDTempatProgram: 1,
			NamaKelompok:    "Kelompok 1",
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PERIODE_BUKAN_KKN", appErr.Code)
	})

	t.Run("gagal: tempat bukan KKN (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findPeriodeFn: func(ctx context.Context, db database.DBTX, id int) (string, string, error) {
				return "kkn", "aktif", nil
			},
			findTempatFn: func(ctx context.Context, db database.DBTX, id int) (string, bool, error) {
				return "asistensi_mengajar", true, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.CreateKelompok(context.Background(), actorAdmin, CreateKelompokRequest{
			IDPeriode:       1,
			IDTempatProgram: 1,
			NamaKelompok:    "Kelompok 1",
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_BUKAN_KKN", appErr.Code)
	})

	t.Run("gagal: tempat KKN tidak aktif (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findPeriodeFn: func(ctx context.Context, db database.DBTX, id int) (string, string, error) {
				return "kkn", "aktif", nil
			},
			findTempatFn: func(ctx context.Context, db database.DBTX, id int) (string, bool, error) {
				return "kkn", false, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.CreateKelompok(context.Background(), actorAdmin, CreateKelompokRequest{
			IDPeriode:       1,
			IDTempatProgram: 1,
			NamaKelompok:    "Kelompok 1",
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_TIDAK_AKTIF", appErr.Code)
	})

	t.Run("sukses: periode kkn dan tempat kkn aktif", func(t *testing.T) {
		repo := &mockKKNRepo{
			findPeriodeFn: func(ctx context.Context, db database.DBTX, id int) (string, string, error) {
				return "kkn", "aktif", nil
			},
			findTempatFn: func(ctx context.Context, db database.DBTX, id int) (string, bool, error) {
				return "kkn", true, nil
			},
		}
		svc := NewService(repo, nil, nil)
		res, err := svc.CreateKelompok(context.Background(), actorAdmin, CreateKelompokRequest{
			IDPeriode:       1,
			IDTempatProgram: 1,
			NamaKelompok:    "Kelompok 1",
		}, "127.0.0.1", "test")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestKKN_AddAnggota_BR06_And_Ketua(t *testing.T) {
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}
	kelompokID := 10
	periodeID := 2

	t.Run("gagal: peserta jenis lain (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDPeriode: periodeID, IDTempatProgram: 5}, nil
			},
			findPesertaFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
				return &PesertaKKNInfo{
					ID:           idPeserta,
					IDPeriode:    periodeID,
					JenisProgram: "magang", // bukan KKN
					Status:       "diajukan",
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.AddAnggota(context.Background(), actorAdmin, kelompokID, AddAnggotaRequest{
			IDPesertaProgram: []int{101},
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PESERTA_BUKAN_KKN", appErr.Code)
	})

	t.Run("gagal: periode peserta berbeda (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDPeriode: periodeID, IDTempatProgram: 5}, nil
			},
			findPesertaFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
				return &PesertaKKNInfo{
					ID:           idPeserta,
					IDPeriode:    999, // periode beda
					JenisProgram: "kkn",
					Status:       "diajukan",
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.AddAnggota(context.Background(), actorAdmin, kelompokID, AddAnggotaRequest{
			IDPesertaProgram: []int{101},
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PERIODE_TIDAK_SESUAI", appErr.Code)
	})

	t.Run("gagal: status peserta bukan diajukan/berjalan (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDPeriode: periodeID, IDTempatProgram: 5}, nil
			},
			findPesertaFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
				return &PesertaKKNInfo{
					ID:           idPeserta,
					IDPeriode:    periodeID,
					JenisProgram: "kkn",
					Status:       "ditolak", // tidak valid
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.AddAnggota(context.Background(), actorAdmin, kelompokID, AddAnggotaRequest{
			IDPesertaProgram: []int{101},
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_PESERTA_TIDAK_SESUAI", appErr.Code)
	})

	t.Run("gagal: peserta sudah tergabung dalam kelompok lain (422)", func(t *testing.T) {
		otherKelompokID := 99
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDPeriode: periodeID, IDTempatProgram: 5}, nil
			},
			findPesertaFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
				return &PesertaKKNInfo{
					ID:            idPeserta,
					IDPeriode:     periodeID,
					JenisProgram:  "kkn",
					Status:        "diajukan",
					IDKelompokKKN: &otherKelompokID,
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.AddAnggota(context.Background(), actorAdmin, kelompokID, AddAnggotaRequest{
			IDPesertaProgram: []int{101},
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PESERTA_SUDAH_MEMILIKI_KELOMPOK", appErr.Code)
	})

	t.Run("gagal: penetapan ketua kedua ditolak (409 Conflict)", func(t *testing.T) {
		existingKetuaID := 101
		newKetuaID := 102
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDPeriode: periodeID, IDTempatProgram: 5}, nil
			},
			findPesertaFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
				return &PesertaKKNInfo{
					ID:           idPeserta,
					IDPeriode:    periodeID,
					JenisProgram: "kkn",
					Status:       "diajukan",
				}, nil
			},
			getKetuaFn: func(ctx context.Context, db database.DBTX, idKel int) (*int, error) {
				return &existingKetuaID, nil // sudah ada ketua 101
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.AddAnggota(context.Background(), actorAdmin, kelompokID, AddAnggotaRequest{
			IDPesertaProgram: []int{newKetuaID},
			IDKetua:          &newKetuaID, // mau ganti ketua jadi 102
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Equal(t, "KETUA_SUDAH_ADA", appErr.Code)
	})
}

func TestKKN_SetPembimbingLapangan_BR06(t *testing.T) {
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}
	kelompokID := 10
	idTempatKelompok := 50

	t.Run("gagal: bukan kepala desa (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDTempatProgram: idTempatKelompok}, nil
			},
			findPLFn: func(ctx context.Context, db database.DBTX, id int) (string, int, error) {
				return "guru_pamong", idTempatKelompok, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.SetPembimbingLapangan(context.Background(), actorAdmin, kelompokID, SetPembimbingLapanganKelompokRequest{
			IDPembimbingLapangan: 20,
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PEMBIMBING_LAPANGAN_BUKAN_KEPALA_DESA", appErr.Code)
	})

	t.Run("gagal: kepala desa dari desa/tempat berbeda (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDTempatProgram: idTempatKelompok}, nil
			},
			findPLFn: func(ctx context.Context, db database.DBTX, id int) (string, int, error) {
				return "kepala_desa", 999, nil // tempat berbeda
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.SetPembimbingLapangan(context.Background(), actorAdmin, kelompokID, SetPembimbingLapanganKelompokRequest{
			IDPembimbingLapangan: 20,
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PEMBIMBING_LAPANGAN_TEMPAT_BERBEDA", appErr.Code)
	})
}

func TestKKN_SetDosenPembimbing_BR07(t *testing.T) {
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}
	kelompokID := 10
	idDosen := 30

	t.Run("gagal: dosen pembimbing sama dengan dosen penguji salah satu anggota (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID, IDTempatProgram: 50}, nil
			},
			findDosenFn: func(ctx context.Context, db database.DBTX, id int) (bool, error) {
				return true, nil
			},
			findAnggotaFn: func(ctx context.Context, db database.DBTX, idKel int) ([]AnggotaKelompok, error) {
				return []AnggotaKelompok{
					{
						IDPesertaProgram: 1,
						NamaMahasiswa:    "Mahasiswa 1",
						IDDosenPenguji:   &idDosen, // pengujinya adalah dosen yang mau dijadikan pembimbing
					},
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		_, err := svc.SetDosenPembimbing(context.Background(), actorAdmin, kelompokID, SetDosenPembimbingKelompokRequest{
			IDDosen: idDosen,
		}, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", appErr.Code)
	})
}

func TestKKN_RemoveAnggota_StatusValidation(t *testing.T) {
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}
	kelompokID := 10
	pesertaID := 100

	t.Run("gagal: mengeluarkan anggota yang sudah berjalan ditolak (422)", func(t *testing.T) {
		repo := &mockKKNRepo{
			findKelompokFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
				return &KelompokKKN{ID: kelompokID}, nil
			},
			findPesertaFn: func(ctx context.Context, db database.DBTX, idP int) (*PesertaKKNInfo, error) {
				return &PesertaKKNInfo{
					ID:            pesertaID,
					IDKelompokKKN: &kelompokID,
					Status:        "berjalan", // sudah berjalan
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		err := svc.RemoveAnggota(context.Background(), actorAdmin, kelompokID, pesertaID, "127.0.0.1", "test")
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_TIDAK_SESUAI", appErr.Code)
	})
}

func TestKKN_GetKelompok_AccessResolution(t *testing.T) {
	kelompokID := 10

	t.Run("gagal: user tidak memiliki relasi dengan kelompok (404)", func(t *testing.T) {
		repo := &mockKKNRepo{
			checkAksesFn: func(ctx context.Context, db database.DBTX, idKel, userID int, roles []string) (bool, error) {
				return false, nil
			},
		}
		svc := NewService(repo, nil, nil)
		actorLain := middleware.Actor{UserID: 999, Roles: []string{"mahasiswa"}}
		_, err := svc.GetKelompok(context.Background(), actorLain, kelompokID)
		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 404, appErr.HTTPStatus)
		assert.Equal(t, "KELOMPOK_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("sukses: dosen pembimbing atau pembimbing lapangan atau anggota berhak melihat (200)", func(t *testing.T) {
		repo := &mockKKNRepo{
			checkAksesFn: func(ctx context.Context, db database.DBTX, idKel, userID int, roles []string) (bool, error) {
				return true, nil
			},
			findDetailFn: func(ctx context.Context, db database.DBTX, id int) (*KelompokDetailResponse, error) {
				return &KelompokDetailResponse{ID: id, NamaKelompok: "Kelompok Desa Tejakula"}, nil
			},
		}
		svc := NewService(repo, nil, nil)
		actorDosen := middleware.Actor{UserID: 50, Roles: []string{"dosen"}}
		res, err := svc.GetKelompok(context.Background(), actorDosen, kelompokID)
		require.NoError(t, err)
		assert.Equal(t, "Kelompok Desa Tejakula", res.NamaKelompok)
	})
}
