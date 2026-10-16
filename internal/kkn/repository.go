package kkn

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mengabstraksikan operasi basis data untuk kelompok KKN dan anggotanya.
type Repository interface {
	CreateKelompok(ctx context.Context, db database.DBTX, k *KelompokKKN) error
	FindKelompokByID(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error)
	FindKelompokDetailByID(ctx context.Context, db database.DBTX, id int) (*KelompokDetailResponse, error)
	ListKelompok(ctx context.Context, db database.DBTX, idPeriode *int, p httpx.Pagination) ([]KelompokSummaryResponse, httpx.PaginationMeta, error)

	FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (jenisProgram string, status string, err error)
	FindTempatByID(ctx context.Context, db database.DBTX, id int) (jenisProgram string, isAktif bool, err error)
	FindDosenByID(ctx context.Context, db database.DBTX, id int) (bool, error)
	FindPembimbingLapanganByID(ctx context.Context, db database.DBTX, id int) (jenis string, idTempat int, err error)

	FindPesertaForKKN(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error)
	FindAnggotaKelompok(ctx context.Context, db database.DBTX, idKelompok int) ([]AnggotaKelompok, error)
	GetKetuaKelompok(ctx context.Context, db database.DBTX, idKelompok int) (*int, error)

	AssignAnggota(ctx context.Context, db database.DBTX, idPeserta int, idKelompok int, idTempat int, idDosen *int, idPL *int, isKetua bool) error
	RemoveAnggota(ctx context.Context, db database.DBTX, idPeserta int) error

	UpdateDosenPembimbingKelompok(ctx context.Context, db database.DBTX, idKelompok int, idDosen int) error
	UpdatePembimbingLapanganKelompok(ctx context.Context, db database.DBTX, idKelompok int, idPL int) error

	CheckAksesKelompok(ctx context.Context, db database.DBTX, idKelompok int, userID int, roles []string) (bool, error)
}

type pgRepository struct{}

// NewRepository membuat instance baru Repository untuk modul KKN.
func NewRepository() Repository {
	return &pgRepository{}
}

