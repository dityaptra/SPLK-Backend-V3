package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/auth"
	"splk-backend/internal/config"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/platform/security"
	"splk-backend/internal/user"
)

// fakeRepository adalah implementasi tiruan auth.Repository untuk unit test tanpa database nyata.
type fakeRepository struct {
	usersByUsername map[string]*auth.User
	usersByID       map[int]*auth.User
	rolesByUserID   map[int][]string
	tokensByHash    map[string]*auth.Token
	logs            []*auth.LogAktivitas
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		usersByUsername: make(map[string]*auth.User),
		usersByID:       make(map[int]*auth.User),
		rolesByUserID:   make(map[int][]string),
		tokensByHash:    make(map[string]*auth.Token),
		logs:            make([]*auth.LogAktivitas, 0),
	}
}

func (f *fakeRepository) FindUserByUsername(ctx context.Context, db database.DBTX, username string) (*auth.User, error) {
	if u, ok := f.usersByUsername[username]; ok {
		return u, nil
	}
	return nil, nil
}

func (f *fakeRepository) FindUserByID(ctx context.Context, db database.DBTX, id int) (*auth.User, error) {
	if u, ok := f.usersByID[id]; ok {
		return u, nil
	}
	return nil, nil
}

func (f *fakeRepository) GetUserRoles(ctx context.Context, db database.DBTX, userID int) ([]string, error) {
	if roles, ok := f.rolesByUserID[userID]; ok {
		return roles, nil
	}
	return []string{}, nil
}

func (f *fakeRepository) CreateToken(ctx context.Context, db database.DBTX, token *auth.Token) error {
	f.tokensByHash[token.TokenHash] = token
	return nil
}

func (f *fakeRepository) FindTokenByHash(ctx context.Context, db database.DBTX, tokenHash string) (*auth.Token, error) {
	if t, ok := f.tokensByHash[tokenHash]; ok {
		return t, nil
	}
	return nil, nil
}

func (f *fakeRepository) DeleteTokenByHash(ctx context.Context, db database.DBTX, tokenHash string) error {
	delete(f.tokensByHash, tokenHash)
	return nil
}

func (f *fakeRepository) DeleteTokensByUserID(ctx context.Context, db database.DBTX, userID int) error {
	for k, v := range f.tokensByHash {
		if v.UserID == userID {
			delete(f.tokensByHash, k)
		}
	}
	return nil
}

func (f *fakeRepository) UpdatePassword(ctx context.Context, db database.DBTX, userID int, passwordHash string) error {
	if u, ok := f.usersByID[userID]; ok {
		u.PasswordHash = passwordHash
	}
	return nil
}

func (f *fakeRepository) CreateLogAktivitas(ctx context.Context, db database.DBTX, log *auth.LogAktivitas) error {
	f.logs = append(f.logs, log)
	return nil
}

// fakeProfileResolver untuk pengujian GetMe
type fakeProfileResolver struct{}

func (f *fakeProfileResolver) GetMahasiswaByUserID(ctx context.Context, db database.DBTX, userID int) (*user.MahasiswaProfile, error) {
	if userID == 1 {
		return &user.MahasiswaProfile{ID: 101, UserID: userID, NIM: "2015051001", Nama: "Mahasiswa Test"}, nil
	}
	return nil, nil
}

func (f *fakeProfileResolver) GetDosenByUserID(ctx context.Context, db database.DBTX, userID int) (*user.DosenProfile, error) {
	return nil, nil
}

func (f *fakeProfileResolver) GetPegawaiByUserID(ctx context.Context, db database.DBTX, userID int) (*user.PegawaiProfile, error) {
	return nil, nil
}

func (f *fakeProfileResolver) GetAdminByUserID(ctx context.Context, db database.DBTX, userID int) (*user.AdminProfile, error) {
	return nil, nil
}

func (f *fakeProfileResolver) GetPembimbingLapanganByUserID(ctx context.Context, db database.DBTX, userID int) (*user.PembimbingLapanganProfile, error) {
	return nil, nil
}

func setupTestService() (auth.Service, *fakeRepository, *clock.MockClock, *config.Config) {
	repo := newFakeRepository()
	clk := clock.NewMockClock(time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC), time.UTC)
	cfg := &config.Config{
		JWTSecret:     "secret_jwt_sangat_panjang_dan_aman_32_karakter",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	// Service tanpa pool database nyata (nil), operasi non-tx akan menggunakan fake repo
	svc := auth.NewService(repo, nil, cfg, clk, &fakeProfileResolver{})
	return svc, repo, clk, cfg
}

