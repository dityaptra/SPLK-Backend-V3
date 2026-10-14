package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"splk-backend/internal/platform/httpx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestPagination(t *testing.T) {
	t.Run("Default dan Batas Maksimal", func(t *testing.T) {
		tests := []struct {
			name           string
			queryString    string
			expectedPage   int
			expectedLimit  int
			expectedOffset int
		}{
			{
				name:           "Tanpa parameter query menghasilkan nilai default",
				queryString:    "",
				expectedPage:   httpx.DefaultPage,
				expectedLimit:  httpx.DefaultLimit,
				expectedOffset: 0,
			},
			{
				name:           "Parameter valid",
				queryString:    "?page=3&limit=15",
				expectedPage:   3,
				expectedLimit:  15,
				expectedOffset: 30, // (3-1) * 15
			},
			{
				name:           "Limit melebihi batas maksimal dipotong ke 100",
				queryString:    "?page=1&limit=500",
				expectedPage:   1,
				expectedLimit:  httpx.MaxLimit,
				expectedOffset: 0,
			},
			{
				name:           "Nilai non-angka dan negatif kembali ke default",
				queryString:    "?page=-5&limit=abc",
				expectedPage:   httpx.DefaultPage,
				expectedLimit:  httpx.DefaultLimit,
				expectedOffset: 0,
			},
			{
				name:           "Page 0 kembali ke default 1",
				queryString:    "?page=0&limit=10",
				expectedPage:   httpx.DefaultPage,
				expectedLimit:  10,
				expectedOffset: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				req, _ := http.NewRequest(http.MethodGet, "/test"+tt.queryString, nil)
				c.Request = req

				p := httpx.ParsePagination(c)
				assert.Equal(t, tt.expectedPage, p.Page)
				assert.Equal(t, tt.expectedLimit, p.Limit)
				assert.Equal(t, tt.expectedOffset, p.Offset())
			})
		}
	})
}
