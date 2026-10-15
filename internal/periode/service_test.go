package periode_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/periode"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/platform/httpx"
)

// mockPeriodeRepo adalah implementasi tiruan periode.Repository untuk unit test.
type mockPeriodeRepo struct {
	periodes   map[int]*periode.Periode
	pegawaiMap map[int]int // userID -> pegawaiID
	nextID     int

	// Counter untuk memverifikasi transaksi selesaikan
	selesaikanTxCalled bool
}

func newMockPeriodeRepo() *mockPeriodeRepo {
	return &mockPeriodeRepo{
		periodes:   make(map[int]*periode.Periode),
		pegawaiMap: make(map[int]int),
		nextID:     1,
	}
}

func (m *mockPeriodeRepo) CreatePeriode(ctx context.Context, db database.DBTX, p *periode.Periode) error {
	p.ID = m.nextID
	m.nextID++
	p.CreatedAt = time.Now()
	// Simpan copy
	stored := *p
	m.periodes[p.ID] = &stored
	return nil
}

func (m *mockPeriodeRepo) FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (*periode.Periode, error) {
	p, ok := m.periodes[id]
	if !ok {
		return nil, nil
	}
	copy := *p
	return &copy, nil
}

func (m *mockPeriodeRepo) ListPeriode(ctx context.Context, db database.DBTX, filter periode.PeriodeFilter, p httpx.Pagination) ([]periode.Periode, int64, error) {
	var list []periode.Periode
	for _, item := range m.periodes {
		if filter.JenisProgram != "" && item.JenisProgram != filter.JenisProgram {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		list = append(list, *item)
	}
	return list, int64(len(list)), nil
}

func (m *mockPeriodeRepo) UpdatePeriode(ctx context.Context, db database.DBTX, id int, req periode.UpdatePeriodeRequest) error {
	p, ok := m.periodes[id]
	if !ok {
		return errors.New("not found")
	}
	mulai, _ := time.Parse("2006-01-02", req.TanggalMulai)
	selesai, _ := time.Parse("2006-01-02", req.TanggalSelesai)
	p.Nama = req.Nama
	p.TahunSemester = req.TahunSemester
	p.TanggalMulai = mulai
	p.TanggalSelesai = selesai
	if req.TanggalBukaDaftar != nil {
		t, _ := time.Parse("2006-01-02", *req.TanggalBukaDaftar)
		p.TanggalBukaDaftar = &t
	}
	if req.TanggalTutupDaftar != nil {
		t, _ := time.Parse("2006-01-02", *req.TanggalTutupDaftar)
		p.TanggalTutupDaftar = &t
	}
	return nil
}

func (m *mockPeriodeRepo) UpdateStatusPeriode(ctx context.Context, db database.DBTX, id int, status string) error {
	p, ok := m.periodes[id]
	if !ok {
		return errors.New("not found")
	}
	p.Status = status
	return nil
}

func (m *mockPeriodeRepo) FindPegawaiIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	id, ok := m.pegawaiMap[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (m *mockPeriodeRepo) SelesaikanPeriodeTx(ctx context.Context, tx pgx.Tx, periodeID int) (int64, int64, error) {
	m.selesaikanTxCalled = true
	return 5, 2, nil
}

// mockActivityLogger mencatat panggilan log aktivitas.
type mockActivityLogger struct {
	logs []string
}

func (l *mockActivityLogger) Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error {
	l.logs = append(l.logs, fmt.Sprintf("%d:%s:%t", userID, action, status))
	return nil
}

func ptrString(s string) *string {
	return &s
}

// TestCreatePeriode_Validation menguji validasi tanggal, jenis program, dan penentuan dibuat_oleh.
func TestCreatePeriode_Validation(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewRealClock(time.UTC)

	tests := []struct {
		name          string
		req           periode.CreatePeriodeRequest
		creatorUserID int
		setupPegawai  bool
		expectedErr   string
		expectPegawai bool
	}{
		{
			name: "sukses membuat periode dengan pembuat pegawai",
			req: periode.CreatePeriodeRequest{
				Nama:           "Magang MBKM Semester Ganjil 2026/2027",
				JenisProgram:   "magang",
				TahunSemester:  "2026/2027-1",
				TanggalMulai:   "2026-08-01",
				TanggalSelesai: "2026-12-31",
			},
			creatorUserID: 1,
			setupPegawai:  true,
			expectPegawai: true,
		},
		{
			name: "sukses membuat periode dengan pembuat admin murni (tanpa pegawai)",
			req: periode.CreatePeriodeRequest{
				Nama:           "KKN Tematik 2026",
				JenisProgram:   "kkn",
				TahunSemester:  "2026/2027-1",
				TanggalMulai:   "2026-07-01",
				TanggalSelesai: "2026-08-31",
			},
			creatorUserID: 2,
			setupPegawai:  false,
			expectPegawai: false,
		},
		{
			name: "sukses dengan tanggal buka dan tutup pendaftaran yang valid",
			req: periode.CreatePeriodeRequest{
				Nama:               "Asistensi Mengajar 2026",
				JenisProgram:       "asistensi_mengajar",
				TahunSemester:      "2026/2027-1",
				TanggalMulai:       "2026-09-01",
				TanggalSelesai:     "2027-01-31",
				TanggalBukaDaftar:  ptrString("2026-07-01"),
				TanggalTutupDaftar: ptrString("2026-07-25"),
			},
			creatorUserID: 1,
			setupPegawai:  true,
			expectPegawai: true,
		},
		{
			name: "gagal: jenis program tidak valid",
			req: periode.CreatePeriodeRequest{
				Nama:           "Studi Independen 2026",
				JenisProgram:   "studi_independen",
				TahunSemester:  "2026/2027-1",
				TanggalMulai:   "2026-08-01",
				TanggalSelesai: "2026-12-31",
			},
			creatorUserID: 1,
			expectedErr:   "JENIS_PROGRAM_TIDAK_VALID",
		},
		{
			name: "gagal: format tanggal mulai tidak valid",
			req: periode.CreatePeriodeRequest{
				Nama:           "Magang 2026",
				JenisProgram:   "magang",
				TahunSemester:  "2026/2027-1",
				TanggalMulai:   "01-08-2026",
				TanggalSelesai: "2026-12-31",
			},
			creatorUserID: 1,
			expectedErr:   "FORMAT_TANGGAL_TIDAK_VALID",
		},
		{
			name: "gagal: tanggal selesai lebih awal dari tanggal mulai (422)",
			req: periode.CreatePeriodeRequest{
				Nama:           "Magang 2026",
				JenisProgram:   "magang",
				TahunSemester:  "2026/2027-1",
				TanggalMulai:   "2026-08-01",
				TanggalSelesai: "2026-07-01",
			},
			creatorUserID: 1,
			expectedErr:   "TANGGAL_TIDAK_VALID",
		},
		{
			name: "gagal: tanggal tutup daftar lebih awal dari tanggal buka daftar (422)",
			req: periode.CreatePeriodeRequest{
				Nama:               "Magang 2026",
				JenisProgram:       "magang",
				TahunSemester:      "2026/2027-1",
				TanggalMulai:       "2026-08-01",
				TanggalSelesai:     "2026-12-31",
				TanggalBukaDaftar:  ptrString("2026-07-20"),
				TanggalTutupDaftar: ptrString("2026-07-10"),
			},
			creatorUserID: 1,
			expectedErr:   "TANGGAL_TIDAK_VALID",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockPeriodeRepo()
			logger := &mockActivityLogger{}
			svc := periode.NewService(repo, nil, clk, logger)

			if tc.setupPegawai {
				repo.pegawaiMap[tc.creatorUserID] = 99
			}

			res, err := svc.CreatePeriode(ctx, tc.req, tc.creatorUserID)

			if tc.expectedErr != "" {
				require.Error(t, err)
				var appErr *httpx.AppError
				require.True(t, errors.As(err, &appErr))
				assert.Equal(t, tc.expectedErr, appErr.Code)
				assert.Equal(t, 422, appErr.HTTPStatus)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, periode.StatusDraft, res.Status)
			assert.Equal(t, tc.req.Nama, res.Nama)
			assert.Equal(t, tc.req.JenisProgram, res.JenisProgram)

			// RULES.md bagian 7 nomor 2: dibuat_oleh hanya bila punya profil pegawai
			if tc.expectPegawai {
				require.NotNil(t, res.DibuatOleh)
				assert.Equal(t, 99, *res.DibuatOleh)
			} else {
				assert.Nil(t, res.DibuatOleh)
			}
		})
	}
}

// TestUpdatePeriode_DraftOnly menguji aturan bahwa update field hanya diizinkan saat status draft.
func TestUpdatePeriode_DraftOnly(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewRealClock(time.UTC)

	repo := newMockPeriodeRepo()
	logger := &mockActivityLogger{}
	svc := periode.NewService(repo, nil, clk, logger)

	// Buat 3 periode dengan status berbeda
	pDraft := &periode.Periode{
		Nama:           "Periode Draft",
		JenisProgram:   "magang",
		TahunSemester:  "2026/2027-1",
		TanggalMulai:   time.Now(),
		TanggalSelesai: time.Now().AddDate(0, 5, 0),
		Status:         periode.StatusDraft,
	}
	require.NoError(t, repo.CreatePeriode(ctx, nil, pDraft))

	pAktif := &periode.Periode{
		Nama:           "Periode Aktif",
		JenisProgram:   "magang",
		TahunSemester:  "2026/2027-1",
		TanggalMulai:   time.Now(),
		TanggalSelesai: time.Now().AddDate(0, 5, 0),
		Status:         periode.StatusAktif,
	}
	require.NoError(t, repo.CreatePeriode(ctx, nil, pAktif))

	pSelesai := &periode.Periode{
		Nama:           "Periode Selesai",
		JenisProgram:   "magang",
		TahunSemester:  "2026/2027-1",
		TanggalMulai:   time.Now(),
		TanggalSelesai: time.Now().AddDate(0, 5, 0),
		Status:         periode.StatusSelesai,
	}
	require.NoError(t, repo.CreatePeriode(ctx, nil, pSelesai))

	updateReq := periode.UpdatePeriodeRequest{
		Nama:           "Nama Baru Periode",
		TahunSemester:  "2026/2027-2",
		TanggalMulai:   "2026-09-01",
		TanggalSelesai: "2027-01-31",
	}

	t.Run("sukses update periode draft", func(t *testing.T) {
		res, err := svc.UpdatePeriode(ctx, pDraft.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Nama Baru Periode", res.Nama)
	})

	t.Run("gagal update periode aktif (422 PERIODE_BUKAN_DRAFT)", func(t *testing.T) {
		_, err := svc.UpdatePeriode(ctx, pAktif.ID, updateReq)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PERIODE_BUKAN_DRAFT", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})

	t.Run("gagal update periode selesai (422 PERIODE_BUKAN_DRAFT)", func(t *testing.T) {
		_, err := svc.UpdatePeriode(ctx, pSelesai.ID, updateReq)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PERIODE_BUKAN_DRAFT", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})

	t.Run("gagal update ID tidak ditemukan (404)", func(t *testing.T) {
		_, err := svc.UpdatePeriode(ctx, 99999, updateReq)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PERIODE_TIDAK_DITEMUKAN", appErr.Code)
		assert.Equal(t, 404, appErr.HTTPStatus)
	})
}

// TestStatusTransitions menguji aturan transisi status satu arah: draft -> aktif -> selesai.
func TestStatusTransitions(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewRealClock(time.UTC)

	repo := newMockPeriodeRepo()
	logger := &mockActivityLogger{}
	svc := periode.NewService(repo, nil, clk, logger)

	// Buat periode draft
	p := &periode.Periode{
		Nama:           "Periode Transisi",
		JenisProgram:   "magang",
		TahunSemester:  "2026/2027-1",
		TanggalMulai:   time.Now(),
		TanggalSelesai: time.Now().AddDate(0, 5, 0),
		Status:         periode.StatusDraft,
	}
	require.NoError(t, repo.CreatePeriode(ctx, nil, p))

	t.Run("aktifkan: draft -> aktif berhasil dan dicatat di log aktivitas", func(t *testing.T) {
		res, err := svc.AktifkanPeriode(ctx, p.ID, 1, "127.0.0.1", "Go-Test")
		require.NoError(t, err)
		assert.Equal(t, periode.StatusAktif, res.Status)

		// Verifikasi audit log tercatat
		require.NotEmpty(t, logger.logs)
		assert.Contains(t, logger.logs[len(logger.logs)-1], "1:aktifkan_periode:true")
	})

	t.Run("aktifkan kembali: aktif -> aktif gagal (422)", func(t *testing.T) {
		_, err := svc.AktifkanPeriode(ctx, p.ID, 1, "127.0.0.1", "Go-Test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "TRANSISI_STATUS_TIDAK_VALID", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})

	t.Run("transisi draft -> selesai langsung pada periode draft lain gagal (422)", func(t *testing.T) {
		p2 := &periode.Periode{
			Nama:           "Periode Draft Lain",
			JenisProgram:   "magang",
			TahunSemester:  "2026/2027-1",
			TanggalMulai:   time.Now(),
			TanggalSelesai: time.Now().AddDate(0, 5, 0),
			Status:         periode.StatusDraft,
		}
		require.NoError(t, repo.CreatePeriode(ctx, nil, p2))

		// Jangan jalankan dengan tx nyata di unit test karena pool = nil
		// tapi kita bisa uji validasi transisi
		assert.False(t, periode.IsTransisiValid(p2.Status, periode.StatusSelesai))
	})
}

// TestRoleVisibility_MahasiswaVsAdmin menguji batasan bahwa mahasiswa hanya melihat periode aktif (PRD 5.3).
func TestRoleVisibility_MahasiswaVsAdmin(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewRealClock(time.UTC)

	repo := newMockPeriodeRepo()
	logger := &mockActivityLogger{}
	svc := periode.NewService(repo, nil, clk, logger)

	// Buat 3 periode
	pDraft := &periode.Periode{Nama: "P-Draft", JenisProgram: "magang", TahunSemester: "2026/2027-1", TanggalMulai: time.Now(), TanggalSelesai: time.Now().AddDate(0, 5, 0), Status: periode.StatusDraft}
	require.NoError(t, repo.CreatePeriode(ctx, nil, pDraft))

	pAktif := &periode.Periode{Nama: "P-Aktif", JenisProgram: "magang", TahunSemester: "2026/2027-1", TanggalMulai: time.Now(), TanggalSelesai: time.Now().AddDate(0, 5, 0), Status: periode.StatusAktif}
	require.NoError(t, repo.CreatePeriode(ctx, nil, pAktif))

	pSelesai := &periode.Periode{Nama: "P-Selesai", JenisProgram: "magang", TahunSemester: "2026/2027-1", TanggalMulai: time.Now(), TanggalSelesai: time.Now().AddDate(0, 5, 0), Status: periode.StatusSelesai}
	require.NoError(t, repo.CreatePeriode(ctx, nil, pSelesai))

	actorMahasiswa := middleware.Actor{UserID: 10, Roles: []string{"mahasiswa"}}
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}
	actorPegawai := middleware.Actor{UserID: 2, Roles: []string{"pegawai"}}

	t.Run("mahasiswa GetByID periode draft mengembalikan 404", func(t *testing.T) {
		_, err := svc.GetPeriodeByID(ctx, pDraft.ID, actorMahasiswa)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("mahasiswa GetByID periode selesai mengembalikan 404", func(t *testing.T) {
		_, err := svc.GetPeriodeByID(ctx, pSelesai.ID, actorMahasiswa)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("mahasiswa GetByID periode aktif mengembalikan data", func(t *testing.T) {
		res, err := svc.GetPeriodeByID(ctx, pAktif.ID, actorMahasiswa)
		require.NoError(t, err)
		assert.Equal(t, pAktif.ID, res.ID)
	})

	t.Run("admin GetByID dapat melihat draft dan selesai", func(t *testing.T) {
		res1, err1 := svc.GetPeriodeByID(ctx, pDraft.ID, actorAdmin)
		require.NoError(t, err1)
		assert.Equal(t, pDraft.ID, res1.ID)

		res2, err2 := svc.GetPeriodeByID(ctx, pSelesai.ID, actorAdmin)
		require.NoError(t, err2)
		assert.Equal(t, pSelesai.ID, res2.ID)
	})

	t.Run("pegawai GetByID dapat melihat draft", func(t *testing.T) {
		res, err := svc.GetPeriodeByID(ctx, pDraft.ID, actorPegawai)
		require.NoError(t, err)
		assert.Equal(t, pDraft.ID, res.ID)
	})

	t.Run("mahasiswa ListPeriode otomatis hanya mengembalikan periode aktif", func(t *testing.T) {
		// Walaupun mahasiswa meminta filter status='draft', service memaksanya ke 'aktif'
		items, _, err := svc.ListPeriode(ctx, periode.PeriodeFilter{Status: "draft"}, httpx.Pagination{Page: 1, Limit: 10}, actorMahasiswa)
		require.NoError(t, err)
		for _, it := range items {
			assert.Equal(t, periode.StatusAktif, it.Status)
		}
	})

	t.Run("admin ListPeriode dapat melihat seluruh periode", func(t *testing.T) {
		items, _, err := svc.ListPeriode(ctx, periode.PeriodeFilter{}, httpx.Pagination{Page: 1, Limit: 10}, actorAdmin)
		require.NoError(t, err)
		assert.Len(t, items, 3)
	})
}

// TestStatusTransitionHelpers menguji fungsi pembantu IsTransisiValid dan CanUpdate.
func TestStatusTransitionHelpers(t *testing.T) {
	// IsTransisiValid
	assert.True(t, periode.IsTransisiValid(periode.StatusDraft, periode.StatusAktif))
	assert.True(t, periode.IsTransisiValid(periode.StatusAktif, periode.StatusSelesai))

	assert.False(t, periode.IsTransisiValid(periode.StatusDraft, periode.StatusSelesai))
	assert.False(t, periode.IsTransisiValid(periode.StatusAktif, periode.StatusDraft))
	assert.False(t, periode.IsTransisiValid(periode.StatusSelesai, periode.StatusAktif))
	assert.False(t, periode.IsTransisiValid(periode.StatusSelesai, periode.StatusDraft))
	assert.False(t, periode.IsTransisiValid("unknown", periode.StatusAktif))

	// CanUpdate
	assert.True(t, periode.CanUpdate(periode.StatusDraft))
	assert.False(t, periode.CanUpdate(periode.StatusAktif))
	assert.False(t, periode.CanUpdate(periode.StatusSelesai))
	assert.False(t, periode.CanUpdate("anything_else"))
}
