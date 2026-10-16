package kkn

import "github.com/gin-gonic/gin"

// RegisterRoutes mendaftarkan rute-rute KKN pada router group Gin.
// adminGroup: rute yang diproteksi hak akses admin.
// authGroup: rute terautentikasi umum (mahasiswa, dosen, pembimbing lapangan, pegawai).
func RegisterRoutes(adminGroup *gin.RouterGroup, authGroup *gin.RouterGroup, h *Handler) {
	if adminGroup != nil {
		kknAdmin := adminGroup.Group("/kkn/kelompok")
		{
			kknAdmin.POST("", h.CreateKelompok)
			kknAdmin.GET("", h.ListKelompok)
			kknAdmin.GET("/:id", h.GetKelompok)
			kknAdmin.POST("/:id/anggota", h.AddAnggota)
			kknAdmin.DELETE("/:id/anggota/:pid", h.RemoveAnggota)
			kknAdmin.PUT("/:id/dosen-pembimbing", h.SetDosenPembimbing)
			kknAdmin.PUT("/:id/pembimbing-lapangan", h.SetPembimbingLapangan)
		}
	}

	if authGroup != nil {
		authGroup.GET("/kkn/kelompok/:id", h.GetKelompok)
	}
}
