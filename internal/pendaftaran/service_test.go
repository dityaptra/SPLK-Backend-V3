package pendaftaran_test

import (
	"context"
	"errors"
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

// mockPendaftaranRepo mengimplementasikan pendaftaran.Repository untuk unit testing.
type mockPendaftaranRepo struct {
	pesertas       map[int]*pendaftaran.PesertaProgram
	mahasiswas     map[int]*pendaftaran.MahasiswaInfo // userID -> MahasiswaInfo
	dosens         map[int]int                        // userID -> dosenID
	pembimbings    map[int]int                        // userID -> plID
	periodes       map[int]*pendaftaran.PeriodeInfo
	periodeDates   map[int][2]*time.Time // [0] = buka, [1] = tutup
	docsByPeserta  map[int][]string
	tempatPrograms map[int]*pendaftaran.TempatProgramCheck
	pembimbingLaps map[int]*pendaftaran.PembimbingLapanganCheck
	dosensCheck    map[int]*pendaftaran.DosenCheck
	nextID         int
}

func newMockPendaftaranRepo() *mockPendaftaranRepo {
	return &mockPendaftaranRepo{
		pesertas:       make(map[int]*pendaftaran.PesertaProgram),
		mahasiswas:     make(map[int]*pendaftaran.MahasiswaInfo),
		dosens:         make(map[int]int),
		pembimbings:    make(map[int]int),
		periodes:       make(map[int]*pendaftaran.PeriodeInfo),
		periodeDates:   make(map[int][2]*time.Time),
		docsByPeserta:  make(map[int][]string),
		tempatPrograms: make(map[int]*pendaftaran.TempatProgramCheck),
		pembimbingLaps: make(map[int]*pendaftaran.PembimbingLapanganCheck),
		dosensCheck:    make(map[int]*pendaftaran.DosenCheck),
		nextID:         1,
	}
}

func (m *mockPendaftaranRepo) CreatePeserta(ctx context.Context, db database.DBTX, p *pendaftaran.PesertaProgram) error {
	p.ID = m.nextID
	m.nextID++
	p.TanggalDaftar = time.Now()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	stored := *p
	m.pesertas[p.ID] = &stored
	return nil
}

func (m *mockPendaftaranRepo) FindPesertaByID(ctx context.Context, db database.DBTX, id int) (*pendaftaran.PesertaProgram, error) {
	p, ok := m.pesertas[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *mockPendaftaranRepo) FindPesertaByMahasiswaAndPeriode(ctx context.Context, db database.DBTX, mhsID, periodeID int, jenisProgram string) (*pendaftaran.PesertaProgram, error) {
	for _, p := range m.pesertas {
		if p.IDMahasiswa == mhsID && p.IDPeriode == periodeID && p.JenisProgram == jenisProgram {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockPendaftaranRepo) UpdateStatusAndAlasan(ctx context.Context, db database.DBTX, id int, status string, alasan *string) error {
	p, ok := m.pesertas[id]
	if !ok {
		return errors.New("not found")
	}
	p.Status = status
	p.AlasanTolak = alasan
	p.UpdatedAt = time.Now()
	return nil
}

func (m *mockPendaftaranRepo) FindMahasiswaByUserID(ctx context.Context, db database.DBTX, userID int) (*pendaftaran.MahasiswaInfo, error) {
	mhs, ok := m.mahasiswas[userID]
	if !ok {
		return nil, nil
	}
	cp := *mhs
	return &cp, nil
}

func (m *mockPendaftaranRepo) FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (*pendaftaran.PeriodeInfo, *time.Time, *time.Time, error) {
	per, ok := m.periodes[id]
	if !ok {
		return nil, nil, nil, nil
	}
	dates := m.periodeDates[id]
	cp := *per
	return &cp, dates[0], dates[1], nil
}

func (m *mockPendaftaranRepo) GetPendaftaranDetail(ctx context.Context, db database.DBTX, id int) (*pendaftaran.PendaftaranDetail, error) {
	p, ok := m.pesertas[id]
	if !ok {
		return nil, nil
	}
	return &pendaftaran.PendaftaranDetail{
		ID:           p.ID,
		JenisProgram: p.JenisProgram,
		Status:       p.Status,
		AlasanTolak:  p.AlasanTolak,
		Mahasiswa: pendaftaran.MahasiswaSummary{
			ID:   p.IDMahasiswa,
			NIM:  "NIM123",
			Nama: "Mhs Test",
		},
		Periode: pendaftaran.PeriodeSummary{
			ID:           p.IDPeriode,
			Nama:         "Periode Test",
			JenisProgram: p.JenisProgram,
			Status:       "aktif",
		},
	}, nil
}

func (m *mockPendaftaranRepo) GetDokumenTypesByPeserta(ctx context.Context, db database.DBTX, idPeserta int, idKelompok *int) ([]string, error) {
	return m.docsByPeserta[idPeserta], nil
}

func (m *mockPendaftaranRepo) ListPendaftaran(ctx context.Context, db database.DBTX, roleScope string, actorProfileID int, filter pendaftaran.PendaftaranFilter, p httpx.Pagination) ([]pendaftaran.PendaftaranListItem, int64, error) {
	var list []pendaftaran.PendaftaranListItem
	for _, it := range m.pesertas {
		list = append(list, pendaftaran.PendaftaranListItem{
			ID:            it.ID,
			IDMahasiswa:   it.IDMahasiswa,
			IDPeriode:     it.IDPeriode,
			JenisProgram:  it.JenisProgram,
			Status:        it.Status,
			TanggalDaftar: it.TanggalDaftar.Format(time.RFC3339),
		})
	}
	return list, int64(len(list)), nil
}

func (m *mockPendaftaranRepo) FindMahasiswaIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	mhs, ok := m.mahasiswas[userID]
	if !ok {
		return nil, nil
	}
	id := mhs.ID
	return &id, nil
}

func (m *mockPendaftaranRepo) FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	id, ok := m.dosens[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (m *mockPendaftaranRepo) FindPembimbingLapanganIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	id, ok := m.pembimbings[userID]
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (m *mockPendaftaranRepo) IsAnggotaKelompokKKN(ctx context.Context, db database.DBTX, mhsID, kelompokID int) (bool, error) {
	for _, p := range m.pesertas {
		if p.IDMahasiswa == mhsID && p.IDKelompokKKN != nil && *p.IDKelompokKKN == kelompokID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockPendaftaranRepo) FindTempatProgram(ctx context.Context, db database.DBTX, id int) (*pendaftaran.TempatProgramCheck, error) {
	t, ok := m.tempatPrograms[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockPendaftaranRepo) FindPembimbingLapangan(ctx context.Context, db database.DBTX, id int) (*pendaftaran.PembimbingLapanganCheck, error) {
	pl, ok := m.pembimbingLaps[id]
	if !ok {
		return nil, nil
	}
	return pl, nil
}

func (m *mockPendaftaranRepo) FindDosen(ctx context.Context, db database.DBTX, id int) (*pendaftaran.DosenCheck, error) {
	d, ok := m.dosensCheck[id]
	if !ok {
		return nil, nil
	}
	return d, nil
}

func (m *mockPendaftaranRepo) UpdatePenempatanMagang(ctx context.Context, db database.DBTX, idPeserta int, idTempat int, idPembimbing int) error {
	p, ok := m.pesertas[idPeserta]
	if !ok {
		return errors.New("peserta not found")
	}
	p.IDTempatProgram = &idTempat
	p.IDPembimbingLapangan = &idPembimbing
	return nil
}

func (m *mockPendaftaranRepo) UpdateDosenPembimbing(ctx context.Context, db database.DBTX, idPeserta int, idDosen int) error {
	p, ok := m.pesertas[idPeserta]
	if !ok {
		return errors.New("peserta not found")
	}
	p.IDDosenPembimbing = &idDosen
	return nil
}

func (m *mockPendaftaranRepo) UpdatePenempatanAM(ctx context.Context, db database.DBTX, idPeserta int, idTempat int, idPembimbing int) error {
	p, ok := m.pesertas[idPeserta]
	if !ok {
		return errors.New("peserta not found")
	}
	p.IDTempatProgram = &idTempat
	p.IDPembimbingLapangan = &idPembimbing
	return nil
}

func (m *mockPendaftaranRepo) UpdateDosenPenguji(ctx context.Context, db database.DBTX, idPeserta int, idDosen int) error {
	p, ok := m.pesertas[idPeserta]
	if !ok {
		return errors.New("peserta not found")
	}
	p.IDDosenPenguji = &idDosen
	return nil
}

// mockAccessService implementasi AccessService untuk pengujian unit service pendaftaran.
type mockAccessService struct {
	relMap map[int]pendaftaran.Relation
}

func (m *mockAccessService) Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
	if rel, ok := m.relMap[idPeserta]; ok {
		return rel, nil
	}
	if actor.HasRole("admin") {
		return pendaftaran.RelAdmin, nil
	}
	if actor.HasRole("pegawai") {
		return pendaftaran.RelPegawai, nil
	}
	if actor.HasRole("mahasiswa") {
		return pendaftaran.RelMahasiswa, nil
	}
	return "", httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
}

// mockFailingKRSValidator implementasi KRSValidator yang gagal.
type mockFailingKRSValidator struct {
	alasan string
}

func (f *mockFailingKRSValidator) Validate(ctx context.Context, m pendaftaran.MahasiswaInfo, p pendaftaran.PeriodeInfo) (bool, string, error) {
	return false, f.alasan, nil
}

// mockFailingEligibilityChecker implementasi EligibilityChecker yang gagal.
type mockFailingEligibilityChecker struct {
	alasan string
}

func (f *mockFailingEligibilityChecker) Check(ctx context.Context, m pendaftaran.MahasiswaInfo, p pendaftaran.PeriodeInfo) (bool, string, error) {
	return false, f.alasan, nil
}

func ptrInt(i int) *int {
	return &i
}

// TestCheckReadiness menguji fungsi murni CheckReadiness untuk ketiga jenis program secara table-driven.
func TestCheckReadiness(t *testing.T) {
	tests := []struct {
		name       string
		input      pendaftaran.ReadinessInput
		expectSiap bool
		checkText  string
	}{
		// 1. Status periode tidak aktif
		{
			name: "periode draft menghasilkan siap=false",
			input: pendaftaran.ReadinessInput{
				JenisProgram:  "magang",
				PeriodeStatus: "draft",
			},
			expectSiap: false,
			checkText:  "periode program belum berstatus aktif",
		},
		// 2. Magang
		{
			name: "magang lengkap -> siap=true",
			input: pendaftaran.ReadinessInput{
				JenisProgram:         "magang",
				PeriodeStatus:        "aktif",
				IDTempatProgram:      ptrInt(1),
				IDPembimbingLapangan: ptrInt(2),
				IDDosenPembimbing:    ptrInt(3),
				DokumenTypes:         []string{"mou", "proposal"},
			},
			expectSiap: true,
		},
		{
			name: "magang tanpa dokumen proposal -> siap=false",
			input: pendaftaran.ReadinessInput{
				JenisProgram:         "magang",
				PeriodeStatus:        "aktif",
				IDTempatProgram:      ptrInt(1),
				IDPembimbingLapangan: ptrInt(2),
				IDDosenPembimbing:    ptrInt(3),
				DokumenTypes:         []string{"mou"},
			},
			expectSiap: false,
			checkText:  "dokumen proposal belum diunggah",
		},
		{
			name: "magang belum ada dosen pembimbing -> siap=false",
			input: pendaftaran.ReadinessInput{
				JenisProgram:         "magang",
				PeriodeStatus:        "aktif",
				IDTempatProgram:      ptrInt(1),
				IDPembimbingLapangan: ptrInt(2),
				DokumenTypes:         []string{"mou", "proposal"},
			},
			expectSiap: false,
			checkText:  "dosen pembimbing belum ditetapkan",
		},
		// 3. Asistensi Mengajar
		{
			name: "asistensi mengajar lengkap -> siap=true",
			input: pendaftaran.ReadinessInput{
				JenisProgram:         "asistensi_mengajar",
				PeriodeStatus:        "aktif",
				IDTempatProgram:      ptrInt(1),
				IDPembimbingLapangan: ptrInt(2),
				IDDosenPembimbing:    ptrInt(3),
				DokumenTypes:         []string{"proker"},
			},
			expectSiap: true,
		},
		{
			name: "asistensi mengajar belum dipetakan sekolah -> siap=false",
			input: pendaftaran.ReadinessInput{
				JenisProgram:         "asistensi_mengajar",
				PeriodeStatus:        "aktif",
				IDPembimbingLapangan: ptrInt(2),
				IDDosenPembimbing:    ptrInt(3),
				DokumenTypes:         []string{"proker"},
			},
			expectSiap: false,
			checkText:  "sekolah penempatan belum dipetakan",
		},
		// 4. KKN
		{
			name: "KKN lengkap -> siap=true",
			input: pendaftaran.ReadinessInput{
				JenisProgram:      "kkn",
				PeriodeStatus:     "aktif",
				IDKelompokKKN:     ptrInt(10),
				IDTempatProgram:   ptrInt(20),
				IDDosenPembimbing: ptrInt(30),
				DokumenTypes:      []string{"proker"},
			},
			expectSiap: true,
		},
		{
			name: "KKN belum ada kelompok -> siap=false",
			input: pendaftaran.ReadinessInput{
				JenisProgram:      "kkn",
				PeriodeStatus:     "aktif",
				IDTempatProgram:   ptrInt(20),
				IDDosenPembimbing: ptrInt(30),
				DokumenTypes:      []string{"proker"},
			},
			expectSiap: false,
			checkText:  "kelompok KKN belum dipetakan",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := pendaftaran.CheckReadiness(tc.input)
			assert.Equal(t, tc.expectSiap, res.Siap)
			if tc.checkText != "" {
				found := false
				for _, k := range res.Kekurangan {
					if k == tc.checkText {
						found = true
						break
					}
				}
				assert.True(t, found, "kekurangan harus memuat: %s, hasil: %v", tc.checkText, res.Kekurangan)
			}
		})
	}
}

// TestDaftar_BusinessRules menguji aturan BR-01, BR-02, BR-03, dan penanganan validasi eksternal.
func TestDaftar_BusinessRules(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(now, time.UTC)

	actorMhs := middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}
	actorAdmin := middleware.Actor{UserID: 2, Roles: []string{"admin"}}

	setupService := func(krsVal pendaftaran.KRSValidator, eligCheck pendaftaran.EligibilityChecker) (*mockPendaftaranRepo, pendaftaran.Service) {
		repo := newMockPendaftaranRepo()
		access := &mockAccessService{relMap: make(map[int]pendaftaran.Relation)}
		if krsVal == nil {
			krsVal = pendaftaran.NewStubKRSValidator()
		}
		if eligCheck == nil {
			eligCheck = pendaftaran.NewStubEligibilityChecker()
		}
		svc := pendaftaran.NewService(repo, nil, clk, krsVal, eligCheck, access, nil)
		return repo, svc
	}

	t.Run("gagal: bukan mahasiswa memanggil endpoint pendaftaran (403)", func(t *testing.T) {
		_, svc := setupService(nil, nil)
		_, err := svc.Daftar(ctx, actorAdmin, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
	})

	t.Run("BR-01: gagal pendaftaran pada periode draft (422)", func(t *testing.T) {
		repo, svc := setupService(nil, nil)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "draft", JenisProgram: "magang"}

		_, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PERIODE_TIDAK_AKTIF", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})

	t.Run("BR-01: gagal pendaftaran sebelum tanggal buka daftar (422)", func(t *testing.T) {
		repo, svc := setupService(nil, nil)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}
		tBuka := now.AddDate(0, 0, 5) // 5 hari di masa depan
		repo.periodeDates[1] = [2]*time.Time{&tBuka, nil}

		_, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PENDAFTARAN_BELUM_DIBUKA", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})

	t.Run("BR-01: gagal pendaftaran sesudah tanggal tutup daftar (422)", func(t *testing.T) {
		repo, svc := setupService(nil, nil)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}
		tTutup := now.AddDate(0, 0, -2) // 2 hari lalu
		repo.periodeDates[1] = [2]*time.Time{nil, &tTutup}

		_, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PENDAFTARAN_SUDAH_DITUTUP", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})

	t.Run("BR-02 & sukses: jenis program ikut jenis periode, status diajukan bila validator lolos", func(t *testing.T) {
		repo, svc := setupService(nil, nil)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "asistensi_mengajar"}
		tBuka := now.AddDate(0, -1, 0)
		tTutup := now.AddDate(0, 1, 0)
		repo.periodeDates[1] = [2]*time.Time{&tBuka, &tTutup}

		detail, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, "asistensi_mengajar", detail.JenisProgram, "BR-02: jenis program pendaftaran harus sama dengan periode")
		assert.Equal(t, pendaftaran.StatusDiajukan, detail.Status)
		assert.Nil(t, detail.AlasanTolak)
	})

	t.Run("BR-03: pendaftaran ganda ditolak 409 Conflict", func(t *testing.T) {
		repo, svc := setupService(nil, nil)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}

		// Pendaftaran pertama sukses
		_, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.NoError(t, err)

		// Pendaftaran kedua pada periode yang sama
		_, err = svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "PENDAFTARAN_GANDA", appErr.Code)
		assert.Equal(t, 409, appErr.HTTPStatus)
	})

	t.Run("validator KRS gagal menghasilkan status ditolak dan alasan terisi", func(t *testing.T) {
		failingKRS := &mockFailingKRSValidator{alasan: "mata kuliah belum diprogram"}
		repo, svc := setupService(failingKRS, nil)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "kkn"}

		detail, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, pendaftaran.StatusDitolak, detail.Status)
		require.NotNil(t, detail.AlasanTolak)
		assert.Contains(t, *detail.AlasanTolak, "mata kuliah belum diprogram")
	})

	t.Run("validator eligibilitas gagal menghasilkan status ditolak", func(t *testing.T) {
		failingElig := &mockFailingEligibilityChecker{alasan: "SKS belum mencapai 80 SKS"}
		repo, svc := setupService(nil, failingElig)
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}

		detail, err := svc.Daftar(ctx, actorMhs, pendaftaran.CreatePendaftaranRequest{IDPeriode: 1})
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, pendaftaran.StatusDitolak, detail.Status)
		require.NotNil(t, detail.AlasanTolak)
		assert.Contains(t, *detail.AlasanTolak, "SKS belum mencapai 80 SKS")
	})
}

