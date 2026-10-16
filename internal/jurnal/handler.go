package jurnal

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul jurnal harian.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler jurnal baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Create menangani endpoint POST /pendaftaran/:id/jurnal.
func (h *Handler) Create(c *gin.Context) {
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

	var req CreateJurnalRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateJurnal(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// Update menangani endpoint PUT /pendaftaran/:id/jurnal/:jid.
func (h *Handler) Update(c *gin.Context) {
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

	jid, err := strconv.Atoi(c.Param("jid"))
	if err != nil || jid <= 0 {
		httpx.WriteError(c, httpx.BadRequest("JID_TIDAK_VALID", "parameter jid harus berupa angka positif"))
		return
	}

	var req UpdateJurnalRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateJurnal(c.Request.Context(), actor, id, jid, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetByID menangani endpoint GET /pendaftaran/:id/jurnal/:jid.
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

	jid, err := strconv.Atoi(c.Param("jid"))
	if err != nil || jid <= 0 {
		httpx.WriteError(c, httpx.BadRequest("JID_TIDAK_VALID", "parameter jid harus berupa angka positif"))
		return
	}

	res, err := h.svc.GetJurnal(c.Request.Context(), actor, id, jid)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// List menangani endpoint GET /pendaftaran/:id/jurnal.
func (h *Handler) List(c *gin.Context) {
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

	p := httpx.ParsePagination(c)
	filter := JurnalFilter{
		TanggalMulai:   c.Query("tanggal_mulai"),
		TanggalSelesai: c.Query("tanggal_selesai"),
		Status:         c.Query("status"),
	}

	items, meta, err := h.svc.ListJurnal(c.Request.Context(), actor, id, filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}
