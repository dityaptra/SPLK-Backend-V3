package pengumuman

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

// Repository mendefinisikan seluruh operasi data akses untuk pengumuman.
type Repository interface {
	Create(ctx context.Context, db database.DBTX, p *Pengumuman) error
	Update(ctx context.Context, db database.DBTX, p *Pengumuman) error
	Delete(ctx context.Context, db database.DBTX, id int) error
	FindByID(ctx context.Context, db database.DBTX, id int) (*PengumumanDetail, error)
	ListAdmin(ctx context.Context, db database.DBTX, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanDetail, int64, error)
	ListForRoles(ctx context.Context, db database.DBTX, roles []string, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanDetail, int64, error)
}

type repository struct{}

// NewRepository membuat instance baru pengumuman Repository.
func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, db database.DBTX, p *Pengumuman) error {
	if p.TanggalTerbit.IsZero() {
		p.TanggalTerbit = time.Now()
	}
	query := `
		INSERT INTO pengumuman (judul, isi, dibuat_oleh, tanggal_terbit, target_role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tanggal_terbit
	`
	return db.QueryRow(ctx, query, p.Judul, p.Isi, p.DibuatOleh, p.TanggalTerbit, p.TargetRole).Scan(&p.ID, &p.TanggalTerbit)
}

func (r *repository) Update(ctx context.Context, db database.DBTX, p *Pengumuman) error {
	query := `
		UPDATE pengumuman
		SET judul = $1, isi = $2, target_role = $3
		WHERE id = $4
	`
	tag, err := db.Exec(ctx, query, p.Judul, p.Isi, p.TargetRole, p.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, db database.DBTX, id int) error {
	query := `DELETE FROM pengumuman WHERE id = $1`
	tag, err := db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, db database.DBTX, id int) (*PengumumanDetail, error) {
	query := `
		SELECT p.id, p.judul, p.isi, p.dibuat_oleh, u.username, p.tanggal_terbit, p.target_role
		FROM pengumuman p
		LEFT JOIN users u ON p.dibuat_oleh = u.id
		WHERE p.id = $1
	`
	var d PengumumanDetail
	err := db.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.Judul,
		&d.Isi,
		&d.DibuatOleh,
		&d.NamaPembuat,
		&d.TanggalTerbit,
		&d.TargetRole,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil detail pengumuman: %w", err)
	}
	return &d, nil
}

func (r *repository) ListAdmin(ctx context.Context, db database.DBTX, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanDetail, int64, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if strings.TrimSpace(filter.TargetRole) != "" {
		conditions = append(conditions, fmt.Sprintf("p.target_role = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.TargetRole))
		argIdx++
	}

	if strings.TrimSpace(filter.Search) != "" {
		conditions = append(conditions, fmt.Sprintf("(p.judul ILIKE '%%' || $%d || '%%' OR p.isi ILIKE '%%' || $%d || '%%')", argIdx, argIdx))
		args = append(args, strings.TrimSpace(filter.Search))
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM pengumuman p %s`, whereClause)
	var total int64
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung pengumuman admin: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.judul, p.isi, p.dibuat_oleh, u.username, p.tanggal_terbit, p.target_role
		FROM pengumuman p
		LEFT JOIN users u ON p.dibuat_oleh = u.id
		%s
		ORDER BY p.tanggal_terbit DESC, p.id DESC
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
		return nil, 0, fmt.Errorf("query list pengumuman admin: %w", err)
	}
	defer rows.Close()

	var list []PengumumanDetail
	for rows.Next() {
		var item PengumumanDetail
		if err := rows.Scan(
			&item.ID,
			&item.Judul,
			&item.Isi,
			&item.DibuatOleh,
			&item.NamaPembuat,
			&item.TanggalTerbit,
			&item.TargetRole,
		); err != nil {
			return nil, 0, fmt.Errorf("scan pengumuman admin: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi pengumuman admin: %w", err)
	}

	return list, total, nil
}

func (r *repository) ListForRoles(ctx context.Context, db database.DBTX, roles []string, filter PengumumanFilter, p httpx.Pagination) ([]PengumumanDetail, int64, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// Filter target role: NULL / kosong (untuk semua) atau sama dengan salah satu peran user
	if len(roles) > 0 {
		conditions = append(conditions, fmt.Sprintf("(p.target_role IS NULL OR p.target_role = '' OR p.target_role = ANY($%d))", argIdx))
		args = append(args, roles)
		argIdx++
	} else {
		conditions = append(conditions, "(p.target_role IS NULL OR p.target_role = '')")
	}

	if strings.TrimSpace(filter.Search) != "" {
		conditions = append(conditions, fmt.Sprintf("(p.judul ILIKE '%%' || $%d || '%%' OR p.isi ILIKE '%%' || $%d || '%%')", argIdx, argIdx))
		args = append(args, strings.TrimSpace(filter.Search))
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM pengumuman p %s`, whereClause)
	var total int64
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung pengumuman user: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.judul, p.isi, p.dibuat_oleh, u.username, p.tanggal_terbit, p.target_role
		FROM pengumuman p
		LEFT JOIN users u ON p.dibuat_oleh = u.id
		%s
		ORDER BY p.tanggal_terbit DESC, p.id DESC
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
		return nil, 0, fmt.Errorf("query list pengumuman user: %w", err)
	}
	defer rows.Close()

	var list []PengumumanDetail
	for rows.Next() {
		var item PengumumanDetail
		if err := rows.Scan(
			&item.ID,
			&item.Judul,
			&item.Isi,
			&item.DibuatOleh,
			&item.NamaPembuat,
			&item.TanggalTerbit,
			&item.TargetRole,
		); err != nil {
			return nil, 0, fmt.Errorf("scan pengumuman user: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi pengumuman user: %w", err)
	}

	return list, total, nil
}
