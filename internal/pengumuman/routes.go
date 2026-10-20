// Package pengumuman menyediakan registrasi rute Gin untuk modul pengumuman.
package pengumuman

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan seluruh endpoint pengumuman ke router Gin.
// adminGroup: grup router yang diproteksi oleh RequireRoles("admin").
// authGroup: grup router untuk pengguna yang terautentikasi (login).
func RegisterRoutes(adminGroup *gin.RouterGroup, authGroup *gin.RouterGroup, h *Handler) {
	// Endpoint tulis dan manajemen oleh admin
	if adminGroup != nil {
		adminGroup.POST("/pengumuman", h.Create)
		adminGroup.PUT("/pengumuman/:id", h.Update)
		adminGroup.DELETE("/pengumuman/:id", h.Delete)
		adminGroup.GET("/pengumuman", h.ListAdmin)
		adminGroup.GET("/pengumuman/:id", h.GetByID)
	}

	// Endpoint baca untuk seluruh pengguna terautentikasi
	if authGroup != nil {
		authGroup.GET("/pengumuman", h.ListUser)
		authGroup.GET("/pengumuman/:id", h.GetByID)
	}
}
