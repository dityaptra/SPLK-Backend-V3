package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

// Repository mendefinisikan seluruh akses database untuk manajemen pengguna dan profil aktor.
type Repository interface {
	// User
	CreateUser(ctx context.Context, db database.DBTX, username, email, passwordHash string) (int, error)
	FindUserByID(ctx context.Context, db database.DBTX, id int) (*UserRecord, error)
	FindUserByUsername(ctx context.Context, db database.DBTX, username string) (*UserRecord, error)
	FindUserByEmail(ctx context.Context, db database.DBTX, email string) (*UserRecord, error)
	SetUserStatus(ctx context.Context, db database.DBTX, id int, isActive bool) error
	UpdatePassword(ctx context.Context, db database.DBTX, id int, passwordHash string) error
	ListUsers(ctx context.Context, db database.DBTX, filter UserFilter, p httpx.Pagination) ([]UserListItem, int64, error)
	AssignRole(ctx context.Context, db database.DBTX, userID int, roleName string) error
	GetUserRoles(ctx context.Context, db database.DBTX, userID int) ([]string, error)
	DeleteTokensByUserID(ctx context.Context, db database.DBTX, userID int) error

	// Mahasiswa
	CreateMahasiswa(ctx context.Context, db database.DBTX, m *MahasiswaProfile) error
	FindMahasiswaByID(ctx context.Context, db database.DBTX, id int) (*MahasiswaDetail, error)
	FindMahasiswaByNIM(ctx context.Context, db database.DBTX, nim string) (*MahasiswaProfile, error)
	ListMahasiswa(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]MahasiswaDetail, int64, error)
	UpdateMahasiswa(ctx context.Context, db database.DBTX, id int, req UpdateMahasiswaRequest) error

	// Dosen
	CreateDosen(ctx context.Context, db database.DBTX, d *DosenProfile) error
	FindDosenByID(ctx context.Context, db database.DBTX, id int) (*DosenDetail, error)
	FindDosenByNIP(ctx context.Context, db database.DBTX, nip string) (*DosenProfile, error)
	ListDosen(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]DosenDetail, int64, error)
	UpdateDosen(ctx context.Context, db database.DBTX, id int, req UpdateDosenRequest) error

	// Pegawai
	CreatePegawai(ctx context.Context, db database.DBTX, peg *PegawaiProfile) error
	FindPegawaiByID(ctx context.Context, db database.DBTX, id int) (*PegawaiDetail, error)
	FindPegawaiByNIP(ctx context.Context, db database.DBTX, nip string) (*PegawaiProfile, error)
	ListPegawai(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]PegawaiDetail, int64, error)
	UpdatePegawai(ctx context.Context, db database.DBTX, id int, req UpdatePegawaiRequest) error

	// Admin
	CreateAdmin(ctx context.Context, db database.DBTX, a *AdminProfile) error
	FindAdminByID(ctx context.Context, db database.DBTX, id int) (*AdminDetail, error)
	FindAdminByNIP(ctx context.Context, db database.DBTX, nip string) (*AdminProfile, error)
	ListAdmin(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]AdminDetail, int64, error)
	UpdateAdmin(ctx context.Context, db database.DBTX, id int, req UpdateAdminRequest) error

	// Pembimbing Lapangan
	CreatePembimbingLapangan(ctx context.Context, db database.DBTX, pl *PembimbingLapanganProfile) error
	FindPembimbingLapanganByID(ctx context.Context, db database.DBTX, id int) (*PembimbingLapanganDetail, error)
	ListPembimbingLapangan(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]PembimbingLapanganDetail, int64, error)
	UpdatePembimbingLapangan(ctx context.Context, db database.DBTX, id int, req UpdatePembimbingLapanganRequest) error

	// Validasi Relasi
	CheckProdiExists(ctx context.Context, db database.DBTX, prodiID int) (bool, error)
	CheckTempatProgram(ctx context.Context, db database.DBTX, tempatID int) (exists bool, jenisProgram string, err error)
}

// UserRecord mewakili baris user untuk query repository.
type UserRecord struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string
	IsActive     bool
}

