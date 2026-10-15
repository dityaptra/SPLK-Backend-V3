package periode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mendefinisikan seluruh akses database untuk manajemen periode.
type Repository interface {
	CreatePeriode(ctx context.Context, db database.DBTX, p *Periode) error
	FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (*Periode, error)
	ListPeriode(ctx context.Context, db database.DBTX, filter PeriodeFilter, p httpx.Pagination) ([]Periode, int64, error)
	UpdatePeriode(ctx context.Context, db database.DBTX, id int, req UpdatePeriodeRequest) error
	UpdateStatusPeriode(ctx context.Context, db database.DBTX, id int, status string) error

	// FindPegawaiIDByUserID mencari id_pegawai milik user; nil jika tidak ada profil pegawai.
	// Digunakan untuk mengisi dibuat_oleh sesuai RULES.md bagian 7 nomor 2.
	FindPegawaiIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error)

	// SelesaikanPeriodeTx dieksekusi dalam transaksi selesaikan:
	//   - peserta_program.status = 'berjalan' -> 'selesai'
	//   - peserta_program.status = 'diajukan' -> 'ditolak' dengan alasan_tolak otomatis (asumsi A12)
	// Mengembalikan jumlah baris yang diubah (berjalan, ditolak).
	SelesaikanPeriodeTx(ctx context.Context, tx pgx.Tx, periodeID int) (jumlahSelesai int64, jumlahDitolak int64, err error)
}

type pgRepository struct{}

// NewRepository membuat implementasi Repository berbasis pgx.
func NewRepository() Repository {
	return &pgRepository{}
}

func (r *pgRepository) CreatePeriode(ctx context.Context, db database.DBTX, p *Periode) error {
	const q = `
		INSERT INTO periode (
			nama, jenis_program, tahun_semester,
			tanggal_mulai, tanggal_selesai,
			tanggal_buka_daftar, tanggal_tutup_daftar,
			status, dibuat_oleh
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	return db.QueryRow(ctx, q,
		p.Nama, p.JenisProgram, p.TahunSemester,
		p.TanggalMulai, p.TanggalSelesai,
		p.TanggalBukaDaftar, p.TanggalTutupDaftar,
		p.Status, p.DibuatOleh,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *pgRepository) FindPeriodeByID(ctx context.Context, db database.DBTX, id int) (*Periode, error) {
	const q = `
		SELECT id, nama, jenis_program, tahun_semester,
		       tanggal_mulai, tanggal_selesai,
		       tanggal_buka_daftar, tanggal_tutup_daftar,
		       status, dibuat_oleh, created_at
		FROM periode
		WHERE id = $1
	`
	p := &Periode{}
	err := db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.Nama, &p.JenisProgram, &p.TahunSemester,
		&p.TanggalMulai, &p.TanggalSelesai,
		&p.TanggalBukaDaftar, &p.TanggalTutupDaftar,
		&p.Status, &p.DibuatOleh, &p.CreatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find periode by id %d: %w", id, err)
	}
	return p, nil
}

func (r *pgRepository) ListPeriode(ctx context.Context, db database.DBTX, filter PeriodeFilter, p httpx.Pagination) ([]Periode, int64, error) {
	// Build klausa WHERE secara dinamis menggunakan allowlist
	conditions := []string{}
	args := []any{}
	argIdx := 1

	if filter.JenisProgram != "" {
		conditions = append(conditions, fmt.Sprintf("jenis_program = $%d", argIdx))
		args = append(args, filter.JenisProgram)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Hitung total
	var total int64
	countQ := fmt.Sprintf("SELECT count(*) FROM periode %s", where)
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count periode: %w", err)
	}

	// Query utama dengan pagination
	offset := (p.Page - 1) * p.Limit
	listArgs := append(args, p.Limit, offset)
	listQ := fmt.Sprintf(`
		SELECT id, nama, jenis_program, tahun_semester,
		       tanggal_mulai, tanggal_selesai,
		       tanggal_buka_daftar, tanggal_tutup_daftar,
		       status, dibuat_oleh, created_at
		FROM periode
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	rows, err := db.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list periode: %w", err)
	}
	defer rows.Close()

	var list []Periode
	for rows.Next() {
		var item Periode
		if err := rows.Scan(
			&item.ID, &item.Nama, &item.JenisProgram, &item.TahunSemester,
			&item.TanggalMulai, &item.TanggalSelesai,
			&item.TanggalBukaDaftar, &item.TanggalTutupDaftar,
			&item.Status, &item.DibuatOleh, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan periode: %w", err)
		}
		list = append(list, item)
	}
	return list, total, nil
}

