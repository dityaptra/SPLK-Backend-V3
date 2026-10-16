// Package pendaftaran menyediakan akses basis data PostgreSQL untuk modul pendaftaran.
package pendaftaran

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mendefinisikan seluruh operasi basis data untuk modul pendaftaran.
type Repository interface {
	CreatePeserta(ctx context.Context, db database.DBTX, p *PesertaProgram) error
	FindPesertaByID(ctx context.Context, db database.DBTX, id int) (*PesertaProgram, error)
	FindPesertaByMahasiswaAndPeriode(ctx context.Context, db database.DBTX, mhsID, periodeID int, jenisProgram string) (*PesertaProgram, error)
	UpdateStatusAndAlasan(ctx context.Context, db database.DBTX, id int, status string, alasan *string) error

	FindMahasiswaByUserID(ctx context.Context, db database.DBTX, userID int) (*MahasiswaInfo, error)
	FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (*PeriodeInfo, *time.Time, *time.Time, error)
	GetPendaftaranDetail(ctx context.Context, db database.DBTX, id int) (*PendaftaranDetail, error)
	GetDokumenTypesByPeserta(ctx context.Context, db database.DBTX, idPeserta int, idKelompok *int) ([]string, error)
	ListPendaftaran(ctx context.Context, db database.DBTX, roleScope string, actorProfileID int, filter PendaftaranFilter, p httpx.Pagination) ([]PendaftaranListItem, int64, error)

	FindMahasiswaIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)
	FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)
	FindPembimbingLapanganIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)
	IsAnggotaKelompokKKN(ctx context.Context, db database.DBTX, mhsID, kelompokID int) (bool, error)

	FindTempatProgram(ctx context.Context, db database.DBTX, id int) (*TempatProgramCheck, error)
	FindPembimbingLapangan(ctx context.Context, db database.DBTX, id int) (*PembimbingLapanganCheck, error)
	FindDosen(ctx context.Context, db database.DBTX, id int) (*DosenCheck, error)
	UpdatePenempatanMagang(ctx context.Context, db database.DBTX, idPeserta int, idTempat int, idPembimbing int) error
	UpdatePenempatanAM(ctx context.Context, db database.DBTX, idPeserta int, idTempat int, idPembimbing int) error
	UpdateDosenPembimbing(ctx context.Context, db database.DBTX, idPeserta int, idDosen int) error
	UpdateDosenPenguji(ctx context.Context, db database.DBTX, idPeserta int, idDosen int) error
}

type pgRepository struct{}

// NewRepository membuat instance Repository pendaftaran berbasis PostgreSQL.
func NewRepository() Repository {
	return &pgRepository{}
}

func (r *pgRepository) CreatePeserta(ctx context.Context, db database.DBTX, p *PesertaProgram) error {
	const q = `
		INSERT INTO peserta_program (
			id_mahasiswa, id_periode, jenis_program,
			id_tempat_program, id_dosen_pembimbing, id_dosen_penguji,
			id_pembimbing_lapangan, id_kelompok_kkn, is_ketua,
			status, alasan_tolak
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, tanggal_daftar, created_at, updated_at
	`
	return db.QueryRow(ctx, q,
		p.IDMahasiswa, p.IDPeriode, p.JenisProgram,
		p.IDTempatProgram, p.IDDosenPembimbing, p.IDDosenPenguji,
		p.IDPembimbingLapangan, p.IDKelompokKKN, p.IsKetua,
		p.Status, p.AlasanTolak,
	).Scan(&p.ID, &p.TanggalDaftar, &p.CreatedAt, &p.UpdatedAt)
}

