package database_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
)

func TestMapPGError(t *testing.T) {
	t.Run("Pemetaan error 23505 unique violation ke 409 Conflict", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "23505",
			Message: "duplicate key value violates unique constraint",
		}

		appErr := database.MapPGError(pgErr)
		require.NotNil(t, appErr)
		assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
		assert.Equal(t, "DATA_DUPLIKAT", appErr.Code)
	})

	t.Run("Pemetaan error 23503 foreign key violation ke 422 Unprocessable", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "23503",
			Message: "insert or update on table violates foreign key constraint",
		}

		appErr := database.MapPGError(pgErr)
		require.NotNil(t, appErr)
		assert.Equal(t, http.StatusUnprocessableEntity, appErr.HTTPStatus)
		assert.Equal(t, "RELASI_TIDAK_VALID", appErr.Code)
	})

	t.Run("Pemetaan error 23514 check violation ke 422 Unprocessable", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "23514",
			Message: "new row violates check constraint",
		}

		appErr := database.MapPGError(pgErr)
		require.NotNil(t, appErr)
		assert.Equal(t, http.StatusUnprocessableEntity, appErr.HTTPStatus)
		assert.Equal(t, "BATASAN_TIDAK_VALID", appErr.Code)
	})

	t.Run("Error pg lain tidak dipetakan otomatis", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "42P01", // undefined_table
			Message: "relation does not exist",
		}

		appErr := database.MapPGError(pgErr)
		assert.Nil(t, appErr)
	})

	t.Run("MapError dengan AppError dan error standar", func(t *testing.T) {
		customErr := httpx.NotFound("TIDAK_DITEMUKAN", "data tidak ada")
		res1 := database.MapError(customErr)
		assert.Equal(t, customErr, res1)

		stdErr := errors.New("koneksi timeout")
		res2 := database.MapError(stdErr)
		assert.Equal(t, stdErr, res2)

		pgErr := &pgconn.PgError{Code: "23505"}
		res3 := database.MapError(pgErr)
		var appErr *httpx.AppError
		require.True(t, errors.As(res3, &appErr))
		assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
	})
}
