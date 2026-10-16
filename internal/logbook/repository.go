package logbook

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mengabstraksikan operasi basis data untuk tabel logbook_bimbingan.
type Repository interface {
	GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error)
	FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, string, error)
	Create(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error
	GetByID(ctx context.Context, db database.DBTX, id int) (*LogbookBimbingan, string, error)
	Update(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error
	List(ctx context.Context, db database.DBTX, idPeserta int, p httpx.Pagination) ([]LogbookResponse, httpx.PaginationMeta, error)
}

type pgRepository struct{}

// NewRepository membuat instance baru logbook Repository.
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

func (r *pgRepository) FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, string, error) {
	const q = `SELECT id, nama FROM dosen WHERE id_user = $1`
	var id int
	var nama string
	err := db.QueryRow(ctx, q, userID).Scan(&id, &nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("find dosen by user_id %d: %w", userID, err)
	}
	return &id, nama, nil
}

func (r *pgRepository) Create(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error {
	const q = `
		INSERT INTO logbook_bimbingan (
			id_peserta_program, id_dosen, tanggal, topik_bimbingan, catatan,
			pertemuan_berikutnya, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	err := db.QueryRow(ctx, q,
		l.IDPesertaProgram, l.IDDosen, l.Tanggal, l.TopikBimbingan, l.Catatan, l.PertemuanBerikutnya,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert logbook_bimbingan: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, db database.DBTX, id int) (*LogbookBimbingan, string, error) {
	const q = `
		SELECT lb.id, lb.id_peserta_program, lb.id_dosen, d.nama AS nama_dosen,
		       lb.tanggal, lb.topik_bimbingan, lb.catatan, lb.pertemuan_berikutnya,
		       lb.created_at, lb.updated_at
		FROM logbook_bimbingan lb
		JOIN dosen d ON lb.id_dosen = d.id
		WHERE lb.id = $1
	`
	var l LogbookBimbingan
	var namaDosen string
	err := db.QueryRow(ctx, q, id).Scan(
		&l.ID, &l.IDPesertaProgram, &l.IDDosen, &namaDosen,
		&l.Tanggal, &l.TopikBimbingan, &l.Catatan, &l.PertemuanBerikutnya,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("ambil logbook by id %d: %w", id, err)
	}
	return &l, namaDosen, nil
}

func (r *pgRepository) Update(ctx context.Context, db database.DBTX, l *LogbookBimbingan) error {
	const q = `
		UPDATE logbook_bimbingan
		SET tanggal = $1, topik_bimbingan = $2, catatan = $3,
		    pertemuan_berikutnya = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`
	res, err := db.Exec(ctx, q,
		l.Tanggal, l.TopikBimbingan, l.Catatan, l.PertemuanBerikutnya, l.ID,
	)
	if err != nil {
		return fmt.Errorf("update logbook_bimbingan %d: %w", l.ID, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) List(ctx context.Context, db database.DBTX, idPeserta int, p httpx.Pagination) ([]LogbookResponse, httpx.PaginationMeta, error) {
	const countQ = `SELECT COUNT(*) FROM logbook_bimbingan WHERE id_peserta_program = $1`
	var total int
	if err := db.QueryRow(ctx, countQ, idPeserta).Scan(&total); err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("count logbook: %w", err)
	}

	const listQ = `
		SELECT lb.id, lb.id_peserta_program, lb.id_dosen, d.nama AS nama_dosen,
		       lb.tanggal, lb.topik_bimbingan, lb.catatan, lb.pertemuan_berikutnya,
		       lb.created_at, lb.updated_at
		FROM logbook_bimbingan lb
		JOIN dosen d ON lb.id_dosen = d.id
		WHERE lb.id_peserta_program = $1
		ORDER BY lb.tanggal DESC, lb.id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(ctx, listQ, idPeserta, p.Limit, p.Offset())
	if err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("list logbook: %w", err)
	}
	defer rows.Close()

	var list []LogbookResponse
	for rows.Next() {
		var l LogbookBimbingan
		var namaDosen string
		if err := rows.Scan(
			&l.ID, &l.IDPesertaProgram, &l.IDDosen, &namaDosen,
			&l.Tanggal, &l.TopikBimbingan, &l.Catatan, &l.PertemuanBerikutnya,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, httpx.PaginationMeta{}, fmt.Errorf("scan logbook: %w", err)
		}
		list = append(list, ToLogbookResponse(l, namaDosen))
	}
	if err := rows.Err(); err != nil {
		return nil, httpx.PaginationMeta{}, fmt.Errorf("iterate logbook: %w", err)
	}

	meta := httpx.PaginationMeta{
		Page:  p.Page,
		Limit: p.Limit,
		Total: int64(total),
	}
	return list, meta, nil
}
