package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/platform/security"
	"splk-backend/internal/tempat"
)

// Service mendefinisikan seluruh use-case manajemen pengguna dan profil aktor.
type Service interface {
	// Mahasiswa
	CreateMahasiswa(ctx context.Context, req CreateMahasiswaRequest) (*MahasiswaDetail, error)
	GetMahasiswaByID(ctx context.Context, id int) (*MahasiswaDetail, error)
	ListMahasiswa(ctx context.Context, p httpx.Pagination) ([]MahasiswaDetail, httpx.PaginationMeta, error)
	UpdateMahasiswa(ctx context.Context, id int, req UpdateMahasiswaRequest) (*MahasiswaDetail, error)

	// Dosen
	CreateDosen(ctx context.Context, req CreateDosenRequest) (*DosenDetail, error)
	GetDosenByID(ctx context.Context, id int) (*DosenDetail, error)
	ListDosen(ctx context.Context, p httpx.Pagination) ([]DosenDetail, httpx.PaginationMeta, error)
	UpdateDosen(ctx context.Context, id int, req UpdateDosenRequest) (*DosenDetail, error)

	// Pegawai
	CreatePegawai(ctx context.Context, req CreatePegawaiRequest) (*PegawaiDetail, error)
	GetPegawaiByID(ctx context.Context, id int) (*PegawaiDetail, error)
	ListPegawai(ctx context.Context, p httpx.Pagination) ([]PegawaiDetail, httpx.PaginationMeta, error)
	UpdatePegawai(ctx context.Context, id int, req UpdatePegawaiRequest) (*PegawaiDetail, error)

	// Admin
	CreateAdmin(ctx context.Context, req CreateAdminRequest) (*AdminDetail, error)
	GetAdminByID(ctx context.Context, id int) (*AdminDetail, error)
	ListAdmin(ctx context.Context, p httpx.Pagination) ([]AdminDetail, httpx.PaginationMeta, error)
	UpdateAdmin(ctx context.Context, id int, req UpdateAdminRequest) (*AdminDetail, error)

	// Pembimbing Lapangan
	CreatePembimbingLapangan(ctx context.Context, req CreatePembimbingLapanganRequest) (*PembimbingLapanganDetail, error)
	GetPembimbingLapanganByID(ctx context.Context, id int) (*PembimbingLapanganDetail, error)
	ListPembimbingLapangan(ctx context.Context, p httpx.Pagination) ([]PembimbingLapanganDetail, httpx.PaginationMeta, error)
	UpdatePembimbingLapangan(ctx context.Context, id int, req UpdatePembimbingLapanganRequest) (*PembimbingLapanganDetail, error)

	// User Umum
	ListUsers(ctx context.Context, filter UserFilter, p httpx.Pagination) ([]UserListItem, httpx.PaginationMeta, error)
	GetUserByID(ctx context.Context, id int) (*UserRecord, []string, error)
	SetUserStatus(ctx context.Context, id int, isActive bool) error
	ResetPassword(ctx context.Context, id int, req ResetPasswordRequest) error
}

type userService struct {
	repo Repository
	pool *pgxpool.Pool
}

// NewService membuat instance Service user baru.
func NewService(repo Repository, pool *pgxpool.Pool) Service {
	return &userService{
		repo: repo,
		pool: pool,
	}
}

// --- Mahasiswa ---

