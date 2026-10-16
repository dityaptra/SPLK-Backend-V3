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

// CreateKelompok menangani endpoint POST /admin/kkn/kelompok.
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

// GetKelompok menangani endpoint GET /admin/kkn/kelompok/:id dan GET /kkn/kelompok/:id.
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

// ListKelompok menangani endpoint GET /admin/kkn/kelompok.
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

// AddAnggota menangani endpoint POST /admin/kkn/kelompok/:id/anggota.
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

// RemoveAnggota menangani endpoint DELETE /admin/kkn/kelompok/:id/anggota/:pid.
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

// SetDosenPembimbing menangani endpoint PUT /admin/kkn/kelompok/:id/dosen-pembimbing.
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

// SetPembimbingLapangan menangani endpoint PUT /admin/kkn/kelompok/:id/pembimbing-lapangan.
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
