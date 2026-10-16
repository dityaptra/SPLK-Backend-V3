package penilaian

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/platform/httpx"
)

// ActivityLogger kontrak pencatatan log aktivitas.
type ActivityLogger interface {
	Log(ctx context.Context, userID int, action string, success bool, ip, userAgent string) error
}

// Service mendefinisikan antarmuka logika bisnis modul penilaian.
type Service interface {
	// Instrumen (Admin)
	CreateInstrumen(ctx context.Context, actor middleware.Actor, req CreateInstrumenRequest, ip, ua string) (*InstrumenResponse, error)
	GetInstrumen(ctx context.Context, id int) (*InstrumenResponse, error)
	UpdateInstrumen(ctx context.Context, actor middleware.Actor, id int, req UpdateInstrumenRequest, ip, ua string) (*InstrumenResponse, error)
	DeleteInstrumen(ctx context.Context, actor middleware.Actor, id int, ip, ua string) error
	ListInstrumen(ctx context.Context, jenisProgram, rolePenilai string) ([]InstrumenResponse, error)
	GetRingkasanBobot(ctx context.Context) (*RingkasanBobotResponse, error)

	// Penilaian Peserta
	GetFormPenilaian(ctx context.Context, actor middleware.Actor, idPendaftaran int) (*FormPenilaianResponse, error)
	SubmitPenilaian(ctx context.Context, actor middleware.Actor, idPendaftaran int, req SubmitPenilaianRequest, ip, ua string) (*SubmitPenilaianResponse, error)
	GetNilai(ctx context.Context, actor middleware.Actor, idPendaftaran int) (*RincianNilaiResponse, error)
	HitungNilaiManual(ctx context.Context, actor middleware.Actor, idPendaftaran int, ip, ua string) (*HitungNilaiManualResponse, error)
}

type service struct {
	pool       *pgxpool.Pool
	repo       Repository
	accessSvc  pendaftaran.AccessService
	calculator ScoreCalculator
	logger     ActivityLogger
}

// NewService membuat instance Service penilaian baru.
func NewService(
	pool *pgxpool.Pool,
	repo Repository,
	accessSvc pendaftaran.AccessService,
	calculator ScoreCalculator,
	logger ActivityLogger,
) Service {
	if calculator == nil {
		calculator = NewScoreCalculator()
	}
	return &service{
		pool:       pool,
		repo:       repo,
		accessSvc:  accessSvc,
		calculator: calculator,
		logger:     logger,
	}
}

func (s *service) runInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if s.pool == nil {
		return fn(nil)
	}
	return database.WithTx(ctx, s.pool, fn)
}

// =========================================================================
// CRUD INSTRUMEN PENILAIAN (Admin)
// =========================================================================

func (s *service) CreateInstrumen(ctx context.Context, actor middleware.Actor, req CreateInstrumenRequest, ip, ua string) (*InstrumenResponse, error) {
	jenis := strings.TrimSpace(req.JenisProgram)
	if jenis != JenisMagang && jenis != JenisAM && jenis != JenisKKN {
		return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_VALID", "jenis program harus magang, asistensi_mengajar, atau kkn")
	}

	role := strings.TrimSpace(req.RolePenilai)
	if role != RoleDosenPembimbing && role != RoleDosenPenguji && role != RolePembimbingLapangan {
		return nil, httpx.Unprocessable("ROLE_PENILAI_TIDAK_VALID", "role penilai harus dosen_pembimbing, dosen_penguji, atau pembimbing_lapangan")
	}

	nama := strings.TrimSpace(req.NamaAspek)
	if nama == "" {
		return nil, httpx.Unprocessable("NAMA_ASPEK_WAJIB", "nama aspek tidak boleh kosong")
	}

	if req.Bobot <= 0 || req.Bobot > 100 {
		return nil, httpx.Unprocessable("BOBOT_TIDAK_VALID", "bobot harus bernilai lebih dari 0 dan maksimal 100")
	}

	item := &InstrumenPenilaian{
		JenisProgram: jenis,
		NamaAspek:    nama,
		Bobot:        req.Bobot,
		RolePenilai:  role,
	}

	if err := s.repo.CreateInstrumen(ctx, s.pool, item); err != nil {
		return nil, fmt.Errorf("simpan instrumen penilaian: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "tambah_instrumen_penilaian", true, ip, ua)
	}

	return &InstrumenResponse{
		ID:           item.ID,
		JenisProgram: item.JenisProgram,
		NamaAspek:    item.NamaAspek,
		Bobot:        item.Bobot,
		RolePenilai:  item.RolePenilai,
	}, nil
}

func (s *service) GetInstrumen(ctx context.Context, id int) (*InstrumenResponse, error) {
	item, err := s.repo.GetInstrumenByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil instrumen penilaian: %w", err)
	}
	if item == nil {
		return nil, httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan")
	}

	return &InstrumenResponse{
		ID:           item.ID,
		JenisProgram: item.JenisProgram,
		NamaAspek:    item.NamaAspek,
		Bobot:        item.Bobot,
		RolePenilai:  item.RolePenilai,
	}, nil
}