func (s *userService) CreateMahasiswa(ctx context.Context, req CreateMahasiswaRequest) (*MahasiswaDetail, error) {
	// Validasi keunikan username dan email
	if existing, _ := s.repo.FindUserByUsername(ctx, s.pool, req.Username); existing != nil {
		return nil, httpx.Conflict("USERNAME_SUDAH_DIGUNAKAN", "username sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindUserByEmail(ctx, s.pool, req.Email); existing != nil {
		return nil, httpx.Conflict("EMAIL_SUDAH_DIGUNAKAN", "email sudah digunakan oleh pengguna lain")
	}
	// Validasi keunikan NIM
	if existing, _ := s.repo.FindMahasiswaByNIM(ctx, s.pool, req.NIM); existing != nil {
		return nil, httpx.Conflict("NIM_SUDAH_DIGUNAKAN", "NIM sudah terdaftar di sistem")
	}
	// Validasi keberadaan prodi
	exists, err := s.repo.CheckProdiExists(ctx, s.pool, req.ProdiID)
	if err != nil {
		return nil, fmt.Errorf("cek prodi: %w", err)
	}
	if !exists {
		return nil, httpx.Unprocessable("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}

	passHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password mahasiswa: %w", err)
	}

	var mID int
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		userID, err := s.repo.CreateUser(ctx, tx, req.Username, req.Email, passHash)
		if err != nil {
			return err
		}

		if err := s.repo.AssignRole(ctx, tx, userID, "mahasiswa"); err != nil {
			return err
		}

		m := &MahasiswaProfile{
			UserID:           userID,
			NIM:              req.NIM,
			Nama:             req.Nama,
			ProdiID:          req.ProdiID,
			Angkatan:         req.Angkatan,
			SemesterSekarang: req.SemesterSekarang,
			NoHP:             req.NoHP,
		}
		if err := s.repo.CreateMahasiswa(ctx, tx, m); err != nil {
			return err
		}
		mID = m.ID
		return nil
	})

	if err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindMahasiswaByID(ctx, s.pool, mID)
}

func (s *userService) GetMahasiswaByID(ctx context.Context, id int) (*MahasiswaDetail, error) {
	m, err := s.repo.FindMahasiswaByID(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("ambil mahasiswa: %w", err)
	}
	if m == nil {
		return nil, httpx.NotFound("MAHASISWA_TIDAK_DITEMUKAN", "data mahasiswa tidak ditemukan")
	}
	return m, nil
}

