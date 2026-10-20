package pengumuman

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul pengumuman.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler baru untuk pengumuman.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Create godoc
// @Summary Buat pengumuman baru (Admin)
// @Description Admin membuat pengumuman baru yang dapat ditargetkan ke peran tertentu atau semua pengguna.
// @Tags Pengumuman
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreatePengumumanRequest true "Data pengumuman"
// @Success 201 {object} httpx.SingleResponse{data=PengumumanResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/pengumuman [post]
func (h *Handler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreatePengumumanRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreatePengumuman(c.Request.Context(), actor, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// Update godoc
// @Summary Perbarui pengumuman (Admin)
// @Description Admin memperbarui judul, isi, atau target peran pengumuman.
// @Tags Pengumuman
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pengumuman"
// @Param request body UpdatePengumumanRequest true "Data perubahan pengumuman"
// @Success 200 {object} httpx.SingleResponse{data=PengumumanResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/pengumuman/{id} [put]
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

	var req UpdatePengumumanRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdatePengumuman(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Delete godoc
// @Summary Hapus pengumuman (Admin)
// @Description Admin menghapus pengumuman berdasarkan ID.
// @Tags Pengumuman
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pengumuman"
// @Success 200 {object} httpx.SingleResponse{data=map[string]string}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /admin/pengumuman/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
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

	if err := h.svc.DeletePengumuman(c.Request.Context(), actor, id, c.ClientIP(), c.Request.UserAgent()); err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{"message": "pengumuman berhasil dihapus"})
}

// GetByID godoc
// @Summary Ambil detail pengumuman
// @Description Mengambil detail satu pengumuman. Pengguna biasa hanya dapat mengakses jika target_role sesuai atau terbuka untuk semua.
// @Tags Pengumuman
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pengumuman"
// @Success 200 {object} httpx.SingleResponse{data=PengumumanResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /pengumuman/{id} [get]
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

	res, err := h.svc.GetPengumumanByID(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// ListAdmin godoc
// @Summary Daftar seluruh pengumuman (Admin)
// @Description Admin melihat seluruh daftar pengumuman sistem dengan filter target role dan pencarian teks.
// @Tags Pengumuman
// @Security BearerAuth
// @Produce json
// @Param target_role query string false "Filter target role"
// @Param search query string false "Pencarian judul atau isi"
// @Param page query int false "Nomor halaman (default: 1)"
// @Param limit query int false "Jumlah data per halaman (default: 20)"
// @Success 200 {object} httpx.ListResponse{data=[]PengumumanResponse}
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Router /admin/pengumuman [get]
func (h *Handler) ListAdmin(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	p := httpx.ParsePagination(c)
	filter := PengumumanFilter{
		TargetRole: c.Query("target_role"),
		Search:     c.Query("search"),
	}

	items, meta, err := h.svc.ListPengumumanAdmin(c.Request.Context(), actor, filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}

// ListUser godoc
// @Summary Daftar pengumuman relevan (Pengguna Login)
// @Description Pengguna melihat daftar pengumuman yang relevan dengan perannya (termasuk pengumuman umum/publik).
// @Tags Pengumuman
// @Security BearerAuth
// @Produce json
// @Param search query string false "Pencarian judul atau isi"
// @Param page query int false "Nomor halaman (default: 1)"
// @Param limit query int false "Jumlah data per halaman (default: 20)"
// @Success 200 {object} httpx.ListResponse{data=[]PengumumanResponse}
// @Failure 401 {object} httpx.AppError
// @Router /pengumuman [get]
func (h *Handler) ListUser(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	p := httpx.ParsePagination(c)
	filter := PengumumanFilter{
		Search: c.Query("search"),
	}

	items, meta, err := h.svc.ListPengumumanUser(c.Request.Context(), actor, filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}
