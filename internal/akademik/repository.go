package akademik

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
)

// Repository mendefinisikan antarmuka database untuk modul akademik.
type Repository interface {
	// Fakultas
	CreateFakultas(ctx context.Context, db database.DBTX, f *Fakultas) error
	FindFakultasByID(ctx context.Context, db database.DBTX, id int) (*Fakultas, error)
	ListFakultas(ctx context.Context, db database.DBTX) ([]Fakultas, error)
	UpdateFakultas(ctx context.Context, db database.DBTX, f *Fakultas) error
	DeleteFakultas(ctx context.Context, db database.DBTX, id int) error
	CountJurusanByFakultasID(ctx context.Context, db database.DBTX, fakultasID int) (int64, error)

	// Jurusan
	CreateJurusan(ctx context.Context, db database.DBTX, j *Jurusan) error
	FindJurusanByID(ctx context.Context, db database.DBTX, id int) (*Jurusan, error)
	ListJurusan(ctx context.Context, db database.DBTX, fakultasID *int) ([]Jurusan, error)
	UpdateJurusan(ctx context.Context, db database.DBTX, j *Jurusan) error
	DeleteJurusan(ctx context.Context, db database.DBTX, id int) error
	CountProdiByJurusanID(ctx context.Context, db database.DBTX, jurusanID int) (int64, error)

	// Program Studi
	CreateProdi(ctx context.Context, db database.DBTX, p *ProgramStudi) error
	FindProdiByID(ctx context.Context, db database.DBTX, id int) (*ProgramStudi, error)
	ListProdi(ctx context.Context, db database.DBTX, jurusanID *int) ([]ProgramStudi, error)
	UpdateProdi(ctx context.Context, db database.DBTX, p *ProgramStudi) error
	DeleteProdi(ctx context.Context, db database.DBTX, id int) error
	CountUsersByProdiID(ctx context.Context, db database.DBTX, prodiID int) (int64, error)
}

// PostgresRepository mengimplementasikan Repository menggunakan driver PostgreSQL.
type PostgresRepository struct{}

// NewRepository membuat instance PostgresRepository baru.
func NewRepository() Repository {
	return &PostgresRepository{}
}

// --- Fakultas ---

func (r *PostgresRepository) CreateFakultas(ctx context.Context, db database.DBTX, f *Fakultas) error {
	query := `INSERT INTO fakultas (nama, kode) VALUES ($1, $2) RETURNING id`
	return db.QueryRow(ctx, query, f.Nama, f.Kode).Scan(&f.ID)
}

func (r *PostgresRepository) FindFakultasByID(ctx context.Context, db database.DBTX, id int) (*Fakultas, error) {
	query := `SELECT id, nama, kode FROM fakultas WHERE id = $1`
	var f Fakultas
	err := db.QueryRow(ctx, query, id).Scan(&f.ID, &f.Nama, &f.Kode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil fakultas %d: %w", id, err)
	}
	return &f, nil
}

func (r *PostgresRepository) ListFakultas(ctx context.Context, db database.DBTX) ([]Fakultas, error) {
	query := `SELECT id, nama, kode FROM fakultas ORDER BY id ASC`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list fakultas: %w", err)
	}
	defer rows.Close()

	var items []Fakultas
	for rows.Next() {
		var f Fakultas
		if err := rows.Scan(&f.ID, &f.Nama, &f.Kode); err != nil {
			return nil, fmt.Errorf("scan fakultas: %w", err)
		}
		items = append(items, f)
	}
	if items == nil {
		items = []Fakultas{}
	}
	return items, nil
}

