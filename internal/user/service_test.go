package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/user"
)

// mockUserRepo adalah implementasi tiruan user.Repository untuk unit testing logika validasi Service.
type mockUserRepo struct {
	usersByUsername map[string]*user.UserRecord
	usersByEmail    map[string]*user.UserRecord
	usersByID       map[int]*user.UserRecord

	mahasiswaByNIM map[string]*user.MahasiswaProfile
	mahasiswaByID  map[int]*user.MahasiswaDetail

	dosenByNIP map[string]*user.DosenProfile
	dosenByID  map[int]*user.DosenDetail

	pegawaiByNIP map[string]*user.PegawaiProfile
	pegawaiByID  map[int]*user.PegawaiDetail

	adminByNIP map[string]*user.AdminProfile
	adminByID  map[int]*user.AdminDetail

	pembimbingByID map[int]*user.PembimbingLapanganDetail

	existingProdiIDs map[int]bool
	tempatProgramMap map[int]string // tempatID -> jenisProgram
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		usersByUsername:  make(map[string]*user.UserRecord),
		usersByEmail:     make(map[string]*user.UserRecord),
		usersByID:        make(map[int]*user.UserRecord),
		mahasiswaByNIM:   make(map[string]*user.MahasiswaProfile),
		mahasiswaByID:    make(map[int]*user.MahasiswaDetail),
		dosenByNIP:       make(map[string]*user.DosenProfile),
		dosenByID:        make(map[int]*user.DosenDetail),
		pegawaiByNIP:     make(map[string]*user.PegawaiProfile),
		pegawaiByID:      make(map[int]*user.PegawaiDetail),
		adminByNIP:       make(map[string]*user.AdminProfile),
		adminByID:        make(map[int]*user.AdminDetail),
		pembimbingByID:   make(map[int]*user.PembimbingLapanganDetail),
		existingProdiIDs: make(map[int]bool),
		tempatProgramMap: make(map[int]string),
	}
}

func (m *mockUserRepo) CreateUser(ctx context.Context, db database.DBTX, username, email, passwordHash string) (int, error) {
	id := len(m.usersByID) + 1
	rec := &user.UserRecord{ID: id, Username: username, Email: email, PasswordHash: passwordHash, IsActive: true}
	m.usersByID[id] = rec
	m.usersByUsername[username] = rec
	m.usersByEmail[email] = rec
	return id, nil
}

func (m *mockUserRepo) FindUserByID(ctx context.Context, db database.DBTX, id int) (*user.UserRecord, error) {
	return m.usersByID[id], nil
}

func (m *mockUserRepo) FindUserByUsername(ctx context.Context, db database.DBTX, username string) (*user.UserRecord, error) {
	return m.usersByUsername[username], nil
}

func (m *mockUserRepo) FindUserByEmail(ctx context.Context, db database.DBTX, email string) (*user.UserRecord, error) {
	return m.usersByEmail[email], nil
}

func (m *mockUserRepo) SetUserStatus(ctx context.Context, db database.DBTX, id int, isActive bool) error {
	u, ok := m.usersByID[id]
	if !ok {
		return nil
	}
	u.IsActive = isActive
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, db database.DBTX, id int, passwordHash string) error {
	u, ok := m.usersByID[id]
	if !ok {
		return nil
	}
	u.PasswordHash = passwordHash
	return nil
}

func (m *mockUserRepo) ListUsers(ctx context.Context, db database.DBTX, filter user.UserFilter, p httpx.Pagination) ([]user.UserListItem, int64, error) {
	var res []user.UserListItem
	for _, u := range m.usersByID {
		res = append(res, user.UserListItem{ID: u.ID, Username: u.Username, Email: u.Email, IsActive: u.IsActive})
	}
	return res, int64(len(res)), nil
}

func (m *mockUserRepo) AssignRole(ctx context.Context, db database.DBTX, userID int, roleName string) error {
	return nil
}

func (m *mockUserRepo) GetUserRoles(ctx context.Context, db database.DBTX, userID int) ([]string, error) {
	return []string{"mahasiswa"}, nil
}

func (m *mockUserRepo) DeleteTokensByUserID(ctx context.Context, db database.DBTX, userID int) error {
	return nil
}