// PostgresRepository mengimplementasikan Repository menggunakan database PostgreSQL.
type PostgresRepository struct{}

// NewRepository membuat instance PostgresRepository baru.
func NewRepository() Repository {
	return &PostgresRepository{}
}

// --- User Implementation ---

func (r *PostgresRepository) CreateUser(ctx context.Context, db database.DBTX, username, email, passwordHash string) (int, error) {
	query := `
		INSERT INTO users (username, email, password_hash, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id
	`
	var id int
	err := db.QueryRow(ctx, query, username, email, passwordHash).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, db database.DBTX, id int) (*UserRecord, error) {
	query := `SELECT id, username, email, password_hash, is_active FROM users WHERE id = $1`
	var u UserRecord
	err := db.QueryRow(ctx, query, id).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id %d: %w", id, err)
	}
	return &u, nil
}

func (r *PostgresRepository) FindUserByUsername(ctx context.Context, db database.DBTX, username string) (*UserRecord, error) {
	query := `SELECT id, username, email, password_hash, is_active FROM users WHERE username = $1`
	var u UserRecord
	err := db.QueryRow(ctx, query, username).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by username %s: %w", username, err)
	}
	return &u, nil
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, db database.DBTX, email string) (*UserRecord, error) {
	query := `SELECT id, username, email, password_hash, is_active FROM users WHERE email = $1`
	var u UserRecord
	err := db.QueryRow(ctx, query, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email %s: %w", email, err)
	}
	return &u, nil
}

