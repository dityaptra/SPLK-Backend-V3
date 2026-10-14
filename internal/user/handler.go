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

func (h *Handler) ListMahasiswa(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListMahasiswa(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

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

func (h *Handler) ListDosen(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListDosen(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

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

func (h *Handler) ListPegawai(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListPegawai(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

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

func (h *Handler) ListAdmin(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListAdmin(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

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

func (h *Handler) ListPembimbingLapangan(c *gin.Context) {
	p := httpx.ParsePagination(c)
	items, meta, err := h.svc.ListPembimbingLapangan(c.Request.Context(), p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.List(c, items, meta)
}

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
