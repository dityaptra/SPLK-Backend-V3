package tempat

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mendefinisikan akses database untuk modul tempat.
type Repository interface {
	Create(ctx context.Context, db database.DBTX, t *TempatProgram) error
	FindByID(ctx context.Context, db database.DBTX, id int) (*TempatProgram, error)
	List(ctx context.Context, db database.DBTX, filter TempatFilter, p httpx.Pagination) ([]TempatProgram, int64, error)
	Update(ctx context.Context, db database.DBTX, t *TempatProgram) error
	SetAktif(ctx context.Context, db database.DBTX, id int, isAktif bool) error
	ListPembimbingByTempatID(ctx context.Context, db database.DBTX, tempatID int) ([]PembimbingLapangan, error)
}

// PostgresRepository mengimplementasikan Repository dengan PostgreSQL.
type PostgresRepository struct{}

// NewRepository membuat instance PostgresRepository baru.
func NewRepository() Repository {
	return &PostgresRepository{}
}

// Create menyimpan tempat program baru ke database.
func (r *PostgresRepository) Create(ctx context.Context, db database.DBTX, t *TempatProgram) error {
	query := `
		INSERT INTO tempat_program (jenis_program, nama, alamat, kota, kontak, is_aktif)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id, is_aktif
	`
	err := db.QueryRow(ctx, query, t.JenisProgram, t.Nama, t.Alamat, t.Kota, t.Kontak).Scan(&t.ID, &t.IsAktif)
	if err != nil {
		return fmt.Errorf("insert tempat_program: %w", err)
	}
	return nil
}

// FindByID mengambil tempat program berdasarkan ID.
func (r *PostgresRepository) FindByID(ctx context.Context, db database.DBTX, id int) (*TempatProgram, error) {
	query := `
		SELECT id, jenis_program, nama, alamat, kota, kontak, is_aktif
		FROM tempat_program
		WHERE id = $1
	`
	var t TempatProgram
	err := db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.JenisProgram, &t.Nama, &t.Alamat, &t.Kota, &t.Kontak, &t.IsAktif,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil tempat_program %d: %w", id, err)
	}
	return &t, nil
}

// List mengambil daftar tempat program dengan filter dan pagination.
func (r *PostgresRepository) List(ctx context.Context, db database.DBTX, filter TempatFilter, p httpx.Pagination) ([]TempatProgram, int64, error) {
	baseQuery := `FROM tempat_program WHERE 1=1`
	args := []any{}
	argIdx := 1

	if filter.JenisProgram != "" {
		baseQuery += fmt.Sprintf(" AND jenis_program = $%d", argIdx)
		args = append(args, filter.JenisProgram)
		argIdx++
	}

	if filter.IsAktif != nil {
		baseQuery += fmt.Sprintf(" AND is_aktif = $%d", argIdx)
		args = append(args, *filter.IsAktif)
		argIdx++
	}

	// Hitung total
	var total int64
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung tempat_program: %w", err)
	}

	// Ambil data berpaginasi
	dataQuery := fmt.Sprintf(`
		SELECT id, jenis_program, nama, alamat, kota, kontak, is_aktif
		%s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query tempat_program: %w", err)
	}
	defer rows.Close()

	var items []TempatProgram
	for rows.Next() {
		var t TempatProgram
		if err := rows.Scan(&t.ID, &t.JenisProgram, &t.Nama, &t.Alamat, &t.Kota, &t.Kontak, &t.IsAktif); err != nil {
			return nil, 0, fmt.Errorf("scan tempat_program: %w", err)
		}
		items = append(items, t)
	}

	if items == nil {
		items = []TempatProgram{}
	}

	return items, total, nil
}

// Update memperbarui data tempat program.
func (r *PostgresRepository) Update(ctx context.Context, db database.DBTX, t *TempatProgram) error {
	query := `
		UPDATE tempat_program
		SET jenis_program = $1, nama = $2, alamat = $3, kota = $4, kontak = $5, is_aktif = $6
		WHERE id = $7
	`
	cmd, err := db.Exec(ctx, query, t.JenisProgram, t.Nama, t.Alamat, t.Kota, t.Kontak, t.IsAktif, t.ID)
	if err != nil {
		return fmt.Errorf("update tempat_program %d: %w", t.ID, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetAktif mengubah status aktif/nonaktif tempat program (soft delete).
func (r *PostgresRepository) SetAktif(ctx context.Context, db database.DBTX, id int, isAktif bool) error {
	query := `UPDATE tempat_program SET is_aktif = $1 WHERE id = $2`
	cmd, err := db.Exec(ctx, query, isAktif, id)
	if err != nil {
		return fmt.Errorf("set aktif tempat_program %d: %w", id, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListPembimbingByTempatID mengambil daftar pembimbing lapangan di tempat tertentu.
func (r *PostgresRepository) ListPembimbingByTempatID(ctx context.Context, db database.DBTX, tempatID int) ([]PembimbingLapangan, error) {
	query := `
		SELECT id, jenis, id_tempat_program, id_user, nama, jabatan, kontak
		FROM pembimbing_lapangan
		WHERE id_tempat_program = $1
		ORDER BY id ASC
	`
	rows, err := db.Query(ctx, query, tempatID)
	if err != nil {
		return nil, fmt.Errorf("ambil pembimbing lapangan tempat %d: %w", tempatID, err)
	}
	defer rows.Close()

	var items []PembimbingLapangan
	for rows.Next() {
		var pl PembimbingLapangan
		if err := rows.Scan(&pl.ID, &pl.Jenis, &pl.TempatProgramID, &pl.UserID, &pl.Nama, &pl.Jabatan, &pl.Kontak); err != nil {
			return nil, fmt.Errorf("scan pembimbing lapangan: %w", err)
		}
		items = append(items, pl)
	}

	if items == nil {
		items = []PembimbingLapangan{}
	}

	return items, nil
}
