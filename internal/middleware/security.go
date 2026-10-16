package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"splk-backend/internal/platform/httpx"
)

// SecurityHeaders menambahkan header keamanan HTTP standar pada response.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// BodySizeLimit membatasi ukuran request body untuk mencegah DoS lewat payload besar.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 10 * 1024 * 1024 // default 10MB jika tidak diatur atau <= 0
	}
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

type rateLimitEntry struct {
	timestamps []time.Time
}

// LoginRateLimiter membatasi jumlah percobaan login per IP dalam rentang waktu tertentu.
// Standar: 5 percobaan per menit per IP, mengembalikan HTTP 429 jika melampaui batas.
type LoginRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	records map[string]*rateLimitEntry
}

// NewLoginRateLimiter membuat rate limiter baru dengan pembersihan entri kedaluwarsa secara berkala.
func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	limiter := &LoginRateLimiter{
		limit:   limit,
		window:  window,
		records: make(map[string]*rateLimitEntry),
	}

	// Goroutine pembersih entri kadaluarsa setiap window durasi
	go func() {
		ticker := time.NewTicker(window)
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return limiter
}

func (l *LoginRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	for ip, entry := range l.records {
		valid := make([]time.Time, 0)
		for _, t := range entry.timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(l.records, ip)
		} else {
			entry.timestamps = valid
		}
	}
}

// Middleware mengembalikan HandlerFunc Gin untuk membatasi request per IP.
func (l *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		cutoff := now.Add(-l.window)

		l.mu.Lock()
		entry, ok := l.records[ip]
		if !ok {
			entry = &rateLimitEntry{timestamps: make([]time.Time, 0, l.limit)}
			l.records[ip] = entry
		}

		// Filter timestamps dalam jendela aktif
		valid := make([]time.Time, 0, len(entry.timestamps))
		for _, t := range entry.timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}

		if len(valid) >= l.limit {
			l.mu.Unlock()
			c.Abort()
			httpx.WriteError(c, httpx.TooManyRequests(
				"TERLALU_BANYAK_PERMINTAAN",
				"terlalu banyak percobaan login dari alamat IP ini. Silakan coba lagi setelah beberapa saat.",
			))
			return
		}

		valid = append(valid, now)
		entry.timestamps = valid
		l.mu.Unlock()

		c.Next()
	}
}