// TestAjukanUlang menguji alur pengajuan ulang pendaftaran ditolak saat periode masih buka.
func TestAjukanUlang(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(now, time.UTC)

	actorMhs := middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}

	repo := newMockPendaftaranRepo()
	access := &mockAccessService{relMap: make(map[int]pendaftaran.Relation)}
	svc := pendaftaran.NewService(repo, nil, clk, pendaftaran.NewStubKRSValidator(), pendaftaran.NewStubEligibilityChecker(), access, nil)

	repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "123", Nama: "Mhs"}
	repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}
	tTutup := now.AddDate(0, 1, 0)
	repo.periodeDates[1] = [2]*time.Time{nil, &tTutup}

	alasanAwal := "KRS belum ada"
	pDitolak := &pendaftaran.PesertaProgram{
		IDMahasiswa:  10,
		IDPeriode:    1,
		JenisProgram: "magang",
		Status:       pendaftaran.StatusDitolak,
		AlasanTolak:  &alasanAwal,
	}
	require.NoError(t, repo.CreatePeserta(ctx, nil, pDitolak))
	access.relMap[pDitolak.ID] = pendaftaran.RelMahasiswa

	t.Run("sukses ajukan ulang: status berubah dari ditolak menjadi diajukan", func(t *testing.T) {
		detail, err := svc.AjukanUlang(ctx, actorMhs, pDitolak.ID)
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, pendaftaran.StatusDiajukan, detail.Status)
		assert.Nil(t, detail.AlasanTolak)
	})

	t.Run("gagal ajukan ulang dari status diajukan (422 STATUS_TIDAK_VALID)", func(t *testing.T) {
		_, err := svc.AjukanUlang(ctx, actorMhs, pDitolak.ID)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "STATUS_TIDAK_VALID", appErr.Code)
		assert.Equal(t, 422, appErr.HTTPStatus)
	})
}

