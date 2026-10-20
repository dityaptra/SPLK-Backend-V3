package penilaian_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/penilaian"
	"splk-backend/internal/platform/httpx"
)

type mockPenilaianRepo struct {
	instrumens map[int]*penilaian.InstrumenPenilaian
	pesertas   map[int]*penilaian.PesertaPenilaianInfo
	penilaians map[string]*penilaian.Penilaian // key: "idPeserta_idInstrumen_idPenilai"
	userRoles  map[string]int                  // key: "idPeserta_role" -> userID
	kelulusans map[int]*penilaian.KelulusanPenguji
	nextInstID int
	nextPenID  int
}

func newMockPenilaianRepo() *mockPenilaianRepo {
	return &mockPenilaianRepo{
		instrumens: make(map[int]*penilaian.InstrumenPenilaian),
		pesertas:   make(map[int]*penilaian.PesertaPenilaianInfo),
		penilaians: make(map[string]*penilaian.Penilaian),
		userRoles:  make(map[string]int),
		kelulusans: make(map[int]*penilaian.KelulusanPenguji),
		nextInstID: 1,
		nextPenID:  1,
	}
}

func (m *mockPenilaianRepo) CreateInstrumen(ctx context.Context, db database.DBTX, i *penilaian.InstrumenPenilaian) error {
	i.ID = m.nextInstID
	m.nextInstID++
	stored := *i
	m.instrumens[i.ID] = &stored
	return nil
}

func (m *mockPenilaianRepo) GetInstrumenByID(ctx context.Context, db database.DBTX, id int) (*penilaian.InstrumenPenilaian, error) {
	it, ok := m.instrumens[id]
	if !ok {
		return nil, nil
	}
	cp := *it
	return &cp, nil
}

func (m *mockPenilaianRepo) UpdateInstrumen(ctx context.Context, db database.DBTX, i *penilaian.InstrumenPenilaian) error {
	if _, ok := m.instrumens[i.ID]; !ok {
		return pgx.ErrNoRows
	}
	stored := *i
	m.instrumens[i.ID] = &stored
	return nil
}

func (m *mockPenilaianRepo) DeleteInstrumen(ctx context.Context, db database.DBTX, id int) error {
	if _, ok := m.instrumens[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.instrumens, id)
	return nil
}

func (m *mockPenilaianRepo) ListInstrumen(ctx context.Context, db database.DBTX, jenisProgram, rolePenilai string) ([]penilaian.InstrumenPenilaian, error) {
	res := make([]penilaian.InstrumenPenilaian, 0)
	for _, it := range m.instrumens {
		if (jenisProgram == "" || it.JenisProgram == jenisProgram) &&
			(rolePenilai == "" || it.RolePenilai == rolePenilai) {
			res = append(res, *it)
		}
	}
	return res, nil
}

func (m *mockPenilaianRepo) GetRingkasanBobot(ctx context.Context, db database.DBTX) ([]penilaian.RingkasanBobotProgram, error) {
	return []penilaian.RingkasanBobotProgram{}, nil
}

func (m *mockPenilaianRepo) GetPesertaPenilaianInfo(ctx context.Context, db database.DBTX, idPeserta int) (*penilaian.PesertaPenilaianInfo, error) {
	p, ok := m.pesertas[idPeserta]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *mockPenilaianRepo) GetPesertaForUpdate(ctx context.Context, tx pgx.Tx, idPeserta int) (*penilaian.PesertaPenilaianInfo, error) {
	return m.GetPesertaPenilaianInfo(ctx, nil, idPeserta)
}

func (m *mockPenilaianRepo) GetPenilaiUserIDByRole(ctx context.Context, db database.DBTX, idPeserta int, role string) (*int, error) {
	key := fmt.Sprintf("%d_%s", idPeserta, role)
	uid, ok := m.userRoles[key]
	if !ok {
		return nil, nil
	}
	return &uid, nil
}

