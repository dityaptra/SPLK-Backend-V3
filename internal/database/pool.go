// Package database menyediakan koneksi pool PostgreSQL menggunakan pgxpool.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/config"
)

// NewPool membuat pool koneksi PostgreSQL dengan timezone UTC.
func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	// Atur timezone koneksi ke UTC sesuai RULES.md bagian 4 nomor 6.
	poolCfg.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("buat pool database: %w", err)
	}

	// Ping untuk memastikan koneksi berjalan
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