func (s *service) UpdateInstrumen(ctx context.Context, actor middleware.Actor, id int, req UpdateInstrumenRequest, ip, ua string) (*InstrumenResponse, error) {
	item, err := s.repo.GetInstrumenByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil instrumen penilaian: %w", err)
	}
	if item == nil {
		return nil, httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan")
	}

	if req.JenisProgram != nil {
		jenis := strings.TrimSpace(*req.JenisProgram)
		if jenis != JenisMagang && jenis != JenisAM && jenis != JenisKKN {
			return nil, httpx.Unprocessable("JENIS_PROGRAM_TIDAK_VALID", "jenis program harus magang, asistensi_mengajar, atau kkn")
		}
		item.JenisProgram = jenis
	}

	if req.RolePenilai != nil {
		role := strings.TrimSpace(*req.RolePenilai)
		if role != RoleDosenPembimbing && role != RoleDosenPenguji && role != RolePembimbingLapangan {
			return nil, httpx.Unprocessable("ROLE_PENILAI_TIDAK_VALID", "role penilai harus dosen_pembimbing, dosen_penguji, atau pembimbing_lapangan")
		}
		item.RolePenilai = role
	}

	if req.NamaAspek != nil {
		nama := strings.TrimSpace(*req.NamaAspek)
		if nama == "" {
			return nil, httpx.Unprocessable("NAMA_ASPEK_WAJIB", "nama aspek tidak boleh kosong")
		}
		item.NamaAspek = nama
	}

	if req.Bobot != nil {
		if *req.Bobot <= 0 || *req.Bobot > 100 {
			return nil, httpx.Unprocessable("BOBOT_TIDAK_VALID", "bobot harus bernilai lebih dari 0 dan maksimal 100")
		}
		item.Bobot = *req.Bobot
	}

	if err := s.repo.UpdateInstrumen(ctx, s.pool, item); err != nil {
		return nil, fmt.Errorf("perbarui instrumen penilaian: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "ubah_instrumen_penilaian", true, ip, ua)
	}

	return &InstrumenResponse{
		ID:           item.ID,
		JenisProgram: item.JenisProgram,
		NamaAspek:    item.NamaAspek,
		Bobot:        item.Bobot,
		RolePenilai:  item.RolePenilai,
	}, nil
}

func (s *service) DeleteInstrumen(ctx context.Context, actor middleware.Actor, id int, ip, ua string) error {
	item, err := s.repo.GetInstrumenByID(ctx, s.pool, id)
	if err != nil {
		return fmt.Errorf("ambil instrumen penilaian: %w", err)
	}
	if item == nil {
		return httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan")
	}

	if err := s.repo.DeleteInstrumen(ctx, s.pool, id); err != nil {
		return fmt.Errorf("hapus instrumen penilaian: %w", err)
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "hapus_instrumen_penilaian", true, ip, ua)
	}

	return nil
}

func (s *service) ListInstrumen(ctx context.Context, jenisProgram, rolePenilai string) ([]InstrumenResponse, error) {
	items, err := s.repo.ListInstrumen(ctx, s.pool, jenisProgram, rolePenilai)
	if err != nil {
		return nil, fmt.Errorf("ambil daftar instrumen: %w", err)
	}

	resp := make([]InstrumenResponse, len(items))
	for i, it := range items {
		resp[i] = InstrumenResponse{
			ID:           it.ID,
			JenisProgram: it.JenisProgram,
			NamaAspek:    it.NamaAspek,
			Bobot:        it.Bobot,
			RolePenilai:  it.RolePenilai,
		}
	}
	return resp, nil
}