func (m *mockPenilaianRepo) GetInstrumenByJenisProgram(ctx context.Context, db database.DBTX, jenisProgram string) ([]penilaian.InstrumenPenilaian, error) {
	return m.ListInstrumen(ctx, db, jenisProgram, "")
}

func (m *mockPenilaianRepo) GetPenilaianByPeserta(ctx context.Context, db database.DBTX, idPeserta int) ([]penilaian.Penilaian, error) {
	res := make([]penilaian.Penilaian, 0)
	for _, p := range m.penilaians {
		if p.IDPesertaProgram == idPeserta {
			res = append(res, *p)
		}
	}
	return res, nil
}

func (m *mockPenilaianRepo) GetNilaiDetailByPeserta(ctx context.Context, db database.DBTX, idPeserta int) ([]penilaian.NilaiDetail, error) {
	res := make([]penilaian.NilaiDetail, 0)
	for _, p := range m.penilaians {
		if p.IDPesertaProgram == idPeserta {
			inst := m.instrumens[p.IDInstrumen]
			namaAspek := ""
			bobot := 0.0
			role := ""
			if inst != nil {
				namaAspek = inst.NamaAspek
				bobot = inst.Bobot
				role = inst.RolePenilai
			}
			res = append(res, penilaian.NilaiDetail{
				IDPenilaian:  p.ID,
				IDInstrumen:  p.IDInstrumen,
				NamaAspek:    namaAspek,
				Bobot:        bobot,
				RolePenilai:  role,
				IDPenilai:    p.IDPenilai,
				NamaPenilai:  "penilai_test",
				Nilai:        p.Nilai,
				Catatan:      p.Catatan,
				TanggalNilai: p.TanggalNilai,
			})
		}
	}
	return res, nil
}

func (m *mockPenilaianRepo) UpsertPenilaian(ctx context.Context, tx pgx.Tx, p *penilaian.Penilaian) error {
	key := fmt.Sprintf("%d_%d_%d", p.IDPesertaProgram, p.IDInstrumen, p.IDPenilai)
	existing, ok := m.penilaians[key]
	if ok {
		existing.Nilai = p.Nilai
		existing.Catatan = p.Catatan
		existing.TanggalNilai = time.Now()
		p.ID = existing.ID
		return nil
	}
	p.ID = m.nextPenID
	m.nextPenID++
	p.TanggalNilai = time.Now()
	stored := *p
	m.penilaians[key] = &stored
	return nil
}

func (m *mockPenilaianRepo) UpdateNilaiAkhir(ctx context.Context, tx pgx.Tx, idPeserta int, nilaiAkhir float64, gradeHuruf string) error {
	p, ok := m.pesertas[idPeserta]
	if !ok {
		return pgx.ErrNoRows
	}
	p.NilaiAkhir = &nilaiAkhir
	p.GradeHuruf = &gradeHuruf
	return nil
}

func (m *mockPenilaianRepo) GetKelulusanPengujiByPeserta(ctx context.Context, db database.DBTX, idPeserta int) (*penilaian.KelulusanPenguji, error) {
	k, ok := m.kelulusans[idPeserta]
	if !ok {
		return nil, nil
	}
	cp := *k
	return &cp, nil
}

func (m *mockPenilaianRepo) UpsertKelulusanPenguji(ctx context.Context, db database.DBTX, k *penilaian.KelulusanPenguji) error {
	k.ID = 1
	k.CreatedAt = time.Now()
	k.UpdatedAt = time.Now()
	stored := *k
	m.kelulusans[k.IDPesertaProgram] = &stored
	return nil
}

func (m *mockPenilaianRepo) GetMahasiswaDanPengujiInfo(ctx context.Context, db database.DBTX, idPeserta int) (nim, namaMhs string, idDosen *int, namaDosen *string, err error) {
	dosenID := 20
	dosenNama := "Dr. Penguji"
	return "2115051001", "Mahasiswa Test", &dosenID, &dosenNama, nil
}