// TestHelperSudahDinilai menguji fungsi bantu SudahDinilai (RULES.md bagian 7 nomor 6).
func TestHelperSudahDinilai(t *testing.T) {
	val := 88.5
	p1 := &pendaftaran.PesertaProgram{Status: pendaftaran.StatusSelesai, NilaiAkhir: &val}
	assert.True(t, pendaftaran.SudahDinilai(p1))

	p2 := &pendaftaran.PesertaProgram{Status: pendaftaran.StatusSelesai, NilaiAkhir: nil}
	assert.False(t, pendaftaran.SudahDinilai(p2))

	p3 := &pendaftaran.PesertaProgram{Status: pendaftaran.StatusBerjalan, NilaiAkhir: &val}
	assert.False(t, pendaftaran.SudahDinilai(p3))

	assert.False(t, pendaftaran.SudahDinilai(nil))
}

// TestAccessService_Resolve menguji evaluasi relasi aktor terhadap pendaftaran sesuai ARCHITECTURE.md bagian 6.
func TestAccessService_Resolve(t *testing.T) {
	ctx := context.Background()
	repo := newMockPendaftaranRepo()
	accessSvc := pendaftaran.NewAccessService(nil, repo)

	// Setup data pendaftaran Magang
	pMagang := &pendaftaran.PesertaProgram{
		IDMahasiswa:          10,
		IDPeriode:            1,
		JenisProgram:         "magang",
		IDDosenPembimbing:    ptrInt(20),
		IDDosenPenguji:       ptrInt(21),
		IDPembimbingLapangan: ptrInt(30),
		Status:               pendaftaran.StatusBerjalan,
	}
	require.NoError(t, repo.CreatePeserta(ctx, nil, pMagang))

	// Setup data pendaftaran KKN untuk menguji anggota kelompok
	pKKN1 := &pendaftaran.PesertaProgram{
		IDMahasiswa:   11,
		IDPeriode:     2,
		JenisProgram:  "kkn",
		IDKelompokKKN: ptrInt(100),
		Status:        pendaftaran.StatusBerjalan,
	}
	require.NoError(t, repo.CreatePeserta(ctx, nil, pKKN1))

	pKKN2 := &pendaftaran.PesertaProgram{
		IDMahasiswa:   12,
		IDPeriode:     2,
		JenisProgram:  "kkn",
		IDKelompokKKN: ptrInt(100),
		Status:        pendaftaran.StatusBerjalan,
	}
	require.NoError(t, repo.CreatePeserta(ctx, nil, pKKN2))

	// Setup profil di mock repo
	repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1} // pemilik Magang
	repo.mahasiswas[2] = &pendaftaran.MahasiswaInfo{ID: 11, UserID: 2} // anggota KKN 1
	repo.mahasiswas[3] = &pendaftaran.MahasiswaInfo{ID: 12, UserID: 3} // anggota KKN 2
	repo.mahasiswas[4] = &pendaftaran.MahasiswaInfo{ID: 13, UserID: 4} // mahasiswa asing

	repo.dosens[10] = 20 // dospem
	repo.dosens[11] = 21 // penguji
	repo.dosens[12] = 22 // dosen lain

	repo.pembimbings[20] = 30 // mentor
	repo.pembimbings[21] = 31 // mentor lain

	t.Run("admin selalu RelAdmin", func(t *testing.T) {
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 99, Roles: []string{"admin"}}, pMagang.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelAdmin, rel)
	})

	t.Run("pegawai selalu RelPegawai", func(t *testing.T) {
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 98, Roles: []string{"pegawai"}}, pMagang.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelPegawai, rel)
	})

	t.Run("mahasiswa pemilik pendaftaran adalah RelMahasiswa", func(t *testing.T) {
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}, pMagang.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelMahasiswa, rel)
	})

	t.Run("dosen pembimbing adalah RelPembimbing", func(t *testing.T) {
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 10, Roles: []string{"dosen"}}, pMagang.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelPembimbing, rel)
	})

	t.Run("dosen penguji adalah RelPenguji", func(t *testing.T) {
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 11, Roles: []string{"dosen"}}, pMagang.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelPenguji, rel)
	})

	t.Run("pembimbing lapangan adalah RelLapangan", func(t *testing.T) {
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 20, Roles: []string{"pembimbing_lapangan"}}, pMagang.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelLapangan, rel)
	})

	t.Run("sesama anggota kelompok KKN adalah RelAnggotaKKN", func(t *testing.T) {
		// Mahasiswa user 3 mengakses pendaftaran pKKN1 milik mahasiswa 11 di kelompok 100
		rel, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 3, Roles: []string{"mahasiswa"}}, pKKN1.ID)
		require.NoError(t, err)
		assert.Equal(t, pendaftaran.RelAnggotaKKN, rel)
	})

	t.Run("mahasiswa lain tanpa relasi mengembalikan 404 (bukan 403)", func(t *testing.T) {
		_, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 4, Roles: []string{"mahasiswa"}}, pMagang.ID)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus, "ARCHITECTURE.md §6: tanpa relasi menghasilkan 404")
	})

	t.Run("dosen lain tanpa relasi mengembalikan 404", func(t *testing.T) {
		_, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 12, Roles: []string{"dosen"}}, pMagang.ID)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("pembimbing lapangan lain tanpa relasi mengembalikan 404", func(t *testing.T) {
		_, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 21, Roles: []string{"pembimbing_lapangan"}}, pMagang.ID)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("pendaftaran ID tidak ditemukan mengembalikan 404", func(t *testing.T) {
		_, err := accessSvc.Resolve(ctx, middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}, 99999)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus)
	})
}