func TestAuthService(t *testing.T) {
	ctx := context.Background()

	t.Run("Login Sukses, Password Salah, User Nonaktif, Pesan Error Identik", func(t *testing.T) {
		svc, repo, _, _ := setupTestService()

		// Siapkan user aktif
		activePass := "Password123"
		activeHash, err := security.HashPassword(activePass)
		require.NoError(t, err)

		userActive := &auth.User{
			ID:           1,
			Username:     "aktif_user",
			Email:        "aktif@undiksha.ac.id",
			PasswordHash: activeHash,
			IsActive:     true,
		}
		repo.usersByUsername[userActive.Username] = userActive
		repo.usersByID[userActive.ID] = userActive
		repo.rolesByUserID[userActive.ID] = []string{"mahasiswa"}

		// Siapkan user non-aktif
		inactivePass := "Password123"
		inactiveHash, err := security.HashPassword(inactivePass)
		require.NoError(t, err)

		userInactive := &auth.User{
			ID:           2,
			Username:     "nonaktif_user",
			Email:        "nonaktif@undiksha.ac.id",
			PasswordHash: inactiveHash,
			IsActive:     false,
		}
		repo.usersByUsername[userInactive.Username] = userInactive
		repo.usersByID[userInactive.ID] = userInactive
		repo.rolesByUserID[userInactive.ID] = []string{"mahasiswa"}

		// 1. Kasus Login Sukses
		resp, err := svc.Login(ctx, auth.LoginRequest{
			Username: "aktif_user",
			Password: activePass,
		}, "127.0.0.1", "curl/7.68.0")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "aktif_user", resp.User.Username)
		assert.Equal(t, []string{"mahasiswa"}, resp.User.Roles)

		// Verifikasi refresh token di hash pada repository
		tokenHash := security.HashToken(resp.RefreshToken)
		assert.Contains(t, repo.tokensByHash, tokenHash, "token di repository harus berupa hash")

		// 2. Kasus Password Salah
		_, errWrongPass := svc.Login(ctx, auth.LoginRequest{
			Username: "aktif_user",
			Password: "WrongPassword456",
		}, "127.0.0.1", "curl/7.68.0")
		require.Error(t, errWrongPass)

		// 3. Kasus User Nonaktif
		_, errInactive := svc.Login(ctx, auth.LoginRequest{
			Username: "nonaktif_user",
			Password: inactivePass,
		}, "127.0.0.1", "curl/7.68.0")
		require.Error(t, errInactive)

		// 4. Kasus User Tidak Ditemukan
		_, errNotFound := svc.Login(ctx, auth.LoginRequest{
			Username: "tidak_ada_user",
			Password: "AnyPassword",
		}, "127.0.0.1", "curl/7.68.0")
		require.Error(t, errNotFound)

		// Verifikasi pesan error ketiganya IDENTIK sesuai kriteria penerimaan
		var appErrWrong, appErrInactive, appErrNotFound *httpx.AppError
		require.True(t, errors.As(errWrongPass, &appErrWrong))
		require.True(t, errors.As(errInactive, &appErrInactive))
		require.True(t, errors.As(errNotFound, &appErrNotFound))

		assert.Equal(t, appErrWrong.Message, appErrInactive.Message)
		assert.Equal(t, appErrWrong.Message, appErrNotFound.Message)
		assert.Equal(t, "username atau password salah", appErrWrong.Message)
		assert.Equal(t, 401, appErrWrong.HTTPStatus)
		assert.Equal(t, 401, appErrInactive.HTTPStatus)
		assert.Equal(t, 401, appErrNotFound.HTTPStatus)

		// Verifikasi audit log: username tak dikenal tidak masuk log_aktivitas
		for _, l := range repo.logs {
			assert.NotEqual(t, "tidak_ada_user", l.NamaUser, "user tidak dikenal tidak boleh masuk log_aktivitas")
		}
	})

	t.Run("Token Kedaluwarsa Ditolak", func(t *testing.T) {
		svc, repo, clk, _ := setupTestService()

		// Buat token yang sudah kedaluwarsa
		expiredTokenStr := "expired_raw_token_123"
		tokenHash := security.HashToken(expiredTokenStr)
		repo.tokensByHash[tokenHash] = &auth.Token{
			ID:        1,
			UserID:    1,
			TokenHash: tokenHash,
			ExpiredAt: clk.Now().Add(-1 * time.Hour), // 1 jam lalu
		}

		_, err := svc.Refresh(ctx, expiredTokenStr)
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "TOKEN_KEDALUWARSA", appErr.Code)

		// Token kedaluwarsa harus dihapus dari storage
		assert.NotContains(t, repo.tokensByHash, tokenHash)
	})

	t.Run("Refresh Token Tidak Dikenal Ditolak", func(t *testing.T) {
		svc, _, _, _ := setupTestService()

		_, err := svc.Refresh(ctx, "token_palsu_yang_tidak_pernah_dibuat")
		require.Error(t, err)
		var appErr *httpx.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "TOKEN_TIDAK_VALID", appErr.Code)
	})

	t.Run("Logout Menghapus Token", func(t *testing.T) {
		svc, repo, clk, _ := setupTestService()

		rawToken := "token_aktif_untuk_logout"
		tokenHash := security.HashToken(rawToken)
		repo.tokensByHash[tokenHash] = &auth.Token{
			ID:        2,
			UserID:    1,
			TokenHash: tokenHash,
			ExpiredAt: clk.Now().Add(24 * time.Hour),
		}

		err := svc.Logout(ctx, 1, rawToken, "127.0.0.1", "Postman")
		require.NoError(t, err)

		// Token harus terhapus
		assert.NotContains(t, repo.tokensByHash, tokenHash)
	})

	t.Run("GetMe Memuat Profil", func(t *testing.T) {
		svc, repo, _, _ := setupTestService()

		userMahasiswa := &auth.User{
			ID:       1,
			Username: "mahasiswa01",
			Email:    "mhs@undiksha.ac.id",
			IsActive: true,
		}
		repo.usersByID[1] = userMahasiswa
		repo.rolesByUserID[1] = []string{"mahasiswa"}

		me, err := svc.GetMe(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, me.UserID)
		assert.Equal(t, "mahasiswa01", me.Username)
		assert.NotNil(t, me.Profiles.MahasiswaID)
		assert.Equal(t, 101, *me.Profiles.MahasiswaID)
	})
}
