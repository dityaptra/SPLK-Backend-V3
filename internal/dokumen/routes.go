package dokumen

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan rute-rute endpoint dokumen pada router group Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	// Rute dokumen pendaftaran peserta
	rg.POST("/pendaftaran/:id/dokumen", h.UploadPeserta)
	rg.GET("/pendaftaran/:id/dokumen", h.ListPeserta)

	// Rute dokumen kelompok KKN
	rg.POST("/kkn/kelompok/:id/dokumen", h.UploadKelompok)
	rg.GET("/kkn/kelompok/:id/dokumen", h.ListKelompok)

	// Rute unduh dan hapus dokumen
	rg.GET("/dokumen/:id/download", h.Download)
	rg.DELETE("/dokumen/:id", h.Delete)
}
