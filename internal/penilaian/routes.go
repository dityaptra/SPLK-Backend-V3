package penilaian

import (
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes mendaftarkan rute penilaian khusus administrator.
func RegisterAdminRoutes(adminGroup *gin.RouterGroup, h *Handler) {
	adminGroup.POST("/instrumen-penilaian", h.CreateInstrumen)
	adminGroup.GET("/instrumen-penilaian", h.ListInstrumen)
	adminGroup.GET("/instrumen-penilaian/ringkasan", h.GetRingkasanBobot)
	adminGroup.GET("/instrumen-penilaian/:id", h.GetInstrumen)
	adminGroup.PUT("/instrumen-penilaian/:id", h.UpdateInstrumen)
	adminGroup.DELETE("/instrumen-penilaian/:id", h.DeleteInstrumen)

	adminGroup.POST("/pendaftaran/:id/hitung-nilai", h.HitungNilaiManual)
}

// RegisterRoutes mendaftarkan rute penilaian pada router group umum terautentikasi.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("/pendaftaran/:id/form-penilaian", h.GetFormPenilaian)
	rg.POST("/pendaftaran/:id/penilaian", h.SubmitPenilaian)
	rg.GET("/pendaftaran/:id/nilai", h.GetNilai)

	// Verifikasi Kelulusan Penguji (Magang - Gambar 4)
	rg.GET("/pendaftaran/:id/verifikasi-penguji", h.GetVerifikasiPenguji)
	rg.POST("/pendaftaran/:id/verifikasi-penguji", h.SubmitVerifikasiPenguji)
}

