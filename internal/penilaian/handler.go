package penilaian

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul penilaian.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler penilaian baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// CreateInstrumen menangani POST /admin/instrumen-penilaian.
func (h *Handler) CreateInstrumen(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreateInstrumenRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateInstrumen(c.Request.Context(), actor, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// GetInstrumen menangani GET /admin/instrumen-penilaian/:id.
func (h *Handler) GetInstrumen(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	res, err := h.svc.GetInstrumen(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// UpdateInstrumen menangani PUT /admin/instrumen-penilaian/:id.
func (h *Handler) UpdateInstrumen(c *gin.Context) {
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

	var req UpdateInstrumenRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateInstrumen(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(c, httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan"))
			return
		}
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// DeleteInstrumen menangani DELETE /admin/instrumen-penilaian/:id.
func (h *Handler) DeleteInstrumen(c *gin.Context) {
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

	if err := h.svc.DeleteInstrumen(c.Request.Context(), actor, id, c.ClientIP(), c.Request.UserAgent()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(c, httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan"))
			return
		}
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{"pesan": "instrumen penilaian berhasil dihapus"})
}

// ListInstrumen menangani GET /admin/instrumen-penilaian.
func (h *Handler) ListInstrumen(c *gin.Context) {
	jenis := c.Query("jenis_program")
	role := c.Query("role_penilai")

	items, err := h.svc.ListInstrumen(c.Request.Context(), jenis, role)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, items)
}

// GetRingkasanBobot menangani GET /admin/instrumen-penilaian/ringkasan.
func (h *Handler) GetRingkasanBobot(c *gin.Context) {
	res, err := h.svc.GetRingkasanBobot(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetFormPenilaian menangani GET /pendaftaran/:id/form-penilaian.
func (h *Handler) GetFormPenilaian(c *gin.Context) {
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

	res, err := h.svc.GetFormPenilaian(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// SubmitPenilaian menangani POST /pendaftaran/:id/penilaian.
func (h *Handler) SubmitPenilaian(c *gin.Context) {
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

	var req SubmitPenilaianRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.SubmitPenilaian(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetNilai menangani GET /pendaftaran/:id/nilai.
func (h *Handler) GetNilai(c *gin.Context) {
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

	res, err := h.svc.GetNilai(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// HitungNilaiManual menangani POST /admin/pendaftaran/:id/hitung-nilai.
func (h *Handler) HitungNilaiManual(c *gin.Context) {
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

	res, err := h.svc.HitungNilaiManual(c.Request.Context(), actor, id, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}