// mockAuditLogger mencatat audit log aktivitas untuk pengujian unit.
type mockAuditLogger struct {
	logs []string
}

func (m *mockAuditLogger) Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error {
	m.logs = append(m.logs, action)
	return nil
}

func TestPenempatanMagang_BusinessRules(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(now, time.UTC)

	actorMhsPemilik := middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}
	actorMhsLain := middleware.Actor{UserID: 2, Roles: []string{"mahasiswa"}}

	setupTest := func() (*mockPendaftaranRepo, *mockAccessService, pendaftaran.Service) {
		repo := newMockPendaftaranRepo()
		access := &mockAccessService{relMap: make(map[int]pendaftaran.Relation)}
		svc := pendaftaran.NewService(repo, nil, clk, pendaftaran.NewStubKRSValidator(), pendaftaran.NewStubEligibilityChecker(), access, nil)

		// Setup master data tempat & mentor
		repo.tempatPrograms[1] = &pendaftaran.TempatProgramCheck{ID: 1, JenisProgram: "magang", IsAktif: true}
		repo.tempatPrograms[2] = &pendaftaran.TempatProgramCheck{ID: 2, JenisProgram: "asistensi_mengajar", IsAktif: true}
		repo.tempatPrograms[3] = &pendaftaran.TempatProgramCheck{ID: 3, JenisProgram: "magang", IsAktif: false}

		repo.pembimbingLaps[10] = &pendaftaran.PembimbingLapanganCheck{ID: 10, Jenis: "mentor", IDTempatProgram: 1}
		repo.pembimbingLaps[20] = &pendaftaran.PembimbingLapanganCheck{ID: 20, Jenis: "guru_pamong", IDTempatProgram: 2}
		repo.pembimbingLaps[30] = &pendaftaran.PembimbingLapanganCheck{ID: 30, Jenis: "mentor", IDTempatProgram: 3} // milik tempat 3

		// Setup pendaftaran magang
		pMagang := &pendaftaran.PesertaProgram{
			IDMahasiswa:  100,
			IDPeriode:    1,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusDiajukan,
		}
		_ = repo.CreatePeserta(ctx, nil, pMagang)
		access.relMap[pMagang.ID] = pendaftaran.RelMahasiswa

		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 100, UserID: 1, NIM: "NIM1", Nama: "Mhs 1"}

		return repo, access, svc
	}

	t.Run("BR-04: gagal tempat program berjenis asistensi_mengajar (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      2, // tempat AM
			IDPembimbingLapangan: 10,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_TIDAK_VALID", appErr.Code)
	})

	t.Run("BR-04: gagal tempat program non-aktif (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      3, // non-aktif
			IDPembimbingLapangan: 30,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_TIDAK_VALID", appErr.Code)
	})

	t.Run("BR-04: gagal pembimbing lapangan bukan mentor (guru_pamong) (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      1,
			IDPembimbingLapangan: 20, // guru_pamong
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PEMBIMBING_TIDAK_VALID", appErr.Code)
	})

	t.Run("BR-04: gagal mentor terdaftar di tempat program lain (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      1,
			IDPembimbingLapangan: 30, // mentor milik tempat 3
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PEMBIMBING_TIDAK_VALID", appErr.Code)
	})

	t.Run("gagal: bukan mahasiswa pemilik pendaftaran (403)", func(t *testing.T) {
		_, access, svc := setupTest()
		// Mahasiswa lain memiliki relasi bukan pemilik
		access.relMap[1] = pendaftaran.RelPembimbing
		_, err := svc.PenempatanMagang(ctx, actorMhsLain, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      1,
			IDPembimbingLapangan: 10,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
	})

	t.Run("gagal: pendaftaran berstatus berjalan menolak perubahan penempatan (422)", func(t *testing.T) {
		repo, _, svc := setupTest()
		repo.pesertas[1].Status = pendaftaran.StatusBerjalan

		_, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      1,
			IDPembimbingLapangan: 10,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_TIDAK_VALID", appErr.Code)
	})

	t.Run("gagal: pendaftaran berjenis bukan magang (422)", func(t *testing.T) {
		repo, _, svc := setupTest()
		repo.pesertas[1].JenisProgram = "asistensi_mengajar"

		_, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      1,
			IDPembimbingLapangan: 10,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "BUKAN_PROGRAM_MAGANG", appErr.Code)
	})

	t.Run("sukses: penempatan magang valid dan pengubahan ulang saat diajukan diizinkan", func(t *testing.T) {
		repo, _, svc := setupTest()

		// Setup master data tambahan untuk pengubahan ulang
		repo.tempatPrograms[4] = &pendaftaran.TempatProgramCheck{ID: 4, JenisProgram: "magang", IsAktif: true}
		repo.pembimbingLaps[40] = &pendaftaran.PembimbingLapanganCheck{ID: 40, Jenis: "mentor", IDTempatProgram: 4}

		// Kali pertama
		detail, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      1,
			IDPembimbingLapangan: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, ptrInt(1), repo.pesertas[1].IDTempatProgram)
		assert.Equal(t, ptrInt(10), repo.pesertas[1].IDPembimbingLapangan)

		// Pengubahan ulang diperbolehkan selama status masih diajukan
		detail2, err := svc.PenempatanMagang(ctx, actorMhsPemilik, 1, pendaftaran.PenempatanMagangRequest{
			IDTempatProgram:      4,
			IDPembimbingLapangan: 40,
		})
		require.NoError(t, err)
		require.NotNil(t, detail2)
		assert.Equal(t, ptrInt(4), repo.pesertas[1].IDTempatProgram)
		assert.Equal(t, ptrInt(40), repo.pesertas[1].IDPembimbingLapangan)
	})
}

