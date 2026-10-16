package jurnal

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan rute endpoint jurnal harian pada router group Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/pendaftaran/:id/jurnal", h.Create)
	rg.PUT("/pendaftaran/:id/jurnal/:jid", h.Update)
	rg.GET("/pendaftaran/:id/jurnal/:jid", h.GetByID)
	rg.GET("/pendaftaran/:id/jurnal", h.List)
}