// Mahasiswa
func (m *mockUserRepo) CreateMahasiswa(ctx context.Context, db database.DBTX, mah *user.MahasiswaProfile) error {
	mah.ID = len(m.mahasiswaByID) + 1
	m.mahasiswaByID[mah.ID] = &user.MahasiswaDetail{
		ID: mah.ID, UserID: mah.UserID, NIM: mah.NIM, Nama: mah.Nama, ProdiID: mah.ProdiID,
	}
	m.mahasiswaByNIM[mah.NIM] = mah
	return nil
}

func (m *mockUserRepo) FindMahasiswaByID(ctx context.Context, db database.DBTX, id int) (*user.MahasiswaDetail, error) {
	return m.mahasiswaByID[id], nil
}

func (m *mockUserRepo) FindMahasiswaByNIM(ctx context.Context, db database.DBTX, nim string) (*user.MahasiswaProfile, error) {
	return m.mahasiswaByNIM[nim], nil
}

func (m *mockUserRepo) ListMahasiswa(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]user.MahasiswaDetail, int64, error) {
	var res []user.MahasiswaDetail
	for _, item := range m.mahasiswaByID {
		res = append(res, *item)
	}
	return res, int64(len(res)), nil
}

func (m *mockUserRepo) UpdateMahasiswa(ctx context.Context, db database.DBTX, id int, req user.UpdateMahasiswaRequest) error {
	if item, ok := m.mahasiswaByID[id]; ok {
		item.NIM = req.NIM
		item.Nama = req.Nama
		item.ProdiID = req.ProdiID
		item.Angkatan = req.Angkatan
	}
	return nil
}

// Dosen
func (m *mockUserRepo) CreateDosen(ctx context.Context, db database.DBTX, d *user.DosenProfile) error {
	d.ID = len(m.dosenByID) + 1
	m.dosenByID[d.ID] = &user.DosenDetail{ID: d.ID, UserID: d.UserID, NIP: d.NIP, Nama: d.Nama, ProdiID: d.ProdiID}
	m.dosenByNIP[d.NIP] = d
	return nil
}

func (m *mockUserRepo) FindDosenByID(ctx context.Context, db database.DBTX, id int) (*user.DosenDetail, error) {
	return m.dosenByID[id], nil
}

func (m *mockUserRepo) FindDosenByNIP(ctx context.Context, db database.DBTX, nip string) (*user.DosenProfile, error) {
	return m.dosenByNIP[nip], nil
}

func (m *mockUserRepo) ListDosen(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]user.DosenDetail, int64, error) {
	var res []user.DosenDetail
	for _, item := range m.dosenByID {
		res = append(res, *item)
	}
	return res, int64(len(res)), nil
}

func (m *mockUserRepo) UpdateDosen(ctx context.Context, db database.DBTX, id int, req user.UpdateDosenRequest) error {
	if item, ok := m.dosenByID[id]; ok {
		item.NIP = req.NIP
		item.Nama = req.Nama
		item.ProdiID = req.ProdiID
	}
	return nil
}

// Pegawai
func (m *mockUserRepo) CreatePegawai(ctx context.Context, db database.DBTX, peg *user.PegawaiProfile) error {
	peg.ID = len(m.pegawaiByID) + 1
	m.pegawaiByID[peg.ID] = &user.PegawaiDetail{ID: peg.ID, UserID: peg.UserID, NIP: peg.NIP, Nama: peg.Nama, Jabatan: peg.Jabatan}
	m.pegawaiByNIP[peg.NIP] = peg
	return nil
}

func (m *mockUserRepo) FindPegawaiByID(ctx context.Context, db database.DBTX, id int) (*user.PegawaiDetail, error) {
	return m.pegawaiByID[id], nil
}

func (m *mockUserRepo) FindPegawaiByNIP(ctx context.Context, db database.DBTX, nip string) (*user.PegawaiProfile, error) {
	return m.pegawaiByNIP[nip], nil
}

func (m *mockUserRepo) ListPegawai(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]user.PegawaiDetail, int64, error) {
	var res []user.PegawaiDetail
	for _, item := range m.pegawaiByID {
		res = append(res, *item)
	}
	return res, int64(len(res)), nil
}

func (m *mockUserRepo) UpdatePegawai(ctx context.Context, db database.DBTX, id int, req user.UpdatePegawaiRequest) error {
	if item, ok := m.pegawaiByID[id]; ok {
		item.NIP = req.NIP
		item.Nama = req.Nama
		item.Jabatan = req.Jabatan
	}
	return nil
}

