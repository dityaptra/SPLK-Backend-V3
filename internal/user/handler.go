package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk manajemen pengguna dan profil aktor.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler user baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// --- Mahasiswa ---

// CreateMahasiswa godoc
// @Summary Tambah akun dan data mahasiswa
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateMahasiswaRequest true "Data mahasiswa baru"
// @Success 201 {object} MahasiswaDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 409 {object} httpx.AppError
// @Router /admin/mahasiswa [post]
func (h *Handler) CreateMahasiswa(c *gin.Context) {
	var req CreateMahasiswaRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.CreateMahasiswa(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetMahasiswaByID godoc
// @Summary Detail mahasiswa berdasarkan ID
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Mahasiswa"
// @Success 200 {object} MahasiswaDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/mahasiswa/{id} [get]
func (h *Handler) GetMahasiswaByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	res, err := h.svc.GetMahasiswaByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListMahasiswa godoc
// @Summary Daftar mahasiswa dengan paginasi
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit data per halaman"
// @Param offset query int false "Offset data"
// @Success 200 {object} httpx.ListResponse{data=[]MahasiswaDetail}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/mahasiswa [get]
func (h *Handler) ListMahasiswa(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListMahasiswa(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

// UpdateMahasiswa godoc
// @Summary Perbarui data mahasiswa
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Mahasiswa"
// @Param request body UpdateMahasiswaRequest true "Data pembaruan mahasiswa"
// @Success 200 {object} MahasiswaDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/mahasiswa/{id} [put]
func (h *Handler) UpdateMahasiswa(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req UpdateMahasiswaRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.UpdateMahasiswa(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// --- Dosen ---

// CreateDosen godoc
// @Summary Tambah akun dan data dosen
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateDosenRequest true "Data dosen baru"
// @Success 201 {object} DosenDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 409 {object} httpx.AppError
// @Router /admin/dosen [post]
func (h *Handler) CreateDosen(c *gin.Context) {
	var req CreateDosenRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.CreateDosen(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetDosenByID godoc
// @Summary Detail dosen berdasarkan ID
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Dosen"
// @Success 200 {object} DosenDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/dosen/{id} [get]
func (h *Handler) GetDosenByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	res, err := h.svc.GetDosenByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListDosen godoc
// @Summary Daftar dosen dengan paginasi
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit data per halaman"
// @Param offset query int false "Offset data"
// @Success 200 {object} httpx.ListResponse{data=[]DosenDetail}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/dosen [get]
func (h *Handler) ListDosen(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListDosen(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

// UpdateDosen godoc
// @Summary Perbarui data dosen
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Dosen"
// @Param request body UpdateDosenRequest true "Data pembaruan dosen"
// @Success 200 {object} DosenDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/dosen/{id} [put]
func (h *Handler) UpdateDosen(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req UpdateDosenRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.UpdateDosen(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// --- Pegawai ---

// CreatePegawai godoc
// @Summary Tambah akun dan data pegawai
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePegawaiRequest true "Data pegawai baru"
// @Success 201 {object} PegawaiDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 409 {object} httpx.AppError
// @Router /admin/pegawai [post]
func (h *Handler) CreatePegawai(c *gin.Context) {
	var req CreatePegawaiRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.CreatePegawai(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetPegawaiByID godoc
// @Summary Detail pegawai berdasarkan ID
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pegawai"
// @Success 200 {object} PegawaiDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/pegawai/{id} [get]
func (h *Handler) GetPegawaiByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	res, err := h.svc.GetPegawaiByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListPegawai godoc
// @Summary Daftar pegawai dengan paginasi
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit data per halaman"
// @Param offset query int false "Offset data"
// @Success 200 {object} httpx.ListResponse{data=[]PegawaiDetail}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/pegawai [get]
func (h *Handler) ListPegawai(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListPegawai(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

// UpdatePegawai godoc
// @Summary Perbarui data pegawai
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pegawai"
// @Param request body UpdatePegawaiRequest true "Data pembaruan pegawai"
// @Success 200 {object} PegawaiDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/pegawai/{id} [put]
func (h *Handler) UpdatePegawai(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req UpdatePegawaiRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.UpdatePegawai(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// --- Admin ---

// CreateAdmin godoc
// @Summary Tambah akun admin baru
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateAdminRequest true "Data admin baru"
// @Success 201 {object} AdminDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 409 {object} httpx.AppError
// @Router /admin/admins [post]
func (h *Handler) CreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.CreateAdmin(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetAdminByID godoc
// @Summary Detail admin berdasarkan ID
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Admin"
// @Success 200 {object} AdminDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/admins/{id} [get]
func (h *Handler) GetAdminByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	res, err := h.svc.GetAdminByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListAdmin godoc
// @Summary Daftar admin dengan paginasi
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit data per halaman"
// @Param offset query int false "Offset data"
// @Success 200 {object} httpx.ListResponse{data=[]AdminDetail}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/admins [get]
func (h *Handler) ListAdmin(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListAdmin(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

// UpdateAdmin godoc
// @Summary Perbarui data profil admin
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Admin"
// @Param request body UpdateAdminRequest true "Data pembaruan admin"
// @Success 200 {object} AdminDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/admins/{id} [put]
func (h *Handler) UpdateAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req UpdateAdminRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.UpdateAdmin(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// --- Pembimbing Lapangan ---

// CreatePembimbingLapangan godoc
// @Summary Tambah akun dan data pembimbing lapangan (mentor, guru pamong, kepala desa)
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePembimbingLapanganRequest true "Data pembimbing lapangan baru"
// @Success 201 {object} PembimbingLapanganDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 409 {object} httpx.AppError
// @Router /admin/pembimbing-lapangan [post]
func (h *Handler) CreatePembimbingLapangan(c *gin.Context) {
	var req CreatePembimbingLapanganRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.CreatePembimbingLapangan(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetPembimbingLapanganByID godoc
// @Summary Detail pembimbing lapangan berdasarkan ID
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pembimbing Lapangan"
// @Success 200 {object} PembimbingLapanganDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/pembimbing-lapangan/{id} [get]
func (h *Handler) GetPembimbingLapanganByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	res, err := h.svc.GetPembimbingLapanganByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListPembimbingLapangan godoc
// @Summary Daftar pembimbing lapangan dengan paginasi
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit data per halaman"
// @Param offset query int false "Offset data"
// @Success 200 {object} httpx.ListResponse{data=[]PembimbingLapanganDetail}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/pembimbing-lapangan [get]
func (h *Handler) ListPembimbingLapangan(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListPembimbingLapangan(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

// UpdatePembimbingLapangan godoc
// @Summary Perbarui data pembimbing lapangan
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pembimbing Lapangan"
// @Param request body UpdatePembimbingLapanganRequest true "Data pembaruan pembimbing lapangan"
// @Success 200 {object} PembimbingLapanganDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/pembimbing-lapangan/{id} [put]
func (h *Handler) UpdatePembimbingLapangan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req UpdatePembimbingLapanganRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	res, err := h.svc.UpdatePembimbingLapangan(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// --- User Manajemen Umum ---

// ListUsers godoc
// @Summary Daftar user umum dengan filter dan paginasi
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param role query string false "Filter berdasarkan role"
// @Param q query string false "Pencarian username atau email"
// @Param limit query int false "Limit data per halaman"
// @Param offset query int false "Offset data"
// @Success 200 {object} httpx.ListResponse{data=[]UserListItem}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	p := httpx.ParsePagination(c)
	filter := UserFilter{
		Role:  c.Query("role"),
		Query: c.Query("q"),
	}
	items, meta, err := h.svc.ListUsers(c.Request.Context(), filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

// GetUserByID godoc
// @Summary Detail user umum berdasarkan ID
// @Tags User
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID User"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/users/{id} [get]
func (h *Handler) GetUserByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	u, roles, err := h.svc.GetUserByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":        u.ID,
		"username":  u.Username,
		"email":     u.Email,
		"is_active": u.IsActive,
		"roles":     roles,
	})
}

// SetUserStatus godoc
// @Summary Aktifkan atau nonaktifkan user
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID User"
// @Param request body UpdateUserStatusRequest true "Status aktif user"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/users/{id}/status [patch]
func (h *Handler) SetUserStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req UpdateUserStatusRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	if err := h.svc.SetUserStatus(c.Request.Context(), id, req.IsActive); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "status user berhasil diperbarui"})
}

// ResetPassword godoc
// @Summary Reset password user oleh admin
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID User"
// @Param request body ResetPasswordRequest true "Password baru"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/users/{id}/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}
	var req ResetPasswordRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), id, req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "password user berhasil di-reset"})
}
