// Package pendaftaran menyediakan pendaftaran rute Gin untuk modul pendaftaran.
package pendaftaran

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan seluruh rute pendaftaran pada router Gin.
// adminGroup: grup router yang diproteksi oleh middleware admin (RequireRoles("admin")).
// authGroup: grup router yang diproteksi oleh middleware autentikasi JWT umum.
func RegisterRoutes(adminGroup *gin.RouterGroup, authGroup *gin.RouterGroup, h *Handler) {
	if adminGroup != nil {
		adminGroup.PUT("/pendaftaran/:id/penempatan", h.AdminPenempatan)
		adminGroup.PUT("/pendaftaran/:id/dosen-penguji", h.SetDosenPenguji)
	}

	if authGroup != nil {
		pendaftaranGroup := authGroup.Group("/pendaftaran")
		{
			pendaftaranGroup.POST("", h.Daftar)
			pendaftaranGroup.POST("/:id/ajukan-ulang", h.AjukanUlang)
			pendaftaranGroup.GET("", h.List)
			pendaftaranGroup.GET("/:id", h.GetByID)
			pendaftaranGroup.PUT("/:id/penempatan-magang", h.PenempatanMagang)
			pendaftaranGroup.PUT("/:id/dosen-pembimbing", h.SetDosenPembimbing)
		}
	}
}