func (s *userService) ListMahasiswa(ctx context.Context, p httpx.Pagination) ([]MahasiswaDetail, httpx.PaginationMeta, error) {
	items, total, err := s.repo.ListMahasiswa(ctx, s.pool, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	return items, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *userService) UpdateMahasiswa(ctx context.Context, id int, req UpdateMahasiswaRequest) (*MahasiswaDetail, error) {
	existing, err := s.repo.FindMahasiswaByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, httpx.NotFound("MAHASISWA_TIDAK_DITEMUKAN", "data mahasiswa tidak ditemukan")
	}

	// Cek keunikan NIM jika diubah
	if existing.NIM != req.NIM {
		if other, _ := s.repo.FindMahasiswaByNIM(ctx, s.pool, req.NIM); other != nil {
			return nil, httpx.Conflict("NIM_SUDAH_DIGUNAKAN", "NIM sudah terdaftar di sistem")
		}
	}

	// Cek prodi
	exists, err := s.repo.CheckProdiExists(ctx, s.pool, req.ProdiID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.Unprocessable("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}

	if err := s.repo.UpdateMahasiswa(ctx, s.pool, id, req); err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindMahasiswaByID(ctx, s.pool, id)
}

// --- Dosen ---

func (s *userService) CreateDosen(ctx context.Context, req CreateDosenRequest) (*DosenDetail, error) {
	if existing, _ := s.repo.FindUserByUsername(ctx, s.pool, req.Username); existing != nil {
		return nil, httpx.Conflict("USERNAME_SUDAH_DIGUNAKAN", "username sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindUserByEmail(ctx, s.pool, req.Email); existing != nil {
		return nil, httpx.Conflict("EMAIL_SUDAH_DIGUNAKAN", "email sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindDosenByNIP(ctx, s.pool, req.NIP); existing != nil {
		return nil, httpx.Conflict("NIP_SUDAH_DIGUNAKAN", "NIP dosen sudah terdaftar di sistem")
	}

	exists, err := s.repo.CheckProdiExists(ctx, s.pool, req.ProdiID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.Unprocessable("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}

	passHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var dID int
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		userID, err := s.repo.CreateUser(ctx, tx, req.Username, req.Email, passHash)
		if err != nil {
			return err
		}

		if err := s.repo.AssignRole(ctx, tx, userID, "dosen"); err != nil {
			return err
		}

		d := &DosenProfile{
			UserID:        userID,
			NIP:           req.NIP,
			Nama:          req.Nama,
			ProdiID:       req.ProdiID,
			GelarDepan:    req.GelarDepan,
			GelarBelakang: req.GelarBelakang,
		}
		if err := s.repo.CreateDosen(ctx, tx, d); err != nil {
			return err
		}
		dID = d.ID
		return nil
	})

	if err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindDosenByID(ctx, s.pool, dID)
}

func (s *userService) GetDosenByID(ctx context.Context, id int) (*DosenDetail, error) {
	d, err := s.repo.FindDosenByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, httpx.NotFound("DOSEN_TIDAK_DITEMUKAN", "data dosen tidak ditemukan")
	}
	return d, nil
}

func (s *userService) ListDosen(ctx context.Context, p httpx.Pagination) ([]DosenDetail, httpx.PaginationMeta, error) {
	items, total, err := s.repo.ListDosen(ctx, s.pool, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	return items, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *userService) UpdateDosen(ctx context.Context, id int, req UpdateDosenRequest) (*DosenDetail, error) {
	existing, err := s.repo.FindDosenByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, httpx.NotFound("DOSEN_TIDAK_DITEMUKAN", "data dosen tidak ditemukan")
	}

	if existing.NIP != req.NIP {
		if other, _ := s.repo.FindDosenByNIP(ctx, s.pool, req.NIP); other != nil {
			return nil, httpx.Conflict("NIP_SUDAH_DIGUNAKAN", "NIP dosen sudah terdaftar di sistem")
		}
	}

	exists, err := s.repo.CheckProdiExists(ctx, s.pool, req.ProdiID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.Unprocessable("PRODI_TIDAK_DITEMUKAN", "program studi tidak ditemukan")
	}

	if err := s.repo.UpdateDosen(ctx, s.pool, id, req); err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindDosenByID(ctx, s.pool, id)
}

// --- Pegawai ---

func (s *userService) CreatePegawai(ctx context.Context, req CreatePegawaiRequest) (*PegawaiDetail, error) {
	if existing, _ := s.repo.FindUserByUsername(ctx, s.pool, req.Username); existing != nil {
		return nil, httpx.Conflict("USERNAME_SUDAH_DIGUNAKAN", "username sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindUserByEmail(ctx, s.pool, req.Email); existing != nil {
		return nil, httpx.Conflict("EMAIL_SUDAH_DIGUNAKAN", "email sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindPegawaiByNIP(ctx, s.pool, req.NIP); existing != nil {
		return nil, httpx.Conflict("NIP_SUDAH_DIGUNAKAN", "NIP pegawai sudah terdaftar di sistem")
	}

	passHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var pegID int
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		userID, err := s.repo.CreateUser(ctx, tx, req.Username, req.Email, passHash)
		if err != nil {
			return err
		}

		if err := s.repo.AssignRole(ctx, tx, userID, "pegawai"); err != nil {
			return err
		}

		peg := &PegawaiProfile{
			UserID:        userID,
			NIP:           req.NIP,
			Nama:          req.Nama,
			GelarDepan:    req.GelarDepan,
			GelarBelakang: req.GelarBelakang,
			Jabatan:       req.Jabatan,
		}
		if err := s.repo.CreatePegawai(ctx, tx, peg); err != nil {
			return err
		}
		pegID = peg.ID
		return nil
	})

	if err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindPegawaiByID(ctx, s.pool, pegID)
}

func (s *userService) GetPegawaiByID(ctx context.Context, id int) (*PegawaiDetail, error) {
	peg, err := s.repo.FindPegawaiByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if peg == nil {
		return nil, httpx.NotFound("PEGAWAI_TIDAK_DITEMUKAN", "data pegawai tidak ditemukan")
	}
	return peg, nil
}

func (s *userService) ListPegawai(ctx context.Context, p httpx.Pagination) ([]PegawaiDetail, httpx.PaginationMeta, error) {
	items, total, err := s.repo.ListPegawai(ctx, s.pool, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	return items, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *userService) UpdatePegawai(ctx context.Context, id int, req UpdatePegawaiRequest) (*PegawaiDetail, error) {
	existing, err := s.repo.FindPegawaiByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, httpx.NotFound("PEGAWAI_TIDAK_DITEMUKAN", "data pegawai tidak ditemukan")
	}

	if existing.NIP != req.NIP {
		if other, _ := s.repo.FindPegawaiByNIP(ctx, s.pool, req.NIP); other != nil {
			return nil, httpx.Conflict("NIP_SUDAH_DIGUNAKAN", "NIP pegawai sudah terdaftar di sistem")
		}
	}

	if err := s.repo.UpdatePegawai(ctx, s.pool, id, req); err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindPegawaiByID(ctx, s.pool, id)
}

// --- Admin ---

func (s *userService) CreateAdmin(ctx context.Context, req CreateAdminRequest) (*AdminDetail, error) {
	if existing, _ := s.repo.FindUserByUsername(ctx, s.pool, req.Username); existing != nil {
		return nil, httpx.Conflict("USERNAME_SUDAH_DIGUNAKAN", "username sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindUserByEmail(ctx, s.pool, req.Email); existing != nil {
		return nil, httpx.Conflict("EMAIL_SUDAH_DIGUNAKAN", "email sudah digunakan oleh pengguna lain")
	}
	if req.NIP != nil && *req.NIP != "" {
		if existing, _ := s.repo.FindAdminByNIP(ctx, s.pool, *req.NIP); existing != nil {
			return nil, httpx.Conflict("NIP_SUDAH_DIGUNAKAN", "NIP admin sudah terdaftar di sistem")
		}
	}

	passHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var aID int
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		userID, err := s.repo.CreateUser(ctx, tx, req.Username, req.Email, passHash)
		if err != nil {
			return err
		}

		if err := s.repo.AssignRole(ctx, tx, userID, "admin"); err != nil {
			return err
		}

		a := &AdminProfile{
			UserID: userID,
			NIP:    req.NIP,
			Nama:   req.Nama,
		}
		if err := s.repo.CreateAdmin(ctx, tx, a); err != nil {
			return err
		}
		aID = a.ID
		return nil
	})

	if err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindAdminByID(ctx, s.pool, aID)
}

func (s *userService) GetAdminByID(ctx context.Context, id int) (*AdminDetail, error) {
	a, err := s.repo.FindAdminByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, httpx.NotFound("ADMIN_TIDAK_DITEMUKAN", "data admin tidak ditemukan")
	}
	return a, nil
}

func (s *userService) ListAdmin(ctx context.Context, p httpx.Pagination) ([]AdminDetail, httpx.PaginationMeta, error) {
	items, total, err := s.repo.ListAdmin(ctx, s.pool, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	return items, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *userService) UpdateAdmin(ctx context.Context, id int, req UpdateAdminRequest) (*AdminDetail, error) {
	existing, err := s.repo.FindAdminByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, httpx.NotFound("ADMIN_TIDAK_DITEMUKAN", "data admin tidak ditemukan")
	}

	if req.NIP != nil && *req.NIP != "" && (existing.NIP == nil || *existing.NIP != *req.NIP) {
		if other, _ := s.repo.FindAdminByNIP(ctx, s.pool, *req.NIP); other != nil {
			return nil, httpx.Conflict("NIP_SUDAH_DIGUNAKAN", "NIP admin sudah terdaftar di sistem")
		}
	}

	if err := s.repo.UpdateAdmin(ctx, s.pool, id, req); err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindAdminByID(ctx, s.pool, id)
}

// --- Pembimbing Lapangan ---

func (s *userService) CreatePembimbingLapangan(ctx context.Context, req CreatePembimbingLapanganRequest) (*PembimbingLapanganDetail, error) {
	if existing, _ := s.repo.FindUserByUsername(ctx, s.pool, req.Username); existing != nil {
		return nil, httpx.Conflict("USERNAME_SUDAH_DIGUNAKAN", "username sudah digunakan oleh pengguna lain")
	}
	if existing, _ := s.repo.FindUserByEmail(ctx, s.pool, req.Email); existing != nil {
		return nil, httpx.Conflict("EMAIL_SUDAH_DIGUNAKAN", "email sudah digunakan oleh pengguna lain")
	}

	// Periksa tempat program dan validasi kesesuaian jenis pembimbing lapangan (asumsi A8)
	exists, jenisProgram, err := s.repo.CheckTempatProgram(ctx, s.pool, req.TempatProgramID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.Unprocessable("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}

	if err := tempat.ValidatePembimbingMatch(req.Jenis, jenisProgram); err != nil {
		return nil, httpx.Unprocessable("JENIS_PEMBIMBING_TIDAK_COCOK", err.Error())
	}

	passHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var plID int
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		userID, err := s.repo.CreateUser(ctx, tx, req.Username, req.Email, passHash)
		if err != nil {
			return err
		}

		if err := s.repo.AssignRole(ctx, tx, userID, "pembimbing_lapangan"); err != nil {
			return err
		}

		// id_user WAJIB terisi lewat endpoint ini (asumsi A8)
		pl := &PembimbingLapanganProfile{
			Jenis:           req.Jenis,
			TempatProgramID: req.TempatProgramID,
			UserID:          &userID,
			Nama:            req.Nama,
			Jabatan:         req.Jabatan,
			Kontak:          req.Kontak,
		}
		if err := s.repo.CreatePembimbingLapangan(ctx, tx, pl); err != nil {
			return err
		}
		plID = pl.ID
		return nil
	})

	if err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindPembimbingLapanganByID(ctx, s.pool, plID)
}

func (s *userService) GetPembimbingLapanganByID(ctx context.Context, id int) (*PembimbingLapanganDetail, error) {
	pl, err := s.repo.FindPembimbingLapanganByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if pl == nil {
		return nil, httpx.NotFound("PEMBIMBING_TIDAK_DITEMUKAN", "data pembimbing lapangan tidak ditemukan")
	}
	return pl, nil
}

func (s *userService) ListPembimbingLapangan(ctx context.Context, p httpx.Pagination) ([]PembimbingLapanganDetail, httpx.PaginationMeta, error) {
	items, total, err := s.repo.ListPembimbingLapangan(ctx, s.pool, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	return items, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *userService) UpdatePembimbingLapangan(ctx context.Context, id int, req UpdatePembimbingLapanganRequest) (*PembimbingLapanganDetail, error) {
	existing, err := s.repo.FindPembimbingLapanganByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, httpx.NotFound("PEMBIMBING_TIDAK_DITEMUKAN", "data pembimbing lapangan tidak ditemukan")
	}

	// Validasi tempat program dan kesesuaian jenis
	exists, jenisProgram, err := s.repo.CheckTempatProgram(ctx, s.pool, req.TempatProgramID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.Unprocessable("TEMPAT_TIDAK_DITEMUKAN", "tempat program tidak ditemukan")
	}

	if err := tempat.ValidatePembimbingMatch(req.Jenis, jenisProgram); err != nil {
		return nil, httpx.Unprocessable("JENIS_PEMBIMBING_TIDAK_COCOK", err.Error())
	}

	if err := s.repo.UpdatePembimbingLapangan(ctx, s.pool, id, req); err != nil {
		return nil, database.MapError(err)
	}

	return s.repo.FindPembimbingLapanganByID(ctx, s.pool, id)
}

// --- User Manajemen Umum ---

func (s *userService) ListUsers(ctx context.Context, filter UserFilter, p httpx.Pagination) ([]UserListItem, httpx.PaginationMeta, error) {
	items, total, err := s.repo.ListUsers(ctx, s.pool, filter, p)
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}
	return items, httpx.PaginationMeta{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *userService) GetUserByID(ctx context.Context, id int) (*UserRecord, []string, error) {
	u, err := s.repo.FindUserByID(ctx, s.pool, id)
	if err != nil {
		return nil, nil, err
	}
	if u == nil {
		return nil, nil, httpx.NotFound("USER_TIDAK_DITEMUKAN", "pengguna tidak ditemukan")
	}

	roles, err := s.repo.GetUserRoles(ctx, s.pool, id)
	if err != nil {
		return nil, nil, err
	}

	return u, roles, nil
}

func (s *userService) SetUserStatus(ctx context.Context, id int, isActive bool) error {
	existing, err := s.repo.FindUserByID(ctx, s.pool, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return httpx.NotFound("USER_TIDAK_DITEMUKAN", "pengguna tidak ditemukan")
	}

	return s.repo.SetUserStatus(ctx, s.pool, id, isActive)
}

func (s *userService) ResetPassword(ctx context.Context, id int, req ResetPasswordRequest) error {
	existing, err := s.repo.FindUserByID(ctx, s.pool, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return httpx.NotFound("USER_TIDAK_DITEMUKAN", "pengguna tidak ditemukan")
	}

	if len(req.PasswordBaru) < 8 {
		return httpx.BadRequest("PASSWORD_TERLALU_PENDEK", "password baru minimal 8 karakter")
	}

	newHash, err := security.HashPassword(req.PasswordBaru)
	if err != nil {
		return err
	}

	// Update password dan cabut seluruh refresh token aktif dalam satu transaksi
	return database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := s.repo.UpdatePassword(ctx, tx, id, newHash); err != nil {
			return err
		}
		return s.repo.DeleteTokensByUserID(ctx, tx, id)
	})
}
