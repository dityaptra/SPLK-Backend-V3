package dokumen

import (
	"context"

	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
)

// AccessResolver mengabstraksikan pengecekan relasi hak akses pendaftaran.
type AccessResolver interface {
	Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error)
}
