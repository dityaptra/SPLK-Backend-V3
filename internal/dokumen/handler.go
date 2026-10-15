package dokumen

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"splk-backend/internal/config"
	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani endpoint HTTP terkait dokumen.
type Handler struct {
	service        Service
	maxUploadBytes int64
}

// NewHandler membuat instance baru dokumen Handler.
func NewHandler(service Service, cfg *config.Config) *Handler {
	maxBytes := int64(5 * 1024 * 1024)
	if cfg != nil && cfg.MaxUploadMB > 0 {
		maxBytes = cfg.MaxUploadMB * 1024 * 1024
	}
	return &Handler{
		service:        service,
		maxUploadBytes: maxBytes,
	}
}

// UploadPeserta menangani POST /pendaftaran/:id/dokumen
func (h *Handler) UploadPeserta(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "pengguna belum terautentikasi"))
		return
	}

	idPeserta, err := strconv.Atoi(c.Param("id"))
	if err != nil || idPeserta <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "ID pendaftaran tidak valid"))
		return
	}

	// Batasi ukuran request body
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes+1024*1024)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "too large") {
			httpx.WriteError(c, httpx.Unprocessable("FILE_TERLALU_BESAR", "ukuran file melebihi batas maksimal"))
			return
		}
		httpx.WriteError(c, httpx.BadRequest("FILE_DIBUTUHKAN", "file multipart wajib disertakan"))
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadBytes {
		httpx.WriteError(c, httpx.Unprocessable("FILE_TERLALU_BESAR", "ukuran file melebihi batas maksimal"))
		return
	}

	jenisDokumen := strings.TrimSpace(c.PostForm("jenis_dokumen"))
	if jenisDokumen == "" {
		httpx.WriteError(c, httpx.BadRequest("JENIS_DOKUMEN_DIBUTUHKAN", "field jenis_dokumen wajib diisi"))
		return
	}

	resp, err := h.service.UploadPesertaDokumen(c.Request.Context(), actor, idPeserta, jenisDokumen, header.Filename, header.Size, file)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// UploadKelompok menangani POST /kkn/kelompok/:id/dokumen
func (h *Handler) UploadKelompok(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "pengguna belum terautentikasi"))
		return
	}

	idKelompok, err := strconv.Atoi(c.Param("id"))
	if err != nil || idKelompok <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "ID kelompok KKN tidak valid"))
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
		httpx.WriteError(c, httpx.BadRequest("FILE_DIBUTUHKAN", "file multipart wajib disertakan"))
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadBytes {
		httpx.WriteError(c, httpx.Unprocessable("FILE_TERLALU_BESAR", "ukuran file melebihi batas maksimal"))
		return
	}

	jenisDokumen := strings.TrimSpace(c.PostForm("jenis_dokumen"))
	if jenisDokumen == "" {
		httpx.WriteError(c, httpx.BadRequest("JENIS_DOKUMEN_DIBUTUHKAN", "field jenis_dokumen wajib diisi"))
		return
	}

	resp, err := h.service.UploadKelompokDokumen(c.Request.Context(), actor, idKelompok, jenisDokumen, header.Filename, header.Size, file)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// ListPeserta menangani GET /pendaftaran/:id/dokumen
func (h *Handler) ListPeserta(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "pengguna belum terautentikasi"))
		return
	}

	idPeserta, err := strconv.Atoi(c.Param("id"))
	if err != nil || idPeserta <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "ID pendaftaran tidak valid"))
		return
	}

	list, err := h.service.ListPesertaDokumen(c.Request.Context(), actor, idPeserta)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ListKelompok menangani GET /kkn/kelompok/:id/dokumen
func (h *Handler) ListKelompok(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "pengguna belum terautentikasi"))
		return
	}

	idKelompok, err := strconv.Atoi(c.Param("id"))
	if err != nil || idKelompok <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "ID kelompok KKN tidak valid"))
		return
	}

	list, err := h.service.ListKelompokDokumen(c.Request.Context(), actor, idKelompok)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// Download menangani GET /dokumen/:id/download
func (h *Handler) Download(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "pengguna belum terautentikasi"))
		return
	}

	idDokumen, err := strconv.Atoi(c.Param("id"))
	if err != nil || idDokumen <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "ID dokumen tidak valid"))
		return
	}

	rc, doc, err := h.service.DownloadDokumen(c.Request.Context(), actor, idDokumen)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	defer rc.Close()

	sanitizedName := SanitizeFilename(doc.NamaFile)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizedName))
	c.Header("Content-Type", doc.MimeType)
	if doc.UkuranFile > 0 {
		c.Header("Content-Length", strconv.FormatInt(doc.UkuranFile, 10))
	}

	_, _ = io.Copy(c.Writer, rc)
}

// Delete menangani DELETE /dokumen/:id
func (h *Handler) Delete(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "pengguna belum terautentikasi"))
		return
	}

	idDokumen, err := strconv.Atoi(c.Param("id"))
	if err != nil || idDokumen <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "ID dokumen tidak valid"))
		return
	}

	if err := h.service.DeleteDokumen(c.Request.Context(), actor, idDokumen); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dokumen berhasil dihapus"})
}
