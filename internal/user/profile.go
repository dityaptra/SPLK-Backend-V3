// Package user menyediakan profil aktor dan resolver profil berdasarkan user_id.
package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"splk-backend/internal/database"
)

// MahasiswaProfile mewakili profil mahasiswa.
type MahasiswaProfile struct {
	ID               int     `json:"id"`
	UserID           int     `json:"id_user"`
	NIM              string  `json:"nim"`
	Nama             string  `json:"nama"`
	ProdiID          int     `json:"id_prodi"`
	Angkatan         int     `json:"angkatan"`
	SemesterSekarang *int    `json:"semester_sekarang,omitempty"`
	NoHP             *string `json:"no_hp,omitempty"`
}

// DosenProfile mewakili profil dosen.
type DosenProfile struct {
	ID            int     `json:"id"`
	UserID        int     `json:"id_user"`
	NIP           string  `json:"nip"`
	Nama          string  `json:"nama"`
	ProdiID       int     `json:"id_prodi"`
	GelarDepan    *string `json:"gelar_depan,omitempty"`
	GelarBelakang *string `json:"gelar_belakang,omitempty"`
}

// PegawaiProfile mewakili profil pegawai (Kejur/PIC).
type PegawaiProfile struct {
	ID            int     `json:"id"`
	UserID        int     `json:"id_user"`
	NIP           string  `json:"nip"`
	Nama          string  `json:"nama"`
	GelarDepan    *string `json:"gelar_depan,omitempty"`
	GelarBelakang *string `json:"gelar_belakang,omitempty"`
	Jabatan       string  `json:"jabatan"`
}

// AdminProfile mewakili profil administrator sistem.
type AdminProfile struct {
	ID     int     `json:"id"`
	UserID int     `json:"id_user"`
	NIP    *string `json:"nip,omitempty"`
	Nama   string  `json:"nama"`
}

// PembimbingLapanganProfile mewakili profil pembimbing lapangan (mentor/guru pamong/kepala desa).
type PembimbingLapanganProfile struct {
	ID              int     `json:"id"`
	Jenis           string  `json:"jenis"`
	TempatProgramID int     `json:"id_tempat_program"`
	UserID          *int    `json:"id_user,omitempty"`
	Nama            string  `json:"nama"`
	Jabatan         *string `json:"jabatan,omitempty"`
	Kontak          *string `json:"kontak,omitempty"`
}

// ProfileResolver adalah antarmuka untuk mengambil profil aktor berdasarkan ID pengguna.
type ProfileResolver interface {
	GetMahasiswaByUserID(ctx context.Context, db database.DBTX, userID int) (*MahasiswaProfile, error)
	GetDosenByUserID(ctx context.Context, db database.DBTX, userID int) (*DosenProfile, error)
	GetPegawaiByUserID(ctx context.Context, db database.DBTX, userID int) (*PegawaiProfile, error)
	GetAdminByUserID(ctx context.Context, db database.DBTX, userID int) (*AdminProfile, error)
	GetPembimbingLapanganByUserID(ctx context.Context, db database.DBTX, userID int) (*PembimbingLapanganProfile, error)
}

// DefaultProfileResolver adalah implementasi PostgreSQL untuk ProfileResolver.
type DefaultProfileResolver struct{}

// NewProfileResolver membuat instance DefaultProfileResolver baru.
func NewProfileResolver() ProfileResolver {
	return &DefaultProfileResolver{}
}

// GetMahasiswaByUserID mengambil profil mahasiswa berdasarkan user ID.
func (r *DefaultProfileResolver) GetMahasiswaByUserID(ctx context.Context, db database.DBTX, userID int) (*MahasiswaProfile, error) {
	query := `
		SELECT id, id_user, nim, nama, id_prodi, angkatan, semester_sekarang, no_hp
		FROM mahasiswa
		WHERE id_user = $1
	`
	var m MahasiswaProfile
	err := db.QueryRow(ctx, query, userID).Scan(
		&m.ID, &m.UserID, &m.NIM, &m.Nama, &m.ProdiID, &m.Angkatan, &m.SemesterSekarang, &m.NoHP,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil profil mahasiswa user %d: %w", userID, err)
	}
	return &m, nil
}

// GetDosenByUserID mengambil profil dosen berdasarkan user ID.
func (r *DefaultProfileResolver) GetDosenByUserID(ctx context.Context, db database.DBTX, userID int) (*DosenProfile, error) {
	query := `
		SELECT id, id_user, nip, nama, id_prodi, gelar_depan, gelar_belakang
		FROM dosen
		WHERE id_user = $1
	`
	var d DosenProfile
	err := db.QueryRow(ctx, query, userID).Scan(
		&d.ID, &d.UserID, &d.NIP, &d.Nama, &d.ProdiID, &d.GelarDepan, &d.GelarBelakang,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil profil dosen user %d: %w", userID, err)
	}
	return &d, nil
}

// GetPegawaiByUserID mengambil profil pegawai berdasarkan user ID.
func (r *DefaultProfileResolver) GetPegawaiByUserID(ctx context.Context, db database.DBTX, userID int) (*PegawaiProfile, error) {
	query := `
		SELECT id, id_user, nip, nama, gelar_depan, gelar_belakang, jabatan
		FROM pegawai
		WHERE id_user = $1
	`
	var p PegawaiProfile
	err := db.QueryRow(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.NIP, &p.Nama, &p.GelarDepan, &p.GelarBelakang, &p.Jabatan,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil profil pegawai user %d: %w", userID, err)
	}
	return &p, nil
}

// GetAdminByUserID mengambil profil admin berdasarkan user ID.
func (r *DefaultProfileResolver) GetAdminByUserID(ctx context.Context, db database.DBTX, userID int) (*AdminProfile, error) {
	query := `
		SELECT id, id_user, nip, nama
		FROM admin
		WHERE id_user = $1
	`
	var a AdminProfile
	err := db.QueryRow(ctx, query, userID).Scan(
		&a.ID, &a.UserID, &a.NIP, &a.Nama,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil profil admin user %d: %w", userID, err)
	}
	return &a, nil
}

// GetPembimbingLapanganByUserID mengambil profil pembimbing lapangan berdasarkan user ID.
func (r *DefaultProfileResolver) GetPembimbingLapanganByUserID(ctx context.Context, db database.DBTX, userID int) (*PembimbingLapanganProfile, error) {
	query := `
		SELECT id, jenis, id_tempat_program, id_user, nama, jabatan, kontak
		FROM pembimbing_lapangan
		WHERE id_user = $1
	`
	var pl PembimbingLapanganProfile
	err := db.QueryRow(ctx, query, userID).Scan(
		&pl.ID, &pl.Jenis, &pl.TempatProgramID, &pl.UserID, &pl.Nama, &pl.Jabatan, &pl.Kontak,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ambil profil pembimbing lapangan user %d: %w", userID, err)
	}
	return &pl, nil
}
