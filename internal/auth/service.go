package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/config"
	"splk-backend/internal/database"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/platform/security"
	"splk-backend/internal/user"
)

// Service mendefinisikan seluruh use-case aturan bisnis autentikasi.
type Service interface {
	Login(ctx context.Context, req LoginRequest, ip, userAgent string) (*LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, userID int, refreshToken string, ip, userAgent string) error
	GetMe(ctx context.Context, userID int) (*MeResponse, error)
	ChangePassword(ctx context.Context, userID int, req ChangePasswordRequest) error
}

type authService struct {
	repo     Repository
	pool     *pgxpool.Pool
	cfg      *config.Config
	clock    clock.Clock
	profiles user.ProfileResolver
}

// NewService membuat instance Service autentikasi baru.
func NewService(repo Repository, pool *pgxpool.Pool, cfg *config.Config, clk clock.Clock, profiles user.ProfileResolver) Service {
	return &authService{
		repo:     repo,
		pool:     pool,
		cfg:      cfg,
		clock:    clk,
		profiles: profiles,
	}
}

// Login memproses autentikasi pengguna dengan username dan password.
// Menerapkan pesan error identik untuk user tidak ditemukan, password salah, atau user non-aktif.
func (s *authService) Login(ctx context.Context, req LoginRequest, ip, userAgent string) (*LoginResponse, error) {
	// Ambil akun berdasarkan username
	u, err := s.repo.FindUserByUsername(ctx, s.pool, req.Username)
	if err != nil {
		return nil, fmt.Errorf("cari user login: %w", err)
	}

	// 1. Username tidak ditemukan
	if u == nil {
		// RULES.md bagian 7 nomor 3: login gagal untuk username tak dikenal tidak masuk log_aktivitas,
		// cukup dicatat ke slog tanpa menyertakan password.
		slog.WarnContext(ctx, "login gagal: username tidak dikenal", "username", req.Username, "ip", ip)
		return nil, httpx.Unauthorized("KREDENSIAL_TIDAK_VALID", "username atau password salah")
	}

	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	var agentPtr *string
	if userAgent != "" {
		agentPtr = &userAgent
	}

	// 2. Akun non-aktif
	if !u.IsActive {
		_ = s.repo.CreateLogAktivitas(ctx, s.pool, &LogAktivitas{
			UserID:       u.ID,
			NamaUser:     u.Username,
			IPUser:       ipPtr,
			AgentBrowser: agentPtr,
			Action:       "login",
			Status:       false,
		})
		slog.WarnContext(ctx, "login ditolak: akun nonaktif", "user_id", u.ID, "username", u.Username)
		return nil, httpx.Unauthorized("KREDENSIAL_TIDAK_VALID", "username atau password salah")
	}

	// 3. Verifikasi password hash
	if !security.CheckPassword(req.Password, u.PasswordHash) {
		_ = s.repo.CreateLogAktivitas(ctx, s.pool, &LogAktivitas{
			UserID:       u.ID,
			NamaUser:     u.Username,
			IPUser:       ipPtr,
			AgentBrowser: agentPtr,
			Action:       "login",
			Status:       false,
		})
		slog.WarnContext(ctx, "login gagal: password salah", "user_id", u.ID, "username", u.Username)
		return nil, httpx.Unauthorized("KREDENSIAL_TIDAK_VALID", "username atau password salah")
	}

	// 4. Login berhasil
	roles, err := s.repo.GetUserRoles(ctx, s.pool, u.ID)
	if err != nil {
		return nil, fmt.Errorf("ambil roles saat login: %w", err)
	}

	_ = s.repo.CreateLogAktivitas(ctx, s.pool, &LogAktivitas{
		UserID:       u.ID,
		NamaUser:     u.Username,
		IPUser:       ipPtr,
		AgentBrowser: agentPtr,
		Action:       "login",
		Status:       true,
	})

	// Buat access token JWT HS256
	accessToken, err := security.GenerateToken(u.ID, roles, s.cfg.JWTAccessTTL, s.cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("buat access token: %w", err)
	}

	// Buat refresh token acak (32 byte base64url)
	rawRefreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("buat refresh token: %w", err)
	}

	// Simpan hash SHA-256 token ke database
	tokenHash := security.HashToken(rawRefreshToken)
	expiredAt := s.clock.Now().Add(s.cfg.JWTRefreshTTL)
	tokenRecord := &Token{
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiredAt: expiredAt,
	}

	if err := s.repo.CreateToken(ctx, s.pool, tokenRecord); err != nil {
		return nil, fmt.Errorf("simpan refresh token: %w", err)
	}

	return &LoginResponse{
		Token:        accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
		User: UserSummary{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Roles:    roles,
		},
	}, nil
}

