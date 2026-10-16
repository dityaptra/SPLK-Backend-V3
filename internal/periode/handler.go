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

// Create godoc
// @Summary Buat periode program baru
// @Tags Periode
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePeriodeRequest true "Data periode"
// @Success 201 {object} PeriodeDetail
// @Failure 400 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /admin/periode [post]
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

// GetByID godoc
// @Summary Detail periode program
// @Tags Periode
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Periode"
// @Success 200 {object} PeriodeDetail
// @Failure 404 {object} httpx.AppError
// @Router /periode/{id} [get]
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

// List godoc
// @Summary Daftar periode program
// @Tags Periode
// @Security BearerAuth
// @Produce json
// @Param jenis_program query string false "Filter jenis program (magang, asistensi_mengajar, kkn)"
// @Param status query string false "Filter status (draft, aktif, selesai)"
// @Param page query int false "Nomor halaman"
// @Param per_page query int false "Jumlah data per halaman"
// @Success 200 {object} map[string]interface{}
// @Router /periode [get]
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

// Update godoc
// @Summary Perbarui draft periode
// @Tags Periode
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Periode"
// @Param request body UpdatePeriodeRequest true "Perubahan data periode"
// @Success 200 {object} PeriodeDetail
// @Failure 404 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /admin/periode/{id} [put]
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

// Aktifkan godoc
// @Summary Aktifkan periode (draft -> aktif)
// @Tags Periode
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Periode"
// @Success 200 {object} PeriodeDetail
// @Failure 422 {object} httpx.AppError
// @Router /admin/periode/{id}/aktifkan [post]
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

// Selesaikan godoc
// @Summary Selesaikan periode (aktif -> selesai)
// @Description Menyelesaikan periode serta mengalihkan status peserta 'berjalan' ke 'selesai' dan peserta 'diajukan' ke 'ditolak'.
// @Tags Periode
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Periode"
// @Success 200 {object} PeriodeDetail
// @Failure 422 {object} httpx.AppError
// @Router /admin/periode/{id}/selesaikan [post]
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
