package logbook

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul logbook bimbingan.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler logbook baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary Isi catatan bimbingan (khusus dosen pembimbing pendaftaran itu)
// @Tags Logbook Bimbingan
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body CreateLogbookRequest true "Catatan bimbingan"
// @Success 201 {object} LogbookResponse
// @Failure 404 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/logbook [post]
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

	var req CreateLogbookRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateLogbook(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// Update godoc
// @Summary Perbarui catatan bimbingan (hanya dosen pembuat)
// @Tags Logbook Bimbingan
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param lid path int true "ID Logbook"
// @Param request body UpdateLogbookRequest true "Perubahan catatan"
// @Success 200 {object} LogbookResponse
// @Failure 404 {object} httpx.AppError
// @Router /pendaftaran/{id}/logbook/{lid} [put]
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

	lid, err := strconv.Atoi(c.Param("lid"))
	if err != nil || lid <= 0 {
		httpx.WriteError(c, httpx.BadRequest("LID_TIDAK_VALID", "parameter lid harus berupa angka positif"))
		return
	}

	var req UpdateLogbookRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateLogbook(c.Request.Context(), actor, id, lid, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetByID godoc
// @Summary Detail logbook bimbingan
// @Tags Logbook Bimbingan
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param lid path int true "ID Logbook"
// @Success 200 {object} LogbookResponse
// @Failure 404 {object} httpx.AppError
// @Router /pendaftaran/{id}/logbook/{lid} [get]
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

	lid, err := strconv.Atoi(c.Param("lid"))
	if err != nil || lid <= 0 {
		httpx.WriteError(c, httpx.BadRequest("LID_TIDAK_VALID", "parameter lid harus berupa angka positif"))
		return
	}

	res, err := h.svc.GetLogbook(c.Request.Context(), actor, id, lid)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// List godoc
// @Summary Daftar logbook bimbingan pendaftaran
// @Tags Logbook Bimbingan
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param page query int false "Nomor halaman"
// @Param per_page query int false "Jumlah data per halaman"
// @Success 200 {object} map[string]interface{}
// @Router /pendaftaran/{id}/logbook [get]
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

	items, meta, err := h.svc.ListLogbook(c.Request.Context(), actor, id, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}