func (r *pgRepository) FindPesertaByID(ctx context.Context, db database.DBTX, id int) (*PesertaProgram, error) {
	const q = `
		SELECT id, id_mahasiswa, id_periode, jenis_program,
		       id_tempat_program, id_dosen_pembimbing, id_dosen_penguji,
		       id_pembimbing_lapangan, id_kelompok_kkn, is_ketua,
		       status, alasan_tolak, tanggal_daftar,
		       tanggal_mulai_aktif, tanggal_selesai_aktif,
		       nilai_akhir, grade_huruf, sks_diakui,
		       created_at, updated_at
		FROM peserta_program
		WHERE id = $1
	`
	p := &PesertaProgram{}
	err := db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.IDMahasiswa, &p.IDPeriode, &p.JenisProgram,
		&p.IDTempatProgram, &p.IDDosenPembimbing, &p.IDDosenPenguji,
		&p.IDPembimbingLapangan, &p.IDKelompokKKN, &p.IsKetua,
		&p.Status, &p.AlasanTolak, &p.TanggalDaftar,
		&p.TanggalMulaiAktif, &p.TanggalSelesaiAktif,
		&p.NilaiAkhir, &p.GradeHuruf, &p.SKSdiakui,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find peserta by id %d: %w", id, err)
	}
	return p, nil
}

func (r *pgRepository) FindPesertaByMahasiswaAndPeriode(ctx context.Context, db database.DBTX, mhsID, periodeID int, jenisProgram string) (*PesertaProgram, error) {
	const q = `
		SELECT id, id_mahasiswa, id_periode, jenis_program,
		       id_tempat_program, id_dosen_pembimbing, id_dosen_penguji,
		       id_pembimbing_lapangan, id_kelompok_kkn, is_ketua,
		       status, alasan_tolak, tanggal_daftar,
		       tanggal_mulai_aktif, tanggal_selesai_aktif,
		       nilai_akhir, grade_huruf, sks_diakui,
		       created_at, updated_at
		FROM peserta_program
		WHERE id_mahasiswa = $1 AND id_periode = $2 AND jenis_program = $3
	`
	p := &PesertaProgram{}
	err := db.QueryRow(ctx, q, mhsID, periodeID, jenisProgram).Scan(
		&p.ID, &p.IDMahasiswa, &p.IDPeriode, &p.JenisProgram,
		&p.IDTempatProgram, &p.IDDosenPembimbing, &p.IDDosenPenguji,
		&p.IDPembimbingLapangan, &p.IDKelompokKKN, &p.IsKetua,
		&p.Status, &p.AlasanTolak, &p.TanggalDaftar,
		&p.TanggalMulaiAktif, &p.TanggalSelesaiAktif,
		&p.NilaiAkhir, &p.GradeHuruf, &p.SKSdiakui,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find peserta by mhs %d periode %d: %w", mhsID, periodeID, err)
	}
	return p, nil
}

