package spd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mendefinisikan seluruh operasi akses data untuk Surat Perjalanan Dinas.
type Repository interface {
	Create(ctx context.Context, db database.DBTX, s *SuratPerjalananDinas) error
	Update(ctx context.Context, db database.DBTX, s *SuratPerjalananDinas) error
	UpdateFileURL(ctx context.Context, db database.DBTX, id int, fileURL string) error
	Delete(ctx context.Context, db database.DBTX, id int) error
	FindByID(ctx context.Context, db database.DBTX, id int) (*SPDDetail, error)
	List(ctx context.Context, db database.DBTX, filter SPDFilter, p httpx.Pagination) ([]SPDDetail, int64, error)

	FindPegawaiIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)
	FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)
	FindMahasiswaIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)
	CheckDosenExists(ctx context.Context, db database.DBTX, idDosen int) (bool, error)
	CheckPesertaExists(ctx context.Context, db database.DBTX, idPeserta int) (bool, error)
}

type repository struct{}

// NewRepository membuat instance baru Repository untuk SPD.
func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, db database.DBTX, s *SuratPerjalananDinas) error {
	query := `
		INSERT INTO surat_perjalanan_dinas (id_dosen, id_peserta_program, tujuan, tanggal_berangkat, tanggal_kembali, file_url, dibuat_oleh)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	return db.QueryRow(ctx, query, s.IDDosen, s.IDPesertaProgram, s.Tujuan, s.TanggalBerangkat, s.TanggalKembali, s.FileURL, s.DibuatOleh).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *repository) Update(ctx context.Context, db database.DBTX, s *SuratPerjalananDinas) error {
	query := `
		UPDATE surat_perjalanan_dinas
		SET id_dosen = $1, id_peserta_program = $2, tujuan = $3, tanggal_berangkat = $4, tanggal_kembali = $5
		WHERE id = $6
	`
	tag, err := db.Exec(ctx, query, s.IDDosen, s.IDPesertaProgram, s.Tujuan, s.TanggalBerangkat, s.TanggalKembali, s.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) UpdateFileURL(ctx context.Context, db database.DBTX, id int, fileURL string) error {
	query := `UPDATE surat_perjalanan_dinas SET file_url = $1 WHERE id = $2`
	tag, err := db.Exec(ctx, query, fileURL, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, db database.DBTX, id int) error {
	query := `DELETE FROM surat_perjalanan_dinas WHERE id = $1`
	tag, err := db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, db database.DBTX, id int) (*SPDDetail, error) {
	query := `
		SELECT 
			s.id, s.id_dosen, d.nama AS nama_dosen, d.nip AS nip_dosen,
			s.id_peserta_program, m.nama AS nama_mahasiswa, m.nim AS nim_mahasiswa, pp.jenis_program,
			s.tujuan, s.tanggal_berangkat, s.tanggal_kembali, s.file_url,
			s.dibuat_oleh, COALESCE(peg.nama, '') AS nama_pegawai, s.created_at
		FROM surat_perjalanan_dinas s
		JOIN dosen d ON s.id_dosen = d.id
		JOIN peserta_program pp ON s.id_peserta_program = pp.id
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		LEFT JOIN pegawai peg ON s.dibuat_oleh = peg.id
		WHERE s.id = $1
	`
	var d SPDDetail
	err := db.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.IDDosen,
		&d.NamaDosen,
		&d.NIPDosen,
		&d.IDPesertaProgram,
		&d.NamaMahasiswa,
		&d.NIMMahasiswa,
		&d.JenisProgram,
		&d.Tujuan,
		&d.TanggalBerangkat,
		&d.TanggalKembali,
		&d.FileURL,
		&d.DibuatOleh,
		&d.NamaPegawai,
		&d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil detail spd: %w", err)
	}
	return &d, nil
}

func (r *repository) List(ctx context.Context, db database.DBTX, filter SPDFilter, p httpx.Pagination) ([]SPDDetail, int64, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.IDDosen > 0 {
		conditions = append(conditions, fmt.Sprintf("s.id_dosen = $%d", argIdx))
		args = append(args, filter.IDDosen)
		argIdx++
	}

	if filter.IDPesertaProgram > 0 {
		conditions = append(conditions, fmt.Sprintf("s.id_peserta_program = $%d", argIdx))
		args = append(args, filter.IDPesertaProgram)
		argIdx++
	}

	if strings.TrimSpace(filter.Search) != "" {
		conditions = append(conditions, fmt.Sprintf("(s.tujuan ILIKE '%%' || $%d || '%%' OR d.nama ILIKE '%%' || $%d || '%%' OR m.nama ILIKE '%%' || $%d || '%%')", argIdx, argIdx, argIdx))
		args = append(args, strings.TrimSpace(filter.Search))
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM surat_perjalanan_dinas s
		JOIN dosen d ON s.id_dosen = d.id
		JOIN peserta_program pp ON s.id_peserta_program = pp.id
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		%s
	`, whereClause)

	var total int64
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung spd: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			s.id, s.id_dosen, d.nama AS nama_dosen, d.nip AS nip_dosen,
			s.id_peserta_program, m.nama AS nama_mahasiswa, m.nim AS nim_mahasiswa, pp.jenis_program,
			s.tujuan, s.tanggal_berangkat, s.tanggal_kembali, s.file_url,
			s.dibuat_oleh, COALESCE(peg.nama, '') AS nama_pegawai, s.created_at
		FROM surat_perjalanan_dinas s
		JOIN dosen d ON s.id_dosen = d.id
		JOIN peserta_program pp ON s.id_peserta_program = pp.id
		JOIN mahasiswa m ON pp.id_mahasiswa = m.id
		LEFT JOIN pegawai peg ON s.dibuat_oleh = peg.id
		%s
		ORDER BY s.tanggal_berangkat DESC, s.id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := p.Offset()
	args = append(args, limit, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query list spd: %w", err)
	}
	defer rows.Close()

	var list []SPDDetail
	for rows.Next() {
		var item SPDDetail
		if err := rows.Scan(
			&item.ID,
			&item.IDDosen,
			&item.NamaDosen,
			&item.NIPDosen,
			&item.IDPesertaProgram,
			&item.NamaMahasiswa,
			&item.NIMMahasiswa,
			&item.JenisProgram,
			&item.Tujuan,
			&item.TanggalBerangkat,
			&item.TanggalKembali,
			&item.FileURL,
			&item.DibuatOleh,
			&item.NamaPegawai,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan spd: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi spd: %w", err)
	}

	return list, total, nil
}

func (r *repository) FindPegawaiIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM pegawai WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari pegawai by user_id %d: %w", userID, err)
	}
	return &id, nil
}

func (r *repository) FindDosenIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM dosen WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari dosen by user_id %d: %w", userID, err)
	}
	return &id, nil
}

func (r *repository) FindMahasiswaIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM mahasiswa WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari mahasiswa by user_id %d: %w", userID, err)
	}
	return &id, nil
}

func (r *repository) CheckDosenExists(ctx context.Context, db database.DBTX, idDosen int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM dosen WHERE id = $1)", idDosen).Scan(&exists)
	return exists, err
}

func (r *repository) CheckPesertaExists(ctx context.Context, db database.DBTX, idPeserta int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM peserta_program WHERE id = $1)", idPeserta).Scan(&exists)
	return exists, err
}
