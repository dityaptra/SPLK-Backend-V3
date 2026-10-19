package auth

// LoginRequest adalah DTO request untuk POST /auth/login.
// Mendukung identifikasi login menggunakan username maupun email.
type LoginRequest struct {
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password" binding:"required"`
}

// UserSummary adalah ringkasan informasi pengguna yang dikembalikan saat login.
type UserSummary struct {
	ID       int      `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// LoginResponse adalah DTO response untuk POST /auth/login.
type LoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int         `json:"expires_in"` // dalam detik
	User         UserSummary `json:"user"`
}

// RefreshRequest adalah DTO request untuk POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshResponse adalah DTO response untuk POST /auth/refresh.
type RefreshResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // dalam detik
}

// LogoutRequest adalah DTO request untuk POST /auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordRequest adalah DTO request untuk PUT /auth/password.
type ChangePasswordRequest struct {
	PasswordLama string `json:"password_lama" binding:"required"`
	PasswordBaru string `json:"password_baru" binding:"required,min=8"`
}

// UserProfiles menyimpan ID entitas profil aktor yang dimiliki oleh pengguna.
type UserProfiles struct {
	MahasiswaID          *int `json:"id_mahasiswa,omitempty"`
	DosenID              *int `json:"id_dosen,omitempty"`
	PegawaiID            *int `json:"id_pegawai,omitempty"`
	AdminID              *int `json:"id_admin,omitempty"`
	PembimbingLapanganID *int `json:"id_pembimbing_lapangan,omitempty"`
}

// MeResponse adalah DTO response untuk GET /auth/me.
type MeResponse struct {
	UserID   int          `json:"user_id"`
	Username string       `json:"username"`
	Email    string       `json:"email"`
	Roles    []string     `json:"roles"`
	Profiles UserProfiles `json:"profiles"`
}
