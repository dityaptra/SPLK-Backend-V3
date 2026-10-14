package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/platform/httpx"
	"splk-backend/internal/platform/security"
)

const ContextKeyActor = "actor"

// Actor merepresentasikan identitas pengguna yang terautentikasi pada konteks request.
type Actor struct {
	UserID int
	Roles  []string
}

// HasRole memeriksa apakah Actor memiliki salah satu dari role yang diberikan.
func (a Actor) HasRole(roles ...string) bool {
	for _, userRole := range a.Roles {
		for _, required := range roles {
			if userRole == required {
				return true
			}
		}
	}
	return false
}

// ActorFrom mengambil data Actor dari context Gin.
func ActorFrom(c *gin.Context) (Actor, bool) {
	val, exists := c.Get(ContextKeyActor)
	if !exists {
		return Actor{}, false
	}
	actor, ok := val.(Actor)
	return actor, ok
}

// Auth adalah middleware yang memvalidasi JWT Bearer token dari header Authorization.
// Jika token valid, Actor disimpan ke dalam context Gin.
// Jika tidak valid atau tidak ada, mengembalikan response HTTP 401 terstandar.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Abort()
			httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "header Authorization diperlukan"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Abort()
			httpx.WriteError(c, httpx.Unauthorized("TOKEN_FORMAT_INVALID", "format token harus 'Bearer <token>'"))
			return
		}

		tokenStr := parts[1]
		claims, err := security.ParseToken(tokenStr, jwtSecret)
		if err != nil {
			c.Abort()
			httpx.WriteError(c, httpx.Unauthorized("TOKEN_TIDAK_VALID", "token akses tidak valid atau telah kedaluwarsa"))
			return
		}

		actor := Actor{
			UserID: claims.Sub,
			Roles:  claims.Roles,
		}

		c.Set(ContextKeyActor, actor)
		c.Next()
	}
}

// RequireRoles adalah middleware otorisasi berbasis role (RBAC).
// Memeriksa apakah Actor memiliki minimal salah satu role yang diizinkan.
// Jika tidak, mengembalikan response HTTP 403 Forbidden terstandar.
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok {
			c.Abort()
			httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
			return
		}

		if !actor.HasRole(roles...) {
			c.Abort()
			httpx.WriteError(c, httpx.Forbidden("AKSES_DITOLAK", "anda tidak memiliki hak akses yang diperlukan"))
			return
		}

		c.Next()
	}
}
