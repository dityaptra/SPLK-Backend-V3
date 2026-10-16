package kkn

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul cabang KKN.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler KKN baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// CreateKelompok godoc
// @Summary Buat kelompok KKN baru
// @Tags KKN
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateKelompokRequest true "Data kelompok KKN"
// @Success 201 {object} KelompokDetailResponse
// @Failure 400 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /admin/kkn/kelompok [post]
func (h *Handler) CreateKelompok(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreateKelompokRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateKelompok(c.Request.Context(), actor, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// GetKelompok godoc
// @Summary Detail kelompok KKN
// @Tags KKN
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Kelompok KKN"
// @Success 200 {object} KelompokDetailResponse
// @Failure 404 {object} httpx.AppError
// @Router /kkn/kelompok/{id} [get]
func (h *Handler) GetKelompok(c *gin.Context) {
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

	res, err := h.svc.GetKelompok(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// ListKelompok godoc
// @Summary Daftar kelompok KKN
// @Tags KKN
// @Security BearerAuth
// @Produce json
// @Param id_periode query int false "Filter periode"
// @Param page query int false "Nomor halaman"
// @Param per_page query int false "Jumlah data per halaman"
// @Success 200 {object} map[string]interface{}
// @Router /admin/kkn/kelompok [get]
func (h *Handler) ListKelompok(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	p := httpx.ParsePagination(c)
	var idPeriode *int
	if idPerStr := c.Query("id_periode"); idPerStr != "" {
		if idPer, err := strconv.Atoi(idPerStr); err == nil && idPer > 0 {
			idPeriode = &idPer
		}
	}

	items, meta, err := h.svc.ListKelompok(c.Request.Context(), actor, idPeriode, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}

// AddAnggota godoc
// @Summary Tambah anggota ke kelompok KKN
// @Tags KKN
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Kelompok KKN"
// @Param request body AddAnggotaRequest true "ID pendaftaran peserta"
// @Success 200 {object} KelompokDetailResponse
// @Failure 422 {object} httpx.AppError
// @Router /admin/kkn/kelompok/{id}/anggota [post]
func (h *Handler) AddAnggota(c *gin.Context) {
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

	var req AddAnggotaRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.AddAnggota(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// RemoveAnggota godoc
// @Summary Keluarkan anggota dari kelompok KKN
// @Tags KKN
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Kelompok KKN"
// @Param pid path int true "ID Pendaftaran Peserta"
// @Success 200 {object} map[string]string
// @Failure 422 {object} httpx.AppError
// @Router /admin/kkn/kelompok/{id}/anggota/{pid} [delete]
func (h *Handler) RemoveAnggota(c *gin.Context) {
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

	pid, err := strconv.Atoi(c.Param("pid"))
	if err != nil || pid <= 0 {
		httpx.WriteError(c, httpx.BadRequest("PID_TIDAK_VALID", "parameter pid harus berupa angka positif"))
		return
	}

	err = h.svc.RemoveAnggota(c.Request.Context(), actor, id, pid, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{"message": "anggota berhasil dikeluarkan dari kelompok KKN"})
}

// SetDosenPembimbing godoc
// @Summary Tetapkan dosen pembimbing kelompok KKN
// @Tags KKN
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Kelompok KKN"
// @Param request body SetDosenPembimbingKelompokRequest true "ID Dosen Pembimbing"
// @Success 200 {object} KelompokDetailResponse
// @Failure 422 {object} httpx.AppError
// @Router /admin/kkn/kelompok/{id}/dosen-pembimbing [put]
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

	var req SetDosenPembimbingKelompokRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.SetDosenPembimbing(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// SetPembimbingLapangan godoc
// @Summary Tetapkan kepala desa / pembimbing lapangan kelompok KKN
// @Tags KKN
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Kelompok KKN"
// @Param request body SetPembimbingLapanganKelompokRequest true "ID Pembimbing Lapangan"
// @Success 200 {object} KelompokDetailResponse
// @Failure 422 {object} httpx.AppError
// @Router /admin/kkn/kelompok/{id}/pembimbing-lapangan [put]
func (h *Handler) SetPembimbingLapangan(c *gin.Context) {
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

	var req SetPembimbingLapanganKelompokRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.SetPembimbingLapangan(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}
