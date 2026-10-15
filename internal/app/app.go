// Package app mengatur inisialisasi aplikasi, dependensi, dan pembuatan router Gin.
package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"splk-backend/internal/akademik"
	"splk-backend/internal/auth"
	"splk-backend/internal/config"
	"splk-backend/internal/middleware"
	"splk-backend/internal/periode"
	"splk-backend/internal/platform/clock"
	"splk-backend/internal/tempat"
	"splk-backend/internal/user"
)

// NewRouter membuat instance router Gin dengan seluruh middleware dasar dan rute modul terpasang.
func NewRouter(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 1. Pasang middleware dasar sesuai ARCHITECTURE.md bagian 5 dan 12
	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.CORS(cfg.CORSAllowedOrigins),
	)

	// 2. Route infrastruktur: /health
	router.GET("/health", HealthHandler(pool))

	// 3. Setup Clock dan Platform Dependencies
	loc := cfg.AppTimezone
	if loc == nil {
		loc = time.UTC
	}
	clk := clock.NewRealClock(loc)

	// 4. Inisialisasi Repositories dan Resolvers
	authRepo := auth.NewRepository()
	profileResolver := user.NewProfileResolver()
	userRepo := user.NewRepository()
	akademikRepo := akademik.NewRepository()
	tempatRepo := tempat.NewRepository()
	periodeRepo := periode.NewRepository()

	// Adapter untuk ActivityLogger periode
	actLogger := &activityLoggerAdapter{authRepo: authRepo, pool: pool}

	// 5. Inisialisasi Services
	authSvc := auth.NewService(authRepo, pool, cfg, clk, profileResolver)
	userSvc := user.NewService(userRepo, pool)
	akademikSvc := akademik.NewService(akademikRepo, pool)
	tempatSvc := tempat.NewService(tempatRepo, pool)
	periodeSvc := periode.NewService(periodeRepo, pool, clk, actLogger)

	// 6. Inisialisasi Handlers
	authHandler := auth.NewHandler(authSvc)
	userHandler := user.NewHandler(userSvc)
	akademikHandler := akademik.NewHandler(akademikSvc)
	tempatHandler := tempat.NewHandler(tempatSvc)
	periodeHandler := periode.NewHandler(periodeSvc)

	// 7. Middleware Keamanan
	authMiddleware := middleware.Auth(cfg.JWTSecret)
	adminRoleMiddleware := middleware.RequireRoles("admin")

	// 8. Rute /api/v1
	apiV1 := router.Group("/api/v1")
	{
		// Modul Auth
		authGroup := apiV1.Group("/auth")
		auth.RegisterRoutes(authGroup, authHandler, authMiddleware)

		// Rute Admin /api/v1/admin (RequireRoles("admin"))
		adminGroup := apiV1.Group("/admin", authMiddleware, adminRoleMiddleware)
		{
			user.RegisterAdminRoutes(adminGroup, userHandler)
			akademik.RegisterRoutes(adminGroup, nil, akademikHandler)
			tempat.RegisterRoutes(adminGroup, nil, tempatHandler)
			periode.RegisterRoutes(adminGroup, nil, periodeHandler)
		}

		// Rute Terautentikasi /api/v1/ (Akademik, Tempat, & Periode baca)
		authGroupGeneral := apiV1.Group("", authMiddleware)
		{
			akademik.RegisterRoutes(nil, authGroupGeneral, akademikHandler)
			tempat.RegisterRoutes(nil, authGroupGeneral, tempatHandler)
			periode.RegisterRoutes(nil, authGroupGeneral, periodeHandler)
		}
	}

	// 9. Alias Rute Langsung (tanpa /api/v1) untuk fleksibilitas klien
	{
		// /auth
		authDirect := router.Group("/auth")
		auth.RegisterRoutes(authDirect, authHandler, authMiddleware)

		// /admin
		adminDirect := router.Group("/admin", authMiddleware, adminRoleMiddleware)
		{
			user.RegisterAdminRoutes(adminDirect, userHandler)
			akademik.RegisterRoutes(adminDirect, nil, akademikHandler)
			tempat.RegisterRoutes(adminDirect, nil, tempatHandler)
			periode.RegisterRoutes(adminDirect, nil, periodeHandler)
		}

		// /tempat-program, /periode, /fakultas, /jurusan, /program-studi langsung
		authDirectGeneral := router.Group("", authMiddleware)
		{
			akademik.RegisterRoutes(nil, authDirectGeneral, akademikHandler)
			tempat.RegisterRoutes(nil, authDirectGeneral, tempatHandler)
			periode.RegisterRoutes(nil, authDirectGeneral, periodeHandler)
		}
	}

	return router
}

// HealthHandler memeriksa konektivitas database dan mengembalikan status sistem.
func HealthHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		if pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "error",
				"database": "down",
			})
			return
		}

		err := pool.Ping(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "health check: database tidak dapat dijangkau", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "error",
				"database": "down",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "up",
		})
	}
}

// activityLoggerAdapter mengadaptasi auth.Repository untuk implementasi periode.ActivityLogger.
type activityLoggerAdapter struct {
	authRepo auth.Repository
	pool     *pgxpool.Pool
}

func (a *activityLoggerAdapter) Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error {
	u, err := a.authRepo.FindUserByID(ctx, a.pool, userID)
	namaUser := "Unknown"
	if err == nil && u != nil {
		namaUser = u.Username
	}

	var ipPtr, uaPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	if userAgent != "" {
		uaPtr = &userAgent
	}

	return a.authRepo.CreateLogAktivitas(ctx, a.pool, &auth.LogAktivitas{
		UserID:       userID,
		NamaUser:     namaUser,
		IPUser:       ipPtr,
		AgentBrowser: uaPtr,
		Action:       action,
		Status:       status,
	})
}
