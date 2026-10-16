package logbook

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan rute endpoint logbook bimbingan pada router group Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/pendaftaran/:id/logbook", h.Create)
	rg.PUT("/pendaftaran/:id/logbook/:lid", h.Update)
	rg.GET("/pendaftaran/:id/logbook/:lid", h.GetByID)
	rg.GET("/pendaftaran/:id/logbook", h.List)
}