// Admin
func (m *mockUserRepo) CreateAdmin(ctx context.Context, db database.DBTX, a *user.AdminProfile) error {
	a.ID = len(m.adminByID) + 1
	m.adminByID[a.ID] = &user.AdminDetail{ID: a.ID, UserID: a.UserID, NIP: a.NIP, Nama: a.Nama}
	if a.NIP != nil {
		m.adminByNIP[*a.NIP] = a
	}
	return nil
}

func (m *mockUserRepo) FindAdminByID(ctx context.Context, db database.DBTX, id int) (*user.AdminDetail, error) {
	return m.adminByID[id], nil
}

func (m *mockUserRepo) FindAdminByNIP(ctx context.Context, db database.DBTX, nip string) (*user.AdminProfile, error) {
	return m.adminByNIP[nip], nil
}

func (m *mockUserRepo) ListAdmin(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]user.AdminDetail, int64, error) {
	var res []user.AdminDetail
	for _, item := range m.adminByID {
		res = append(res, *item)
	}
	return res, int64(len(res)), nil
}

func (m *mockUserRepo) UpdateAdmin(ctx context.Context, db database.DBTX, id int, req user.UpdateAdminRequest) error {
	if item, ok := m.adminByID[id]; ok {
		item.NIP = req.NIP
		item.Nama = req.Nama
	}
	return nil
}

// Pembimbing Lapangan
func (m *mockUserRepo) CreatePembimbingLapangan(ctx context.Context, db database.DBTX, pl *user.PembimbingLapanganProfile) error {
	pl.ID = len(m.pembimbingByID) + 1
	m.pembimbingByID[pl.ID] = &user.PembimbingLapanganDetail{
		ID:              pl.ID,
		UserID:          pl.UserID,
		Jenis:           pl.Jenis,
		TempatProgramID: pl.TempatProgramID,
		Nama:            pl.Nama,
	}
	return nil
}

func (m *mockUserRepo) FindPembimbingLapanganByID(ctx context.Context, db database.DBTX, id int) (*user.PembimbingLapanganDetail, error) {
	return m.pembimbingByID[id], nil
}

func (m *mockUserRepo) ListPembimbingLapangan(ctx context.Context, db database.DBTX, p httpx.Pagination) ([]user.PembimbingLapanganDetail, int64, error) {
	var res []user.PembimbingLapanganDetail
	for _, item := range m.pembimbingByID {
		res = append(res, *item)
	}
	return res, int64(len(res)), nil
}

func (m *mockUserRepo) UpdatePembimbingLapangan(ctx context.Context, db database.DBTX, id int, req user.UpdatePembimbingLapanganRequest) error {
	if item, ok := m.pembimbingByID[id]; ok {
		item.Jenis = req.Jenis
		item.TempatProgramID = req.TempatProgramID
		item.Nama = req.Nama
	}
	return nil
}

func (m *mockUserRepo) CheckProdiExists(ctx context.Context, db database.DBTX, prodiID int) (bool, error) {
	return m.existingProdiIDs[prodiID], nil
}

func (m *mockUserRepo) CheckTempatProgram(ctx context.Context, db database.DBTX, tempatID int) (bool, string, error) {
	jp, ok := m.tempatProgramMap[tempatID]
	return ok, jp, nil
}

// --- Test Cases ---

