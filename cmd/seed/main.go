// Package main adalah seeder untuk membuat akun admin pertama secara idempoten.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/config"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/security"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("gagal memuat konfigurasi", "error", err)
		os.Exit(1)
	}

	username := cfg.SeedAdminUsername
	if username == "" {
		username = "admin"
	}
	email := cfg.SeedAdminEmail
	if email == "" {
		email = "admin@splk.undiksha.ac.id"
	}
	password := cfg.SeedAdminPassword
	if password == "" {
		password = "Admin12345678"
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		slog.Error("gagal koneksi database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	err = database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		// 1. Ambil ID role 'admin'
		var roleAdminID int
		err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE nama = 'admin'").Scan(&roleAdminID)
		if err != nil {
			return fmt.Errorf("ambil role admin: %w", err)
		}

		// 2. Cek apakah user sudah ada
		var userID int
		err = tx.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("cek user admin: %w", err)
		}

		if errors.Is(err, pgx.ErrNoRows) {
			// User belum ada: buat baru
			passwordHash, err := security.HashPassword(password)
			if err != nil {
				return fmt.Errorf("hash password admin: %w", err)
			}

			insertUserQuery := `
				INSERT INTO users (username, email, password_hash, is_active)
				VALUES ($1, $2, $3, true)
				RETURNING id
			`
			err = tx.QueryRow(ctx, insertUserQuery, username, email, passwordHash).Scan(&userID)
			if err != nil {
				return fmt.Errorf("insert user admin: %w", err)
			}
			slog.Info("user admin berhasil dibuat", "user_id", userID, "username", username)
		} else {
			slog.Info("user admin sudah ada, memastikan role dan profil", "user_id", userID, "username", username)
		}

		// 3. Pastikan relasi user_roles terisi
		insertRoleQuery := `
			INSERT INTO user_roles (id_user, id_role)
			VALUES ($1, $2)
			ON CONFLICT (id_user, id_role) DO NOTHING
		`
		if _, err := tx.Exec(ctx, insertRoleQuery, userID, roleAdminID); err != nil {
			return fmt.Errorf("insert user_roles: %w", err)
		}

		// 4. Pastikan profil admin terisi
		adminName := "Administrator SPLK"
		insertAdminQuery := `
			INSERT INTO admin (id_user, nama, nip)
			VALUES ($1, $2, 'ADMIN001')
			ON CONFLICT (id_user) DO NOTHING
		`
		if _, err := tx.Exec(ctx, insertAdminQuery, userID, adminName); err != nil {
			return fmt.Errorf("insert profil admin: %w", err)
		}

		return nil
	})

	if err != nil {
		slog.Error("gagal menjalankan seed admin", "error", err)
		os.Exit(1)
	}

	fmt.Println("Seeder admin selesai dengan sukses!")
}
