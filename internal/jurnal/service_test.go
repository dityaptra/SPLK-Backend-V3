package jurnal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/platform/httpx"
)

type mockJurnalRepo struct {
	infoFn   func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error)
	existsFn func(ctx context.Context, db database.DBTX, idPeserta int, tanggal time.Time, excludeID int) (bool, error)
	createFn func(ctx context.Context, db database.DBTX, j *JurnalHarian) error
	getFn    func(ctx context.Context, db database.DBTX, id int) (*JurnalHarian, error)
	updateFn func(ctx context.Context, db database.DBTX, j *JurnalHarian) error
	listFn   func(ctx context.Context, db database.DBTX, idPeserta int, filter JurnalFilter, p httpx.Pagination) ([]JurnalHarian, httpx.PaginationMeta, error)
}

func (m *mockJurnalRepo) GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
	if m.infoFn != nil {
		return m.infoFn(ctx, db, idPeserta)
	}
	return &PendaftaranInfo{
		IDPesertaProgram: idPeserta,
		StatusPeserta:    pendaftaran.StatusBerjalan,
		IDPeriode:        1,
		TanggalMulai:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		TanggalSelesai:   time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
	}, nil
}

func (m *mockJurnalRepo) ExistsByDate(ctx context.Context, db database.DBTX, idPeserta int, tanggal time.Time, excludeID int) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, db, idPeserta, tanggal, excludeID)
	}
	return false, nil
}

func (m *mockJurnalRepo) Create(ctx context.Context, db database.DBTX, j *JurnalHarian) error {
	if m.createFn != nil {
		return m.createFn(ctx, db, j)
	}
	j.ID = 10
	j.CreatedAt = time.Now()
	j.UpdatedAt = time.Now()
	return nil
}

func (m *mockJurnalRepo) GetByID(ctx context.Context, db database.DBTX, id int) (*JurnalHarian, error) {
	if m.getFn != nil {
		return m.getFn(ctx, db, id)
	}
	return nil, nil
}

func (m *mockJurnalRepo) Update(ctx context.Context, db database.DBTX, j *JurnalHarian) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, db, j)
	}
	j.UpdatedAt = time.Now()
	return nil
}

func (m *mockJurnalRepo) List(ctx context.Context, db database.DBTX, idPeserta int, filter JurnalFilter, p httpx.Pagination) ([]JurnalHarian, httpx.PaginationMeta, error) {
	if m.listFn != nil {
		return m.listFn(ctx, db, idPeserta, filter, p)
	}
	return nil, httpx.PaginationMeta{}, nil
}

type mockAccessService struct {
	resolveFn func(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error)
}

func (m *mockAccessService) Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
	if m.resolveFn != nil {
		return m.resolveFn(ctx, actor, idPeserta)
	}
	return pendaftaran.RelMahasiswa, nil
}

