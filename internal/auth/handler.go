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

// Login menangani endpoint POST /auth/login.
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

// Refresh menangani endpoint POST /auth/refresh.
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

// Logout menangani endpoint POST /auth/logout.
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

// Me menangani endpoint GET /auth/me.
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

// ChangePassword menangani endpoint PUT /auth/password.
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
