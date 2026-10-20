package penilaian

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
)

// Repository antarmuka akses data untuk modul penilaian.
type Repository interface {
	CreateInstrumen(ctx context.Context, db database.DBTX, i *InstrumenPenilaian) error
	GetInstrumenByID(ctx context.Context, db database.DBTX, id int) (*InstrumenPenilaian, error)
	UpdateInstrumen(ctx context.Context, db database.DBTX, i *InstrumenPenilaian) error
	DeleteInstrumen(ctx context.Context, db database.DBTX, id int) error
	ListInstrumen(ctx context.Context, db database.DBTX, jenisProgram, rolePenilai string) ([]InstrumenPenilaian, error)
	GetRingkasanBobot(ctx context.Context, db database.DBTX) ([]RingkasanBobotProgram, error)

	GetPesertaPenilaianInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaPenilaianInfo, error)
	GetPesertaForUpdate(ctx context.Context, tx pgx.Tx, idPeserta int) (*PesertaPenilaianInfo, error)
	GetPenilaiUserIDByRole(ctx context.Context, db database.DBTX, idPeserta int, role string) (*int, error)
	GetInstrumenByJenisProgram(ctx context.Context, db database.DBTX, jenisProgram string) ([]InstrumenPenilaian, error)
	GetPenilaianByPeserta(ctx context.Context, db database.DBTX, idPeserta int) ([]Penilaian, error)
	GetNilaiDetailByPeserta(ctx context.Context, db database.DBTX, idPeserta int) ([]NilaiDetail, error)
	UpsertPenilaian(ctx context.Context, tx pgx.Tx, p *Penilaian) error
	UpdateNilaiAkhir(ctx context.Context, tx pgx.Tx, idPeserta int, nilaiAkhir float64, gradeHuruf string) error

	// Kelulusan Penguji (Magang)
	GetKelulusanPengujiByPeserta(ctx context.Context, db database.DBTX, idPeserta int) (*KelulusanPenguji, error)
	UpsertKelulusanPenguji(ctx context.Context, db database.DBTX, k *KelulusanPenguji) error
	GetMahasiswaDanPengujiInfo(ctx context.Context, db database.DBTX, idPeserta int) (nim, namaMhs string, idDosen *int, namaDosen *string, err error)
}

type repository struct{}

// NewRepository membuat instance Repository penilaian baru.
func NewRepository() Repository {
	return &repository{}
}

func (r *repository) CreateInstrumen(ctx context.Context, db database.DBTX, i *InstrumenPenilaian) error {
	query := `
		INSERT INTO instrumen_penilaian (jenis_program, nama_aspek, bobot, role_penilai)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	return db.QueryRow(ctx, query, i.JenisProgram, i.NamaAspek, i.Bobot, i.RolePenilai).Scan(&i.ID)
}

func (r *repository) GetInstrumenByID(ctx context.Context, db database.DBTX, id int) (*InstrumenPenilaian, error) {
	query := `
		SELECT id, jenis_program, nama_aspek, bobot, role_penilai
		FROM instrumen_penilaian
		WHERE id = $1
	`
	var i InstrumenPenilaian
	err := db.QueryRow(ctx, query, id).Scan(&i.ID, &i.JenisProgram, &i.NamaAspek, &i.Bobot, &i.RolePenilai)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *repository) UpdateInstrumen(ctx context.Context, db database.DBTX, i *InstrumenPenilaian) error {
	query := `
		UPDATE instrumen_penilaian
		SET jenis_program = $1, nama_aspek = $2, bobot = $3, role_penilai = $4
		WHERE id = $5
	`
	cmd, err := db.Exec(ctx, query, i.JenisProgram, i.NamaAspek, i.Bobot, i.RolePenilai, i.ID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) DeleteInstrumen(ctx context.Context, db database.DBTX, id int) error {
	cmd, err := db.Exec(ctx, "DELETE FROM instrumen_penilaian WHERE id = $1", id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) ListInstrumen(ctx context.Context, db database.DBTX, jenisProgram, rolePenilai string) ([]InstrumenPenilaian, error) {
	query := `
		SELECT id, jenis_program, nama_aspek, bobot, role_penilai
		FROM instrumen_penilaian
		WHERE ($1 = '' OR jenis_program = $1)
		  AND ($2 = '' OR role_penilai = $2)
		ORDER BY jenis_program, role_penilai, id
	`
	rows, err := db.Query(ctx, query, jenisProgram, rolePenilai)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]InstrumenPenilaian, 0)
	for rows.Next() {
		var i InstrumenPenilaian
		if err := rows.Scan(&i.ID, &i.JenisProgram, &i.NamaAspek, &i.Bobot, &i.RolePenilai); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *repository) GetRingkasanBobot(ctx context.Context, db database.DBTX) ([]RingkasanBobotProgram, error) {
	query := `
		SELECT jenis_program, role_penilai, COUNT(*), COALESCE(SUM(bobot), 0)
		FROM instrumen_penilaian
		GROUP BY jenis_program, role_penilai
		ORDER BY jenis_program, role_penilai
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progMap := make(map[string]*RingkasanBobotProgram)
	progOrder := make([]string, 0)

	// Pastikan jenis program standar muncul di ringkasan
	defaultPrograms := []string{JenisMagang, JenisAM, JenisKKN}
	for _, p := range defaultPrograms {
		progMap[p] = &RingkasanBobotProgram{
			JenisProgram:          p,
			TotalBobotKeseluruhan: 0,
			PerRole:               make([]RingkasanBobotRole, 0),
		}
		progOrder = append(progOrder, p)
	}

	for rows.Next() {
		var jenis, role string
		var jml int
		var sumBobot float64
		if err := rows.Scan(&jenis, &role, &jml, &sumBobot); err != nil {
			return nil, err
		}

		entry, ok := progMap[jenis]
		if !ok {
			entry = &RingkasanBobotProgram{
				JenisProgram:          jenis,
				TotalBobotKeseluruhan: 0,
				PerRole:               make([]RingkasanBobotRole, 0),
			}
			progMap[jenis] = entry
			progOrder = append(progOrder, jenis)
		}

		entry.PerRole = append(entry.PerRole, RingkasanBobotRole{
			RolePenilai:     role,
			TotalBobot:      sumBobot,
			JumlahInstrumen: jml,
		})
		entry.TotalBobotKeseluruhan += sumBobot
	}

	res := make([]RingkasanBobotProgram, 0, len(progOrder))
	for _, p := range progOrder {
		res = append(res, *progMap[p])
	}
	return res, rows.Err()
}

func (r *repository) GetPesertaPenilaianInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaPenilaianInfo, error) {
	query := `
		SELECT id, id_mahasiswa, id_periode, jenis_program, status,
		       id_dosen_pembimbing, id_dosen_penguji, id_pembimbing_lapangan,
		       nilai_akhir, grade_huruf, sks_diakui
		FROM peserta_program
		WHERE id = $1
	`
	var p PesertaPenilaianInfo
	err := db.QueryRow(ctx, query, idPeserta).Scan(
		&p.ID, &p.IDMahasiswa, &p.IDPeriode, &p.JenisProgram, &p.Status,
		&p.IDDosenPembimbing, &p.IDDosenPenguji, &p.IDPembimbingLapangan,
		&p.NilaiAkhir, &p.GradeHuruf, &p.SksDiakui,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetPesertaForUpdate(ctx context.Context, tx pgx.Tx, idPeserta int) (*PesertaPenilaianInfo, error) {
	query := `
		SELECT id, id_mahasiswa, id_periode, jenis_program, status,
		       id_dosen_pembimbing, id_dosen_penguji, id_pembimbing_lapangan,
		       nilai_akhir, grade_huruf, sks_diakui
		FROM peserta_program
		WHERE id = $1
		FOR UPDATE
	`
	var p PesertaPenilaianInfo
	err := tx.QueryRow(ctx, query, idPeserta).Scan(
		&p.ID, &p.IDMahasiswa, &p.IDPeriode, &p.JenisProgram, &p.Status,
		&p.IDDosenPembimbing, &p.IDDosenPenguji, &p.IDPembimbingLapangan,
		&p.NilaiAkhir, &p.GradeHuruf, &p.SksDiakui,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetPenilaiUserIDByRole(ctx context.Context, db database.DBTX, idPeserta int, role string) (*int, error) {
	var query string
	switch role {
	case RoleDosenPembimbing:
		query = `
			SELECT d.id_user
			FROM peserta_program pp
			JOIN dosen d ON pp.id_dosen_pembimbing = d.id
			WHERE pp.id = $1
		`
	case RoleDosenPenguji:
		query = `
			SELECT d.id_user
			FROM peserta_program pp
			JOIN dosen d ON pp.id_dosen_penguji = d.id
			WHERE pp.id = $1
		`
	case RolePembimbingLapangan:
		query = `
			SELECT pl.id_user
			FROM peserta_program pp
			JOIN pembimbing_lapangan pl ON pp.id_pembimbing_lapangan = pl.id
			WHERE pp.id = $1
		`
	default:
		return nil, fmt.Errorf("role penilai tidak dikenal: %s", role)
	}

	var userID *int
	err := db.QueryRow(ctx, query, idPeserta).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return userID, nil
}

func (r *repository) GetInstrumenByJenisProgram(ctx context.Context, db database.DBTX, jenisProgram string) ([]InstrumenPenilaian, error) {
	query := `
		SELECT id, jenis_program, nama_aspek, bobot, role_penilai
		FROM instrumen_penilaian
		WHERE jenis_program = $1
		ORDER BY role_penilai, id
	`
	rows, err := db.Query(ctx, query, jenisProgram)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]InstrumenPenilaian, 0)
	for rows.Next() {
		var i InstrumenPenilaian
		if err := rows.Scan(&i.ID, &i.JenisProgram, &i.NamaAspek, &i.Bobot, &i.RolePenilai); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *repository) GetPenilaianByPeserta(ctx context.Context, db database.DBTX, idPeserta int) ([]Penilaian, error) {
	query := `
		SELECT id, id_peserta_program, id_instrumen, id_penilai, nilai, catatan, tanggal_nilai
		FROM penilaian
		WHERE id_peserta_program = $1
	`
	rows, err := db.Query(ctx, query, idPeserta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Penilaian, 0)
	for rows.Next() {
		var p Penilaian
		if err := rows.Scan(&p.ID, &p.IDPesertaProgram, &p.IDInstrumen, &p.IDPenilai, &p.Nilai, &p.Catatan, &p.TanggalNilai); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *repository) GetNilaiDetailByPeserta(ctx context.Context, db database.DBTX, idPeserta int) ([]NilaiDetail, error) {
	query := `
		SELECT p.id, ip.id, ip.nama_aspek, ip.bobot, ip.role_penilai,
		       p.id_penilai, COALESCE(u.username, ''), p.nilai, p.catatan, p.tanggal_nilai
		FROM penilaian p
		JOIN instrumen_penilaian ip ON p.id_instrumen = ip.id
		LEFT JOIN users u ON p.id_penilai = u.id
		WHERE p.id_peserta_program = $1
		ORDER BY ip.role_penilai, ip.id
	`
	rows, err := db.Query(ctx, query, idPeserta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NilaiDetail, 0)
	for rows.Next() {
		var d NilaiDetail
		if err := rows.Scan(&d.IDPenilaian, &d.IDInstrumen, &d.NamaAspek, &d.Bobot, &d.RolePenilai, &d.IDPenilai, &d.NamaPenilai, &d.Nilai, &d.Catatan, &d.TanggalNilai); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

func (r *repository) UpsertPenilaian(ctx context.Context, tx pgx.Tx, p *Penilaian) error {
	query := `
		INSERT INTO penilaian (id_peserta_program, id_instrumen, id_penilai, nilai, catatan, tanggal_nilai)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (id_peserta_program, id_instrumen, id_penilai)
		DO UPDATE SET nilai = EXCLUDED.nilai, catatan = EXCLUDED.catatan, tanggal_nilai = CURRENT_TIMESTAMP
		RETURNING id
	`
	return tx.QueryRow(ctx, query, p.IDPesertaProgram, p.IDInstrumen, p.IDPenilai, p.Nilai, p.Catatan).Scan(&p.ID)
}

func (r *repository) UpdateNilaiAkhir(ctx context.Context, tx pgx.Tx, idPeserta int, nilaiAkhir float64, gradeHuruf string) error {
	query := `
		UPDATE peserta_program
		SET nilai_akhir = $1, grade_huruf = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := tx.Exec(ctx, query, nilaiAkhir, gradeHuruf, idPeserta)
	return err
}

func (r *repository) GetKelulusanPengujiByPeserta(ctx context.Context, db database.DBTX, idPeserta int) (*KelulusanPenguji, error) {
	query := `
		SELECT id, id_peserta_program, id_dosen_penguji, status_kelulusan, COALESCE(catatan, ''), created_at, updated_at
		FROM kelulusan_penguji
		WHERE id_peserta_program = $1
	`
	var k KelulusanPenguji
	err := db.QueryRow(ctx, query, idPeserta).Scan(
		&k.ID, &k.IDPesertaProgram, &k.IDDosenPenguji, &k.StatusKelulusan, &k.Catatan, &k.CreatedAt, &k.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *repository) UpsertKelulusanPenguji(ctx context.Context, db database.DBTX, k *KelulusanPenguji) error {
	query := `
		INSERT INTO kelulusan_penguji (id_peserta_program, id_dosen_penguji, status_kelulusan, catatan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id_peserta_program)
		DO UPDATE SET id_dosen_penguji = EXCLUDED.id_dosen_penguji,
		              status_kelulusan = EXCLUDED.status_kelulusan,
		              catatan = EXCLUDED.catatan,
		              updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(ctx, query, k.IDPesertaProgram, k.IDDosenPenguji, k.StatusKelulusan, k.Catatan).Scan(
		&k.ID, &k.CreatedAt, &k.UpdatedAt,
	)
}

func (r *repository) GetMahasiswaDanPengujiInfo(ctx context.Context, db database.DBTX, idPeserta int) (nim, namaMhs string, idDosen *int, namaDosen *string, err error) {
	query := `
		SELECT 
			m.nim, 
			u.nama, 
			pp.id_dosen_penguji, 
			ud.nama
		FROM peserta_program pp
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		JOIN users u ON m.id_user = u.id
		LEFT JOIN dosen d ON pp.id_dosen_penguji = d.id
		LEFT JOIN users ud ON d.id_user = ud.id
		WHERE pp.id = $1
	`
	err = db.QueryRow(ctx, query, idPeserta).Scan(&nim, &namaMhs, &idDosen, &namaDosen)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, nil, nil
	}
	return nim, namaMhs, idDosen, namaDosen, err
}

