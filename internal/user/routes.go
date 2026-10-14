package user

import "github.com/gin-gonic/gin"

// RegisterAdminRoutes mendaftarkan seluruh rute manajemen akun dan pengguna pada grup admin.
// Seluruh rute ini berada di bawah otorisasi RequireRoles("admin").
func RegisterAdminRoutes(adminGroup *gin.RouterGroup, h *Handler) {
	// Mahasiswa
	adminGroup.POST("/mahasiswa", h.CreateMahasiswa)
	adminGroup.GET("/mahasiswa", h.ListMahasiswa)
	adminGroup.GET("/mahasiswa/:id", h.GetMahasiswaByID)
	adminGroup.PUT("/mahasiswa/:id", h.UpdateMahasiswa)

	// Dosen
	adminGroup.POST("/dosen", h.CreateDosen)
	adminGroup.GET("/dosen", h.ListDosen)
	adminGroup.GET("/dosen/:id", h.GetDosenByID)
	adminGroup.PUT("/dosen/:id", h.UpdateDosen)

	// Pegawai
	adminGroup.POST("/pegawai", h.CreatePegawai)
	adminGroup.GET("/pegawai", h.ListPegawai)
	adminGroup.GET("/pegawai/:id", h.GetPegawaiByID)
	adminGroup.PUT("/pegawai/:id", h.UpdatePegawai)

	// Admin
	adminGroup.POST("/admins", h.CreateAdmin)
	adminGroup.GET("/admins", h.ListAdmin)
	adminGroup.GET("/admins/:id", h.GetAdminByID)
	adminGroup.PUT("/admins/:id", h.UpdateAdmin)

	// Pembimbing Lapangan
	adminGroup.POST("/pembimbing-lapangan", h.CreatePembimbingLapangan)
	adminGroup.GET("/pembimbing-lapangan", h.ListPembimbingLapangan)
	adminGroup.GET("/pembimbing-lapangan/:id", h.GetPembimbingLapanganByID)
	adminGroup.PUT("/pembimbing-lapangan/:id", h.UpdatePembimbingLapangan)

	// Manajemen User Umum
	adminGroup.GET("/users", h.ListUsers)
	adminGroup.GET("/users/:id", h.GetUserByID)
	adminGroup.PATCH("/users/:id/status", h.SetUserStatus)
	adminGroup.POST("/users/:id/reset-password", h.ResetPassword)
}