// GetRingkasanBobot menampilkan total bobot per (jenis_program, role_penilai) dan total keseluruhan.
// Catatan: tidak memaksa total 100 sesuai ASUMSI A4.
func (s *service) GetRingkasanBobot(ctx context.Context) (*RingkasanBobotResponse, error) {
	programs, err := s.repo.GetRingkasanBobot(ctx, s.pool)
	if err != nil {
		return nil, fmt.Errorf("ambil ringkasan bobot: %w", err)
	}

	return &RingkasanBobotResponse{
		Programs: programs,
	}, nil
}

// =========================================================================
// FORM PENILAIAN & PENGISIAN PENILAIAN
// =========================================================================

func (s *service) GetFormPenilaian(ctx context.Context, actor middleware.Actor, idPendaftaran int) (*FormPenilaianResponse, error) {
	// 1. Tentukan relasi pemanggil terhadap pendaftaran
	rel, err := s.accessSvc.Resolve(ctx, actor, idPendaftaran)
	if err != nil {
		return nil, err
	}

	rolePenilai := string(rel)
	if rolePenilai != RoleDosenPembimbing && rolePenilai != RoleDosenPenguji && rolePenilai != RolePembimbingLapangan {
		return nil, httpx.Forbidden("AKSES_DITOLAK", "hanya penilai sah (dosen pembimbing, penguji, atau pembimbing lapangan) yang dapat mengakses form penilaian")
	}

	// 2. Ambil info pendaftaran
	peserta, err := s.repo.GetPesertaPenilaianInfo(ctx, s.pool, idPendaftaran)
	if err != nil {
		return nil, fmt.Errorf("ambil data peserta: %w", err)
	}
	if peserta == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// 3. Ambil instrumen yang cocok dengan program peserta dan role pemanggil
	allInstrumen, err := s.repo.ListInstrumen(ctx, s.pool, peserta.JenisProgram, rolePenilai)
	if err != nil {
		return nil, fmt.Errorf("ambil instrumen penilaian: %w", err)
	}

	// 4. Ambil penilaian yang sudah pernah diisi oleh pemanggil
	existingNilai, err := s.repo.GetPenilaianByPeserta(ctx, s.pool, idPendaftaran)
	if err != nil {
		return nil, fmt.Errorf("ambil penilaian peserta: %w", err)
	}

	nilaiMap := make(map[int]Penilaian)
	for _, n := range existingNilai {
		if n.IDPenilai == actor.UserID {
			nilaiMap[n.IDInstrumen] = n
		}
	}

	terkunci := peserta.NilaiAkhir != nil

	itemsResp := make([]FormPenilaianItemResponse, len(allInstrumen))
	for i, inst := range allInstrumen {
		var nilaiPtr *float64
		var catatanPtr *string
		sudah := false

		if val, ok := nilaiMap[inst.ID]; ok {
			v := val.Nilai
			nilaiPtr = &v
			catatanPtr = val.Catatan
			sudah = true
		}

		itemsResp[i] = FormPenilaianItemResponse{
			IDInstrumen:  inst.ID,
			NamaAspek:    inst.NamaAspek,
			Bobot:        inst.Bobot,
			RolePenilai:  inst.RolePenilai,
			Nilai:        nilaiPtr,
			Catatan:      catatanPtr,
			SudahDinilai: sudah,
		}
	}

	return &FormPenilaianResponse{
		PendaftaranID: idPendaftaran,
		JenisProgram:  peserta.JenisProgram,
		RolePenilai:   rolePenilai,
		Terkunci:      terkunci,
		NilaiAkhir:    peserta.NilaiAkhir,
		GradeHuruf:    peserta.GradeHuruf,
		Items:         itemsResp,
	}, nil
}