func (r *pgRepository) UpdateStatusAndAlasan(ctx context.Context, db database.DBTX, id int, status string, alasan *string) error {
	const q = `
		UPDATE peserta_program
		SET status = $1, alasan_tolak = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := db.Exec(ctx, q, status, alasan, id)
	return err
}

func (r *pgRepository) FindMahasiswaByUserID(ctx context.Context, db database.DBTX, userID int) (*MahasiswaInfo, error) {
	const q = `
		SELECT id, id_user, nim, nama, id_prodi, angkatan
		FROM mahasiswa
		WHERE id_user = $1
	`
	m := &MahasiswaInfo{}
	err := db.QueryRow(ctx, q, userID).Scan(&m.ID, &m.UserID, &m.NIM, &m.Nama, &m.IDProdi, &m.Angkatan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find mahasiswa by user %d: %w", userID, err)
	}
	return m, nil
}

func (r *pgRepository) FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (*PeriodeInfo, *time.Time, *time.Time, error) {
	const q = `
		SELECT id, nama, jenis_program, tahun_semester, status,
		       tanggal_buka_daftar, tanggal_tutup_daftar
		FROM periode
		WHERE id = $1
	`
	p := &PeriodeInfo{}
	var buka, tutup *time.Time
	err := db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Nama, &p.JenisProgram, &p.TahunSemester, &p.Status,
		&buka, &tutup,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("find periode by id %d: %w", id, err)
	}
	return p, buka, tutup, nil
}

func (r *pgRepository) GetPendaftaranDetail(ctx context.Context, db database.DBTX, id int) (*PendaftaranDetail, error) {
	const q = `
		SELECT
			pp.id, pp.jenis_program, pp.status, pp.alasan_tolak,
			pp.tanggal_daftar, pp.tanggal_mulai_aktif, pp.tanggal_selesai_aktif,
			pp.nilai_akhir, pp.grade_huruf, pp.sks_diakui,
			pp.created_at, pp.updated_at,
			-- Mahasiswa
			m.id, m.nim, m.nama, m.angkatan, COALESCE(ps.nama, ''),
			-- Periode
			per.id, per.nama, per.jenis_program, per.tahun_semester, per.status,
			-- Tempat Program
			tp.id, tp.nama, tp.jenis_program, tp.alamat,
			-- Dosen Pembimbing
			dp.id, dp.nip, dp.nama,
			-- Dosen Penguji
			dq.id, dq.nip, dq.nama,
			-- Pembimbing Lapangan
			pl.id, pl.nama, pl.jenis,
			-- Kelompok KKN
			kk.id, kk.nama_kelompok
		FROM peserta_program pp
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		LEFT JOIN program_studi ps ON m.id_prodi = ps.id
		JOIN periode per ON pp.id_periode = per.id
		LEFT JOIN tempat_program tp ON pp.id_tempat_program = tp.id
		LEFT JOIN dosen dp ON pp.id_dosen_pembimbing = dp.id
		LEFT JOIN dosen dq ON pp.id_dosen_penguji = dq.id
		LEFT JOIN pembimbing_lapangan pl ON pp.id_pembimbing_lapangan = pl.id
		LEFT JOIN kelompok_kkn kk ON pp.id_kelompok_kkn = kk.id
		WHERE pp.id = $1
	`

	var (
		res                        PendaftaranDetail
		tDaftar, cAt, uAt          time.Time
		tMulaiAktif, tSelesaiAktif *time.Time
		tpID                       *int
		tpNama, tpJenis, tpAlamat  *string
		dpID                       *int
		dpNIP, dpNama              *string
		dqID                       *int
		dqNIP, dqNama              *string
		plID                       *int
		plNama, plJenis            *string
		kkID                       *int
		kkNama                     *string
	)

	err := db.QueryRow(ctx, q, id).Scan(
		&res.ID, &res.JenisProgram, &res.Status, &res.AlasanTolak,
		&tDaftar, &tMulaiAktif, &tSelesaiAktif,
		&res.NilaiAkhir, &res.GradeHuruf, &res.SKSdiakui,
		&cAt, &uAt,
		// Mahasiswa
		&res.Mahasiswa.ID, &res.Mahasiswa.NIM, &res.Mahasiswa.Nama, &res.Mahasiswa.Angkatan, &res.Mahasiswa.Prodi,
		// Periode
		&res.Periode.ID, &res.Periode.Nama, &res.Periode.JenisProgram, &res.Periode.TahunSemester, &res.Periode.Status,
		// Tempat
		&tpID, &tpNama, &tpJenis, &tpAlamat,
		// Dosen Pembimbing
		&dpID, &dpNIP, &dpNama,
		// Dosen Penguji
		&dqID, &dqNIP, &dqNama,
		// Pembimbing Lapangan
		&plID, &plNama, &plJenis,
		// Kelompok KKN
		&kkID, &kkNama,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("detail pendaftaran by id %d: %w", id, err)
	}

	res.TanggalDaftar = tDaftar.Format(time.RFC3339)
	res.CreatedAt = cAt.Format(time.RFC3339)
	res.UpdatedAt = uAt.Format(time.RFC3339)

	if tMulaiAktif != nil {
		s := tMulaiAktif.Format("2006-01-02")
		res.TanggalMulaiAktif = &s
	}
	if tSelesaiAktif != nil {
		s := tSelesaiAktif.Format("2006-01-02")
		res.TanggalSelesaiAktif = &s
	}

	if tpID != nil {
		res.TempatProgram = &TempatSummary{
			ID:           *tpID,
			Nama:         stringValue(tpNama),
			JenisProgram: stringValue(tpJenis),
			Alamat:       stringValue(tpAlamat),
		}
	}
	if dpID != nil {
		res.DosenPembimbing = &DosenSummary{
			ID:   *dpID,
			NIP:  stringValue(dpNIP),
			Nama: stringValue(dpNama),
		}
	}
	if dqID != nil {
		res.DosenPenguji = &DosenSummary{
			ID:   *dqID,
			NIP:  stringValue(dqNIP),
			Nama: stringValue(dqNama),
		}
	}
	if plID != nil {
		res.PembimbingLapangan = &PembimbingLapanganSummary{
			ID:    *plID,
			Nama:  stringValue(plNama),
			Jenis: stringValue(plJenis),
		}
	}
	if kkID != nil {
		res.KelompokKKN = &KelompokKKNSummary{
			ID:           *kkID,
			NamaKelompok: stringValue(kkNama),
		}
	}

	return &res, nil
}

func (r *pgRepository) GetDokumenTypesByPeserta(ctx context.Context, db database.DBTX, idPeserta int, idKelompok *int) ([]string, error) {
	const q = `
		SELECT DISTINCT jenis_dokumen
		FROM dokumen
		WHERE id_peserta_program = $1
		   OR ($2::int IS NOT NULL AND id_kelompok_kkn = $2)
	`
	rows, err := db.Query(ctx, q, idPeserta, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("ambil jenis dokumen: %w", err)
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan jenis dokumen: %w", err)
		}
		types = append(types, t)
	}
	return types, nil
}

func (r *pgRepository) ListPendaftaran(ctx context.Context, db database.DBTX, roleScope string, actorProfileID int, filter PendaftaranFilter, p httpx.Pagination) ([]PendaftaranListItem, int64, error) {
	conditions := []string{}
	args := []any{}
	argIdx := 1

	// Filter berdasarkan cakupan role (Scope)
	switch roleScope {
	case "mahasiswa":
		conditions = append(conditions, fmt.Sprintf("pp.id_mahasiswa = $%d", argIdx))
		args = append(args, actorProfileID)
		argIdx++
	case "dosen":
		conditions = append(conditions, fmt.Sprintf("(pp.id_dosen_pembimbing = $%d OR pp.id_dosen_penguji = $%d)", argIdx, argIdx))
		args = append(args, actorProfileID)
		argIdx++
	case "pembimbing_lapangan":
		conditions = append(conditions, fmt.Sprintf("pp.id_pembimbing_lapangan = $%d", argIdx))
		args = append(args, actorProfileID)
		argIdx++
	case "admin", "pegawai":
		// Bebas melihat semua
	}

	// Filter query parameter
	if filter.IDPeriode != nil {
		conditions = append(conditions, fmt.Sprintf("pp.id_periode = $%d", argIdx))
		args = append(args, *filter.IDPeriode)
		argIdx++
	}
	if filter.JenisProgram != "" {
		conditions = append(conditions, fmt.Sprintf("pp.jenis_program = $%d", argIdx))
		args = append(args, filter.JenisProgram)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("pp.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Q != "" {
		conditions = append(conditions, fmt.Sprintf("(m.nama ILIKE '%%' || $%d || '%%' OR m.nim ILIKE '%%' || $%d || '%%')", argIdx, argIdx))
		args = append(args, filter.Q)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQ := fmt.Sprintf(`
		SELECT count(*)
		FROM peserta_program pp
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		JOIN periode per ON pp.id_periode = per.id
		%s
	`, whereClause)

	var total int64
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pendaftaran: %w", err)
	}

	// Query data dengan paginasi
	offset := (p.Page - 1) * p.Limit
	listArgs := append(args, p.Limit, offset)
	listQ := fmt.Sprintf(`
		SELECT
			pp.id, pp.id_mahasiswa, m.nim, m.nama,
			pp.id_periode, per.nama, pp.jenis_program,
			pp.status, pp.tanggal_daftar, pp.nilai_akhir, pp.grade_huruf
		FROM peserta_program pp
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		JOIN periode per ON pp.id_periode = per.id
		%s
		ORDER BY pp.tanggal_daftar DESC, pp.id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := db.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pendaftaran: %w", err)
	}
	defer rows.Close()

	var items []PendaftaranListItem
	for rows.Next() {
		var item PendaftaranListItem
		var tDaftar time.Time
		if err := rows.Scan(
			&item.ID, &item.IDMahasiswa, &item.NIMMahasiswa, &item.NamaMahasiswa,
			&item.IDPeriode, &item.NamaPeriode, &item.JenisProgram,
			&item.Status, &tDaftar, &item.NilaiAkhir, &item.GradeHuruf,
		); err != nil {
			return nil, 0, fmt.Errorf("scan item pendaftaran: %w", err)
		}
		item.TanggalDaftar = tDaftar.Format(time.RFC3339)
		items = append(items, item)
	}

	return items, total, nil
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *pgRepository) FindMahasiswaIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find mahasiswa id by user %d: %w", userID, err)
	}
	return &id, nil
}

