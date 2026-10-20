package penilaian

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul penilaian.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler penilaian baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// CreateInstrumen godoc
// @Summary Tambah instrumen penilaian (admin)
// @Tags Penilaian
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateInstrumenRequest true "Data instrumen baru"
// @Success 201 {object} InstrumenResponse
// @Failure 400 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /admin/instrumen-penilaian [post]
func (h *Handler) CreateInstrumen(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req CreateInstrumenRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.CreateInstrumen(c.Request.Context(), actor, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.Created(c, res)
}

// GetInstrumen godoc
// @Summary Detail instrumen penilaian
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Instrumen"
// @Success 200 {object} InstrumenResponse
// @Failure 404 {object} httpx.AppError
// @Router /admin/instrumen-penilaian/{id} [get]
func (h *Handler) GetInstrumen(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.WriteError(c, httpx.BadRequest("ID_TIDAK_VALID", "parameter id harus berupa angka positif"))
		return
	}

	res, err := h.svc.GetInstrumen(c.Request.Context(), id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// UpdateInstrumen godoc
// @Summary Perbarui instrumen penilaian
// @Tags Penilaian
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Instrumen"
// @Param request body UpdateInstrumenRequest true "Data perubahan instrumen"
// @Success 200 {object} InstrumenResponse
// @Failure 404 {object} httpx.AppError
// @Router /admin/instrumen-penilaian/{id} [put]
func (h *Handler) UpdateInstrumen(c *gin.Context) {
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

	var req UpdateInstrumenRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.UpdateInstrumen(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(c, httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan"))
			return
		}
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// DeleteInstrumen godoc
// @Summary Hapus instrumen penilaian
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Instrumen"
// @Success 200 {object} map[string]string
// @Failure 404 {object} httpx.AppError
// @Router /admin/instrumen-penilaian/{id} [delete]
func (h *Handler) DeleteInstrumen(c *gin.Context) {
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

	if err := h.svc.DeleteInstrumen(c.Request.Context(), actor, id, c.ClientIP(), c.Request.UserAgent()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(c, httpx.NotFound("INSTRUMEN_TIDAK_DITEMUKAN", "instrumen penilaian tidak ditemukan"))
			return
		}
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{"pesan": "instrumen penilaian berhasil dihapus"})
}

// ListInstrumen godoc
// @Summary Daftar instrumen penilaian
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param jenis_program query string false "Filter jenis program"
// @Param role_penilai query string false "Filter role penilai"
// @Success 200 {array} InstrumenResponse
// @Router /admin/instrumen-penilaian [get]
func (h *Handler) ListInstrumen(c *gin.Context) {
	jenis := c.Query("jenis_program")
	role := c.Query("role_penilai")

	items, err := h.svc.ListInstrumen(c.Request.Context(), jenis, role)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, items)
}

// GetRingkasanBobot godoc
// @Summary Ringkasan total bobot per program dan role penilai
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Success 200 {object} RingkasanBobotResponse
// @Router /admin/instrumen-penilaian/ringkasan [get]
func (h *Handler) GetRingkasanBobot(c *gin.Context) {
	res, err := h.svc.GetRingkasanBobot(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetFormPenilaian godoc
// @Summary Form instrumen penilaian untuk penilai sah
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Success 200 {object} FormPenilaianResponse
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /pendaftaran/{id}/form-penilaian [get]
func (h *Handler) GetFormPenilaian(c *gin.Context) {
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

	res, err := h.svc.GetFormPenilaian(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// SubmitPenilaian godoc
// @Summary Kirim penilaian peserta (penilai sah)
// @Description Mengisi penilaian per instrumen. Bila seluruh instrumen dari seluruh penilai telah terisi, nilai akhir dihitung otomatis dan dikunci.
// @Tags Penilaian
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body SubmitPenilaianRequest true "Daftar nilai instrumen"
// @Success 200 {object} SubmitPenilaianResponse
// @Failure 403 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/penilaian [post]
func (h *Handler) SubmitPenilaian(c *gin.Context) {
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

	var req SubmitPenilaianRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.SubmitPenilaian(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetNilai godoc
// @Summary Rincian nilai dan nilai akhir peserta
// @Description Menampilkan rincian nilai. Mahasiswa pemilik hanya dapat melihat rincian setelah nilai akhir terisi.
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Success 200 {object} RincianNilaiResponse
// @Failure 403 {object} httpx.AppError
// @Router /pendaftaran/{id}/nilai [get]
func (h *Handler) GetNilai(c *gin.Context) {
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

	res, err := h.svc.GetNilai(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// HitungNilaiManual godoc
// @Summary Hitung manual nilai akhir oleh admin
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Success 200 {object} HitungNilaiManualResponse
// @Failure 422 {object} httpx.AppError
// @Router /admin/pendaftaran/{id}/hitung-nilai [post]
func (h *Handler) HitungNilaiManual(c *gin.Context) {
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

	res, err := h.svc.HitungNilaiManual(c.Request.Context(), actor, id, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// GetVerifikasiPenguji godoc
// @Summary Ambil form/status verifikasi kelulusan dosen penguji (Gambar 4)
// @Description Mengambil data verifikasi kelulusan mahasiswa magang oleh dosen penguji
// @Tags Penilaian
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Success 200 {object} VerifikasiPengujiResponse
// @Failure 403 {object} httpx.AppError
// @Failure 404 {object} httpx.AppError
// @Router /pendaftaran/{id}/verifikasi-penguji [get]
func (h *Handler) GetVerifikasiPenguji(c *gin.Context) {
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

	res, err := h.svc.GetVerifikasiPenguji(c.Request.Context(), actor, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// SubmitVerifikasiPenguji godoc
// @Summary Simpan penentuan kelulusan oleh dosen penguji (Gambar 4)
// @Description Menyimpan keputusan status kelulusan (Lulus / Tidak Lulus) dan catatan oleh dosen penguji pada program magang
// @Tags Penilaian
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID Pendaftaran"
// @Param request body SubmitVerifikasiPengujiRequest true "Data verifikasi kelulusan"
// @Success 200 {object} VerifikasiPengujiResponse
// @Failure 403 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /pendaftaran/{id}/verifikasi-penguji [post]
func (h *Handler) SubmitVerifikasiPenguji(c *gin.Context) {
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

	var req SubmitVerifikasiPengujiRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.SubmitVerifikasiPenguji(c.Request.Context(), actor, id, req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