func TestSetDosenPembimbing_BusinessRules(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(now, time.UTC)

	actorAdmin := middleware.Actor{UserID: 2, Roles: []string{"admin"}}
	actorPegawai := middleware.Actor{UserID: 3, Roles: []string{"pegawai"}}
	actorMhs := middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}

	setupTest := func() (*mockPendaftaranRepo, *mockAuditLogger, pendaftaran.Service) {
		repo := newMockPendaftaranRepo()
		access := &mockAccessService{relMap: make(map[int]pendaftaran.Relation)}
		logger := &mockAuditLogger{}
		svc := pendaftaran.NewService(repo, nil, clk, pendaftaran.NewStubKRSValidator(), pendaftaran.NewStubEligibilityChecker(), access, logger)

		repo.dosensCheck[100] = &pendaftaran.DosenCheck{ID: 100, Nama: "Dr. Pembimbing"}
		repo.dosensCheck[200] = &pendaftaran.DosenCheck{ID: 200, Nama: "Dr. Penguji"}

		// Pendaftaran Magang
		pMagang := &pendaftaran.PesertaProgram{
			ID:             1,
			IDMahasiswa:    10,
			IDPeriode:      1,
			JenisProgram:   "magang",
			Status:         pendaftaran.StatusDiajukan,
			IDDosenPenguji: ptrInt(200), // Sudah ada dosen penguji 200
		}
		repo.pesertas[1] = pMagang
		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 10, UserID: 1, NIM: "NIM1", Nama: "Mhs 1"}

		// Pendaftaran AM
		pAM := &pendaftaran.PesertaProgram{
			ID:           2,
			IDMahasiswa:  11,
			IDPeriode:    2,
			JenisProgram: "asistensi_mengajar",
			Status:       pendaftaran.StatusDiajukan,
		}
		repo.pesertas[2] = pAM
		repo.periodes[2] = &pendaftaran.PeriodeInfo{ID: 2, Status: "aktif", JenisProgram: "asistensi_mengajar"}
		repo.mahasiswas[11] = &pendaftaran.MahasiswaInfo{ID: 11, UserID: 11, NIM: "NIM2", Nama: "Mhs 2"}

		return repo, logger, svc
	}

	t.Run("gagal: mahasiswa tidak boleh memanggil endpoint dosen-pembimbing (403)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.SetDosenPembimbing(ctx, actorMhs, 1, pendaftaran.SetDosenPembimbingRequest{IDDosen: 100}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
	})

	t.Run("gagal: pegawai tidak boleh menetapkan dosen pada pendaftaran AM (403)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.SetDosenPembimbing(ctx, actorPegawai, 2, pendaftaran.SetDosenPembimbingRequest{IDDosen: 100}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
	})

	t.Run("gagal: dosen tidak ditemukan (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.SetDosenPembimbing(ctx, actorAdmin, 1, pendaftaran.SetDosenPembimbingRequest{IDDosen: 999}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "DOSEN_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("BR-07: gagal id_dosen pembimbing sama dengan id_dosen_penguji (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		// Pendaftaran 1 memiliki IDDosenPenguji = 200
		_, err := svc.SetDosenPembimbing(ctx, actorAdmin, 1, pendaftaran.SetDosenPembimbingRequest{IDDosen: 200}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", appErr.Code)
	})

	t.Run("sukses: pegawai menetapkan dosen pada pendaftaran magang dan dicatat di log_aktivitas", func(t *testing.T) {
		repo, logger, svc := setupTest()
		detail, err := svc.SetDosenPembimbing(ctx, actorPegawai, 1, pendaftaran.SetDosenPembimbingRequest{IDDosen: 100}, "127.0.0.1", "agent")
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, ptrInt(100), repo.pesertas[1].IDDosenPembimbing)

		// Verifikasi audit log
		require.Equal(t, 1, len(logger.logs))
		assert.Equal(t, "tetapkan_dosen_pembimbing", logger.logs[0])
	})

	t.Run("sukses: admin menetapkan dosen pada pendaftaran AM", func(t *testing.T) {
		repo, logger, svc := setupTest()
		detail, err := svc.SetDosenPembimbing(ctx, actorAdmin, 2, pendaftaran.SetDosenPembimbingRequest{IDDosen: 100}, "127.0.0.1", "agent")
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, ptrInt(100), repo.pesertas[2].IDDosenPembimbing)

		require.Equal(t, 1, len(logger.logs))
		assert.Equal(t, "tetapkan_dosen_pembimbing", logger.logs[0])
	})
}

