// Package pendaftaran menyediakan pendaftaran rute Gin untuk modul pendaftaran.
package pendaftaran

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan seluruh rute pendaftaran pada router Gin.
// authGroup adalah grup router yang sudah diproteksi oleh middleware autentikasi JWT.
func RegisterRoutes(authGroup *gin.RouterGroup, h *Handler) {
	if authGroup == nil {
		return
	}

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
