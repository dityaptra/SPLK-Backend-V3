// Package periode menyediakan registrasi rute Gin untuk modul periode program.
package periode

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan seluruh endpoint periode program ke router Gin.
// adminGroup: grup router yang diproteksi oleh RequireRoles("admin").
// authGroup: grup router untuk pengguna yang terautentikasi (login).
func RegisterRoutes(adminGroup *gin.RouterGroup, authGroup *gin.RouterGroup, h *Handler) {
	// Endpoint tulis dan kelola oleh admin
	if adminGroup != nil {
		adminGroup.POST("/periode", h.Create)
		adminGroup.PUT("/periode/:id", h.Update)
		adminGroup.POST("/periode/:id/aktifkan", h.Aktifkan)
		adminGroup.POST("/periode/:id/selesaikan", h.Selesaikan)
		adminGroup.GET("/periode", h.List)
		adminGroup.GET("/periode/:id", h.GetByID)
	}

	// Endpoint baca terbuka untuk seluruh aktor terautentikasi (mahasiswa, dosen, pegawai, admin)
	if authGroup != nil {
		authGroup.GET("/periode", h.List)
		authGroup.GET("/periode/:id", h.GetByID)
	}
}
