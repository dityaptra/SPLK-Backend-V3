package jurnal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mengabstraksikan operasi basis data untuk tabel jurnal_harian.
type Repository interface {
	GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error)
	ExistsByDate(ctx context.Context, db database.DBTX, idPeserta int, tanggal time.Time, excludeID int) (bool, error)
	Create(ctx context.Context, db database.DBTX, j *JurnalHarian) error
	GetByID(ctx context.Context, db database.DBTX, id int) (*JurnalHarian, error)
	Update(ctx context.Context, db database.DBTX, j *JurnalHarian) error
	List(ctx context.Context, db database.DBTX, idPeserta int, filter JurnalFilter, p httpx.Pagination) ([]JurnalHarian, httpx.PaginationMeta, error)
}

type pgRepository struct{}

// NewRepository membuat instance baru jurnal Repository.
func NewRepository() Repository {
	return &pgRepository{}
}

func (r *pgRepository) GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
	const q = `
		SELECT pp.id, pp.status, p.id, p.tanggal_mulai, p.tanggal_selesai
		FROM peserta_program pp
		JOIN periode p ON pp.id_periode = p.id
		WHERE pp.id = $1
	`
	var info PendaftaranInfo
	err := db.QueryRow(ctx, q, idPeserta).Scan(
		&info.IDPesertaProgram, &info.StatusPeserta, &info.IDPeriode,
		&info.TanggalMulai, &info.TanggalSelesai,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil info pendaftaran %d: %w", idPeserta, err)
	}
	return &info, nil
}

func (r *pgRepository) ExistsByDate(ctx context.Context, db database.DBTX, idPeserta int, tanggal time.Time, excludeID int) (bool, error) {
	const q = `
		SELECT COUNT(*)
		FROM jurnal_harian
		WHERE id_peserta_program = $1 AND tanggal = $2 AND id <> $3
	`
	var count int
	if err := db.QueryRow(ctx, q, idPeserta, tanggal, excludeID).Scan(&count); err != nil {
		return false, fmt.Errorf("cek duplikat tanggal jurnal: %w", err)
	}
	return count > 0, nil
}

func (r *pgRepository) Create(ctx context.Context, db database.DBTX, j *JurnalHarian) error {
	const q = `
		INSERT INTO jurnal_harian (
			id_peserta_program, tanggal, judul, isi, kendala, solusi, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	err := db.QueryRow(ctx, q,
		j.IDPesertaProgram, j.Tanggal, j.Judul, j.Isi, j.Kendala, j.Solusi, j.Status,
	).Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert jurnal_harian: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, db database.DBTX, id int) (*JurnalHarian, error) {
	const q = `
		SELECT id, id_peserta_program, tanggal, judul, isi, kendala, solusi, status,
		       catatan_reviewer, reviewed_by, reviewed_at, created_at, updated_at
		FROM jurnal_harian
		WHERE id = $1
	`
	var j JurnalHarian
	err := db.QueryRow(ctx, q, id).Scan(
		&j.ID, &j.IDPesertaProgram, &j.Tanggal, &j.Judul, &j.Isi,
		&j.Kendala, &j.Solusi, &j.Status, &j.CatatanReviewer,
		&j.ReviewedBy, &j.ReviewedAt, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil jurnal by id %d: %w", id, err)
	}
	return &j, nil
}

func (r *pgRepository) Update(ctx context.Context, db database.DBTX, j *JurnalHarian) error {
	const q = `
		UPDATE jurnal_harian
		SET tanggal = $1, judul = $2, isi = $3, kendala = $4, solusi = $5,
		    status = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`
	res, err := db.Exec(ctx, q,
		j.Tanggal, j.Judul, j.Isi, j.Kendala, j.Solusi, j.Status, j.ID,
	)
	if err != nil {
		return fmt.Errorf("update jurnal_harian %d: %w", j.ID, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) List(ctx context.Context, db database.DBTX, idPeserta int, filter JurnalFilter, p httpx.Pagination) ([]JurnalHarian, httpx.PaginationMeta, error) {
	countQ := `SELECT COUNT(*) FROM jurnal_harian WHERE id_peserta_program = $1`
	listQ := `
		SELECT id, id_peserta_program, tanggal, judul, isi, kendala, solusi, status,
		       catatan_reviewer, reviewed_by, reviewed_at, created_at, updated_at
		FROM jurnal_harian
		WHERE id_peserta_program = $1
	`
	args := []any{idPeserta}
	argIdx := 2

	if filter.TanggalMulai != "" {
		countQ += fmt.Sprintf(" AND tanggal >= $%d", argIdx)
		listQ += fmt.Sprintf(" AND tanggal >= $%d", argIdx)
		args = append(args, filter.TanggalMulai)
		argIdx++
	}
	if filter.TanggalSelesai != "" {
		countQ += fmt.Sprintf(" AND tanggal <= $%d", argIdx)
		listQ += fmt.Sprintf(" AND tanggal <= $%d", argIdx)
		args = append(args, filter.TanggalSelesai)
		argIdx++
	}
	if filter.Status != "" {
		countQ += fmt.Sprintf(" AND status = $%d", argIdx)
		listQ += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	var total int
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("count jurnal: %w", err)
	}

	listQ += fmt.Sprintf(" ORDER BY tanggal DESC, id DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, p.Limit, p.Offset())

	rows, err := db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("list jurnal: %w", err)
	}
	defer rows.Close()

	var list []JurnalHarian
	for rows.Next() {
		var j JurnalHarian
		if err := rows.Scan(
			&j.ID, &j.IDPesertaProgram, &j.Tanggal, &j.Judul, &j.Isi,
			&j.Kendala, &j.Solusi, &j.Status, &j.CatatanReviewer,
			&j.ReviewedBy, &j.ReviewedAt, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, httpx.PaginationMeta{}, fmt.Errorf("scan jurnal: %w", err)
		}
		list = append(list, j)
	}
	if err := rows.Err(); err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("iterate jurnal: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: int64(total),
	}
	return list, meta, nil
}