func TestJurnal_Create_Rules(t *testing.T) {
	mockNow := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	mockClk := clock.NewMockClock(mockNow, time.UTC)
	actorMhs := middleware.Actor{UserID: 2, Roles: []string{"mahasiswa"}}

	t.Run("gagal: status pendaftaran bukan berjalan (422)", func(t *testing.T) {
		repo := &mockJurnalRepo{
			infoFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
				return &PendaftaranInfo{
					IDPesertaProgram: idPeserta,
					StatusPeserta:    pendaftaran.StatusDiajukan, // belum berjalan
					IDPeriode:        1,
					TanggalMulai:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
					TanggalSelesai:   time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				}, nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.CreateJurnal(context.Background(), actorMhs, 1, CreateJurnalRequest{
			Tanggal: "2026-10-15",
			Judul:   "Hari Pertama",
			Isi:     "Kegiatan...",
			Status:  "draft",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_TIDAK_SESUAI", appErr.Code)
	})

	t.Run("gagal: tanggal jurnal di masa depan (422)", func(t *testing.T) {
		repo := &mockJurnalRepo{}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.CreateJurnal(context.Background(), actorMhs, 1, CreateJurnalRequest{
			Tanggal: "2026-10-17", // masa depan (hari ini 2026-10-16)
			Judul:   "Hari Esok",
			Isi:     "Kegiatan...",
			Status:  "draft",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TANGGAL_MASA_DEPAN", appErr.Code)
	})

	t.Run("gagal: tanggal di luar rentang periode (422)", func(t *testing.T) {
		repo := &mockJurnalRepo{
			infoFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
				return &PendaftaranInfo{
					IDPesertaProgram: idPeserta,
					StatusPeserta:    pendaftaran.StatusBerjalan,
					IDPeriode:        1,
					TanggalMulai:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), // mulai 1 September
					TanggalSelesai:   time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				}, nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.CreateJurnal(context.Background(), actorMhs, 1, CreateJurnalRequest{
			Tanggal: "2026-08-15", // sebelum 1 September
			Judul:   "Terlalu Awal",
			Isi:     "Kegiatan...",
			Status:  "draft",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TANGGAL_DILUAR_PERIODE", appErr.Code)
	})

	t.Run("gagal: duplikat tanggal untuk peserta yang sama - BR-09 (409 Conflict)", func(t *testing.T) {
		repo := &mockJurnalRepo{
			existsFn: func(ctx context.Context, db database.DBTX, idPeserta int, tanggal time.Time, excludeID int) (bool, error) {
				return true, nil // sudah ada jurnal tanggal ini
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.CreateJurnal(context.Background(), actorMhs, 1, CreateJurnalRequest{
			Tanggal: "2026-10-15",
			Judul:   "Hari yang Sama",
			Isi:     "Kegiatan...",
			Status:  "draft",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Equal(t, "JURNAL_DUPLIKAT_TANGGAL", appErr.Code)
	})

	t.Run("gagal: bukan pemilik pendaftaran (403)", func(t *testing.T) {
		access := &mockAccessService{
			resolveFn: func(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
				return pendaftaran.RelPembimbing, nil // dosen, bukan mahasiswa pemilik
			},
		}
		repo := &mockJurnalRepo{}
		svc := NewService(repo, nil, mockClk, access, nil)
		_, err := svc.CreateJurnal(context.Background(), actorMhs, 1, CreateJurnalRequest{
			Tanggal: "2026-10-15",
			Judul:   "Hari Pertama",
			Isi:     "Kegiatan...",
			Status:  "draft",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 403, appErr.HTTPStatus)
		assert.Equal(t, "HAK_AKSES_DITOLAK", appErr.Code)
	})

	t.Run("sukses: jurnal draft berhasil dibuat (201)", func(t *testing.T) {
		repo := &mockJurnalRepo{}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		res, err := svc.CreateJurnal(context.Background(), actorMhs, 1, CreateJurnalRequest{
			Tanggal: "2026-10-15",
			Judul:   "Hari Pertama",
			Isi:     "Kegiatan berjalan lancar.",
			Status:  "draft",
		}, "127.0.0.1", "test")

		require.NoError(t, err)
		assert.Equal(t, "Hari Pertama", res.Judul)
		assert.Equal(t, "draft", res.Status)
	})
}

func TestJurnal_Update_Rules(t *testing.T) {
	mockNow := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	mockClk := clock.NewMockClock(mockNow, time.UTC)
	actorMhs := middleware.Actor{UserID: 2, Roles: []string{"mahasiswa"}}

	t.Run("gagal: edit jurnal yang sudah disubmit ditolak (422)", func(t *testing.T) {
		repo := &mockJurnalRepo{
			getFn: func(ctx context.Context, db database.DBTX, id int) (*JurnalHarian, error) {
				return &JurnalHarian{
					ID:               id,
					IDPesertaProgram: 1,
					Tanggal:          time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
					Status:           StatusDisubmit, // sudah disubmit!
				}, nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.UpdateJurnal(context.Background(), actorMhs, 1, 10, UpdateJurnalRequest{
			Judul:  "Revisi Judul",
			Isi:    "Revisi Isi",
			Status: "disubmit",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "JURNAL_BUKAN_DRAFT", appErr.Code)
	})

	t.Run("sukses: edit jurnal berstatus draft", func(t *testing.T) {
		repo := &mockJurnalRepo{
			getFn: func(ctx context.Context, db database.DBTX, id int) (*JurnalHarian, error) {
				return &JurnalHarian{
					ID:               id,
					IDPesertaProgram: 1,
					Tanggal:          time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
					Judul:            "Judul Asli",
					Isi:              "Isi Asli",
					Status:           StatusDraft,
				}, nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		res, err := svc.UpdateJurnal(context.Background(), actorMhs, 1, 10, UpdateJurnalRequest{
			Judul:  "Judul Diubah",
			Isi:    "Isi Diubah",
			Status: "disubmit",
		}, "127.0.0.1", "test")

		require.NoError(t, err)
		assert.Equal(t, "Judul Diubah", res.Judul)
		assert.Equal(t, "disubmit", res.Status)
	})
}
