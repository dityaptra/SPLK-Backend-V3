// Package periode menyediakan handler HTTP untuk modul periode program.
package periode

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk manajemen periode program.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Create menangani endpoint POST /admin/periode.
func (h *Handler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreatePeriodeRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreatePeriode(c.Request.Context(), req, actor.UserID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// GetByID menangani endpoint GET /periode/:id.
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

	res, err := h.svc.GetPeriodeByID(c.Request.Context(), id, actor)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// List menangani endpoint GET /periode.
func (h *Handler) List(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	p := httpx.ParsePagination(c)
	filter := PeriodeFilter{
		JenisProgram: c.Query("jenis_program"),
		Status:       c.Query("status"),
	}

	items, meta, err := h.svc.ListPeriode(c.Request.Context(), filter, p, actor)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}

// Update menangani endpoint PUT /admin/periode/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	var req UpdatePeriodeRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdatePeriode(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Aktifkan menangani endpoint POST /admin/periode/:id/aktifkan.
func (h *Handler) Aktifkan(c *gin.Context) {
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	res, err := h.svc.AktifkanPeriode(c.Request.Context(), id, actor.UserID, ip, ua)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Selesaikan menangani endpoint POST /admin/periode/:id/selesaikan.
func (h *Handler) Selesaikan(c *gin.Context) {
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	res, err := h.svc.SelesaikanPeriode(c.Request.Context(), id, actor.UserID, ip, ua)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}
