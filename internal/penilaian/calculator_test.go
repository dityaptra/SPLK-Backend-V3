package penilaian_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"splk-backend/internal/penilaian"
)

func TestScoreCalculator_TableDriven(t *testing.T) {
	calc := penilaian.NewScoreCalculator()

	tests := []struct {
		name          string
		items         []penilaian.NilaiInstrumen
		expectedNilai float64
		expectedHuruf string
	}{
		{
			name:          "kosong (0 instrumen)",
			items:         []penilaian.NilaiInstrumen{},
			expectedNilai: 0,
			expectedHuruf: "E",
		},
		{
			name: "satu instrumen nilai 85 bobot 100",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, NamaAspek: "Laporan", Bobot: 100, Nilai: 85},
			},
			expectedNilai: 85,
			expectedHuruf: "A",
		},
		{
			name: "satu instrumen nilai 92.5 bobot 50 (bobot tak ber-total 100)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, NamaAspek: "Laporan", Bobot: 50, Nilai: 92.5},
			},
			expectedNilai: 92.5,
			expectedHuruf: "A",
		},
		{
			name: "bobot tak ber-total 100 (total bobot 35)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 10, Nilai: 80},
				{IDInstrumen: 2, Bobot: 25, Nilai: 70},
			},
			// (10*80 + 25*70) / 35 = (800 + 1750) / 35 = 2550 / 35 = 72.85714... -> round to 72.86
			expectedNilai: 72.86,
			expectedHuruf: "B",
		},
		{
			name: "pembulatan 2 desimal (round up)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 1, Nilai: 85.556},
			},
			expectedNilai: 85.56,
			expectedHuruf: "A",
		},
		{
			name: "pembulatan 2 desimal (round down)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 1, Nilai: 85.554},
			},
			expectedNilai: 85.55,
			expectedHuruf: "A",
		},
		{
			name: "batas huruf A (tepat 85.00)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 85.00},
			},
			expectedNilai: 85.00,
			expectedHuruf: "A",
		},
		{
			name: "batas huruf B (84.99)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 84.99},
			},
			expectedNilai: 84.99,
			expectedHuruf: "B",
		},
		{
			name: "batas huruf B (tepat 70.00)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 70.00},
			},
			expectedNilai: 70.00,
			expectedHuruf: "B",
		},
		{
			name: "batas huruf C (69.99)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 69.99},
			},
			expectedNilai: 69.99,
			expectedHuruf: "C",
		},
		{
			name: "batas huruf C (tepat 55.00)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 55.00},
			},
			expectedNilai: 55.00,
			expectedHuruf: "C",
		},
		{
			name: "batas huruf D (54.99)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 54.99},
			},
			expectedNilai: 54.99,
			expectedHuruf: "D",
		},
		{
			name: "batas huruf D (tepat 40.00)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 40.00},
			},
			expectedNilai: 40.00,
			expectedHuruf: "D",
		},
		{
			name: "batas huruf E (39.99)",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 39.99},
			},
			expectedNilai: 39.99,
			expectedHuruf: "E",
		},
		{
			name: "nilai 0",
			items: []penilaian.NilaiInstrumen{
				{IDInstrumen: 1, Bobot: 100, Nilai: 0},
			},
			expectedNilai: 0,
			expectedHuruf: "E",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nilai, huruf := calc.Hitung(tt.items)
			assert.InDelta(t, tt.expectedNilai, nilai, 0.0001)
			assert.Equal(t, tt.expectedHuruf, huruf)
		})
	}
}
