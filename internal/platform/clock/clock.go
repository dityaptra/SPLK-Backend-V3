// Package clock menyediakan abstraksi waktu (Clock) untuk memudahkan pengujian unit
// dan perhitungan tanggal berdasarkan zona waktu yang dikonfigurasi (APP_TIMEZONE).
package clock

import (
	"sync"
	"time"
)

// Clock adalah interface abstraksi waktu yang dapat disuntikkan ke service.
type Clock interface {
	// Now mengembalikan waktu saat ini dalam format UTC.
	Now() time.Time
	// Today mengembalikan tanggal hari ini pada zona waktu yang dikonfigurasi tanpa komponen jam (00:00:00).
	Today() time.Time
}

// RealClock adalah implementasi Clock menggunakan waktu sistem aktual.
type RealClock struct {
	loc *time.Location
}

// NewRealClock membuat instance RealClock dengan zona waktu tertentu.
func NewRealClock(loc *time.Location) Clock {
	if loc == nil {
		loc = time.UTC
	}
	return &RealClock{loc: loc}
}

// Now mengembalikan waktu sistem saat ini dalam format UTC.
func (r *RealClock) Now() time.Time {
	return time.Now().UTC()
}

// Today mengembalikan tanggal saat ini pada zona waktu r.loc dengan jam 00:00:00.
func (r *RealClock) Today() time.Time {
	t := time.Now().In(r.loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, r.loc)
}

// MockClock adalah implementasi tiruan Clock untuk keperluan pengujian.
type MockClock struct {
	mu  sync.RWMutex
	now time.Time
	loc *time.Location
}

// NewMockClock membuat instance MockClock dengan waktu awal dan zona waktu tertentu.
func NewMockClock(initialTime time.Time, loc *time.Location) *MockClock {
	if loc == nil {
		loc = time.UTC
	}
	return &MockClock{
		now: initialTime,
		loc: loc,
	}
}

// Now mengembalikan waktu tiruan saat ini dalam format UTC.
func (m *MockClock) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.now.UTC()
}

// Today mengembalikan tanggal tiruan saat ini pada zona waktu m.loc dengan jam 00:00:00.
func (m *MockClock) Today() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t := m.now.In(m.loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, m.loc)
}

// SetNow memperbarui waktu tiruan.
func (m *MockClock) SetNow(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = t
}

// Advance memajukan waktu tiruan sebanyak durasi d.
func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = m.now.Add(d)
}
