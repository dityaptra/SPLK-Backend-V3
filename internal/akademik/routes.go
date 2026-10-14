package akademik

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan endpoint akademik untuk admin dan user terautentikasi.
func RegisterRoutes(adminGroup *gin.RouterGroup, authGroup *gin.RouterGroup, h *Handler) {
	// Endpoint Admin (RequireRoles("admin"))
	if adminGroup != nil {
		// Fakultas
		adminGroup.POST("/fakultas", h.CreateFakultas)
		adminGroup.GET("/fakultas", h.ListFakultas)
		adminGroup.GET("/fakultas/:id", h.GetFakultasByID)
		adminGroup.PUT("/fakultas/:id", h.UpdateFakultas)
		adminGroup.DELETE("/fakultas/:id", h.DeleteFakultas)

		// Jurusan
		adminGroup.POST("/jurusan", h.CreateJurusan)
		adminGroup.GET("/jurusan", h.ListJurusan)
		adminGroup.GET("/jurusan/:id", h.GetJurusanByID)
		adminGroup.PUT("/jurusan/:id", h.UpdateJurusan)
		adminGroup.DELETE("/jurusan/:id", h.DeleteJurusan)

		// Program Studi
		adminGroup.POST("/program-studi", h.CreateProdi)
		adminGroup.GET("/program-studi", h.ListProdi)
		adminGroup.GET("/program-studi/:id", h.GetProdiByID)
		adminGroup.PUT("/program-studi/:id", h.UpdateProdi)
		adminGroup.DELETE("/program-studi/:id", h.DeleteProdi)
	}

	// Endpoint Baca untuk User Terautentikasi
	if authGroup != nil {
		authGroup.GET("/fakultas", h.ListFakultas)
		authGroup.GET("/fakultas/:id", h.GetFakultasByID)

		authGroup.GET("/jurusan", h.ListJurusan)
		authGroup.GET("/jurusan/:id", h.GetJurusanByID)

		authGroup.GET("/program-studi", h.ListProdi)
		authGroup.GET("/program-studi/:id", h.GetProdiByID)
	}
}
