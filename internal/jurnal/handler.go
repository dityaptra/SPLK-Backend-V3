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

// Create godoc
// @Summary Isi jurnal harian (mahasiswa pemilik)
// @Tags Jurnal Harian
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body CreateJurnalRequest true "Data jurnal harian"
// @Success 201 {object} JurnalResponse
// @Failure 400 {object} httpx.AppError
// @Failure 409 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/jurnal [post]
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

// Update godoc
// @Summary Perbarui jurnal harian draft
// @Tags Jurnal Harian
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param jid path int true "ID Jurnal"
// @Param request body UpdateJurnalRequest true "Perubahan data jurnal"
// @Success 200 {object} JurnalResponse
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/jurnal/{jid} [put]
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

// GetByID godoc
// @Summary Detail jurnal harian
// @Tags Jurnal Harian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param jid path int true "ID Jurnal"
// @Success 200 {object} JurnalResponse
// @Failure 404 {object} httpx.AppError
// @Router /pendaftaran/{id}/jurnal/{jid} [get]
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

// List godoc
// @Summary Daftar jurnal harian
// @Tags Jurnal Harian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param tanggal_mulai query string false "Filter tanggal mulai (YYYY-MM-DD)"
// @Param tanggal_selesai query string false "Filter tanggal selesai (YYYY-MM-DD)"
// @Param status query string false "Filter status (draft, disubmit)"
// @Param page query int false "Nomor halaman"
// @Param per_page query int false "Jumlah data per halaman"
// @Success 200 {object} map[string]interface{}
// @Router /pendaftaran/{id}/jurnal [get]
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
