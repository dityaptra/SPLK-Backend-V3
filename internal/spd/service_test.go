package spd_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/spd"
)

type mockSPDRepo struct {
	spds           map[int]*spd.SPDDetail
	pegawaiMap     map[int]int // userID -> pegawaiID
	dosenMap       map[int]int // userID -> dosenID
	mahasiswaMap   map[int]int // userID -> mahasiswaID
	validDosens    map[int]bool
	validPesertas  map[int]bool
	nextID         int
}

func newMockSPDRepo() *mockSPDRepo {
	return &mockSPDRepo{
		spds:          make(map[int]*spd.SPDDetail),
		pegawaiMap:    make(map[int]int),
		dosenMap:      make(map[int]int),
		mahasiswaMap:  make(map[int]int),
		validDosens:   make(map[int]bool),
		validPesertas: make(map[int]bool),
		nextID:        1,
	}
}

func (m *mockSPDRepo) Create(ctx context.Context, db database.DBTX, s *spd.SuratPerjalananDinas) error {
	s.ID = m.nextID
	m.nextID++
	s.CreatedAt = time.Now()

	d := &spd.SPDDetail{
		ID:               s.ID,
		IDDosen:          s.IDDosen,
		NamaDosen:        "Dr. Dosen Tester, M.Kom.",
		NIPDosen:         "19850101",
		IDPesertaProgram: s.IDPesertaProgram,
		NamaMahasiswa:    "Mahasiswa Tester",
		NIMMahasiswa:     "2215101001",
		JenisProgram:     "magang",
		Tujuan:           s.Tujuan,
		TanggalBerangkat: s.TanggalBerangkat,
		TanggalKembali:   s.TanggalKembali,
		FileURL:          s.FileURL,
		DibuatOleh:       s.DibuatOleh,
		NamaPegawai:      "Pegawai Admin Tester",
		CreatedAt:        s.CreatedAt,
	}
	m.spds[s.ID] = d
	return nil
}