func TestAdminPenempatan_BusinessRules(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewMockClock(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), time.UTC)

	actorAdmin := middleware.Actor{UserID: 99, Roles: []string{"admin"}}
	actorPegawai := middleware.Actor{UserID: 88, Roles: []string{"pegawai"}}
	actorMhs := middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}

	setupTest := func() (*mockPendaftaranRepo, pendaftaran.Service) {
		repo := newMockPendaftaranRepo()
		access := &mockAccessService{relMap: make(map[int]pendaftaran.Relation)}
		svc := pendaftaran.NewService(repo, nil, clk, pendaftaran.NewStubKRSValidator(), pendaftaran.NewStubEligibilityChecker(), access, nil)

		// Setup master tempat program
		repo.tempatPrograms[1] = &pendaftaran.TempatProgramCheck{ID: 1, JenisProgram: "magang", IsAktif: true}
		repo.tempatPrograms[2] = &pendaftaran.TempatProgramCheck{ID: 2, JenisProgram: "asistensi_mengajar", IsAktif: true}
		repo.tempatPrograms[3] = &pendaftaran.TempatProgramCheck{ID: 3, JenisProgram: "asistensi_mengajar", IsAktif: false}
		repo.tempatPrograms[4] = &pendaftaran.TempatProgramCheck{ID: 4, JenisProgram: "asistensi_mengajar", IsAktif: true} // sekolah lain

		// Setup master pembimbing lapangan
		repo.pembimbingLaps[10] = &pendaftaran.PembimbingLapanganCheck{ID: 10, Jenis: "mentor", IDTempatProgram: 1}
		repo.pembimbingLaps[20] = &pendaftaran.PembimbingLapanganCheck{ID: 20, Jenis: "guru_pamong", IDTempatProgram: 2}
		repo.pembimbingLaps[21] = &pendaftaran.PembimbingLapanganCheck{ID: 21, Jenis: "mentor", IDTempatProgram: 2}      // bukan guru_pamong
		repo.pembimbingLaps[40] = &pendaftaran.PembimbingLapanganCheck{ID: 40, Jenis: "guru_pamong", IDTempatProgram: 4} // milik sekolah 4

		// Setup pendaftaran
		// Pendaftaran 1: Magang
		pMagang := &pendaftaran.PesertaProgram{
			IDMahasiswa:  100,
			IDPeriode:    1,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusDiajukan,
		}
		_ = repo.CreatePeserta(ctx, nil, pMagang)

		// Pendaftaran 2: Asistensi Mengajar
		pAM := &pendaftaran.PesertaProgram{
			IDMahasiswa:  101,
			IDPeriode:    2,
			JenisProgram: "asistensi_mengajar",
			Status:       pendaftaran.StatusDiajukan,
		}
		_ = repo.CreatePeserta(ctx, nil, pAM)

		// Pendaftaran 3: KKN
		pKKN := &pendaftaran.PesertaProgram{
			IDMahasiswa:  102,
			IDPeriode:    3,
			JenisProgram: "kkn",
			Status:       pendaftaran.StatusDiajukan,
		}
		_ = repo.CreatePeserta(ctx, nil, pKKN)

		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "magang"}
		repo.periodes[2] = &pendaftaran.PeriodeInfo{ID: 2, Status: "aktif", JenisProgram: "asistensi_mengajar"}
		repo.periodes[3] = &pendaftaran.PeriodeInfo{ID: 3, Status: "aktif", JenisProgram: "kkn"}

		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 100, UserID: 1, NIM: "NIM1", Nama: "Mhs 1"}
		repo.mahasiswas[2] = &pendaftaran.MahasiswaInfo{ID: 101, UserID: 2, NIM: "NIM2", Nama: "Mhs 2"}
		repo.mahasiswas[3] = &pendaftaran.MahasiswaInfo{ID: 102, UserID: 3, NIM: "NIM3", Nama: "Mhs 3"}

		return repo, svc
	}

	t.Run("gagal: bukan admin memanggil admin penempatan (403)", func(t *testing.T) {
		_, svc := setupTest()
		// Mahasiswa memanggil
		_, err := svc.AdminPenempatan(ctx, actorMhs, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)

		// Pegawai memanggil
		_, err = svc.AdminPenempatan(ctx, actorPegawai, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
	})

	t.Run("BR-05: gagal tempat program berjenis magang (422)", func(t *testing.T) {
		_, svc := setupTest()
		_, err := svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      1, // tempat magang
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_TIDAK_VALID", appErr.Code)
	})

	t.Run("BR-05: gagal tempat program non-aktif (422)", func(t *testing.T) {
		_, svc := setupTest()
		_, err := svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      3, // tempat AM non-aktif
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_TIDAK_VALID", appErr.Code)
	})

	t.Run("BR-05: gagal pembimbing lapangan bukan guru_pamong (mentor) (422)", func(t *testing.T) {
		_, svc := setupTest()
		_, err := svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 21, // mentor
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PEMBIMBING_TIDAK_VALID", appErr.Code)
	})

	t.Run("BR-05: gagal guru_pamong terdaftar di sekolah penempatan lain (422)", func(t *testing.T) {
		_, svc := setupTest()
		_, err := svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 40, // guru pamong sekolah 4
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PEMBIMBING_TIDAK_VALID", appErr.Code)
	})

	t.Run("gagal: pendaftaran peserta magang ditolak pada endpoint penempatan admin (422)", func(t *testing.T) {
		_, svc := setupTest()
		_, err := svc.AdminPenempatan(ctx, actorAdmin, 1, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "JENIS_PROGRAM_TIDAK_SESUAI", appErr.Code)
		assert.Contains(t, appErr.Message, "magang dipilih mandiri")
	})

	t.Run("gagal: pendaftaran peserta KKN ditolak pada endpoint penempatan admin (422)", func(t *testing.T) {
		_, svc := setupTest()
		_, err := svc.AdminPenempatan(ctx, actorAdmin, 3, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "JENIS_PROGRAM_TIDAK_SESUAI", appErr.Code)
		assert.Contains(t, appErr.Message, "kelompok KKN")
	})

	t.Run("gagal: pendaftaran berstatus berjalan menolak perubahan penempatan (422)", func(t *testing.T) {
		repo, svc := setupTest()
		repo.pesertas[2].Status = pendaftaran.StatusBerjalan

		_, err := svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_TIDAK_VALID", appErr.Code)
	})

	t.Run("sukses: admin memetakan penempatan AM valid dan perubahan ulang saat diajukan diizinkan", func(t *testing.T) {
		repo, svc := setupTest()
		detail, err := svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, ptrInt(2), repo.pesertas[2].IDTempatProgram)
		assert.Equal(t, ptrInt(20), repo.pesertas[2].IDPembimbingLapangan)

		// Pengubahan ulang saat diajukan diizinkan
		detail, err = svc.AdminPenempatan(ctx, actorAdmin, 2, pendaftaran.AdminPenempatanRequest{
			IDTempatProgram:      2,
			IDPembimbingLapangan: 20,
		})
		require.NoError(t, err)
		require.NotNil(t, detail)
	})
}