func (s *service) SubmitPenilaian(ctx context.Context, actor middleware.Actor, idPendaftaran int, req SubmitPenilaianRequest, ip, ua string) (*SubmitPenilaianResponse, error) {
	if len(req.Items) == 0 {
		return nil, httpx.BadRequest("ITEMS_KOSONG", "daftar nilai yang dikirim tidak boleh kosong")
	}

	// Evaluasi relasi pemanggil di awal
	rel, err := s.accessSvc.Resolve(ctx, actor, idPendaftaran)
	if err != nil {
		return nil, err
	}

	rolePenilai := string(rel)
	if rolePenilai != RoleDosenPembimbing && rolePenilai != RoleDosenPenguji && rolePenilai != RolePembimbingLapangan {
		return nil, httpx.Forbidden("PENILAI_TIDAK_SAH", "hanya penilai sah yang dapat mengirimkan penilaian")
	}

	var res SubmitPenilaianResponse
	res.PendaftaranID = idPendaftaran

	err = s.runInTx(ctx, func(tx pgx.Tx) error {
		// 1. Kunci baris peserta FOR UPDATE untuk konsistensi konkurensi (ARCHITECTURE.md Bagian 8)
		peserta, err := s.repo.GetPesertaForUpdate(ctx, tx, idPendaftaran)
		if err != nil {
			return fmt.Errorf("kunci data peserta: %w", err)
		}
		if peserta == nil {
			return httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
		}

		// 2. BR-10: Status pendaftaran harus 'selesai'
		if peserta.Status != pendaftaran.StatusSelesai {
			return httpx.Unprocessable("STATUS_TIDAK_SESUAI", "penilaian hanya dapat dilakukan pada pendaftaran berstatus selesai")
		}

		// 3. BR-11: Jika nilai akhir sudah terhitung, penilaian terkunci
		if peserta.NilaiAkhir != nil {
			return httpx.Unprocessable("NILAI_SUDAH_TERKUNCI", "penilaian pada pendaftaran ini sudah selesai dan nilai akhir telah terkunci")
		}

		// 4. Pastikan penilai adalah user sah untuk role tersebut pada peserta ini (BR-15)
		validUserID, err := s.repo.GetPenilaiUserIDByRole(ctx, tx, idPendaftaran, rolePenilai)
		if err != nil {
			return fmt.Errorf("validasi user penilai: %w", err)
		}
		if validUserID == nil || *validUserID != actor.UserID {
			return httpx.Forbidden("PENILAI_TIDAK_SAH", "anda bukan penilai yang ditugaskan untuk pendaftaran ini")
		}

		// 5. Validasi dan upsert setiap item penilaian
		for _, it := range req.Items {
			if it.Nilai == nil {
				return httpx.Unprocessable("NILAI_WAJIB", "nilai wajib diisi")
			}
			nilai := *it.Nilai
			// BR-10: Nilai harus antara 0 sampai 100
			if nilai < 0 || nilai > 100 {
				return httpx.Unprocessable("NILAI_DI_LUAR_RENTANG", "nilai harus berada di antara 0 sampai 100")
			}

			inst, err := s.repo.GetInstrumenByID(ctx, tx, it.IDInstrumen)
			if err != nil {
				return fmt.Errorf("ambil instrumen %d: %w", it.IDInstrumen, err)
			}
			if inst == nil {
				return httpx.Unprocessable("INSTRUMEN_TIDAK_DITEMUKAN", fmt.Sprintf("instrumen penilaian id %d tidak ditemukan", it.IDInstrumen))
			}

			// Validasi instrumen cocok jenis program
			if inst.JenisProgram != peserta.JenisProgram {
				return httpx.Unprocessable("INSTRUMEN_TIDAK_COCOK", fmt.Sprintf("instrumen id %d tidak cocok dengan jenis program pendaftaran (%s)", it.IDInstrumen, peserta.JenisProgram))
			}

			// Validasi instrumen cocok role penilai
			if inst.RolePenilai != rolePenilai {
				return httpx.Unprocessable("ROLE_INSTRUMEN_TIDAK_COCOK", fmt.Sprintf("instrumen id %d memiliki role %s, tidak cocok dengan role penilai pemanggil (%s)", it.IDInstrumen, inst.RolePenilai, rolePenilai))
			}

			// Upsert penilaian
			pen := &Penilaian{
				IDPesertaProgram: idPendaftaran,
				IDInstrumen:      it.IDInstrumen,
				IDPenilai:        actor.UserID,
				Nilai:            nilai,
				Catatan:          it.Catatan,
			}
			if err := s.repo.UpsertPenilaian(ctx, tx, pen); err != nil {
				return fmt.Errorf("upsert penilaian instrumen %d: %w", it.IDInstrumen, err)
			}
		}

		// 6. Evaluasi kelengkapan seluruh instrumen program peserta (ARCHITECTURE.md Bagian 8)
		programInstrumen, err := s.repo.GetInstrumenByJenisProgram(ctx, tx, peserta.JenisProgram)
		if err != nil {
			return fmt.Errorf("ambil instrumen program: %w", err)
		}

		if len(programInstrumen) == 0 {
			// Jika instrumen belum diatur, tidak ada penghitungan otomatis
			res.SemuaInstrumenTerisi = false
			res.Pesan = "penilaian berhasil disimpan, instrumen program belum diatur sehingga nilai akhir belum dihitung"
			return nil
		}

		allPenilaian, err := s.repo.GetPenilaianByPeserta(ctx, tx, idPendaftaran)
		if err != nil {
			return fmt.Errorf("ambil seluruh penilaian peserta: %w", err)
		}

		// Map penilaian berdasarkan (id_instrumen, id_penilai)
		penMap := make(map[string]Penilaian)
		for _, p := range allPenilaian {
			key := fmt.Sprintf("%d_%d", p.IDInstrumen, p.IDPenilai)
			penMap[key] = p
		}

		// Periksa setiap instrumen program apakah sudah diisi oleh penilai sah masing-masing
		semuaLengkap := true
		nilaiCalcItems := make([]NilaiInstrumen, 0, len(programInstrumen))

		for _, inst := range programInstrumen {
			expectedUID, err := s.repo.GetPenilaiUserIDByRole(ctx, tx, idPendaftaran, inst.RolePenilai)
			if err != nil {
				return fmt.Errorf("ambil penilai role %s: %w", inst.RolePenilai, err)
			}
			if expectedUID == nil {
				semuaLengkap = false
				break
			}

			key := fmt.Sprintf("%d_%d", inst.ID, *expectedUID)
			val, ok := penMap[key]
			if !ok {
				semuaLengkap = false
				break
			}

			nilaiCalcItems = append(nilaiCalcItems, NilaiInstrumen{
				IDInstrumen: inst.ID,
				NamaAspek:   inst.NamaAspek,
				RolePenilai: inst.RolePenilai,
				Bobot:       inst.Bobot,
				Nilai:       val.Nilai,
			})
		}

		if semuaLengkap {
			// BR-11: Hitung dan simpan nilai_akhir dan grade_huruf
			nilaiAkhir, gradeHuruf := s.calculator.Hitung(nilaiCalcItems)
			if err := s.repo.UpdateNilaiAkhir(ctx, tx, idPendaftaran, nilaiAkhir, gradeHuruf); err != nil {
				return fmt.Errorf("simpan nilai akhir: %w", err)
			}

			res.SemuaInstrumenTerisi = true
			res.NilaiAkhir = &nilaiAkhir
			res.GradeHuruf = &gradeHuruf
			res.Pesan = "penilaian berhasil disimpan dan seluruh instrumen telah lengkap. Nilai akhir berhasil dihitung dan dikunci."
		} else {
			res.SemuaInstrumenTerisi = false
			res.Pesan = "penilaian berhasil disimpan. Menunggu penilai lain melengkapi instrumen sebelum nilai akhir dihitung."
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "submit_penilaian", true, ip, ua)
	}

	return &res, nil
}

// GetNilai menampilkan rincian nilai per instrumen dan nilai akhir.
// Mahasiswa pemilik hanya dapat melihat rincian setelah nilai_akhir terisi.
func (s *service) GetNilai(ctx context.Context, actor middleware.Actor, idPendaftaran int) (*RincianNilaiResponse, error) {
	rel, err := s.accessSvc.Resolve(ctx, actor, idPendaftaran)
	if err != nil {
		return nil, err
	}

	peserta, err := s.repo.GetPesertaPenilaianInfo(ctx, s.pool, idPendaftaran)
	if err != nil {
		return nil, fmt.Errorf("ambil data peserta: %w", err)
	}
	if peserta == nil {
		return nil, httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
	}

	// Aturan pembatasan untuk mahasiswa pemilik:
	// Mahasiswa pemilik hanya melihat rincian setelah nilai_akhir terisi.
	if rel == pendaftaran.RelMahasiswa && peserta.NilaiAkhir == nil {
		return nil, httpx.Forbidden("NILAI_BELUM_TERSEDIA", "rincian nilai belum tersedia karena proses penilaian belum selesai")
	}

	details, err := s.repo.GetNilaiDetailByPeserta(ctx, s.pool, idPendaftaran)
	if err != nil {
		return nil, fmt.Errorf("ambil rincian nilai: %w", err)
	}

	items := make([]NilaiItemDetailResponse, len(details))
	for i, d := range details {
		items[i] = NilaiItemDetailResponse{
			IDInstrumen:  d.IDInstrumen,
			NamaAspek:    d.NamaAspek,
			Bobot:        d.Bobot,
			RolePenilai:  d.RolePenilai,
			Nilai:        d.Nilai,
			Catatan:      d.Catatan,
			NamaPenilai:  d.NamaPenilai,
			TanggalNilai: d.TanggalNilai.Format("2006-01-02 15:04:05"),
		}
	}

	return &RincianNilaiResponse{
		PendaftaranID: idPendaftaran,
		JenisProgram:  peserta.JenisProgram,
		StatusPeserta: peserta.Status,
		NilaiAkhir:    peserta.NilaiAkhir,
		GradeHuruf:    peserta.GradeHuruf,
		Items:         items,
	}, nil
}

// HitungNilaiManual endpoint POST /admin/pendaftaran/:id/hitung-nilai untuk menghitung ulang manual oleh admin.
// Bila instrumen jenis program kosong atau penilaian belum lengkap, balas 422 dengan kode error jelas.
func (s *service) HitungNilaiManual(ctx context.Context, actor middleware.Actor, idPendaftaran int, ip, ua string) (*HitungNilaiManualResponse, error) {
	var res HitungNilaiManualResponse
	res.PendaftaranID = idPendaftaran

	err := s.runInTx(ctx, func(tx pgx.Tx) error {
		peserta, err := s.repo.GetPesertaForUpdate(ctx, tx, idPendaftaran)
		if err != nil {
			return fmt.Errorf("kunci data peserta: %w", err)
		}
		if peserta == nil {
			return httpx.NotFound("PENDAFTARAN_TIDAK_DITEMUKAN", "pendaftaran tidak ditemukan")
		}

		if peserta.Status != pendaftaran.StatusSelesai {
			return httpx.Unprocessable("STATUS_TIDAK_SESUAI", "hitung nilai hanya dapat dilakukan pada pendaftaran berstatus selesai")
		}

		// Ambil template instrumen untuk program peserta
		programInstrumen, err := s.repo.GetInstrumenByJenisProgram(ctx, tx, peserta.JenisProgram)
		if err != nil {
			return fmt.Errorf("ambil instrumen program: %w", err)
		}

		if len(programInstrumen) == 0 {
			return httpx.Unprocessable("INSTRUMEN_BELUM_DIATUR", "instrumen penilaian untuk jenis program ini belum diatur")
		}

		allPenilaian, err := s.repo.GetPenilaianByPeserta(ctx, tx, idPendaftaran)
		if err != nil {
			return fmt.Errorf("ambil seluruh penilaian peserta: %w", err)
		}

		penMap := make(map[string]Penilaian)
		for _, p := range allPenilaian {
			key := fmt.Sprintf("%d_%d", p.IDInstrumen, p.IDPenilai)
			penMap[key] = p
		}

		nilaiCalcItems := make([]NilaiInstrumen, 0, len(programInstrumen))
		for _, inst := range programInstrumen {
			expectedUID, err := s.repo.GetPenilaiUserIDByRole(ctx, tx, idPendaftaran, inst.RolePenilai)
			if err != nil {
				return fmt.Errorf("ambil penilai role %s: %w", inst.RolePenilai, err)
			}
			if expectedUID == nil {
				return httpx.Unprocessable("PENILAIAN_BELUM_LENGKAP", fmt.Sprintf("penilai untuk role %s belum ditugaskan pada pendaftaran ini", inst.RolePenilai))
			}

			key := fmt.Sprintf("%d_%d", inst.ID, *expectedUID)
			val, ok := penMap[key]
			if !ok {
				return httpx.Unprocessable("PENILAIAN_BELUM_LENGKAP", fmt.Sprintf("instrumen '%s' belum dinilai oleh penilai berwenang (%s)", inst.NamaAspek, inst.RolePenilai))
			}

			nilaiCalcItems = append(nilaiCalcItems, NilaiInstrumen{
				IDInstrumen: inst.ID,
				NamaAspek:   inst.NamaAspek,
				RolePenilai: inst.RolePenilai,
				Bobot:       inst.Bobot,
				Nilai:       val.Nilai,
			})
		}

		nilaiAkhir, gradeHuruf := s.calculator.Hitung(nilaiCalcItems)
		if err := s.repo.UpdateNilaiAkhir(ctx, tx, idPendaftaran, nilaiAkhir, gradeHuruf); err != nil {
			return fmt.Errorf("simpan nilai akhir: %w", err)
		}

		res.NilaiAkhir = nilaiAkhir
		res.GradeHuruf = gradeHuruf
		res.Pesan = "penghitungan nilai akhir berhasil"
		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.logger != nil {
		_ = s.logger.Log(ctx, actor.UserID, "hitung_nilai_manual", true, ip, ua)
	}

	return &res, nil
}