func (m *mockSPDRepo) Update(ctx context.Context, db database.DBTX, s *spd.SuratPerjalananDinas) error {
	existing, ok := m.spds[s.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	existing.IDDosen = s.IDDosen
	existing.IDPesertaProgram = s.IDPesertaProgram
	existing.Tujuan = s.Tujuan
	existing.TanggalBerangkat = s.TanggalBerangkat
	existing.TanggalKembali = s.TanggalKembali
	return nil
}

func (m *mockSPDRepo) UpdateFileURL(ctx context.Context, db database.DBTX, id int, fileURL string) error {
	existing, ok := m.spds[id]
	if !ok {
		return pgx.ErrNoRows
	}
	existing.FileURL = &fileURL
	return nil
}

func (m *mockSPDRepo) Delete(ctx context.Context, db database.DBTX, id int) error {
	if _, ok := m.spds[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.spds, id)
	return nil
}

func (m *mockSPDRepo) FindByID(ctx context.Context, db database.DBTX, id int) (*spd.SPDDetail, error) {
	d, ok := m.spds[id]
	if !ok {
		return nil, nil
	}
	cp := *d
	return &cp, nil
}

func (m *mockSPDRepo) List(ctx context.Context, db database.DBTX, filter spd.SPDFilter, p httpx.Pagination) ([]spd.SPDDetail, int64, error) {
	var list []spd.SPDDetail
	for _, item := range m.spds {
		if filter.IDDosen > 0 && item.IDDosen != filter.IDDosen {
			continue
		}
		if filter.IDPesertaProgram > 0 && item.IDPesertaProgram != filter.IDPesertaProgram {
			continue
		}
		if filter.Search != "" && !strings.Contains(item.Tujuan, filter.Search) {
			continue
		}
		list = append(list, *item)
	}
	return list, int64(len(list)), nil
}

func (m *mockSPDRepo) FindPegawaiIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	id, ok := m.pegawaiMap[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (m *mockSPDRepo) FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	id, ok := m.dosenMap[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (m *mockSPDRepo) FindMahasiswaIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	id, ok := m.mahasiswaMap[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (m *mockSPDRepo) CheckDosenExists(ctx context.Context, db database.DBTX, idDosen int) (bool, error) {
	return m.validDosens[idDosen], nil
}

func (m *mockSPDRepo) CheckPesertaExists(ctx context.Context, db database.DBTX, idPeserta int) (bool, error) {
	return m.validPesertas[idPeserta], nil
}

type mockStorage struct {
	files map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{files: make(map[string][]byte)}
}

func (m *mockStorage) Save(ctx context.Context, relPath string, r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	m.files[relPath] = data
	return int64(len(data)), nil
}

func (m *mockStorage) Open(ctx context.Context, relPath string) (io.ReadCloser, error) {
	data, ok := m.files[relPath]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorage) Delete(ctx context.Context, relPath string) error {
	delete(m.files, relPath)
	return nil
}

type mockActivityLogger struct{}

func (m *mockActivityLogger) Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error {
	return nil
}

func TestSPDService(t *testing.T) {
	ctx := context.Background()
	repo := newMockSPDRepo()
	st := newMockStorage()
	logger := &mockActivityLogger{}
	svc := spd.NewService(repo, nil, st, logger, nil)

	// Aktor & Profile setup
	actorPegawai := middleware.Actor{UserID: 10, Roles: []string{"pegawai"}}
	repo.pegawaiMap[10] = 100 // Pegawai ID 100

	actorDosen := middleware.Actor{UserID: 20, Roles: []string{"dosen"}}
	repo.dosenMap[20] = 200 // Dosen ID 200

	actorMhs := middleware.Actor{UserID: 30, Roles: []string{"mahasiswa"}}
	repo.mahasiswaMap[30] = 300 // Mahasiswa ID 300

	actorAdminWithoutPegawai := middleware.Actor{UserID: 99, Roles: []string{"admin"}}

	repo.validDosens[200] = true
	repo.validPesertas[500] = true

	t.Run("CreateSPD - Gagal jika bukan pegawai/admin", func(t *testing.T) {
		req := spd.CreateSPDRequest{
			IDDosen:          200,
			IDPesertaProgram: 500,
			Tujuan:           "Kunjungan Monitoring PT Telkom Denpasar",
			TanggalBerangkat: "2026-11-01",
			TanggalKembali:   "2026-11-02",
		}
		_, err := svc.CreateSPD(ctx, actorMhs, req, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, "HAK_AKSES_DITOLAK", appErr.Code)
	})

	t.Run("CreateSPD - Gagal jika tidak memiliki profil pegawai", func(t *testing.T) {
		req := spd.CreateSPDRequest{
			IDDosen:          200,
			IDPesertaProgram: 500,
			Tujuan:           "Kunjungan Monitoring",
			TanggalBerangkat: "2026-11-01",
			TanggalKembali:   "2026-11-02",
		}
		_, err := svc.CreateSPD(ctx, actorAdminWithoutPegawai, req, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, "PROFIL_PEGAWAI_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("CreateSPD - Gagal jika tanggal kembali sebelum berangkat", func(t *testing.T) {
		req := spd.CreateSPDRequest{
			IDDosen:          200,
			IDPesertaProgram: 500,
			Tujuan:           "Kunjungan Monitoring",
			TanggalBerangkat: "2026-11-05",
			TanggalKembali:   "2026-11-01",
		}
		_, err := svc.CreateSPD(ctx, actorPegawai, req, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, "RENTANG_TANGGAL_TIDAK_VALID", appErr.Code)
	})

	t.Run("CreateSPD - Gagal jika dosen atau peserta tidak ada", func(t *testing.T) {
		req := spd.CreateSPDRequest{
			IDDosen:          999,
			IDPesertaProgram: 500,
			Tujuan:           "Kunjungan Monitoring",
			TanggalBerangkat: "2026-11-01",
			TanggalKembali:   "2026-11-02",
		}
		_, err := svc.CreateSPD(ctx, actorPegawai, req, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, "DOSEN_TIDAK_DITEMUKAN", appErr.Code)
	})

	var createdSPDID int
	t.Run("CreateSPD - Sukses oleh Pegawai Sah", func(t *testing.T) {
		req := spd.CreateSPDRequest{
			IDDosen:          200,
			IDPesertaProgram: 500,
			Tujuan:           "Supervisi dan Monitoring Magang Industri di Denpasar",
			TanggalBerangkat: "2026-11-10",
			TanggalKembali:   "2026-11-12",
		}
		res, err := svc.CreateSPD(ctx, actorPegawai, req, "127.0.0.1", "agent")
		require.NoError(t, err)
		assert.Positive(t, res.ID)
		assert.Equal(t, 200, res.IDDosen)
		assert.Equal(t, 500, res.IDPesertaProgram)
		assert.Equal(t, 100, res.DibuatOleh)
		createdSPDID = res.ID
	})

	t.Run("Upload & Download File SPD", func(t *testing.T) {
		// Buat dummy PDF
		dummyPDF := []byte("%PDF-1.4\n%test spd pdf content\n%%EOF")
		reader := bytes.NewReader(dummyPDF)

		uploadRes, err := svc.UploadFile(ctx, actorPegawai, createdSPDID, "surat_tugas.pdf", int64(len(dummyPDF)), reader)
		require.NoError(t, err)
		assert.NotNil(t, uploadRes.FileURL)

		// Download oleh Pegawai (sukses)
		rc, detail, err := svc.DownloadFile(ctx, actorPegawai, createdSPDID)
		require.NoError(t, err)
		assert.Equal(t, createdSPDID, detail.ID)
		readBytes, _ := io.ReadAll(rc)
		rc.Close()
		assert.Equal(t, dummyPDF, readBytes)

		// Download oleh Dosen bersangkutan (sukses)
		rcDosen, _, err := svc.DownloadFile(ctx, actorDosen, createdSPDID)
		require.NoError(t, err)
		rcDosen.Close()

		// Download oleh Dosen Lain (ditolak 404/Not Found)
		actorOtherDosen := middleware.Actor{UserID: 22, Roles: []string{"dosen"}}
		repo.dosenMap[22] = 222
		_, _, err = svc.DownloadFile(ctx, actorOtherDosen, createdSPDID)
		require.Error(t, err)
	})

	t.Run("Update & Delete SPD", func(t *testing.T) {
		reqUpdate := spd.UpdateSPDRequest{
			IDDosen:          200,
			IDPesertaProgram: 500,
			Tujuan:           "Tujuan Diperbarui: Monitoring Lanjutan",
			TanggalBerangkat: "2026-11-11",
			TanggalKembali:   "2026-11-13",
		}
		updated, err := svc.UpdateSPD(ctx, actorPegawai, createdSPDID, reqUpdate, "127.0.0.1", "agent")
		require.NoError(t, err)
		assert.Equal(t, "Tujuan Diperbarui: Monitoring Lanjutan", updated.Tujuan)

		err = svc.DeleteSPD(ctx, actorPegawai, createdSPDID, "127.0.0.1", "agent")
		require.NoError(t, err)

		_, err = svc.GetSPDByID(ctx, actorPegawai, createdSPDID)
		require.Error(t, err)
	})
}