type mockAccessService struct {
	relations map[string]pendaftaran.Relation // key: "userID_idPeserta"
}

func newMockAccessService() *mockAccessService {
	return &mockAccessService{
		relations: make(map[string]pendaftaran.Relation),
	}
}

func (m *mockAccessService) Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
	if actor.HasRole("admin") {
		return pendaftaran.RelAdmin, nil
	}
	if actor.HasRole("pegawai") {
		return pendaftaran.RelPegawai, nil
	}
	key := fmt.Sprintf("%d_%d", actor.UserID, idPeserta)
	rel, ok := m.relations[key]
	if !ok {
		return "", httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}
	return rel, nil
}

func TestPenilaian_BusinessRules(t *testing.T) {
	ctx := context.Background()

	setupEnv := func() (*mockPenilaianRepo, *mockAccessService, penilaian.Service) {
		repo := newMockPenilaianRepo()
		accessSvc := newMockAccessService()
		svc := penilaian.NewService(nil, repo, accessSvc, penilaian.NewScoreCalculator(), nil)
		return repo, accessSvc, svc
	}

	actorPembimbing := middleware.Actor{UserID: 101, Roles: []string{"dosen"}}
	_ = middleware.Actor{UserID: 102, Roles: []string{"dosen"}}
	_ = middleware.Actor{UserID: 103, Roles: []string{"pembimbing_lapangan"}}
	actorDosenLain := middleware.Actor{UserID: 999, Roles: []string{"dosen"}}
	actorMhsPemilik := middleware.Actor{UserID: 201, Roles: []string{"mahasiswa"}}
	actorAdmin := middleware.Actor{UserID: 1, Roles: []string{"admin"}}

	t.Run("gagal: status pendaftaran bukan selesai (BR-10)", func(t *testing.T) {
		repo, accessSvc, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusBerjalan, // bukan selesai
		}
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID
		accessSvc.relations[fmt.Sprintf("%d_%d", actorPembimbing.UserID, pesertaID)] = pendaftaran.RelPembimbing

		val := 85.0
		req := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{
				{IDInstrumen: 1, Nilai: &val},
			},
		}

		_, err := svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, req, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "STATUS_TIDAK_SESUAI", appErr.Code)
	})

	t.Run("gagal: penilai bukan relasi sah yang ditugaskan (BR-15)", func(t *testing.T) {
		repo, _, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusSelesai,
		}
		// dosen lain tidak ada relasi di accessSvc
		val := 85.0
		req := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{
				{IDInstrumen: 1, Nilai: &val},
			},
		}

		_, err := svc.SubmitPenilaian(ctx, actorDosenLain, pesertaID, req, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 404, appErr.HTTPStatus) // ditolak oleh Resolve (tidak membocorkan IDOR)
	})

	t.Run("gagal: nilai di luar rentang 0 sampai 100 (BR-10)", func(t *testing.T) {
		repo, accessSvc, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusSelesai,
		}
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID
		accessSvc.relations[fmt.Sprintf("%d_%d", actorPembimbing.UserID, pesertaID)] = pendaftaran.RelPembimbing

		valNeg := -5.0
		reqNeg := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{{IDInstrumen: 1, Nilai: &valNeg}},
		}
		_, err := svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, reqNeg, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "NILAI_DI_LUAR_RENTANG", appErr.Code)

		val105 := 105.0
		req105 := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{{IDInstrumen: 1, Nilai: &val105}},
		}
		_, err = svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, req105, "127.0.0.1", "test")
		require.Error(t, err)
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "NILAI_DI_LUAR_RENTANG", appErr.Code)
	})

	t.Run("gagal: instrumen jenis program salah (BR-10)", func(t *testing.T) {
		repo, accessSvc, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang", // program peserta adalah magang
			Status:       pendaftaran.StatusSelesai,
		}
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID
		accessSvc.relations[fmt.Sprintf("%d_%d", actorPembimbing.UserID, pesertaID)] = pendaftaran.RelPembimbing

		// Instrumen dibuat untuk program kkn
		instKKN := &penilaian.InstrumenPenilaian{
			ID:           10,
			JenisProgram: "kkn",
			NamaAspek:    "Laporan KKN",
			Bobot:        50,
			RolePenilai:  penilaian.RoleDosenPembimbing,
		}
		repo.instrumens[instKKN.ID] = instKKN

		val := 80.0
		req := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{{IDInstrumen: instKKN.ID, Nilai: &val}},
		}
		_, err := svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, req, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "INSTRUMEN_TIDAK_COCOK", appErr.Code)
	})

	t.Run("gagal: instrumen role penilai salah (BR-10)", func(t *testing.T) {
		repo, accessSvc, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusSelesai,
		}
		// Pemanggil adalah dosen pembimbing
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID
		accessSvc.relations[fmt.Sprintf("%d_%d", actorPembimbing.UserID, pesertaID)] = pendaftaran.RelPembimbing

		// Instrumen untuk dosen penguji
		instPenguji := &penilaian.InstrumenPenilaian{
			ID:           20,
			JenisProgram: "magang",
			NamaAspek:    "Presentasi Ujian",
			Bobot:        50,
			RolePenilai:  penilaian.RoleDosenPenguji,
		}
		repo.instrumens[instPenguji.ID] = instPenguji

		val := 90.0
		req := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{{IDInstrumen: instPenguji.ID, Nilai: &val}},
		}
		_, err := svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, req, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "ROLE_INSTRUMEN_TIDAK_COCOK", appErr.Code)
	})

	t.Run("gagal: nilai terkunci setelah terhitung (BR-11)", func(t *testing.T) {
		repo, accessSvc, svc := setupEnv()
		pesertaID := 1
		nilaiAkhir := 88.5
		grade := "A"
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusSelesai,
			NilaiAkhir:   &nilaiAkhir, // nilai akhir SUDAH terisi
			GradeHuruf:   &grade,
		}
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID
		accessSvc.relations[fmt.Sprintf("%d_%d", actorPembimbing.UserID, pesertaID)] = pendaftaran.RelPembimbing

		val := 95.0
		req := penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{{IDInstrumen: 1, Nilai: &val}},
		}
		_, err := svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, req, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "NILAI_SUDAH_TERKUNCI", appErr.Code)
	})

	t.Run("mahasiswa melihat nilai: 403 saat belum selesai, 200 saat sudah selesai", func(t *testing.T) {
		repo, accessSvc, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusSelesai,
			NilaiAkhir:   nil, // belum terisi
		}
		accessSvc.relations[fmt.Sprintf("%d_%d", actorMhsPemilik.UserID, pesertaID)] = pendaftaran.RelMahasiswa

		// Sebelum nilai_akhir terisi -> 403 NILAI_BELUM_TERSEDIA
		_, err := svc.GetNilai(ctx, actorMhsPemilik, pesertaID)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 403, appErr.HTTPStatus)
		assert.Equal(t, "NILAI_BELUM_TERSEDIA", appErr.Code)

		// Setelah nilai_akhir terisi -> 200 OK
		nAkhir := 86.0
		gHuruf := "A"
		repo.pesertas[pesertaID].NilaiAkhir = &nAkhir
		repo.pesertas[pesertaID].GradeHuruf = &gHuruf

		res, err := svc.GetNilai(ctx, actorMhsPemilik, pesertaID)
		require.NoError(t, err)
		assert.Equal(t, 86.0, *res.NilaiAkhir)
		assert.Equal(t, "A", *res.GradeHuruf)
	})

	t.Run("hitung nilai manual admin: 422 bila instrumen kosong atau penilaian belum lengkap", func(t *testing.T) {
		repo, _, svc := setupEnv()
		pesertaID := 1
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:           pesertaID,
			JenisProgram: "magang",
			Status:       pendaftaran.StatusSelesai,
		}

		// Kasus 1: Instrumen kosong -> 422 INSTRUMEN_BELUM_DIATUR
		_, err := svc.HitungNilaiManual(ctx, actorAdmin, pesertaID, "127.0.0.1", "test")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "INSTRUMEN_BELUM_DIATUR", appErr.Code)

		// Kasus 2: Instrumen ada, tapi penilai belum menilai -> 422 PENILAIAN_BELUM_LENGKAP
		inst1 := &penilaian.InstrumenPenilaian{ID: 1, JenisProgram: "magang", NamaAspek: "Aspek 1", Bobot: 50, RolePenilai: penilaian.RoleDosenPembimbing}
		repo.instrumens[1] = inst1
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID

		_, err = svc.HitungNilaiManual(ctx, actorAdmin, pesertaID, "127.0.0.1", "test")
		require.Error(t, err)
		require.True(t, errorsAs(err, &appErr))
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PENILAIAN_BELUM_LENGKAP", appErr.Code)
	})

	t.Run("Magang: Hitung nilai akhir dengan proporsi 60% pembimbing dan 40% mentor", func(t *testing.T) {
		repo := newMockPenilaianRepo()
		accessSvc := newMockAccessService()
		svc := penilaian.NewService(nil, repo, accessSvc, nil, nil)
		ctx := context.Background()

		pesertaID := 10
		idDosen := 101
		idLapangan := 102
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:                   pesertaID,
			IDMahasiswa:          1,
			JenisProgram:         "magang",
			Status:               pendaftaran.StatusSelesai,
			IDDosenPembimbing:    &idDosen,
			IDPembimbingLapangan: &idLapangan,
		}

		actorPembimbing := middleware.Actor{UserID: 201, Roles: []string{"dosen"}}
		actorLapangan := middleware.Actor{UserID: 202, Roles: []string{"pembimbing_lapangan"}}

		accessSvc.relations[fmt.Sprintf("%d_%d", actorPembimbing.UserID, pesertaID)] = pendaftaran.RelPembimbing
		accessSvc.relations[fmt.Sprintf("%d_%d", actorLapangan.UserID, pesertaID)] = pendaftaran.RelLapangan

		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RoleDosenPembimbing)] = actorPembimbing.UserID
		repo.userRoles[fmt.Sprintf("%d_%s", pesertaID, penilaian.RolePembimbingLapangan)] = actorLapangan.UserID

		// Instrumen Pembimbing (Total 100%)
		inst1 := &penilaian.InstrumenPenilaian{ID: 1, JenisProgram: "magang", NamaAspek: "Pembimbingan", Bobot: 100, RolePenilai: penilaian.RoleDosenPembimbing}
		repo.instrumens[1] = inst1

		// Instrumen Lapangan (Total 100%)
		inst2 := &penilaian.InstrumenPenilaian{ID: 2, JenisProgram: "magang", NamaAspek: "Kinerja Lapangan", Bobot: 100, RolePenilai: penilaian.RolePembimbingLapangan}
		repo.instrumens[2] = inst2

		// Pembimbing mengisi nilai 80
		nilaiPemb := 80.0
		res1, err := svc.SubmitPenilaian(ctx, actorPembimbing, pesertaID, penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{
				{IDInstrumen: 1, Nilai: &nilaiPemb},
			},
		}, "127.0.0.1", "test")
		require.NoError(t, err)
		assert.False(t, res1.SemuaInstrumenTerisi)

		// Mentor lapangan mengisi nilai 90
		nilaiLap := 90.0
		res2, err := svc.SubmitPenilaian(ctx, actorLapangan, pesertaID, penilaian.SubmitPenilaianRequest{
			Items: []penilaian.SubmitPenilaianItem{
				{IDInstrumen: 2, Nilai: &nilaiLap},
			},
		}, "127.0.0.1", "test")
		require.NoError(t, err)
		assert.True(t, res2.SemuaInstrumenTerisi)
		require.NotNil(t, res2.NilaiAkhir)
		// 60% * 80 + 40% * 90 = 48 + 36 = 84.0
		assert.Equal(t, 84.0, *res2.NilaiAkhir)
		require.NotNil(t, res2.GradeHuruf)
		assert.Equal(t, "B", *res2.GradeHuruf)
	})

	t.Run("Verifikasi Penguji: Submit dan Get status kelulusan (Gambar 4)", func(t *testing.T) {
		repo := newMockPenilaianRepo()
		accessSvc := newMockAccessService()
		svc := penilaian.NewService(nil, repo, accessSvc, nil, nil)
		ctx := context.Background()

		pesertaID := 15
		idPenguji := 105
		repo.pesertas[pesertaID] = &penilaian.PesertaPenilaianInfo{
			ID:             pesertaID,
			IDMahasiswa:    1,
			JenisProgram:   "magang",
			Status:         pendaftaran.StatusBerjalan,
			IDDosenPenguji: &idPenguji,
		}

		actorPenguji := middleware.Actor{UserID: 205, Roles: []string{"dosen"}}
		actorLain := middleware.Actor{UserID: 999, Roles: []string{"mahasiswa"}}
		accessSvc.relations[fmt.Sprintf("%d_%d", actorPenguji.UserID, pesertaID)] = pendaftaran.RelPenguji
		accessSvc.relations[fmt.Sprintf("%d_%d", actorLain.UserID, pesertaID)] = pendaftaran.RelMahasiswa

		// Actor lain dilarang verifikasi
		_, err := svc.SubmitVerifikasiPenguji(ctx, actorLain, pesertaID, penilaian.SubmitVerifikasiPengujiRequest{
			StatusKelulusan: "Lulus",
			Catatan:         "Mahasiswa lulus",
		}, "127.0.0.1", "test")
		require.Error(t, err)

		// Input status tidak valid
		_, err = svc.SubmitVerifikasiPenguji(ctx, actorPenguji, pesertaID, penilaian.SubmitVerifikasiPengujiRequest{
			StatusKelulusan: "Abrakadabra",
			Catatan:         "Catatan",
		}, "127.0.0.1", "test")
		require.Error(t, err)

		// Sukses verifikasi oleh penguji
		resSubmit, err := svc.SubmitVerifikasiPenguji(ctx, actorPenguji, pesertaID, penilaian.SubmitVerifikasiPengujiRequest{
			StatusKelulusan: "Lulus",
			Catatan:         "Mahasiswa lulus dengan sangat memuaskan saat ujian",
		}, "127.0.0.1", "test")
		require.NoError(t, err)
		assert.Equal(t, "Lulus", resSubmit.StatusKelulusan)
		assert.Equal(t, "Mahasiswa lulus dengan sangat memuaskan saat ujian", resSubmit.Catatan)
		assert.True(t, resSubmit.CanVerify)

		// Get verifikasi penguji
		resGet, err := svc.GetVerifikasiPenguji(ctx, actorPenguji, pesertaID)
		require.NoError(t, err)
		assert.Equal(t, "Lulus", resGet.StatusKelulusan)
		assert.Equal(t, "2115051001", resGet.NIM)
		assert.Equal(t, "Mahasiswa Test", resGet.NamaMahasiswa)
		assert.True(t, resGet.CanVerify)
	})
}

func errorsAs(err error, target interface{}) bool {
	var appErr *httpx.AppError
	if errorsAsAppErr(err, &appErr) {
		if ptr, ok := target.(**httpx.AppError); ok {
			*ptr = appErr
			return true
		}
	}
	return false
}

func errorsAsAppErr(err error, target **httpx.AppError) bool {
	if appErr, ok := err.(*httpx.AppError); ok {
		*target = appErr
		return true
	}
	return false
}
