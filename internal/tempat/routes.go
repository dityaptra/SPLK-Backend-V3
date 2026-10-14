package tempat

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan seluruh endpoint tempat program.
// adminGroup berada di bawah otorisasi RequireRoles("admin").
// authGroup terbuka untuk semua user login.
func RegisterRoutes(adminGroup *gin.RouterGroup, authGroup *gin.RouterGroup, h *Handler) {
	// Endpoint kelola oleh admin
	if adminGroup != nil {
		adminGroup.POST("/tempat-program", h.Create)
		adminGroup.GET("/tempat-program", h.List)
		adminGroup.GET("/tempat-program/:id", h.GetByID)
		adminGroup.PUT("/tempat-program/:id", h.Update)
		adminGroup.DELETE("/tempat-program/:id", h.Delete)
	}

	// Endpoint baca terbuka untuk semua user terautentikasi
	if authGroup != nil {
		authGroup.GET("/tempat-program", h.List)
		authGroup.GET("/tempat-program/:id", h.GetByID)
		authGroup.GET("/tempat-program/:id/pembimbing-lapangan", h.ListPembimbing)
	}
}