func (r *pgRepository) FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM dosen WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find dosen id by user %d: %w", userID, err)
	}
	return &id, nil
}

func (r *pgRepository) FindPembimbingLapanganIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM pembimbing_lapangan WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find pembimbing lapangan id by user %d: %w", userID, err)
	}
	return &id, nil
}

func (r *pgRepository) IsAnggotaKelompokKKN(ctx context.Context, db database.DBTX, mhsID, kelompokID int) (bool, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT count(*)
		FROM peserta_program
		WHERE id_mahasiswa = $1 AND id_kelompok_kkn = $2
	`, mhsID, kelompokID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("cek anggota kelompok kkn: %w", err)
	}
	return count > 0, nil
}

func (r *pgRepository) FindTempatProgram(ctx context.Context, db database.DBTX, id int) (*TempatProgramCheck, error) {
	const q = `
		SELECT id, jenis_program, is_aktif
		FROM tempat_program
		WHERE id = $1
	`
	var t TempatProgramCheck
	err := db.QueryRow(ctx, q, id).Scan(&t.ID, &t.JenisProgram, &t.IsAktif)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find tempat program %d: %w", id, err)
	}
	return &t, nil
}

func (r *pgRepository) FindPembimbingLapangan(ctx context.Context, db database.DBTX, id int) (*PembimbingLapanganCheck, error) {
	const q = `
		SELECT id, jenis, id_tempat_program
		FROM pembimbing_lapangan
		WHERE id = $1
	`
	var pl PembimbingLapanganCheck
	err := db.QueryRow(ctx, q, id).Scan(&pl.ID, &pl.Jenis, &pl.IDTempatProgram)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find pembimbing lapangan %d: %w", id, err)
	}
	return &pl, nil
}

func (r *pgRepository) FindDosen(ctx context.Context, db database.DBTX, id int) (*DosenCheck, error) {
	const q = `
		SELECT id, nama
		FROM dosen
		WHERE id = $1
	`
	var d DosenCheck
	err := db.QueryRow(ctx, q, id).Scan(&d.ID, &d.Nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find dosen %d: %w", id, err)
	}
	return &d, nil
}

func (r *pgRepository) UpdatePenempatanMagang(ctx context.Context, db database.DBTX, idPeserta int, idTempat int, idPembimbing int) error {
	const q = `
		UPDATE peserta_program
		SET id_tempat_program = $1,
		    id_pembimbing_lapangan = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	res, err := db.Exec(ctx, q, idTempat, idPembimbing, idPeserta)
	if err != nil {
		return fmt.Errorf("update penempatan magang peserta %d: %w", idPeserta, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) UpdateDosenPembimbing(ctx context.Context, db database.DBTX, idPeserta int, idDosen int) error {
	const q = `
		UPDATE peserta_program
		SET id_dosen_pembimbing = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	res, err := db.Exec(ctx, q, idDosen, idPeserta)
	if err != nil {
		return fmt.Errorf("update dosen pembimbing peserta %d: %w", idPeserta, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) UpdatePenempatanAM(ctx context.Context, db database.DBTX, idPeserta int, idTempat int, idPembimbing int) error {
	const q = `
		UPDATE peserta_program
		SET id_tempat_program = $1,
		    id_pembimbing_lapangan = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	res, err := db.Exec(ctx, q, idTempat, idPembimbing, idPeserta)
	if err != nil {
		return fmt.Errorf("update penempatan asistensi mengajar peserta %d: %w", idPeserta, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) UpdateDosenPenguji(ctx context.Context, db database.DBTX, idPeserta int, idDosen int) error {
	const q = `
		UPDATE peserta_program
		SET id_dosen_penguji = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	res, err := db.Exec(ctx, q, idDosen, idPeserta)
	if err != nil {
		return fmt.Errorf("update dosen penguji peserta %d: %w", idPeserta, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
