// Package spd menyediakan registrasi rute Gin untuk modul Surat Perjalanan Dinas.
package spd

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan seluruh endpoint Surat Perjalanan Dinas ke router Gin.
// pegawaiOrAdminMiddleware: middleware otorisasi yang membatasi hak tulis hanya untuk admin dan pegawai.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pegawaiOrAdminMiddleware gin.HandlerFunc) {
	// Operasi tulis (Create, Update, Delete, UploadFile) dibatasi untuk pegawai & admin
	writeGroup := rg.Group("", pegawaiOrAdminMiddleware)
	{
		writeGroup.POST("/spd", h.Create)
		writeGroup.PUT("/spd/:id", h.Update)
		writeGroup.DELETE("/spd/:id", h.Delete)
		writeGroup.POST("/spd/:id/file", h.UploadFile)
	}

	// Operasi baca dan unduh (List, GetByID, DownloadFile) terbuka untuk seluruh aktor terautentikasi dengan filter otorisasi
	rg.GET("/spd", h.List)
	rg.GET("/spd/:id", h.GetByID)
	rg.GET("/spd/:id/download", h.DownloadFile)
}