func (r *PostgresRepository) UpdateFakultas(ctx context.Context, db database.DBTX, f *Fakultas) error {
	query := `UPDATE fakultas SET nama = $1, kode = $2 WHERE id = $3`
	cmd, err := db.Exec(ctx, query, f.Nama, f.Kode, f.ID)
	if err != nil {
		return fmt.Errorf("update fakultas %d: %w", f.ID, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) DeleteFakultas(ctx context.Context, db database.DBTX, id int) error {
	query := `DELETE FROM fakultas WHERE id = $1`
	cmd, err := db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete fakultas %d: %w", id, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) CountJurusanByFakultasID(ctx context.Context, db database.DBTX, fakultasID int) (int64, error) {
	var count int64
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM jurusan WHERE id_fakultas = $1`, fakultasID).Scan(&count)
	return count, err
}

// --- Jurusan ---

func (r *PostgresRepository) CreateJurusan(ctx context.Context, db database.DBTX, j *Jurusan) error {
	query := `INSERT INTO jurusan (id_fakultas, nama, kode_jurusan) VALUES ($1, $2, $3) RETURNING id`
	return db.QueryRow(ctx, query, j.FakultasID, j.Nama, j.KodeJurusan).Scan(&j.ID)
}

func (r *PostgresRepository) FindJurusanByID(ctx context.Context, db database.DBTX, id int) (*Jurusan, error) {
	query := `SELECT id, id_fakultas, nama, kode_jurusan FROM jurusan WHERE id = $1`
	var j Jurusan
	err := db.QueryRow(ctx, query, id).Scan(&j.ID, &j.FakultasID, &j.Nama, &j.KodeJurusan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil jurusan %d: %w", id, err)
	}
	return &j, nil
}

func (r *PostgresRepository) ListJurusan(ctx context.Context, db database.DBTX, fakultasID *int) ([]Jurusan, error) {
	query := `SELECT id, id_fakultas, nama, kode_jurusan FROM jurusan`
	args := []any{}
	if fakultasID != nil {
		query += ` WHERE id_fakultas = $1`
		args = append(args, *fakultasID)
	}
	query += ` ORDER BY id ASC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jurusan: %w", err)
	}
	defer rows.Close()

	var items []Jurusan
	for rows.Next() {
		var j Jurusan
		if err := rows.Scan(&j.ID, &j.FakultasID, &j.Nama, &j.KodeJurusan); err != nil {
			return nil, fmt.Errorf("scan jurusan: %w", err)
		}
		items = append(items, j)
	}
	if items == nil {
		items = []Jurusan{}
	}
	return items, nil
}

func (r *PostgresRepository) UpdateJurusan(ctx context.Context, db database.DBTX, j *Jurusan) error {
	query := `UPDATE jurusan SET id_fakultas = $1, nama = $2, kode_jurusan = $3 WHERE id = $4`
	cmd, err := db.Exec(ctx, query, j.FakultasID, j.Nama, j.KodeJurusan, j.ID)
	if err != nil {
		return fmt.Errorf("update jurusan %d: %w", j.ID, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) DeleteJurusan(ctx context.Context, db database.DBTX, id int) error {
	query := `DELETE FROM jurusan WHERE id = $1`
	cmd, err := db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete jurusan %d: %w", id, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) CountProdiByJurusanID(ctx context.Context, db database.DBTX, jurusanID int) (int64, error) {
	var count int64
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM program_studi WHERE id_jurusan = $1`, jurusanID).Scan(&count)
	return count, err
}

// --- Program Studi ---

func (r *PostgresRepository) CreateProdi(ctx context.Context, db database.DBTX, p *ProgramStudi) error {
	query := `INSERT INTO program_studi (id_jurusan, nama, kode) VALUES ($1, $2, $3) RETURNING id`
	return db.QueryRow(ctx, query, p.JurusanID, p.Nama, p.Kode).Scan(&p.ID)
}

func (r *PostgresRepository) FindProdiByID(ctx context.Context, db database.DBTX, id int) (*ProgramStudi, error) {
	query := `SELECT id, id_jurusan, nama, kode FROM program_studi WHERE id = $1`
	var p ProgramStudi
	err := db.QueryRow(ctx, query, id).Scan(&p.ID, &p.JurusanID, &p.Nama, &p.Kode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil program_studi %d: %w", id, err)
	}
	return &p, nil
}

func (r *PostgresRepository) ListProdi(ctx context.Context, db database.DBTX, jurusanID *int) ([]ProgramStudi, error) {
	query := `SELECT id, id_jurusan, nama, kode FROM program_studi`
	args := []any{}
	if jurusanID != nil {
		query += ` WHERE id_jurusan = $1`
		args = append(args, *jurusanID)
	}
	query += ` ORDER BY id ASC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list prodi: %w", err)
	}
	defer rows.Close()

	var items []ProgramStudi
	for rows.Next() {
		var p ProgramStudi
		if err := rows.Scan(&p.ID, &p.JurusanID, &p.Nama, &p.Kode); err != nil {
			return nil, fmt.Errorf("scan prodi: %w", err)
		}
		items = append(items, p)
	}
	if items == nil {
		items = []ProgramStudi{}
	}
	return items, nil
}

func (r *PostgresRepository) UpdateProdi(ctx context.Context, db database.DBTX, p *ProgramStudi) error {
	query := `UPDATE program_studi SET id_jurusan = $1, nama = $2, kode = $3 WHERE id = $4`
	cmd, err := db.Exec(ctx, query, p.JurusanID, p.Nama, p.Kode, p.ID)
	if err != nil {
		return fmt.Errorf("update program_studi %d: %w", p.ID, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) DeleteProdi(ctx context.Context, db database.DBTX, id int) error {
	query := `DELETE FROM program_studi WHERE id = $1`
	cmd, err := db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete program_studi %d: %w", id, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) CountUsersByProdiID(ctx context.Context, db database.DBTX, prodiID int) (int64, error) {
	var countMahasiswa, countDosen int64
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM mahasiswa WHERE id_prodi = $1`, prodiID).Scan(&countMahasiswa)
	if err != nil {
		return 0, err
	}
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM dosen WHERE id_prodi = $1`, prodiID).Scan(&countDosen)
	if err != nil {
		return 0, err
	}
	return countMahasiswa + countDosen, nil
}
