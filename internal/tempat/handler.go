package tempat

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul tempat.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary Tambah tempat pelaksanaan program
// @Tags Tempat Program
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateTempatRequest true "Data tempat program"
// @Success 201 {object} TempatProgram
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/tempat-program [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateTempatRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// GetByID godoc
// @Summary Detail tempat pelaksanaan program
// @Tags Tempat Program
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Tempat Program"
// @Success 200 {object} TempatProgram
// @Failure 404 {object} httpx.AppError
// @Router /tempat-program/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	res, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// List godoc
// @Summary Daftar tempat pelaksanaan program
// @Tags Tempat Program
// @Security BearerAuth
// @Produce json
// @Param jenis_program query string false "Filter jenis program (magang, asistensi_mengajar, kkn)"
// @Param is_aktif query bool false "Filter status keaktifan"
// @Param page query int false "Nomor halaman"
// @Param per_page query int false "Jumlah data per halaman"
// @Success 200 {object} map[string]interface{}
// @Router /tempat-program [get]
func (h *Handler) List(c *gin.Context) {
	p := httpx.ParsePagination(c)
	filter := TempatFilter{
		JenisProgram: c.Query("jenis_program"),
	}

	if isAktifStr := c.Query("is_aktif"); isAktifStr != "" {
		isAktif, err := strconv.ParseBool(isAktifStr)
		if err == nil {
			filter.IsAktif = &isAktif
		}
	}

	items, meta, err := h.svc.List(c.Request.Context(), filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}

// Update godoc
// @Summary Perbarui data tempat program
// @Tags Tempat Program
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Tempat Program"
// @Param request body UpdateTempatRequest true "Data perubahan"
// @Success 200 {object} TempatProgram
// @Failure 404 {object} httpx.AppError
// @Router /admin/tempat-program/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	var req UpdateTempatRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Delete godoc
// @Summary Nonaktifkan tempat program (soft delete)
// @Tags Tempat Program
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Tempat Program"
// @Success 200 {object} map[string]string
// @Failure 404 {object} httpx.AppError
// @Router /admin/tempat-program/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{"message": "tempat program berhasil dinonaktifkan"})
}

// ListPembimbing godoc
// @Summary Daftar pembimbing lapangan pada suatu tempat
// @Tags Tempat Program
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Tempat Program"
// @Success 200 {array} PembimbingLapangan
// @Router /tempat-program/{id}/pembimbing-lapangan [get]
func (h *Handler) ListPembimbing(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	items, err := h.svc.ListPembimbing(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, items)
}
