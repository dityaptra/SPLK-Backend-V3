// Package auth menangani autentikasi pengguna, manajemen token JWT, rotasi refresh token,
// perubahan password, profil aktor, dan audit log aktivitas.
package auth

import "time"

// User mewakili baris dalam tabel users.
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Role mewakili baris dalam tabel roles.
type Role struct {
	ID        int     `json:"id"`
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi,omitempty"`
}

// Token mewakili baris dalam tabel tokens (refresh token).
type Token struct {
	ID        int       `json:"id"`
	UserID    int       `json:"id_user"`
	TokenHash string    `json:"token_hash"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
}

// LogAktivitas mewakili baris audit log dalam tabel log_aktivitas.
type LogAktivitas struct {
	ID           int       `json:"id"`
	UserID       int       `json:"id_user"`
	NamaUser     string    `json:"nama_user"`
	IPUser       *string   `json:"ip_user,omitempty"`
	AgentBrowser *string   `json:"agent_browser,omitempty"`
	Action       string    `json:"action"`
	Status       bool      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