func TestUserService_Validations(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	svc := user.NewService(repo, nil)

	// Isi data awal
	repo.existingProdiIDs[1] = true
	repo.tempatProgramMap[10] = "magang"
	repo.tempatProgramMap[20] = "asistensi_mengajar"
	repo.tempatProgramMap[30] = "kkn"

	repo.usersByUsername["existing_user"] = &user.UserRecord{ID: 1, Username: "existing_user", Email: "existing@undiksha.ac.id"}
	repo.usersByEmail["existing@undiksha.ac.id"] = &user.UserRecord{ID: 1, Username: "existing_user", Email: "existing@undiksha.ac.id"}
	repo.mahasiswaByNIM["2015051001"] = &user.MahasiswaProfile{ID: 1, UserID: 1, NIM: "2015051001"}
	repo.dosenByNIP["198501012010121001"] = &user.DosenProfile{ID: 1, UserID: 2, NIP: "198501012010121001"}
	repo.pegawaiByNIP["198801012015011001"] = &user.PegawaiProfile{ID: 1, UserID: 3, NIP: "198801012015011001"}
	adminNIP := "199001012020011001"
	repo.adminByNIP[adminNIP] = &user.AdminProfile{ID: 1, UserID: 4, NIP: &adminNIP}

	t.Run("CreateMahasiswa - Duplicate Username", func(t *testing.T) {
		req := user.CreateMahasiswaRequest{
			Username: "existing_user",
			Email:    "mhs_baru@undiksha.ac.id",
			Password: "password123",
			NIM:      "2015051099",
			Nama:     "Budi",
			ProdiID:  1,
			Angkatan: 2024,
		}
		_, err := svc.CreateMahasiswa(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Equal(t, "USERNAME_SUDAH_DIGUNAKAN", appErr.Code)
	})

	t.Run("CreateMahasiswa - Duplicate Email", func(t *testing.T) {
		req := user.CreateMahasiswaRequest{
			Username: "mhs_baru",
			Email:    "existing@undiksha.ac.id",
			Password: "password123",
			NIM:      "2015051099",
			Nama:     "Budi",
			ProdiID:  1,
			Angkatan: 2024,
		}
		_, err := svc.CreateMahasiswa(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Equal(t, "EMAIL_SUDAH_DIGUNAKAN", appErr.Code)
	})

	t.Run("CreateMahasiswa - Duplicate NIM", func(t *testing.T) {
		req := user.CreateMahasiswaRequest{
			Username: "mhs_unik",
			Email:    "mhs_unik@undiksha.ac.id",
			Password: "password123",
			NIM:      "2015051001",
			Nama:     "Budi",
			ProdiID:  1,
			Angkatan: 2024,
		}
		_, err := svc.CreateMahasiswa(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Equal(t, "NIM_SUDAH_DIGUNAKAN", appErr.Code)
	})

	t.Run("CreateMahasiswa - Prodi Not Found", func(t *testing.T) {
		req := user.CreateMahasiswaRequest{
			Username: "mhs_unik2",
			Email:    "mhs_unik2@undiksha.ac.id",
			Password: "password123",
			NIM:      "2015051098",
			Nama:     "Budi",
			ProdiID:  999, // tidak terdaftar
			Angkatan: 2024,
		}
		_, err := svc.CreateMahasiswa(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "PRODI_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("CreateDosen - Duplicate NIP", func(t *testing.T) {
		req := user.CreateDosenRequest{
			Username: "dosen_baru",
			Email:    "dosen_baru@undiksha.ac.id",
			Password: "password123",
			NIP:      "198501012010121001",
			Nama:     "Dr. Wayan",
			ProdiID:  1,
		}
		_, err := svc.CreateDosen(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 409, appErr.HTTPStatus)
		assert.Equal(t, "NIP_SUDAH_DIGUNAKAN", appErr.Code)
	})

	t.Run("CreatePembimbingLapangan - Tempat Program Tidak Ditemukan", func(t *testing.T) {
		req := user.CreatePembimbingLapanganRequest{
			Username:        "mentor_baru",
			Email:           "mentor@pt.com",
			Password:        "password123",
			Jenis:           "mentor",
			TempatProgramID: 9999, // tidak terdaftar
			Nama:            "Pak Made",
		}
		_, err := svc.CreatePembimbingLapangan(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "TEMPAT_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("CreatePembimbingLapangan - Mismatch Jenis Program (Mentor di KKN)", func(t *testing.T) {
		req := user.CreatePembimbingLapanganRequest{
			Username:        "mentor_kkn",
			Email:           "mentor_kkn@desa.id",
			Password:        "password123",
			Jenis:           "mentor",
			TempatProgramID: 30, // Tempat 30 adalah KKN
			Nama:            "Pak Made",
		}
		_, err := svc.CreatePembimbingLapangan(ctx, req)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 422, appErr.HTTPStatus)
		assert.Equal(t, "JENIS_PEMBIMBING_TIDAK_COCOK", appErr.Code)
	})

	t.Run("GetMahasiswaByID - Not Found", func(t *testing.T) {
		_, err := svc.GetMahasiswaByID(ctx, 9999)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("SetUserStatus - User Not Found", func(t *testing.T) {
		err := svc.SetUserStatus(ctx, 9999, false)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 404, appErr.HTTPStatus)
	})

	t.Run("ResetPassword - User Not Found", func(t *testing.T) {
		err := svc.ResetPassword(ctx, 9999, user.ResetPasswordRequest{PasswordBaru: "newpass123"})
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, 404, appErr.HTTPStatus)
	})
}
