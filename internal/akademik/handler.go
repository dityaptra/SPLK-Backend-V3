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

func (h *Handler) ListFakultas(c *gin.Context) {
	items, err := h.svc.ListFakultas(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, items)
}

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
