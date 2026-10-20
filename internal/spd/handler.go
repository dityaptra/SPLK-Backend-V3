package spd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/config"
	"splk-backend/internal/dokumen"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul Surat Perjalanan Dinas.
type Handler struct {
	svc            Service
	maxUploadBytes int64
}

// NewHandler membuat instance Handler baru untuk SPD.
func NewHandler(svc Service, cfg *config.Config) *Handler {
	maxBytes := int64(10 * 1024 * 1024)
	if cfg != nil && cfg.MaxUploadMB > 0 {
		maxBytes = cfg.MaxUploadMB * 1024 * 1024
	}
	return &Handler{
		svc:            svc,
		maxUploadBytes: maxBytes,
	}
}

// Create godoc
// @Summary Buat Surat Perjalanan Dinas (Pegawai/Admin)
// @Description Pegawai atau admin membuat surat perjalanan dinas untuk dosen dan peserta program.
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateSPDRequest true "Data SPD"
// @Success 201 {object} httpx.SingleResponse{data=SPDResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /spd [post]
func (h *Handler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreateSPDRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateSPD(c.Request.Context(), actor, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// Update godoc
// @Summary Perbarui Surat Perjalanan Dinas (Pegawai/Admin)
// @Description Pegawai atau admin memperbarui data surat perjalanan dinas.
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID SPD"
// @Param request body UpdateSPDRequest true "Data perubahan SPD"
// @Success 200 {object} httpx.SingleResponse{data=SPDResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /spd/{id} [put]
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

	var req UpdateSPDRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateSPD(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Delete godoc
// @Summary Hapus Surat Perjalanan Dinas (Pegawai/Admin)
// @Description Pegawai atau admin menghapus surat perjalanan dinas dan berkas fisiknya.
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID SPD"
// @Success 200 {object} httpx.SingleResponse{data=map[string]string}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /spd/{id} [delete]
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

	if err := h.svc.DeleteSPD(c.Request.Context(), actor, id, c.ClientIP(), c.Request.UserAgent()); err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{"message": "surat perjalanan dinas berhasil dihapus"})
}

// GetByID godoc
// @Summary Ambil detail Surat Perjalanan Dinas
// @Description Mengambil detail satu SPD. Terbuka untuk Pegawai, Admin, Dosen terkait, dan Mahasiswa terkait.
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID SPD"
// @Success 200 {object} httpx.SingleResponse{data=SPDResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /spd/{id} [get]
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

	res, err := h.svc.GetSPDByID(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// List godoc
// @Summary Daftar Surat Perjalanan Dinas
// @Description Melihat daftar SPD. Admin dan Pegawai melihat seluruh data; Dosen melihat SPD tugasnya; Mahasiswa melihat SPD kegiatannya.
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Produce json
// @Param id_dosen query int false "Filter ID Dosen"
// @Param id_peserta_program query int false "Filter ID Peserta Program"
// @Param search query string false "Pencarian tujuan/nama dosen/nama mahasiswa"
// @Param page query int false "Nomor halaman (default: 1)"
// @Param limit query int false "Jumlah data per halaman (default: 20)"
// @Success 200 {object} httpx.ListResponse{data=[]SPDResponse}
// @Failure 401 {object} httpx.AppError
// @Router /spd [get]
func (h *Handler) List(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	p := httpx.ParsePagination(c)
	idDosen, _ := strconv.Atoi(c.Query("id_dosen"))
	idPeserta, _ := strconv.Atoi(c.Query("id_peserta_program"))

	filter := SPDFilter{
		IDDosen:          idDosen,
		IDPesertaProgram: idPeserta,
		Search:           c.Query("search"),
	}

	items, meta, err := h.svc.ListSPD(c.Request.Context(), actor, filter, p)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.List(c, items, meta)
}

// UploadFile godoc
// @Summary Unggah berkas fisik Surat Perjalanan Dinas (PDF/Docx/Gambar)
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID SPD"
// @Param file formData file true "File dokumen SPD (maks 10MB)"
// @Success 200 {object} httpx.SingleResponse{data=SPDResponse}
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /spd/{id}/file [post]
func (h *Handler) UploadFile(c *gin.Context) {
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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes+1024*1024)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "too large") {
			httpx.WriteError(c, httpx.Unprocessable("FILE_TERLALU_BESAR", "ukuran file melebihi batas maksimal"))
			return
		}
		httpx.WriteError(c, httpx.BadRequest("FILE_DIBUTUHKAN", "file berkas wajib disertakan"))
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadBytes {
		httpx.WriteError(c, httpx.Unprocessable("FILE_TERLALU_BESAR", "ukuran file melebihi batas maksimal"))
		return
	}

	res, err := h.svc.UploadFile(c.Request.Context(), actor, id, header.Filename, header.Size, file)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// DownloadFile godoc
// @Summary Unduh berkas fisik Surat Perjalanan Dinas
// @Tags Surat Perjalanan Dinas
// @Security BearerAuth
// @Produce octet-stream
// @Param id path int true "ID SPD"
// @Success 200 {file} binary
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /spd/{id}/download [get]
func (h *Handler) DownloadFile(c *gin.Context) {
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

	rc, spdDetail, err := h.svc.DownloadFile(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	defer rc.Close()

	filename := fmt.Sprintf("SPD_%d_%s%s", spdDetail.ID, spdDetail.NamaDosen, filepath.Ext(*spdDetail.FileURL))
	sanitizedName := dokumen.SanitizeFilename(filename)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizedName))
	c.Header("Content-Type", "application/octet-stream")
	_, _ = io.Copy(c.Writer, rc)
}
