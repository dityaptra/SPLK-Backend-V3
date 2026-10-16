// @title SPLK Backend API
// @version 1.0
// @description Dokumentasi REST API Sistem Pengelolaan Magang, Asistensi Mengajar, dan KKN (SPLK) UPA TIK Undiksha.
// @termsOfService http://swagger.io/terms/

// @contact.name UPA TIK Undiksha
// @contact.email tik@undiksha.ac.id

// @license.name Proprietary
// @license.url https://undiksha.ac.id

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Format token: "Bearer <token_jwt>"

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"splk-backend/internal/app"
	"splk-backend/internal/config"
	"splk-backend/internal/database"
)

func main() {
	// 1. Load konfigurasi
	cfg, err := config.Load()
	if err != nil {
		slog.Error("gagal memuat konfigurasi", "error", err)
		os.Exit(1)
	}

	// 2. Setup logger
	setupLogger(cfg)
	logger := slog.Default()

	logger.Info("memulai server SPLK", "env", cfg.AppEnv, "port", cfg.AppPort)

	// 3. Buat pool database
	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		logger.Error("gagal membuat pool database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info("koneksi database berhasil")

	// 4. Inisialisasi router aplikasi melalui internal/app
	router := app.NewRouter(cfg, pool, logger)

	// 5. Jalankan server HTTP
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.AppPort),
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("server berjalan", "addr", srv.Addr)

	// 6. Tunggu sinyal shutdown
	sig := <-quit
	logger.Info("menerima sinyal shutdown", "signal", sig)

	// Graceful shutdown dengan timeout 10 detik
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("gagal shutdown server", "error", err)
		os.Exit(1)
	}

	logger.Info("server berhenti")
}

// setupLogger mengatur slog sesuai lingkungan: JSON untuk production, teks untuk development.
func setupLogger(cfg *config.Config) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if cfg.AppEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
