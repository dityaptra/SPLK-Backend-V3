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

// Create menangani endpoint POST /admin/tempat-program.
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

// GetByID menangani endpoint GET /tempat-program/:id.
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

// List menangani endpoint GET /tempat-program.
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

// Update menangani endpoint PUT /admin/tempat-program/:id.
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

// Delete menangani endpoint DELETE /admin/tempat-program/:id (soft delete is_aktif=false).
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

// ListPembimbing menangani endpoint GET /tempat-program/:id/pembimbing-lapangan.
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
