package dokumen

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"splk-backend/internal/database"
)

// Repository mengabstraksikan operasi basis data untuk tabel dokumen dan relasi kelompok KKN.
type Repository interface {
	Create(ctx context.Context, db database.DBTX, doc *Dokumen) error
	GetByID(ctx context.Context, db database.DBTX, id int) (*Dokumen, error)
	ListByPesertaID(ctx context.Context, db database.DBTX, idPeserta int) ([]Dokumen, error)
	ListByKelompokID(ctx context.Context, db database.DBTX, idKelompok int) ([]Dokumen, error)
	Delete(ctx context.Context, db database.DBTX, id int) error
	GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error)
	CheckKelompokAccess(ctx context.Context, db database.DBTX, idKelompok int, userID int, userRoles []string, isWrite bool) (bool, error)
	GetKelompokMemberIDs(ctx context.Context, db database.DBTX, idKelompok int) ([]int, error)
}

type pgRepository struct{}

// NewRepository membuat instance baru dokumen Repository.
func NewRepository() Repository {
	return &pgRepository{}
}

func (r *pgRepository) Create(ctx context.Context, db database.DBTX, doc *Dokumen) error {
	const q = `
		INSERT INTO dokumen (
			id_peserta_program, jenis_dokumen, nama_file, file_url,
			ukuran_file, mime_type, diunggah_oleh, id_kelompok_kkn
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, uploaded_at, is_verified, verified_by, verified_at
	`
	err := db.QueryRow(ctx, q,
		doc.IDPesertaProgram, doc.JenisDokumen, doc.NamaFile, doc.FileURL,
		doc.UkuranFile, doc.MimeType, doc.DiunggahOleh, doc.IDKelompokKKN,
	).Scan(&doc.ID, &doc.UploadedAt, &doc.IsVerified, &doc.VerifiedBy, &doc.VerifiedAt)
	if err != nil {
		return fmt.Errorf("insert dokumen: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(ctx context.Context, db database.DBTX, id int) (*Dokumen, error) {
	const q = `
		SELECT id, id_peserta_program, jenis_dokumen, nama_file, file_url,
		       ukuran_file, mime_type, diunggah_oleh, uploaded_at, id_kelompok_kkn,
		       is_verified, verified_by, verified_at
		FROM dokumen
		WHERE id = $1
	`
	var d Dokumen
	err := db.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.IDPesertaProgram, &d.JenisDokumen, &d.NamaFile, &d.FileURL,
		&d.UkuranFile, &d.MimeType, &d.DiunggahOleh, &d.UploadedAt, &d.IDKelompokKKN,
		&d.IsVerified, &d.VerifiedBy, &d.VerifiedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query dokumen %d: %w", id, err)
	}
	return &d, nil
}

func (r *pgRepository) ListByPesertaID(ctx context.Context, db database.DBTX, idPeserta int) ([]Dokumen, error) {
	const q = `
		SELECT id, id_peserta_program, jenis_dokumen, nama_file, file_url,
		       ukuran_file, mime_type, diunggah_oleh, uploaded_at, id_kelompok_kkn,
		       is_verified, verified_by, verified_at
		FROM dokumen
		WHERE id_peserta_program = $1
		ORDER BY uploaded_at ASC, id ASC
	`
	rows, err := db.Query(ctx, q, idPeserta)
	if err != nil {
		return nil, fmt.Errorf("list dokumen peserta %d: %w", idPeserta, err)
	}
	defer rows.Close()

	var list []Dokumen
	for rows.Next() {
		var d Dokumen
		err := rows.Scan(
			&d.ID, &d.IDPesertaProgram, &d.JenisDokumen, &d.NamaFile, &d.FileURL,
			&d.UkuranFile, &d.MimeType, &d.DiunggahOleh, &d.UploadedAt, &d.IDKelompokKKN,
			&d.IsVerified, &d.VerifiedBy, &d.VerifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dokumen: %w", err)
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *pgRepository) ListByKelompokID(ctx context.Context, db database.DBTX, idKelompok int) ([]Dokumen, error) {
	const q = `
		SELECT id, id_peserta_program, jenis_dokumen, nama_file, file_url,
		       ukuran_file, mime_type, diunggah_oleh, uploaded_at, id_kelompok_kkn,
		       is_verified, verified_by, verified_at
		FROM dokumen
		WHERE id_kelompok_kkn = $1
		ORDER BY uploaded_at ASC, id ASC
	`
	rows, err := db.Query(ctx, q, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("list dokumen kelompok %d: %w", idKelompok, err)
	}
	defer rows.Close()

	var list []Dokumen
	for rows.Next() {
		var d Dokumen
		err := rows.Scan(
			&d.ID, &d.IDPesertaProgram, &d.JenisDokumen, &d.NamaFile, &d.FileURL,
			&d.UkuranFile, &d.MimeType, &d.DiunggahOleh, &d.UploadedAt, &d.IDKelompokKKN,
			&d.IsVerified, &d.VerifiedBy, &d.VerifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dokumen kelompok: %w", err)
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *pgRepository) Delete(ctx context.Context, db database.DBTX, id int) error {
	res, err := db.Exec(ctx, "DELETE FROM dokumen WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete dokumen %d: %w", id, err)
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgRepository) GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
	const q = `
		SELECT id, jenis_program, status, id_mahasiswa
		FROM peserta_program
		WHERE id = $1
	`
	var p PendaftaranInfo
	err := db.QueryRow(ctx, q, idPeserta).Scan(&p.ID, &p.JenisProgram, &p.Status, &p.IDMahasiswa)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query pendaftaran info %d: %w", idPeserta, err)
	}
	return &p, nil
}

func (r *pgRepository) CheckKelompokAccess(ctx context.Context, db database.DBTX, idKelompok int, userID int, userRoles []string, isWrite bool) (bool, error) {
	for _, role := range userRoles {
		if role == "admin" {
			return true, nil
		}
		if !isWrite && role == "pegawai" {
			return true, nil
		}
	}

	// 1. Cek apakah user adalah mahasiswa anggota kelompok KKN ini
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

	// Untuk operasi tulis (unggah dokumen kelompok), hanya anggota kelompok atau admin yang berhak
	if isWrite {
		return false, nil
	}

	// Untuk operasi baca (download/list), periksa juga dosen pembimbing & pembimbing lapangan kelompok
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

	var lapanganCount int
	err = db.QueryRow(ctx, `
		SELECT count(*)
		FROM kelompok_kkn k
		JOIN pembimbing_lapangan pl ON k.id_pembimbing_lapangan = pl.id
		WHERE k.id = $1 AND pl.id_user = $2
	`, idKelompok, userID).Scan(&lapanganCount)
	if err != nil {
		return false, fmt.Errorf("cek pembimbing lapangan kelompok: %w", err)
	}
	if lapanganCount > 0 {
		return true, nil
	}

	return false, nil
}

func (r *pgRepository) GetKelompokMemberIDs(ctx context.Context, db database.DBTX, idKelompok int) ([]int, error) {
	const q = `SELECT id FROM peserta_program WHERE id_kelompok_kkn = $1 AND status = 'diajukan'`
	rows, err := db.Query(ctx, q, idKelompok)
	if err != nil {
		return nil, fmt.Errorf("query kelompok members: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan kelompok member id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kelompok members: %w", err)
	}
	return ids, nil
}
