// Package config membaca dan memvalidasi konfigurasi dari environment variable.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config menyimpan seluruh konfigurasi aplikasi.
type Config struct {
	AppEnv      string // "development" atau "production"
	AppPort     string
	AppTimezone *time.Location

	DatabaseURL   string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	UploadDir   string
	MaxUploadMB int64

	CORSAllowedOrigins []string
	LogLevel           string

	// Hanya untuk cmd/seed
	SeedAdminUsername string
	SeedAdminEmail    string
	SeedAdminPassword string
}

// Load membaca konfigurasi dari environment variable. Jika file .env ada,
// variabel dimuat terlebih dahulu dengan godotenv.
func Load() (*Config, error) {
	// Muat .env hanya bila file ada (development). Jangan gagal bila tidak ada.
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			return nil, fmt.Errorf("gagal memuat .env: %w", err)
		}
	}

	cfg := &Config{}

	// APP_ENV (default: development)
	cfg.AppEnv = getEnv("APP_ENV", "development")
	if cfg.AppEnv != "development" && cfg.AppEnv != "production" {
		return nil, fmt.Errorf("APP_ENV harus 'development' atau 'production', diterima: %q", cfg.AppEnv)
	}

	// APP_PORT (wajib berupa port valid)
	cfg.AppPort = getEnv("APP_PORT", "8080")
	port, err := strconv.Atoi(cfg.AppPort)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("APP_PORT harus bilangan bulat antara 1-65535, diterima: %q", cfg.AppPort)
	}

	// APP_TIMEZONE (wajib bisa dimuat)
	tzName := getEnv("APP_TIMEZONE", "Asia/Makassar")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("APP_TIMEZONE %q tidak valid: %w", tzName, err)
	}
	cfg.AppTimezone = loc

	// DATABASE_URL (wajib)
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL wajib diisi")
	}

	// JWT_SECRET (wajib, minimal 32 karakter)
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET wajib diisi dan minimal 32 karakter (diterima: %d karakter)", len(cfg.JWTSecret))
	}

	// JWT_ACCESS_TTL (default: 15m)
	accessTTL := getEnv("JWT_ACCESS_TTL", "15m")
	cfg.JWTAccessTTL, err = time.ParseDuration(accessTTL)
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL %q tidak valid: %w", accessTTL, err)
	}

	// JWT_REFRESH_TTL (default: 168h = 7 hari)
	refreshTTL := getEnv("JWT_REFRESH_TTL", "168h")
	cfg.JWTRefreshTTL, err = time.ParseDuration(refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL %q tidak valid: %w", refreshTTL, err)
	}

	// UPLOAD_DIR (default: ./uploads)
	cfg.UploadDir = getEnv("UPLOAD_DIR", "./uploads")

	// MAX_UPLOAD_MB (wajib > 0, default: 5)
	maxUpload := getEnv("MAX_UPLOAD_MB", "5")
	cfg.MaxUploadMB, err = strconv.ParseInt(maxUpload, 10, 64)
	if err != nil || cfg.MaxUploadMB <= 0 {
		return nil, fmt.Errorf("MAX_UPLOAD_MB harus bilangan bulat > 0, diterima: %q", maxUpload)
	}

	// CORS_ALLOWED_ORIGINS (dipisah koma)
	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.CORSAllowedOrigins = strings.Split(origins, ",")
	for i := range cfg.CORSAllowedOrigins {
		cfg.CORSAllowedOrigins[i] = strings.TrimSpace(cfg.CORSAllowedOrigins[i])
	}

	// LOG_LEVEL (default: info)
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	// Seed admin (opsional, hanya untuk cmd/seed)
	cfg.SeedAdminUsername = os.Getenv("SEED_ADMIN_USERNAME")
	cfg.SeedAdminEmail = os.Getenv("SEED_ADMIN_EMAIL")
	cfg.SeedAdminPassword = os.Getenv("SEED_ADMIN_PASSWORD")

	return cfg, nil
}

// getEnv mengembalikan nilai variabel lingkungan atau fallback bila kosong.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
