// Package database menyediakan abstraksi koneksi pool, transaksi, dan pemetaan error PostgreSQL.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/platform/httpx"
)

// DBTX adalah interface yang dipenuhi oleh *pgxpool.Pool maupun pgx.Tx,
// sehingga repository dapat digunakan di dalam maupun di luar transaksi.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx menjalankan fungsi fn di dalam sebuah transaksi database pgx.
// Jika fn mengembalikan error, transaksi di-rollback.
// Jika terjadi panic, transaksi di-rollback dan panic diteruskan kembali (panic-safe).
// Jika fn selesai tanpa error, transaksi di-commit.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return fmt.Errorf("rollback gagal (%v) setelah error: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaksi: %w", err)
	}

	return nil
}

// MapPGError menerjemahkan error pgconn.PgError ke *httpx.AppError sesuai RULES.md bagian 4 dan 5:
// - 23505 (unique_violation) -> 409 Conflict
// - 23503 (foreign_key_violation) -> 422 Unprocessable
// - 23514 (check_violation) -> 422 Unprocessable
// Mengembalikan nil jika error bukan salah satu kode di atas.
func MapPGError(err error) *httpx.AppError {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}

	switch pgErr.Code {
	case "23505":
		return httpx.Conflict("DATA_DUPLIKAT", "data yang dimasukkan sudah ada atau melanggar keunikan")
	case "23503":
		return httpx.Unprocessable("RELASI_TIDAK_VALID", "referensi data terkait tidak ditemukan atau tidak valid")
	case "23514":
		return httpx.Unprocessable("BATASAN_TIDAK_VALID", "data yang dimasukkan melanggar batasan sistem")
	default:
		return nil
	}
}

// MapError memeriksa apakah error merupakan *httpx.AppError atau error pgconn yang dapat dipetakan.
// Jika tidak dikenali, mengembalikan error aslinya.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	if mapped := MapPGError(err); mapped != nil {
		return mapped
	}

	return err
}
