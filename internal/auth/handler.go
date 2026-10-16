package auth

import (
	"github.com/gin-gonic/gin"

	"splk-backend/internal/middleware"
	"splk-backend/internal/platform/httpx"
)

// Handler menangani request HTTP untuk modul autentikasi.
type Handler struct {
	svc Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Login godoc
// @Summary Login akun pengguna
// @Description Autentikasi dengan username dan password, mengembalikan access token dan refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Kredensial login"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 429 {object} httpx.AppError
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	res, err := h.svc.Login(c.Request.Context(), req, ip, userAgent)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Refresh godoc
// @Summary Rotasi refresh token
// @Description Menukar refresh token yang valid dengan access token baru dan refresh token baru.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} RefreshResponse
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// Logout godoc
// @Summary Logout pengguna
// @Description Mencabut refresh token dan mencatat aktivitas logout.
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "Refresh token yang akan dicabut"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req LogoutRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	err := h.svc.Logout(c.Request.Context(), actor.UserID, req.RefreshToken, ip, userAgent)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.NoContent(c)
}

// Me godoc
// @Summary Profil pengguna saat ini
// @Description Mengembalikan profil singkat, roles, dan profil spesifik (mahasiswa/dosen/pegawai/lapangan).
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} MeResponse
// @Failure 401 {object} httpx.AppError
// @Router /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	res, err := h.svc.GetMe(c.Request.Context(), actor.UserID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, res)
}

// ChangePassword godoc
// @Summary Ubah password sendiri
// @Description Mengubah password akun pengguna yang sedang login.
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "Password lama dan baru"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpx.AppError
// @Failure 401 {object} httpx.AppError
// @Failure 422 {object} httpx.AppError
// @Router /auth/password [put]
func (h *Handler) ChangePassword(c *gin.Context) {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		httpx.WriteError(c, httpx.Unauthorized("UNAUTHORIZED", "autentikasi diperlukan"))
		return
	}

	var req ChangePasswordRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	err := h.svc.ChangePassword(c.Request.Context(), actor.UserID, req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"message": "password berhasil diubah",
	})
}