func (r *pgRepository) CreateKelompok(ctx context.Context, db database.DBTX, k *KelompokKKN) error {
	const q = `
		INSERT INTO kelompok_kkn (id_periode, id_tempat_program, nama_kelompok, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	err := db.QueryRow(ctx, q, k.IDPeriode, k.IDTempatProgram, k.NamaKelompok).Scan(&k.ID, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert kelompok_kkn: %w", err)
	}
	return nil
}

func (r *pgRepository) FindKelompokByID(ctx context.Context, db database.DBTX, id int) (*KelompokKKN, error) {
	const q = `
		SELECT id, id_periode, id_tempat_program, id_dosen_pembimbing,
		       id_pembimbing_lapangan, nama_kelompok, created_at, updated_at
		FROM kelompok_kkn
		WHERE id = $1
	`
	var k KelompokKKN
	err := db.QueryRow(ctx, q, id).Scan(
		&k.ID, &k.IDPeriode, &k.IDTempatProgram, &k.IDDosenPembimbing,
		&k.IDPembimbingLapangan, &k.NamaKelompok, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find kelompok by id %d: %w", id, err)
	}
	return &k, nil
}

func (r *pgRepository) FindKelompokDetailByID(ctx context.Context, db database.DBTX, id int) (*KelompokDetailResponse, error) {
	const q = `
		SELECT k.id, k.id_periode, p.nama AS nama_periode,
		       k.id_tempat_program, tp.nama AS nama_tempat, tp.alamat AS alamat_tempat,
		       k.id_dosen_pembimbing, d.nama AS nama_dosen_pembimbing,
		       k.id_pembimbing_lapangan, pl.nama AS nama_pembimbing_lapangan,
		       k.nama_kelompok, k.created_at, k.updated_at
		FROM kelompok_kkn k
		JOIN periode p ON k.id_periode = p.id
		JOIN tempat_program tp ON k.id_tempat_program = tp.id
		LEFT JOIN dosen d ON k.id_dosen_pembimbing = d.id
		LEFT JOIN pembimbing_lapangan pl ON k.id_pembimbing_lapangan = pl.id
		WHERE k.id = $1
	`
	var resp KelompokDetailResponse
	err := db.QueryRow(ctx, q, id).Scan(
		&resp.ID, &resp.IDPeriode, &resp.NamaPeriode,
		&resp.IDTempatProgram, &resp.NamaTempat, &resp.AlamatTempat,
		&resp.IDDosenPembimbing, &resp.NamaDosenPembimbing,
		&resp.IDPembimbingLapangan, &resp.NamaPembimbingLapangan,
		&resp.NamaKelompok, &resp.CreatedAt, &resp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find kelompok detail by id %d: %w", id, err)
	}

	// Ambil anggota
	anggotaList, err := r.FindAnggotaKelompok(ctx, db, id)
	if err != nil {
		return nil, err
	}

	resp.Anggota = make([]AnggotaKelompokResponse, 0, len(anggotaList))
	for _, a := range anggotaList {
		item := AnggotaKelompokResponse{
			IDPesertaProgram: a.IDPesertaProgram,
			IDMahasiswa:      a.IDMahasiswa,
			NIM:              a.NIM,
			Nama:             a.NamaMahasiswa,
			Prodi:            a.NamaProdi,
			IsKetua:          a.IsKetua,
			Status:           a.Status,
			TanggalDaftar:    a.TanggalDaftar,
		}
		resp.Anggota = append(resp.Anggota, item)
		if a.IsKetua {
			ketuaCopy := item
			resp.Ketua = &ketuaCopy
		}
	}

	return &resp, nil
}

func (r *pgRepository) ListKelompok(ctx context.Context, db database.DBTX, idPeriode *int, p httpx.Pagination) ([]KelompokSummaryResponse, httpx.PaginationMeta, error) {
	const countQ = `
		SELECT COUNT(*)
		FROM kelompok_kkn
		WHERE ($1::int IS NULL OR id_periode = $1)
	`
	var total int
	if err := db.QueryRow(ctx, countQ, idPeriode).Scan(&total); err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("count kelompok: %w", err)
	}

	const listQ = `
		SELECT k.id, k.id_periode, p.nama AS nama_periode,
		       k.id_tempat_program, tp.nama AS nama_tempat, tp.alamat AS alamat_tempat,
		       k.id_dosen_pembimbing, d.nama AS nama_dosen_pembimbing,
		       k.id_pembimbing_lapangan, pl.nama AS nama_pembimbing_lapangan,
		       k.nama_kelompok,
		       (SELECT COUNT(*) FROM peserta_program pp WHERE pp.id_kelompok_kkn = k.id) AS jumlah_anggota,
		       k.created_at, k.updated_at
		FROM kelompok_kkn k
		JOIN periode p ON k.id_periode = p.id
		JOIN tempat_program tp ON k.id_tempat_program = tp.id
		LEFT JOIN dosen d ON k.id_dosen_pembimbing = d.id
		LEFT JOIN pembimbing_lapangan pl ON k.id_pembimbing_lapangan = pl.id
		WHERE ($1::int IS NULL OR k.id_periode = $1)
		ORDER BY k.id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(ctx, listQ, idPeriode, p.Limit, p.Offset())
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("list kelompok: %w", err)
	}
	defer rows.Close()

	var list []KelompokSummaryResponse
	for rows.Next() {
		var item KelompokSummaryResponse
		if err := rows.Scan(
			&item.ID, &item.IDPeriode, &item.NamaPeriode,
			&item.IDTempatProgram, &item.NamaTempat, &item.AlamatTempat,
			&item.IDDosenPembimbing, &item.NamaDosenPembimbing,
			&item.IDPembimbingLapangan, &item.NamaPembimbingLapangan,
			&item.NamaKelompok, &item.JumlahAnggota,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, httpx.PaginationMeta{}, fmt.Errorf("scan kelompok summary: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("iterate kelompok summary: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: int64(total),
	}
	return list, meta, nil
}

func (r *pgRepository) FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (string, string, error) {
	const q = `SELECT jenis_program, status FROM periode WHERE id = $1`
	var jenis, status string
	err := db.QueryRow(ctx, q, id).Scan(&jenis, &status)
	if err != nil {
		return "", "", err
	}
	return jenis, status, nil
}

func (r *pgRepository) FindTempatByID(ctx context.Context, db database.DBTX, id int) (string, bool, error) {
	const q = `SELECT jenis_program, is_aktif FROM tempat_program WHERE id = $1`
	var jenis string
	var aktif bool
	err := db.QueryRow(ctx, q, id).Scan(&jenis, &aktif)
	if err != nil {
		return "", false, err
	}
	return jenis, aktif, nil
}

func (r *pgRepository) FindDosenByID(ctx context.Context, db database.DBTX, id int) (bool, error) {
	const q = `SELECT COUNT(*) FROM dosen WHERE id = $1`
	var count int
	if err := db.QueryRow(ctx, q, id).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *pgRepository) FindPembimbingLapanganByID(ctx context.Context, db database.DBTX, id int) (string, int, error) {
	const q = `SELECT jenis, id_tempat_program FROM pembimbing_lapangan WHERE id = $1`
	var jenis string
	var idTempat int
	err := db.QueryRow(ctx, q, id).Scan(&jenis, &idTempat)
	if err != nil {
		return "", 0, err
	}
	return jenis, idTempat, nil
}

func (r *pgRepository) FindPesertaForKKN(ctx context.Context, db database.DBTX, idPeserta int) (*PesertaKKNInfo, error) {
	const q = `
		SELECT id, id_mahasiswa, id_periode, jenis_program, status,
		       id_kelompok_kkn, id_dosen_pembimbing, id_dosen_penguji,
		       id_pembimbing_lapangan, is_ketua
		FROM peserta_program
		WHERE id = $1
	`
	var p PesertaKKNInfo
	err := db.QueryRow(ctx, q, idPeserta).Scan(
		&p.ID, &p.IDMahasiswa, &p.IDPeriode, &p.JenisProgram, &p.Status,
		&p.IDKelompokKKN, &p.IDDosenPembimbing, &p.IDDosenPenguji,
		&p.IDPembimbingLapangan, &p.IsKetua,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find peserta for kkn %d: %w", idPeserta, err)
	}
	return &p, nil
}

func (r *pgRepository) FindAnggotaKelompok(ctx context.Context, db database.DBTX, idKelompok int) ([]AnggotaKelompok, error) {
	const q = `
		SELECT pp.id, pp.id_mahasiswa, m.nim, m.nama, COALESCE(ps.nama, ''),
		       pp.is_ketua, pp.status, pp.tanggal_daftar,
		       pp.id_dosen_pembimbing, pp.id_pembimbing_lapangan, pp.id_dosen_penguji
		FROM peserta_program pp
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		LEFT JOIN program_studi ps ON m.id_prodi = ps.id
		WHERE pp.id_kelompok_kkn = $1
		ORDER BY pp.id ASC
	`
	rows, err := db.Query(ctx, q, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("query anggota kelompok: %w", err)
	}
	defer rows.Close()

	var list []AnggotaKelompok
	for rows.Next() {
		var a AnggotaKelompok
		if err := rows.Scan(
			&a.IDPesertaProgram, &a.IDMahasiswa, &a.NIM, &a.NamaMahasiswa, &a.NamaProdi,
			&a.IsKetua, &a.Status, &a.TanggalDaftar,
			&a.IDDosenPembimbing, &a.IDPembimbingLapangan, &a.IDDosenPenguji,
		); err != nil {
			return nil, fmt.Errorf("scan anggota kelompok: %w", err)
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate anggota kelompok: %w", err)
	}
	return list, nil
}

func (r *pgRepository) GetKetuaKelompok(ctx context.Context, db database.DBTX, idKelompok int) (*int, error) {
	const q = `
		SELECT id
		FROM peserta_program
		WHERE id_kelompok_kkn = $1 AND is_ketua = true
		LIMIT 1
	`
	var id int
	err := db.QueryRow(ctx, q, idKelompok).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ketua kelompok %d: %w", idKelompok, err)
	}
	return &id, nil
}

func (r *pgRepository) AssignAnggota(ctx context.Context, db database.DBTX, idPeserta int, idKelompok int, idTempat int, idDosen *int, idPL *int, isKetua bool) error {
	const q = `
		UPDATE peserta_program
		SET id_kelompok_kkn = $1,
		    id_tempat_program = $2,
		    id_dosen_pembimbing = COALESCE($3, id_dosen_pembimbing),
		    id_pembimbing_lapangan = COALESCE($4, id_pembimbing_lapangan),
		    is_ketua = $5,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`
	res, err := db.Exec(ctx, q, idKelompok, idTempat, idDosen, idPL, isKetua, idPeserta)
	if err != nil {
		return fmt.Errorf("assign anggota %d: %w", idPeserta, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) RemoveAnggota(ctx context.Context, db database.DBTX, idPeserta int) error {
	const q = `
		UPDATE peserta_program
		SET id_kelompok_kkn = NULL,
		    is_ketua = false,
		    id_tempat_program = NULL,
		    id_dosen_pembimbing = NULL,
		    id_pembimbing_lapangan = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	res, err := db.Exec(ctx, q, idPeserta)
	if err != nil {
		return fmt.Errorf("remove anggota %d: %w", idPeserta, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) UpdateDosenPembimbingKelompok(ctx context.Context, db database.DBTX, idKelompok int, idDosen int) error {
	const qKelompok = `
		UPDATE kelompok_kkn
		SET id_dosen_pembimbing = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	res, err := db.Exec(ctx, qKelompok, idDosen, idKelompok)
	if err != nil {
		return fmt.Errorf("update kelompok dosen pembimbing: %w", err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	const qPeserta = `
		UPDATE peserta_program
		SET id_dosen_pembimbing = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id_kelompok_kkn = $2
	`
	if _, err := db.Exec(ctx, qPeserta, idDosen, idKelompok); err != nil {
		return fmt.Errorf("update peserta dosen pembimbing: %w", err)
	}

	return nil
}

func (r *pgRepository) UpdatePembimbingLapanganKelompok(ctx context.Context, db database.DBTX, idKelompok int, idPL int) error {
	const qKelompok = `
		UPDATE kelompok_kkn
		SET id_pembimbing_lapangan = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	res, err := db.Exec(ctx, qKelompok, idPL, idKelompok)
	if err != nil {
		return fmt.Errorf("update kelompok pembimbing lapangan: %w", err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	const qPeserta = `
		UPDATE peserta_program
		SET id_pembimbing_lapangan = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id_kelompok_kkn = $2
	`
	if _, err := db.Exec(ctx, qPeserta, idPL, idKelompok); err != nil {
		return fmt.Errorf("update peserta pembimbing lapangan: %w", err)
	}

	return nil
}

func (r *pgRepository) CheckAksesKelompok(ctx context.Context, db database.DBTX, idKelompok int, userID int, roles []string) (bool, error) {
	for _, role := range roles {
		if role == "admin" || role == "pegawai" {
			return true, nil
		}
	}

	// 1. Mahasiswa anggota kelompok
	var mhsCount int
	err := db.QueryRow(ctx, `
		SELECT count(*)
		FROM peserta_program pp
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		WHERE pp.id_kelompok_kkn = $1 AND m.id_user = $2
	`, idKelompok, userID).Scan(&mhsCount)
	if err != nil {
		return false, fmt.Errorf("cek mahasiswa kelompok: %w", err)
	}
	if mhsCount > 0 {
		return true, nil
	}

	// 2. Dosen pembimbing kelompok
	var dosenCount int
	err = db.QueryRow(ctx, `
		SELECT count(*)
		FROM kelompok_kkn k
		JOIN dosen d ON k.id_dosen_pembimbing = d.id
		WHERE k.id = $1 AND d.id_user = $2
	`, idKelompok, userID).Scan(&dosenCount)
	if err != nil {
		return false, fmt.Errorf("cek dosen kelompok: %w", err)
	}
	if dosenCount > 0 {
		return true, nil
	}

	// 3. Pembimbing lapangan kelompok
	var plCount int
	err = db.QueryRow(ctx, `
		SELECT count(*)
		FROM kelompok_kkn k
		JOIN pembimbing_lapangan pl ON k.id_pembimbing_lapangan = pl.id
		WHERE k.id = $1 AND pl.id_user = $2
	`, idKelompok, userID).Scan(&plCount)
	if err != nil {
		return false, fmt.Errorf("cek pembimbing lapangan kelompok: %w", err)
	}
	if plCount > 0 {
		return true, nil
	}

	return false, nil
}
