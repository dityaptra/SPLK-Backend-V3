package akademik

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul hierarki akademik.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler akademik baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// --- Fakultas Handlers ---

// CreateFakultas godoc
// @Summary Tambah fakultas baru
// @Tags Akademik
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateFakultasRequest true "Data fakultas"
// @Success 201 {object} Fakultas
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/fakultas [post]
func (h *Handler) CreateFakultas(c *gin.Context) {
	var req CreateFakultasRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateFakultas(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetFakultasByID godoc
// @Summary Detail fakultas
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Fakultas"
// @Success 200 {object} Fakultas
// @Failure 404 {object} httpx.AppError
// @Router /fakultas/{id} [get]
func (h *Handler) GetFakultasByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	res, err := h.svc.GetFakultasByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListFakultas godoc
// @Summary Daftar seluruh fakultas
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Success 200 {array} Fakultas
// @Router /fakultas [get]
func (h *Handler) ListFakultas(c *gin.Context) {
	items, err := h.svc.ListFakultas(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, items)
}

// UpdateFakultas godoc
// @Summary Perbarui data fakultas
// @Tags Akademik
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Fakultas"
// @Param request body UpdateFakultasRequest true "Perubahan data fakultas"
// @Success 200 {object} Fakultas
// @Failure 400 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/fakultas/{id} [put]
func (h *Handler) UpdateFakultas(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	var req UpdateFakultasRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateFakultas(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// DeleteFakultas godoc
// @Summary Hapus fakultas
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Fakultas"
// @Success 200 {object} map[string]string
// @Failure 404 {object} httpx.AppError
// @Router /admin/fakultas/{id} [delete]
func (h *Handler) DeleteFakultas(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	if err := h.svc.DeleteFakultas(c.Request.Context(), id); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "fakultas berhasil dihapus"})
}

// --- Jurusan Handlers ---

// CreateJurusan godoc
// @Summary Tambah jurusan baru
// @Tags Akademik
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateJurusanRequest true "Data jurusan"
// @Success 201 {object} Jurusan
// @Failure 400 {object} httpx.AppError
// @Router /admin/jurusan [post]
func (h *Handler) CreateJurusan(c *gin.Context) {
	var req CreateJurusanRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateJurusan(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetJurusanByID godoc
// @Summary Detail jurusan
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Jurusan"
// @Success 200 {object} Jurusan
// @Failure 404 {object} httpx.AppError
// @Router /jurusan/{id} [get]
func (h *Handler) GetJurusanByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	res, err := h.svc.GetJurusanByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListJurusan godoc
// @Summary Daftar seluruh jurusan
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id_fakultas query int false "Filter berdasarkan ID Fakultas"
// @Success 200 {array} Jurusan
// @Router /jurusan [get]
func (h *Handler) ListJurusan(c *gin.Context) {
	var fakID *int
	if fStr := c.Query("id_fakultas"); fStr != "" {
		if id, err := strconv.Atoi(fStr); err == nil && id > 0 {
			fakID = &id
		}
	}

	items, err := h.svc.ListJurusan(c.Request.Context(), fakID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, items)
}

// UpdateJurusan godoc
// @Summary Perbarui data jurusan
// @Tags Akademik
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Jurusan"
// @Param request body UpdateJurusanRequest true "Perubahan data jurusan"
// @Success 200 {object} Jurusan
// @Failure 404 {object} httpx.AppError
// @Router /admin/jurusan/{id} [put]
func (h *Handler) UpdateJurusan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	var req UpdateJurusanRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateJurusan(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// DeleteJurusan godoc
// @Summary Hapus jurusan
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Jurusan"
// @Success 200 {object} map[string]string
// @Failure 404 {object} httpx.AppError
// @Router /admin/jurusan/{id} [delete]
func (h *Handler) DeleteJurusan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	if err := h.svc.DeleteJurusan(c.Request.Context(), id); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "jurusan berhasil dihapus"})
}

// --- Program Studi Handlers ---

// CreateProdi godoc
// @Summary Tambah program studi baru
// @Tags Akademik
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateProdiRequest true "Data program studi"
// @Success 201 {object} ProgramStudi
// @Failure 400 {object} httpx.AppError
// @Router /admin/program-studi [post]
func (h *Handler) CreateProdi(c *gin.Context) {
	var req CreateProdiRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateProdi(c.Request.Context(), req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, res)
}

// GetProdiByID godoc
// @Summary Detail program studi
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Program Studi"
// @Success 200 {object} ProgramStudi
// @Failure 404 {object} httpx.AppError
// @Router /program-studi/{id} [get]
func (h *Handler) GetProdiByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	res, err := h.svc.GetProdiByID(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// ListProdi godoc
// @Summary Daftar seluruh program studi
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id_jurusan query int false "Filter berdasarkan ID Jurusan"
// @Success 200 {array} ProgramStudi
// @Router /program-studi [get]
func (h *Handler) ListProdi(c *gin.Context) {
	var jurID *int
	if jStr := c.Query("id_jurusan"); jStr != "" {
		if id, err := strconv.Atoi(jStr); err == nil && id > 0 {
			jurID = &id
		}
	}

	items, err := h.svc.ListProdi(c.Request.Context(), jurID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, items)
}

// UpdateProdi godoc
// @Summary Perbarui data program studi
// @Tags Akademik
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Program Studi"
// @Param request body UpdateProdiRequest true "Perubahan data program studi"
// @Success 200 {object} ProgramStudi
// @Failure 404 {object} httpx.AppError
// @Router /admin/program-studi/{id} [put]
func (h *Handler) UpdateProdi(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	var req UpdateProdiRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateProdi(c.Request.Context(), id, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, res)
}

// DeleteProdi godoc
// @Summary Hapus program studi
// @Tags Akademik
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Program Studi"
// @Success 200 {object} map[string]string
// @Failure 404 {object} httpx.AppError
// @Router /admin/program-studi/{id} [delete]
func (h *Handler) DeleteProdi(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus angka positif"))
		return
	}

	if err := h.svc.DeleteProdi(c.Request.Context(), id); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "program studi berhasil dihapus"})
}