func (r *PostgresRepository) SetUserStatus(ctx context.Context, db database.DBTX, id int, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	cmd, err := db.Exec(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("set status user %d: %w", id, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, db database.DBTX, id int, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	cmd, err := db.Exec(ctx, query, passwordHash, id)
	if err != nil {
		return fmt.Errorf("update password user %d: %w", id, err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) DeleteTokensByUserID(ctx context.Context, db database.DBTX, userID int) error {
	_, err := db.Exec(ctx, `DELETE FROM tokens WHERE id_user = $1`, userID)
	return err
}

func (r *PostgresRepository) AssignRole(ctx context.Context, db database.DBTX, userID int, roleName string) error {
	query := `
		INSERT INTO user_roles (id_user, id_role)
		SELECT $1, id FROM roles WHERE nama = $2
		ON CONFLICT (id_user, id_role) DO NOTHING
	`
	cmd, err := db.Exec(ctx, query, userID, roleName)
	if err != nil {
		return fmt.Errorf("assign role '%s' to user %d: %w", roleName, userID, err)
	}
	if cmd.RowsAffected() == 0 {
		// Periksa apakah role ada
		var exists bool
		_ = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE nama = $1)`, roleName).Scan(&exists)
		if !exists {
			return fmt.Errorf("role '%s' tidak ditemukan di database", roleName)
		}
	}
	return nil
}

func (r *PostgresRepository) GetUserRoles(ctx context.Context, db database.DBTX, userID int) ([]string, error) {
	query := `
		SELECT ro.nama
		FROM user_roles ur
		JOIN roles ro ON ur.id_role = ro.id
		WHERE ur.id_user = $1
		ORDER BY ro.id ASC
	`
	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ambil roles user %d: %w", userID, err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, err
		}
		roles = append(roles, roleName)
	}
	if roles == nil {
		roles = []string{}
	}
	return roles, nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context, db database.DBTX, filter UserFilter, p httpx.Pagination) ([]UserListItem, int64, error) {
	baseQuery := `
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.id_user
		LEFT JOIN roles ro ON ur.id_role = ro.id
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if filter.Role != "" {
		baseQuery += fmt.Sprintf(" AND ro.nama = $%d", argIdx)
		args = append(args, filter.Role)
		argIdx++
	}

	if filter.Query != "" {
		likePattern := "%" + strings.ToLower(filter.Query) + "%"
		baseQuery += fmt.Sprintf(" AND (LOWER(u.username) LIKE $%d OR LOWER(u.email) LIKE $%d)", argIdx, argIdx)
		args = append(args, likePattern)
		argIdx++
	}

	// Hitung total user unik
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT u.id) %s", baseQuery)
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung users: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.username, u.email, u.is_active, u.created_at,
		       COALESCE(array_agg(ro.nama) FILTER (WHERE ro.nama IS NOT NULL), '{}') as roles
		%s
		GROUP BY u.id, u.username, u.email, u.is_active, u.created_at
		ORDER BY u.id ASC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, p.Limit, p.Offset())

	rows, err := db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query list users: %w", err)
	}
	defer rows.Close()

	var items []UserListItem
	for rows.Next() {
		var item UserListItem
		var createdAt any
		if err := rows.Scan(&item.ID, &item.Username, &item.Email, &item.IsActive, &createdAt, &item.Roles); err != nil {
			return nil, 0, fmt.Errorf("scan user item: %w", err)
		}
		item.CreatedAt = fmt.Sprintf("%v", createdAt)
		items = append(items, item)
	}
	if items == nil {
		items = []UserListItem{}
	}
	return items, total, nil
}

// --- Mahasiswa ---

func (r *PostgresRepository) CreateMahasiswa(ctx context.Context, db database.DBTX, m *MahasiswaProfile) error {
	query := `
		INSERT INTO mahasiswa (id_user, nim, nama, id_prodi, angkatan, semester_sekarang, no_hp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return db.QueryRow(ctx, query, m.UserID, m.NIM, m.Nama, m.ProdiID, m.Angkatan, m.SemesterSekarang, m.NoHP).Scan(&m.ID)
}

func (r *PostgresRepository) FindMahasiswaByID(ctx context.Context, db database.DBTX, id int) (*MahasiswaDetail, error) {
	query := `
		SELECT m.id, m.id_user, u.username, u.email, u.is_active, m.nim, m.nama,
		       m.id_prodi, COALESCE(p.nama, ''), m.angkatan, m.semester_sekarang, m.no_hp
		FROM mahasiswa m
		JOIN users u ON m.id_user = u.id
		LEFT JOIN program_studi p ON m.id_prodi = p.id
		WHERE m.id = $1
	`
	var d MahasiswaDetail
	err := db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.Username, &d.Email, &d.IsActive, &d.NIM, &d.Nama,
		&d.ProdiID, &d.NamaProdi, &d.Angkatan, &d.SemesterSekarang, &d.NoHP,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil mahasiswa %d: %w", id, err)
	}
	return &d, nil
}

func (r *PostgresRepository) FindMahasiswaByNIM(ctx context.Context, db database.DBTX, nim string) (*MahasiswaProfile, error) {
	query := `SELECT id, id_user, nim, nama, id_prodi, angkatan, semester_sekarang, no_hp FROM mahasiswa WHERE nim = $1`
	var m MahasiswaProfile
	err := db.QueryRow(ctx, query, nim).Scan(&m.ID, &m.UserID, &m.NIM, &m.Nama, &m.ProdiID, &m.Angkatan, &m.SemesterSekarang, &m.NoHP)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari mahasiswa by nim %s: %w", nim, err)
	}
	return &m, nil
}

func (r *PostgresRepository) ListMahasiswa(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]MahasiswaDetail, int64, error) {
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM mahasiswa`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT m.id, m.id_user, u.username, u.email, u.is_active, m.nim, m.nama,
		       m.id_prodi, COALESCE(p.nama, ''), m.angkatan, m.semester_sekarang, m.no_hp
		FROM mahasiswa m
		JOIN users u ON m.id_user = u.id
		LEFT JOIN program_studi p ON m.id_prodi = p.id
		ORDER BY m.id ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.Query(ctx, query, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []MahasiswaDetail
	for rows.Next() {
		var d MahasiswaDetail
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Username, &d.Email, &d.IsActive, &d.NIM, &d.Nama,
			&d.ProdiID, &d.NamaProdi, &d.Angkatan, &d.SemesterSekarang, &d.NoHP,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, d)
	}
	if items == nil {
		items = []MahasiswaDetail{}
	}
	return items, total, nil
}

func (r *PostgresRepository) UpdateMahasiswa(ctx context.Context, db database.DBTX, id int, req UpdateMahasiswaRequest) error {
	query := `
		UPDATE mahasiswa
		SET nim = $1, nama = $2, id_prodi = $3, angkatan = $4, semester_sekarang = $5, no_hp = $6
		WHERE id = $7
	`
	cmd, err := db.Exec(ctx, query, req.NIM, req.Nama, req.ProdiID, req.Angkatan, req.SemesterSekarang, req.NoHP, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Dosen ---

func (r *PostgresRepository) CreateDosen(ctx context.Context, db database.DBTX, d *DosenProfile) error {
	query := `
		INSERT INTO dosen (id_user, nip, nama, id_prodi, gelar_depan, gelar_belakang)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return db.QueryRow(ctx, query, d.UserID, d.NIP, d.Nama, d.ProdiID, d.GelarDepan, d.GelarBelakang).Scan(&d.ID)
}

func (r *PostgresRepository) FindDosenByID(ctx context.Context, db database.DBTX, id int) (*DosenDetail, error) {
	query := `
		SELECT d.id, d.id_user, u.username, u.email, u.is_active, d.nip, d.nama,
		       d.id_prodi, COALESCE(p.nama, ''), d.gelar_depan, d.gelar_belakang
		FROM dosen d
		JOIN users u ON d.id_user = u.id
		LEFT JOIN program_studi p ON d.id_prodi = p.id
		WHERE d.id = $1
	`
	var dt DosenDetail
	err := db.QueryRow(ctx, query, id).Scan(
		&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive, &dt.NIP, &dt.Nama,
		&dt.ProdiID, &dt.NamaProdi, &dt.GelarDepan, &dt.GelarBelakang,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil dosen %d: %w", id, err)
	}
	return &dt, nil
}

func (r *PostgresRepository) FindDosenByNIP(ctx context.Context, db database.DBTX, nip string) (*DosenProfile, error) {
	query := `SELECT id, id_user, nip, nama, id_prodi, gelar_depan, gelar_belakang FROM dosen WHERE nip = $1`
	var d DosenProfile
	err := db.QueryRow(ctx, query, nip).Scan(&d.ID, &d.UserID, &d.NIP, &d.Nama, &d.ProdiID, &d.GelarDepan, &d.GelarBelakang)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *PostgresRepository) ListDosen(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]DosenDetail, int64, error) {
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM dosen`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT d.id, d.id_user, u.username, u.email, u.is_active, d.nip, d.nama,
		       d.id_prodi, COALESCE(p.nama, ''), d.gelar_depan, d.gelar_belakang
		FROM dosen d
		JOIN users u ON d.id_user = u.id
		LEFT JOIN program_studi p ON d.id_prodi = p.id
		ORDER BY d.id ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.Query(ctx, query, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []DosenDetail
	for rows.Next() {
		var dt DosenDetail
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive, &dt.NIP, &dt.Nama,
			&dt.ProdiID, &dt.NamaProdi, &dt.GelarDepan, &dt.GelarBelakang,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, dt)
	}
	if items == nil {
		items = []DosenDetail{}
	}
	return items, total, nil
}

func (r *PostgresRepository) UpdateDosen(ctx context.Context, db database.DBTX, id int, req UpdateDosenRequest) error {
	query := `
		UPDATE dosen
		SET nip = $1, nama = $2, id_prodi = $3, gelar_depan = $4, gelar_belakang = $5
		WHERE id = $6
	`
	cmd, err := db.Exec(ctx, query, req.NIP, req.Nama, req.ProdiID, req.GelarDepan, req.GelarBelakang, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Pegawai ---

func (r *PostgresRepository) CreatePegawai(ctx context.Context, db database.DBTX, peg *PegawaiProfile) error {
	query := `
		INSERT INTO pegawai (id_user, nip, nama, gelar_depan, gelar_belakang, jabatan)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return db.QueryRow(ctx, query, peg.UserID, peg.NIP, peg.Nama, peg.GelarDepan, peg.GelarBelakang, peg.Jabatan).Scan(&peg.ID)
}

func (r *PostgresRepository) FindPegawaiByID(ctx context.Context, db database.DBTX, id int) (*PegawaiDetail, error) {
	query := `
		SELECT p.id, p.id_user, u.username, u.email, u.is_active, p.nip, p.nama,
		       p.gelar_depan, p.gelar_belakang, p.jabatan
		FROM pegawai p
		JOIN users u ON p.id_user = u.id
		WHERE p.id = $1
	`
	var dt PegawaiDetail
	err := db.QueryRow(ctx, query, id).Scan(
		&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive, &dt.NIP, &dt.Nama,
		&dt.GelarDepan, &dt.GelarBelakang, &dt.Jabatan,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil pegawai %d: %w", id, err)
	}
	return &dt, nil
}

func (r *PostgresRepository) FindPegawaiByNIP(ctx context.Context, db database.DBTX, nip string) (*PegawaiProfile, error) {
	query := `SELECT id, id_user, nip, nama, gelar_depan, gelar_belakang, jabatan FROM pegawai WHERE nip = $1`
	var peg PegawaiProfile
	err := db.QueryRow(ctx, query, nip).Scan(&peg.ID, &peg.UserID, &peg.NIP, &peg.Nama, &peg.GelarDepan, &peg.GelarBelakang, &peg.Jabatan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &peg, nil
}

func (r *PostgresRepository) ListPegawai(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]PegawaiDetail, int64, error) {
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM pegawai`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT p.id, p.id_user, u.username, u.email, u.is_active, p.nip, p.nama,
		       p.gelar_depan, p.gelar_belakang, p.jabatan
		FROM pegawai p
		JOIN users u ON p.id_user = u.id
		ORDER BY p.id ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.Query(ctx, query, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []PegawaiDetail
	for rows.Next() {
		var dt PegawaiDetail
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive, &dt.NIP, &dt.Nama,
			&dt.GelarDepan, &dt.GelarBelakang, &dt.Jabatan,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, dt)
	}
	if items == nil {
		items = []PegawaiDetail{}
	}
	return items, total, nil
}

func (r *PostgresRepository) UpdatePegawai(ctx context.Context, db database.DBTX, id int, req UpdatePegawaiRequest) error {
	query := `
		UPDATE pegawai
		SET nip = $1, nama = $2, gelar_depan = $3, gelar_belakang = $4, jabatan = $5
		WHERE id = $6
	`
	cmd, err := db.Exec(ctx, query, req.NIP, req.Nama, req.GelarDepan, req.GelarBelakang, req.Jabatan, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Admin ---

func (r *PostgresRepository) CreateAdmin(ctx context.Context, db database.DBTX, a *AdminProfile) error {
	query := `
		INSERT INTO admin (id_user, nip, nama)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	return db.QueryRow(ctx, query, a.UserID, a.NIP, a.Nama).Scan(&a.ID)
}

func (r *PostgresRepository) FindAdminByID(ctx context.Context, db database.DBTX, id int) (*AdminDetail, error) {
	query := `
		SELECT a.id, a.id_user, u.username, u.email, u.is_active, a.nip, a.nama
		FROM admin a
		JOIN users u ON a.id_user = u.id
		WHERE a.id = $1
	`
	var dt AdminDetail
	err := db.QueryRow(ctx, query, id).Scan(&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive, &dt.NIP, &dt.Nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil admin %d: %w", id, err)
	}
	return &dt, nil
}

func (r *PostgresRepository) FindAdminByNIP(ctx context.Context, db database.DBTX, nip string) (*AdminProfile, error) {
	query := `SELECT id, id_user, nip, nama FROM admin WHERE nip = $1`
	var a AdminProfile
	err := db.QueryRow(ctx, query, nip).Scan(&a.ID, &a.UserID, &a.NIP, &a.Nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *PostgresRepository) ListAdmin(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]AdminDetail, int64, error) {
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM admin`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT a.id, a.id_user, u.username, u.email, u.is_active, a.nip, a.nama
		FROM admin a
		JOIN users u ON a.id_user = u.id
		ORDER BY a.id ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.Query(ctx, query, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []AdminDetail
	for rows.Next() {
		var dt AdminDetail
		if err := rows.Scan(&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive, &dt.NIP, &dt.Nama); err != nil {
			return nil, 0, err
		}
		items = append(items, dt)
	}
	if items == nil {
		items = []AdminDetail{}
	}
	return items, total, nil
}

func (r *PostgresRepository) UpdateAdmin(ctx context.Context, db database.DBTX, id int, req UpdateAdminRequest) error {
	query := `UPDATE admin SET nip = $1, nama = $2 WHERE id = $3`
	cmd, err := db.Exec(ctx, query, req.NIP, req.Nama, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Pembimbing Lapangan ---

func (r *PostgresRepository) CreatePembimbingLapangan(ctx context.Context, db database.DBTX, pl *PembimbingLapanganProfile) error {
	query := `
		INSERT INTO pembimbing_lapangan (jenis, id_tempat_program, id_user, nama, jabatan, kontak)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return db.QueryRow(ctx, query, pl.Jenis, pl.TempatProgramID, pl.UserID, pl.Nama, pl.Jabatan, pl.Kontak).Scan(&pl.ID)
}

func (r *PostgresRepository) FindPembimbingLapanganByID(ctx context.Context, db database.DBTX, id int) (*PembimbingLapanganDetail, error) {
	query := `
		SELECT pl.id, pl.id_user, COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(u.is_active, false),
		       pl.jenis, pl.id_tempat_program, COALESCE(tp.nama, ''), pl.nama, pl.jabatan, pl.kontak
		FROM pembimbing_lapangan pl
		LEFT JOIN users u ON pl.id_user = u.id
		LEFT JOIN tempat_program tp ON pl.id_tempat_program = tp.id
		WHERE pl.id = $1
	`
	var dt PembimbingLapanganDetail
	err := db.QueryRow(ctx, query, id).Scan(
		&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive,
		&dt.Jenis, &dt.TempatProgramID, &dt.NamaTempat, &dt.Nama, &dt.Jabatan, &dt.Kontak,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil pembimbing lapangan %d: %w", id, err)
	}
	return &dt, nil
}

func (r *PostgresRepository) ListPembimbingLapangan(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]PembimbingLapanganDetail, int64, error) {
	var total int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM pembimbing_lapangan`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT pl.id, pl.id_user, COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(u.is_active, false),
		       pl.jenis, pl.id_tempat_program, COALESCE(tp.nama, ''), pl.nama, pl.jabatan, pl.kontak
		FROM pembimbing_lapangan pl
		LEFT JOIN users u ON pl.id_user = u.id
		LEFT JOIN tempat_program tp ON pl.id_tempat_program = tp.id
		ORDER BY pl.id ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := db.Query(ctx, query, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []PembimbingLapanganDetail
	for rows.Next() {
		var dt PembimbingLapanganDetail
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Username, &dt.Email, &dt.IsActive,
			&dt.Jenis, &dt.TempatProgramID, &dt.NamaTempat, &dt.Nama, &dt.Jabatan, &dt.Kontak,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, dt)
	}
	if items == nil {
		items = []PembimbingLapanganDetail{}
	}
	return items, total, nil
}

func (r *PostgresRepository) UpdatePembimbingLapangan(ctx context.Context, db database.DBTX, id int, req UpdatePembimbingLapanganRequest) error {
	query := `
		UPDATE pembimbing_lapangan
		SET jenis = $1, id_tempat_program = $2, nama = $3, jabatan = $4, kontak = $5
		WHERE id = $6
	`
	cmd, err := db.Exec(ctx, query, req.Jenis, req.TempatProgramID, req.Nama, req.Jabatan, req.Kontak, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Validasi Relasi ---

func (r *PostgresRepository) CheckProdiExists(ctx context.Context, db database.DBTX, prodiID int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM program_studi WHERE id = $1)`, prodiID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) CheckTempatProgram(ctx context.Context, db database.DBTX, tempatID int) (bool, string, error) {
	var jenis string
	err := db.QueryRow(ctx, `SELECT jenis_program FROM tempat_program WHERE id = $1`, tempatID).Scan(&jenis)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, jenis, nil
}
