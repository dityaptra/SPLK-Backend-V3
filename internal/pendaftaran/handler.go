// Package pendaftaran menyediakan handler HTTP untuk modul pendaftaran program.
package pendaftaran

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul pendaftaran.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler pendaftaran baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Daftar godoc
// @Summary Pendaftaran program oleh mahasiswa
// @Tags Pendaftaran
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePendaftaranRequest true "Data pendaftaran program"
// @Success 201 {object} PendaftaranDetail
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran [post]
func (h *Handler) Daftar(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreatePendaftaranRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.Daftar(c.Request.Context(), actor, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// AjukanUlang godoc
// @Summary Pengajuan ulang pendaftaran yang ditolak
// @Tags Pendaftaran
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Success 200 {object} PendaftaranDetail
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/ajukan-ulang [post]
func (h *Handler) AjukanUlang(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	res, err := h.svc.AjukanUlang(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetByID godoc
// @Summary Detail pendaftaran program
// @Tags Pendaftaran
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Success 200 {object} PendaftaranDetail
// @Failure 404 {object} httpx.AppError
// @Router /pendaftaran/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	res, err := h.svc.GetDetail(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// List godoc
// @Summary Daftar pendaftaran program
// @Tags Pendaftaran
// @Security BearerAuth
// @Produce json
// @Param jenis_program query string false "Filter jenis program"
// @Param status query string false "Filter status (diajukan, ditolak, berjalan, selesai)"
// @Param id_periode query int false "Filter periode"
// @Param q query string false "Pencarian nama atau NIM"
// @Param page query int false "Nomor halaman"
// @Param per_page query int false "Jumlah data per halaman"
// @Success 200 {object} map[string]interface{}
// @Router /pendaftaran [get]
func (h *Handler) List(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	p := httpx.ParsePagination(c)
	filter := PendaftaranFilter{
		JenisProgram: c.Query("jenis_program"),
		Status:       c.Query("status"),
		Q:            c.Query("q"),
	}

	if idPerStr := c.Query("id_periode"); idPerStr != "" {
		if idPer, err := strconv.Atoi(idPerStr); err == nil && idPer > 0 {
			filter.IDPeriode = &idPer
		}
	}

	items, meta, err := h.svc.List(c.Request.Context(), actor, filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}

// PenempatanMagang godoc
// @Summary Pemilihan tempat magang dan mentor oleh mahasiswa
// @Tags Pendaftaran
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body PenempatanMagangRequest true "Pilihan tempat magang dan mentor"
// @Success 200 {object} PendaftaranDetail
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/penempatan-magang [put]
func (h *Handler) PenempatanMagang(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	var req PenempatanMagangRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.PenempatanMagang(c.Request.Context(), actor, id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// SetDosenPembimbing godoc
// @Summary Penetapan dosen pembimbing oleh admin/pegawai
// @Tags Pendaftaran
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body SetDosenPembimbingRequest true "ID Dosen Pembimbing"
// @Success 200 {object} PendaftaranDetail
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/dosen-pembimbing [put]
func (h *Handler) SetDosenPembimbing(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	var req SetDosenPembimbingRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	res, err := h.svc.SetDosenPembimbing(c.Request.Context(), actor, id, req, ip, userAgent)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// AdminPenempatan godoc
// @Summary Pemetaan sekolah penempatan dan guru pamong AM oleh admin
// @Tags Pendaftaran
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body AdminPenempatanRequest true "Penempatan sekolah dan guru pamong"
// @Success 200 {object} PendaftaranDetail
// @Failure 422 {object} httpx.AppError
// @Router /admin/pendaftaran/{id}/penempatan [put]
func (h *Handler) AdminPenempatan(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "tidak terautentikasi"))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	var req AdminPenempatanRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.AdminPenempatan(c.Request.Context(), actor, id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// SetDosenPenguji godoc
// @Summary Penetapan dosen penguji oleh admin
// @Tags Pendaftaran
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body SetDosenPengujiRequest true "ID Dosen Penguji"
// @Success 200 {object} PendaftaranDetail
// @Failure 422 {object} httpx.AppError
// @Router /admin/pendaftaran/{id}/dosen-penguji [put]
func (h *Handler) SetDosenPenguji(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "tidak terautentikasi"))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	var req SetDosenPengujiRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	res, err := h.svc.SetDosenPenguji(c.Request.Context(), actor, id, req, ip, userAgent)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}