func TestSetDosenPenguji_BusinessRules(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewMockClock(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), time.UTC)

	actorAdmin := middleware.Actor{UserID: 99, Roles: []string{"admin"}}
	actorPegawai := middleware.Actor{UserID: 88, Roles: []string{"pegawai"}}
	actorMhs := middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}

	setupTest := func() (*mockPendaftaranRepo, *mockAuditLogger, pendaftaran.Service) {
		repo := newMockPendaftaranRepo()
		logger := &mockAuditLogger{}
		access := &mockAccessService{relMap: make(map[int]pendaftaran.Relation)}
		svc := pendaftaran.NewService(repo, nil, clk, pendaftaran.NewStubKRSValidator(), pendaftaran.NewStubEligibilityChecker(), access, logger)

		// Setup master dosen
		repo.dosensCheck[100] = &pendaftaran.DosenCheck{ID: 100, Nama: "Dr. Pembimbing"}
		repo.dosensCheck[200] = &pendaftaran.DosenCheck{ID: 200, Nama: "Dr. Penguji"}

		// Pendaftaran 1 memiliki dosen pembimbing = 100
		idDosenPembimbing := 100
		p := &pendaftaran.PesertaProgram{
			IDMahasiswa:       100,
			IDPeriode:         1,
			JenisProgram:      "asistensi_mengajar",
			Status:            pendaftaran.StatusDiajukan,
			IDDosenPembimbing: &idDosenPembimbing,
		}
		_ = repo.CreatePeserta(ctx, nil, p)

		repo.periodes[1] = &pendaftaran.PeriodeInfo{ID: 1, Status: "aktif", JenisProgram: "asistensi_mengajar"}
		repo.mahasiswas[1] = &pendaftaran.MahasiswaInfo{ID: 100, UserID: 1, NIM: "NIM1", Nama: "Mhs 1"}

		return repo, logger, svc
	}

	t.Run("gagal: bukan admin memanggil set dosen penguji (403)", func(t *testing.T) {
		_, _, svc := setupTest()
		// Mahasiswa memanggil
		_, err := svc.SetDosenPenguji(ctx, actorMhs, 1, pendaftaran.SetDosenPengujiRequest{IDDosen: 200}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)

		// Pegawai memanggil
		_, err = svc.SetDosenPenguji(ctx, actorPegawai, 1, pendaftaran.SetDosenPengujiRequest{IDDosen: 200}, "127.0.0.1", "agent")
		require.Error(t, err)
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
	})

	t.Run("gagal: dosen tidak ditemukan (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		_, err := svc.SetDosenPenguji(ctx, actorAdmin, 1, pendaftaran.SetDosenPengujiRequest{IDDosen: 999}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "DOSEN_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("BR-07: gagal id_dosen penguji sama dengan id_dosen_pembimbing (422)", func(t *testing.T) {
		_, _, svc := setupTest()
		// Pendaftaran 1 memiliki IDDosenPembimbing = 100
		_, err := svc.SetDosenPenguji(ctx, actorAdmin, 1, pendaftaran.SetDosenPengujiRequest{IDDosen: 100}, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "DOSEN_PEMBIMBING_SAMA_DENGAN_PENGUJI", appErr.Code)
	})

	t.Run("sukses: admin menetapkan dosen penguji dan dicatat di log_aktivitas", func(t *testing.T) {
		repo, logger, svc := setupTest()
		detail, err := svc.SetDosenPenguji(ctx, actorAdmin, 1, pendaftaran.SetDosenPengujiRequest{IDDosen: 200}, "127.0.0.1", "agent")
		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, ptrInt(200), repo.pesertas[1].IDDosenPenguji)

		// Verifikasi audit log
		require.Equal(t, 1, len(logger.logs))
		assert.Equal(t, "tetapkan_dosen_penguji", logger.logs[0])
	})
}
