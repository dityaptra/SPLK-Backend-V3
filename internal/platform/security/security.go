// Package security menyediakan utilitas hashing password bcrypt, manajemen JWT HS256,
// dan pembuatan serta hashing refresh token.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost adalah cost hashing bcrypt sesuai RULES.md bagian 6.
	BcryptCost = 12
)

var (
	ErrInvalidToken = errors.New("token tidak valid atau telah kedaluwarsa")
	ErrInvalidAlg   = errors.New("algoritma signing token tidak diizinkan")
)

// HashPassword membuat hash bcrypt dengan cost 12 dari plain password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword membandingkan password plain dengan hash bcrypt.
// Mengembalikan true jika cocok, false jika tidak cocok.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Claims adalah payload klaim JWT akses SPLK.
type Claims struct {
	Sub   int      `json:"sub"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateToken membuat access token JWT baru dengan algoritma HS256.
func GenerateToken(userID int, roles []string, ttl time.Duration, secret string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Sub:   userID,
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return signed, nil
}

// ParseToken memvalidasi dan mem-parse JWT access token.
// Menolak secara tegas algoritma selain HS256 (termasuk 'none' atau RS256).
func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		// Validasi algoritma signing strictly HS256
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAlg, token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, ErrInvalidAlg) {
			return nil, ErrInvalidAlg
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateRefreshToken menghasilkan 32 byte acak yang di-encode base64url tanpa padding.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken menghasilkan nilai hash SHA-256 heksadesimal deterministik dari sebuah token string.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