// Refresh menukarkan refresh token lama dengan access token baru dan refresh token baru (rotasi).
func (s *authService) Refresh(ctx context.Context, rawRefreshToken string) (*RefreshResponse, error) {
	tokenHash := security.HashToken(rawRefreshToken)

	tokenRecord, err := s.repo.FindTokenByHash(ctx, s.pool, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("cari token refresh: %w", err)
	}
	if tokenRecord == nil {
		return nil, httpx.Unauthorized("TOKEN_TIDAK_VALID", "refresh token tidak valid")
	}

	// Periksa kedaluwarsa
	if tokenRecord.ExpiredAt.Before(s.clock.Now()) {
		_ = s.repo.DeleteTokenByHash(ctx, s.pool, tokenHash)
		return nil, httpx.Unauthorized("TOKEN_KEDALUWARSA", "refresh token telah kedaluwarsa")
	}

	var newAccessToken string
	var newRawRefreshToken string

	// Lakukan rotasi token dalam satu transaksi
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		// 1. Hapus token lama
		if err := s.repo.DeleteTokenByHash(ctx, tx, tokenHash); err != nil {
			return fmt.Errorf("hapus token lama: %w", err)
		}

		// 2. Muat user dan pastikan masih aktif
		u, err := s.repo.FindUserByID(ctx, tx, tokenRecord.UserID)
		if err != nil {
			return fmt.Errorf("ambil user saat refresh: %w", err)
		}
		if u == nil || !u.IsActive {
			return httpx.Unauthorized("AKUN_TIDAK_AKTIF", "akun tidak ditemukan atau telah dinonaktifkan")
		}

		roles, err := s.repo.GetUserRoles(ctx, tx, u.ID)
		if err != nil {
			return fmt.Errorf("ambil roles saat refresh: %w", err)
		}

		// 3. Generate token baru
		newAccessToken, err = security.GenerateToken(u.ID, roles, s.cfg.JWTAccessTTL, s.cfg.JWTSecret)
		if err != nil {
			return fmt.Errorf("buat access token baru: %w", err)
		}

		newRawRefreshToken, err = security.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("buat refresh token baru: %w", err)
		}

		// 4. Simpan hash token baru
		newTokenRecord := &Token{
			UserID:    u.ID,
			TokenHash: security.HashToken(newRawRefreshToken),
			ExpiredAt: s.clock.Now().Add(s.cfg.JWTRefreshTTL),
		}
		if err := s.repo.CreateToken(ctx, tx, newTokenRecord); err != nil {
			return fmt.Errorf("simpan token baru: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		Token:        newAccessToken,
		RefreshToken: newRawRefreshToken,
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

// Logout menghapus refresh token pengguna dan mencatat log audit.
func (s *authService) Logout(ctx context.Context, userID int, rawRefreshToken string, ip, userAgent string) error {
	tokenHash := security.HashToken(rawRefreshToken)

	// Hapus token dari database
	if err := s.repo.DeleteTokenByHash(ctx, s.pool, tokenHash); err != nil {
		return fmt.Errorf("hapus token saat logout: %w", err)
	}

	u, _ := s.repo.FindUserByID(ctx, s.pool, userID)
	userName := ""
	if u != nil {
		userName = u.Username
	}

	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	var agentPtr *string
	if userAgent != "" {
		agentPtr = &userAgent
	}

	_ = s.repo.CreateLogAktivitas(ctx, s.pool, &LogAktivitas{
		UserID:       userID,
		NamaUser:     userName,
		IPUser:       ipPtr,
		AgentBrowser: agentPtr,
		Action:       "logout",
		Status:       true,
	})

	return nil
}

// GetMe mengembalikan profil lengkap pengguna beserta daftar ID profil aktor yang dimiliki.
func (s *authService) GetMe(ctx context.Context, userID int) (*MeResponse, error) {
	u, err := s.repo.FindUserByID(ctx, s.pool, userID)
	if err != nil {
		return nil, fmt.Errorf("ambil user me: %w", err)
	}
	if u == nil {
		return nil, httpx.NotFound("USER_TIDAK_DITEMUKAN", "pengguna tidak ditemukan")
	}

	roles, err := s.repo.GetUserRoles(ctx, s.pool, u.ID)
	if err != nil {
		return nil, fmt.Errorf("ambil roles me: %w", err)
	}

	// Muat ID profil dari seluruh kemungkinan aktor
	var profiles UserProfiles

	if s.profiles != nil {
		if m, err := s.profiles.GetMahasiswaByUserID(ctx, s.pool, u.ID); err == nil && m != nil {
			profiles.MahasiswaID = &m.ID
		}
		if d, err := s.profiles.GetDosenByUserID(ctx, s.pool, u.ID); err == nil && d != nil {
			profiles.DosenID = &d.ID
		}
		if p, err := s.profiles.GetPegawaiByUserID(ctx, s.pool, u.ID); err == nil && p != nil {
			profiles.PegawaiID = &p.ID
		}
		if a, err := s.profiles.GetAdminByUserID(ctx, s.pool, u.ID); err == nil && a != nil {
			profiles.AdminID = &a.ID
		}
		if pl, err := s.profiles.GetPembimbingLapanganByUserID(ctx, s.pool, u.ID); err == nil && pl != nil {
			profiles.PembimbingLapanganID = &pl.ID
		}
	}

	return &MeResponse{
		UserID:   u.ID,
		Username: u.Username,
		Email:    u.Email,
		Roles:    roles,
		Profiles: profiles,
	}, nil
}

// ChangePassword mengganti password pengguna dan menghapus seluruh refresh token miliknya.
func (s *authService) ChangePassword(ctx context.Context, userID int, req ChangePasswordRequest) error {
	if len(req.PasswordBaru) < 8 {
		return httpx.BadRequest("PASSWORD_TERLALU_PENDEK", "password baru minimal 8 karakter")
	}

	u, err := s.repo.FindUserByID(ctx, s.pool, userID)
	if err != nil {
		return fmt.Errorf("ambil user change password: %w", err)
	}
	if u == nil {
		return httpx.NotFound("USER_TIDAK_DITEMUKAN", "pengguna tidak ditemukan")
	}

	// Cek kecocokan password lama
	if !security.CheckPassword(req.PasswordLama, u.PasswordHash) {
		return httpx.BadRequest("PASSWORD_LAMA_SALAH", "password lama tidak sesuai")
	}

	// Hash password baru dengan bcrypt cost 12
	newHash, err := security.HashPassword(req.PasswordBaru)
	if err != nil {
		return fmt.Errorf("hash password baru: %w", err)
	}

	// Perbarui password dan hapus seluruh token aktif dalam satu transaksi
	return database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := s.repo.UpdatePassword(ctx, tx, userID, newHash); err != nil {
			return fmt.Errorf("update password hash: %w", err)
		}

		if err := s.repo.DeleteTokensByUserID(ctx, tx, userID); err != nil {
			return fmt.Errorf("hapus tokens: %w", err)
		}

		return nil
	})
}
