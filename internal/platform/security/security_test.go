package security_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/platform/security"
)

const testSecret = "kunci_rahasia_testing_yang_panjang_dan_aman_32_karakter"

func TestSecurity(t *testing.T) {
	t.Run("Bcrypt Password Hashing", func(t *testing.T) {
		password := "AdminPassword123"
		hash, err := security.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)

		assert.True(t, security.CheckPassword(password, hash))
		assert.False(t, security.CheckPassword("SalahPassword", hash))
	})

	t.Run("JWT Valid", func(t *testing.T) {
		userID := 42
		roles := []string{"mahasiswa", "admin"}
		ttl := 15 * time.Minute

		tokenStr, err := security.GenerateToken(userID, roles, ttl, testSecret)
		require.NoError(t, err)
		assert.NotEmpty(t, tokenStr)

		claims, err := security.ParseToken(tokenStr, testSecret)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.Sub)
		assert.Equal(t, roles, claims.Roles)
	})

	t.Run("JWT Kedaluwarsa", func(t *testing.T) {
		userID := 10
		roles := []string{"dosen"}
		ttl := -5 * time.Minute // sudah kedaluwarsa

		tokenStr, err := security.GenerateToken(userID, roles, ttl, testSecret)
		require.NoError(t, err)

		_, err = security.ParseToken(tokenStr, testSecret)
		assert.Error(t, err)
	})

	t.Run("JWT Tanda Tangan Salah", func(t *testing.T) {
		userID := 10
		roles := []string{"dosen"}
		ttl := 15 * time.Minute

		tokenStr, err := security.GenerateToken(userID, roles, ttl, testSecret)
		require.NoError(t, err)

		wrongSecret := "kunci_rahasia_lain_yang_berbeda_dan_cukup_panjang_32"
		_, err = security.ParseToken(tokenStr, wrongSecret)
		assert.Error(t, err)
	})

	t.Run("JWT Menolak Algoritma None", func(t *testing.T) {
		// Buat token tanpa tanda tangan (alg: none)
		unsignedToken := jwt.NewWithClaims(jwt.SigningMethodNone, security.Claims{
			Sub:   99,
			Roles: []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			},
		})
		noneTokenStr, err := unsignedToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = security.ParseToken(noneTokenStr, testSecret)
		assert.Error(t, err, "harus menolak algoritma none")
	})

	t.Run("HashToken Deterministik", func(t *testing.T) {
		tokenA := "random_refresh_token_string_12345"
		hash1 := security.HashToken(tokenA)
		hash2 := security.HashToken(tokenA)

		assert.Equal(t, hash1, hash2, "hash harus deterministik untuk token yang sama")
		assert.Len(t, hash1, 64, "SHA-256 hex string harus sepanjang 64 karakter")

		tokenB := "random_refresh_token_string_67890"
		hashB := security.HashToken(tokenB)
		assert.NotEqual(t, hash1, hashB, "token berbeda harus menghasilkan hash berbeda")
	})

	t.Run("GenerateRefreshToken", func(t *testing.T) {
		token1, err := security.GenerateRefreshToken()
		require.NoError(t, err)
		assert.NotEmpty(t, token1)

		token2, err := security.GenerateRefreshToken()
		require.NoError(t, err)
		assert.NotEqual(t, token1, token2, "refresh token berurutan harus unik")
	})
}