func (r *pgRepository) UpdatePeriode(ctx context.Context, db database.DBTX, id int, req UpdatePeriodeRequest) error {
	const q = `
		UPDATE periode SET
			nama = $1,
			tahun_semester = $2,
			tanggal_mulai = $3,
			tanggal_selesai = $4,
			tanggal_buka_daftar = $5,
			tanggal_tutup_daftar = $6
		WHERE id = $7
	`
	// Parse tanggal dari string ke time.Time
	mulai, err := time.Parse("2006-01-02", req.TanggalMulai)
	if err != nil {
		return fmt.Errorf("parse tanggal_mulai: %w", err)
	}
	selesai, err := time.Parse("2006-01-02", req.TanggalSelesai)
	if err != nil {
		return fmt.Errorf("parse tanggal_selesai: %w", err)
	}
	var buka, tutup *time.Time
	if req.TanggalBukaDaftar != nil {
		t, err := time.Parse("2006-01-02", *req.TanggalBukaDaftar)
		if err != nil {
			return fmt.Errorf("parse tanggal_buka_daftar: %w", err)
		}
		buka = &t
	}
	if req.TanggalTutupDaftar != nil {
		t, err := time.Parse("2006-01-02", *req.TanggalTutupDaftar)
		if err != nil {
			return fmt.Errorf("parse tanggal_tutup_daftar: %w", err)
		}
		tutup = &t
	}

	_, err = db.Exec(ctx, q,
		req.Nama, req.TahunSemester,
		mulai, selesai, buka, tutup,
		id,
	)
	return err
}

func (r *pgRepository) UpdateStatusPeriode(ctx context.Context, db database.DBTX, id int, status string) error {
	_, err := db.Exec(ctx, "UPDATE periode SET status = $1 WHERE id = $2", status, id)
	return err
}

func (r *pgRepository) FindPegawaiIDByUserID(ctx context.Context, db database.DBTX, userID int) (*int, error) {
	var id int
	err := db.QueryRow(ctx, "SELECT id FROM pegawai WHERE id_user = $1", userID).Scan(&id)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find pegawai by user_id %d: %w", userID, err)
	}
	return &id, nil
}

// SelesaikanPeriodeTx mengeksekusi update massal peserta_program dalam transaksi yang sudah dimulai.
// ASUMSI: A12 - alasan_tolak otomatis diisi dengan pesan standar saat periode diselesaikan.
func (r *pgRepository) SelesaikanPeriodeTx(ctx context.Context, tx pgx.Tx, periodeID int) (jumlahSelesai int64, jumlahDitolak int64, err error) {
	// 1. Selesaikan peserta yang sedang berjalan
	tag, err := tx.Exec(ctx, `
		UPDATE peserta_program
		SET status = 'selesai'
		WHERE id_periode = $1 AND status = 'berjalan'
	`, periodeID)
	if err != nil {
		return 0, 0, fmt.Errorf("selesaikan peserta berjalan: %w", err)
	}
	jumlahSelesai = tag.RowsAffected()

	// 2. Tolak peserta yang masih diajukan (A12: alasan otomatis karena periode ditutup)
	// ASUMSI: A12 - peserta dengan status 'diajukan' saat periode diselesaikan ditolak otomatis.
	tag, err = tx.Exec(ctx, `
		UPDATE peserta_program
		SET status = 'ditolak',
		    alasan_tolak = 'Periode program telah diselesaikan oleh administrator sebelum pendaftaran diproses.'
		WHERE id_periode = $1 AND status = 'diajukan'
	`, periodeID)
	if err != nil {
		return jumlahSelesai, 0, fmt.Errorf("tolak peserta diajukan: %w", err)
	}
	jumlahDitolak = tag.RowsAffected()

	return jumlahSelesai, jumlahDitolak, nil
}

// isNoRows memeriksa apakah error adalah pgx.ErrNoRows.
func isNoRows(err error) bool {
	return err != nil && (err.Error() == pgx.ErrNoRows.Error() || err == pgx.ErrNoRows)
}

// MapListToPeriodeDetail mengonversi slice Periode ke slice PeriodeDetail.
func MapListToPeriodeDetail(list []Periode) []PeriodeDetail {
	result := make([]PeriodeDetail, 0, len(list))
	for _, p := range list {
		result = append(result, MapToPeriodeDetail(p))
	}
	return result
}

// MapToPeriodeDetail mengonversi Periode ke PeriodeDetail (format tanggal YYYY-MM-DD).
func MapToPeriodeDetail(p Periode) PeriodeDetail {
	d := PeriodeDetail{
		ID:             p.ID,
		Nama:           p.Nama,
		JenisProgram:   p.JenisProgram,
		TahunSemester:  p.TahunSemester,
		TanggalMulai:   p.TanggalMulai.Format("2006-01-02"),
		TanggalSelesai: p.TanggalSelesai.Format("2006-01-02"),
		Status:         p.Status,
		DibuatOleh:     p.DibuatOleh,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
	}
	if p.TanggalBukaDaftar != nil {
		s := p.TanggalBukaDaftar.Format("2006-01-02")
		d.TanggalBukaDaftar = &s
	}
	if p.TanggalTutupDaftar != nil {
		s := p.TanggalTutupDaftar.Format("2006-01-02")
		d.TanggalTutupDaftar = &s
	}
	return d
}
