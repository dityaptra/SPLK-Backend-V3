package logbook

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

type mockLogbookRepo struct {
	infoFn      func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error)
	findDosenFn func(ctx context.Context, db database.DBTX, userID int) (*int, string, error)
	createFn    func(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error
	getFn       func(ctx context.Context, db database.DBTX, id int) (*LogbookBimbingan, string, error)
	updateFn    func(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error
	listFn      func(ctx context.Context, db database.DBTX, idPeserta int, p httpx.Pagination) ([]LogbookResponse, httpx.PaginationMeta, error)
}

func (m *mockLogbookRepo) GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
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

func (m *mockLogbookRepo) FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, string, error) {
	if m.findDosenFn != nil {
		return m.findDosenFn(ctx, db, userID)
	}
	id := 50
	return &id, "Dr. Dosen Pembimbing", nil
}

func (m *mockLogbookRepo) Create(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error {
	if m.createFn != nil {
		return m.createFn(ctx, db, l)
	}
	l.ID = 100
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()
	return nil
}

func (m *mockLogbookRepo) GetByID(ctx context.Context, db database.DBTX, id int) (*LogbookBimbingan, string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, db, id)
	}
	return nil, "", nil
}

func (m *mockLogbookRepo) Update(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, db, l)
	}
	l.UpdatedAt = time.Now()
	return nil
}

func (m *mockLogbookRepo) List(ctx context.Context, db database.DBTX, idPeserta int, p httpx.Pagination) ([]LogbookResponse, httpx.PaginationMeta, error) {
	if m.listFn != nil {
		return m.listFn(ctx, db, idPeserta, p)
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
	return pendaftaran.RelPembimbing, nil
}

func TestLogbook_Create_Rules(t *testing.T) {
	mockNow := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	mockClk := clock.NewMockClock(mockNow, time.UTC)
	actorDosen := middleware.Actor{UserID: 15, Roles: []string{"dosen"}}

	t.Run("gagal: dosen lain (bukan pembimbing pendaftaran itu) ditolak (404)", func(t *testing.T) {
		access := &mockAccessService{
			resolveFn: func(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
				return "", httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
			},
		}
		repo := &mockLogbookRepo{}
		svc := NewService(repo, nil, mockClk, access, nil)
		_, err := svc.CreateLogbook(context.Background(), actorDosen, 1, CreateLogbookRequest{
			Tanggal:        "2026-10-15",
			TopikBimbingan: "Bimbingan Perdana",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("gagal: status pendaftaran bukan berjalan (422)", func(t *testing.T) {
		repo := &mockLogbookRepo{
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
		_, err := svc.CreateLogbook(context.Background(), actorDosen, 1, CreateLogbookRequest{
			Tanggal:        "2026-10-15",
			TopikBimbingan: "Bimbingan Perdana",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_TIDAK_SESUAI", appErr.Code)
	})

	t.Run("gagal: tanggal logbook di masa depan (422)", func(t *testing.T) {
		repo := &mockLogbookRepo{}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.CreateLogbook(context.Background(), actorDosen, 1, CreateLogbookRequest{
			Tanggal:        "2026-10-17", // masa depan
			TopikBimbingan: "Bimbingan Esok",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TANGGAL_MASA_DEPAN", appErr.Code)
	})

	t.Run("gagal: tanggal di luar periode (422)", func(t *testing.T) {
		repo := &mockLogbookRepo{
			infoFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
				return &PendaftaranInfo{
					IDPesertaProgram: idPeserta,
					StatusPeserta:    pendaftaran.StatusBerjalan,
					IDPeriode:        1,
					TanggalMulai:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
					TanggalSelesai:   time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				}, nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.CreateLogbook(context.Background(), actorDosen, 1, CreateLogbookRequest{
			Tanggal:        "2026-08-15", // sebelum 1 September
			TopikBimbingan: "Bimbingan Awal",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TANGGAL_DILUAR_PERIODE", appErr.Code)
	})

	t.Run("sukses: dosen pembimbing berhasil mengisi logbook (201)", func(t *testing.T) {
		repo := &mockLogbookRepo{}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		res, err := svc.CreateLogbook(context.Background(), actorDosen, 1, CreateLogbookRequest{
			Tanggal:        "2026-10-15",
			TopikBimbingan: "Evaluasi Rencana Proker",
		}, "127.0.0.1", "test")

		require.NoError(t, err)
		assert.Equal(t, "Evaluasi Rencana Proker", res.TopikBimbingan)
		assert.Equal(t, 50, res.IDDosen)
	})
}

func TestLogbook_Update_Rules(t *testing.T) {
	mockNow := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	mockClk := clock.NewMockClock(mockNow, time.UTC)
	actorDosen := middleware.Actor{UserID: 15, Roles: []string{"dosen"}}

	t.Run("gagal: dosen lain yang bukan pembuatnya tidak bisa mengubah logbook (404)", func(t *testing.T) {
		repo := &mockLogbookRepo{
			findDosenFn: func(ctx context.Context, db database.DBTX, userID int) (*int, string, error) {
				id := 50 // dosen pemanggil ber-ID 50
				return &id, "Dr. Dosen 50", nil
			},
			getFn: func(ctx context.Context, db database.DBTX, id int) (*LogbookBimbingan, string, error) {
				return &LogbookBimbingan{
					ID:               id,
					IDPesertaProgram: 1,
					IDDosen:          99, // dibuat oleh dosen ber-ID 99
					Tanggal:          time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
					TopikBimbingan:   "Topik Lama",
				}, "Dr. Dosen 99", nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		_, err := svc.UpdateLogbook(context.Background(), actorDosen, 1, 100, UpdateLogbookRequest{
			TopikBimbingan: "Topik Baru",
		}, "127.0.0.1", "test")

		require.Error(t, err)
		appErr, ok := err.(*httpx.AppError)
		require.True(t, ok)
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("sukses: dosen pembuat dapat memperbarui logbook (200)", func(t *testing.T) {
		repo := &mockLogbookRepo{
			findDosenFn: func(ctx context.Context, db database.DBTX, userID int) (*int, string, error) {
				id := 50
				return &id, "Dr. Dosen 50", nil
			},
			getFn: func(ctx context.Context, db database.DBTX, id int) (*LogbookBimbingan, string, error) {
				return &LogbookBimbingan{
					ID:               id,
					IDPesertaProgram: 1,
					IDDosen:          50,
					Tanggal:          time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
					TopikBimbingan:   "Topik Lama",
				}, "Dr. Dosen 50", nil
			},
		}
		svc := NewService(repo, nil, mockClk, &mockAccessService{}, nil)
		res, err := svc.UpdateLogbook(context.Background(), actorDosen, 1, 100, UpdateLogbookRequest{
			TopikBimbingan: "Topik Direvisi",
		}, "127.0.0.1", "test")

		require.NoError(t, err)
		assert.Equal(t, "Topik Direvisi", res.TopikBimbingan)
	})
}
