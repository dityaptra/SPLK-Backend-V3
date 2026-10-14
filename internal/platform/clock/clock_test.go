package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/platform/clock"
)

func TestClock(t *testing.T) {
	makassarLoc, err := time.LoadLocation("Asia/Makassar")
	require.NoError(t, err)

	t.Run("Today pada zona Asia/Makassar di sekitar pergantian hari UTC", func(t *testing.T) {
		// Kasus 1: Pukul 15:59:00 UTC pada 14 Oktober 2026
		// Di Asia/Makassar (UTC+8), waktu lokal adalah 23:59:00 pada 14 Oktober 2026.
		// Today() harus menghasilkan tanggal 14 Oktober 2026 jam 00:00:00.
		timeBeforeMidnightUTC := time.Date(2026, time.October, 14, 15, 59, 0, 0, time.UTC)
		mockClock1 := clock.NewMockClock(timeBeforeMidnightUTC, makassarLoc)

		today1 := mockClock1.Today()
		assert.Equal(t, 2026, today1.Year())
		assert.Equal(t, time.October, today1.Month())
		assert.Equal(t, 14, today1.Day())
		assert.Equal(t, 0, today1.Hour())
		assert.Equal(t, 0, today1.Minute())
		assert.Equal(t, 0, today1.Second())
		assert.Equal(t, "Asia/Makassar", today1.Location().String())

		// Kasus 2: Pukul 16:01:00 UTC pada 14 Oktober 2026
		// Di UTC tanggal masih 14 Oktober.
		// Tetapi di Asia/Makassar (UTC+8), waktu lokal sudah 00:01:00 pada 15 Oktober 2026.
		// Today() harus menghasilkan tanggal 15 Oktober 2026 jam 00:00:00.
		timeAfterMidnightUTC := time.Date(2026, time.October, 14, 16, 1, 0, 0, time.UTC)
		mockClock2 := clock.NewMockClock(timeAfterMidnightUTC, makassarLoc)

		today2 := mockClock2.Today()
		assert.Equal(t, 2026, today2.Year())
		assert.Equal(t, time.October, today2.Month())
		assert.Equal(t, 15, today2.Day(), "harus sudah berganti hari ke 15 di zona Asia/Makassar")
		assert.Equal(t, 0, today2.Hour())
		assert.Equal(t, 0, today2.Minute())
		assert.Equal(t, 0, today2.Second())
		assert.Equal(t, "Asia/Makassar", today2.Location().String())
	})

	t.Run("RealClock Now mengembalikan UTC", func(t *testing.T) {
		rc := clock.NewRealClock(makassarLoc)
		now := rc.Now()
		assert.Equal(t, time.UTC, now.Location())

		today := rc.Today()
		assert.Equal(t, 0, today.Hour())
		assert.Equal(t, 0, today.Minute())
		assert.Equal(t, 0, today.Second())
		assert.Equal(t, "Asia/Makassar", today.Location().String())
	})
}
