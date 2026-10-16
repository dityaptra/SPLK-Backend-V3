package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan seluruh route untuk modul autentikasi pada grup router yang diberikan.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc, loginMiddlewares ...gin.HandlerFunc) {
	// Endpoint publik (dengan rate limiter opsional untuk login)
	loginHandlers := append(loginMiddlewares, h.Login)
	rg.POST("/login", loginHandlers...)
	rg.POST("/refresh", h.Refresh)

	// Endpoint terproteksi yang mewajibkan token JWT valid
	protected := rg.Group("", authMiddleware)
	{
		protected.POST("/logout", h.Logout)
		protected.GET("/me", h.Me)
		protected.PUT("/password", h.ChangePassword)
	}
}
