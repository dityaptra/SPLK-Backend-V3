package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
)

// Repository adalah antarmuka akses database untuk modul autentikasi.
type Repository interface {
	FindUserByUsername(ctx context.Context, db database.DBTX, username string) (*User, error)
	FindUserByID(ctx context.Context, db database.DBTX, id int) (*User, error)
	GetUserRoles(ctx context.Context, db database.DBTX, userID int) ([]string, error)
	CreateToken(ctx context.Context, db database.DBTX, token *Token) error
	FindTokenByHash(ctx context.Context, db database.DBTX, tokenHash string) (*Token, error)
	DeleteTokenByHash(ctx context.Context, db database.DBTX, tokenHash string) error
	DeleteTokensByUserID(ctx context.Context, db database.DBTX, userID int) error
	UpdatePassword(ctx context.Context, db database.DBTX, userID int, passwordHash string) error
	CreateLogAktivitas(ctx context.Context, db database.DBTX, log *LogAktivitas) error
}

// PostgresRepository mengimplementasikan Repository menggunakan driver pgx.
type PostgresRepository struct{}

// NewRepository membuat instance PostgresRepository baru.
func NewRepository() Repository {
	return &PostgresRepository{}
}

// FindUserByUsername mencari akun pengguna berdasarkan username.
func (r *PostgresRepository) FindUserByUsername(ctx context.Context, db database.DBTX, username string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`
	var u User
	err := db.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari user by username %s: %w", username, err)
	}
	return &u, nil
}

// FindUserByID mencari akun pengguna berdasarkan ID.
func (r *PostgresRepository) FindUserByID(ctx context.Context, db database.DBTX, id int) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u User
	err := db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari user by id %d: %w", id, err)
	}
	return &u, nil
}

// GetUserRoles mengambil daftar nama role yang dimiliki user melalui JOIN user_roles dan roles.
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
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, roleName)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterasi roles user %d: %w", userID, rows.Err())
	}

	return roles, nil
}

// CreateToken menyimpan hash refresh token baru ke tabel tokens.
func (r *PostgresRepository) CreateToken(ctx context.Context, db database.DBTX, token *Token) error {
	query := `
		INSERT INTO tokens (id_user, token_hash, expired_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := db.QueryRow(ctx, query, token.UserID, token.TokenHash, token.ExpiredAt).Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("simpan token: %w", err)
	}
	return nil
}

// FindTokenByHash mencari refresh token berdasarkan hash SHA-256.
func (r *PostgresRepository) FindTokenByHash(ctx context.Context, db database.DBTX, tokenHash string) (*Token, error) {
	query := `
		SELECT id, id_user, token_hash, expired_at, created_at
		FROM tokens
		WHERE token_hash = $1
	`
	var t Token
	err := db.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiredAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari token by hash: %w", err)
	}
	return &t, nil
}

// DeleteTokenByHash menghapus token berdasarkan hash (saat rotasi atau logout).
func (r *PostgresRepository) DeleteTokenByHash(ctx context.Context, db database.DBTX, tokenHash string) error {
	query := `DELETE FROM tokens WHERE token_hash = $1`
	_, err := db.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("hapus token by hash: %w", err)
	}
	return nil
}

// DeleteTokensByUserID menghapus seluruh refresh token milik user (misal saat reset atau ganti password).
func (r *PostgresRepository) DeleteTokensByUserID(ctx context.Context, db database.DBTX, userID int) error {
	query := `DELETE FROM tokens WHERE id_user = $1`
	_, err := db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("hapus tokens user %d: %w", userID, err)
	}
	return nil
}

// UpdatePassword memperbarui hash password pengguna di tabel users.
func (r *PostgresRepository) UpdatePassword(ctx context.Context, db database.DBTX, userID int, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := db.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password user %d: %w", userID, err)
	}
	return nil
}

// CreateLogAktivitas mencatat baris audit ke tabel log_aktivitas.
func (r *PostgresRepository) CreateLogAktivitas(ctx context.Context, db database.DBTX, log *LogAktivitas) error {
	query := `
		INSERT INTO log_aktivitas (id_user, nama_user, ip_user, agent_browser, action, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := db.QueryRow(ctx, query, log.UserID, log.NamaUser, log.IPUser, log.AgentBrowser, log.Action, log.Status).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("catat log aktivitas user %d: %w", log.UserID, err)
	}
	return nil
}
