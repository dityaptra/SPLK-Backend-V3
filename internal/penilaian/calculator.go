package penilaian

import (
	"math"
)

// NilaiInstrumen memuat data nilai dan bobot suatu instrumen untuk dihitung nilai akhirnya.
type NilaiInstrumen struct {
	IDInstrumen int
	NamaAspek   string
	RolePenilai string
	Bobot       float64
	Nilai       float64
}

// ScoreCalculator mendefinisikan kontrak penghitungan nilai akhir dan penentuan grade huruf.
// Interface ini memungkinkan formula penilaian dan standar huruf dikonfigurasi ulang di kemudian hari.
type ScoreCalculator interface {
	Hitung(items []NilaiInstrumen) (nilaiAkhir float64, gradeHuruf string)
	HitungProgram(jenisProgram string, items []NilaiInstrumen) (nilaiAkhir float64, gradeHuruf string)
}

// defaultScoreCalculator mengimplementasikan ScoreCalculator standar SPLK.
type defaultScoreCalculator struct{}

// NewScoreCalculator membuat instance default ScoreCalculator.
func NewScoreCalculator() ScoreCalculator {
	return &defaultScoreCalculator{}
}

// Hitung menghitung nilai akhir secara default.
func (c *defaultScoreCalculator) Hitung(items []NilaiInstrumen) (float64, string) {
	return c.HitungProgram("", items)
}

// HitungProgram menghitung nilai akhir dengan aturan khusus per program:
// - Magang: 60% Dosen Pembimbing + 40% Pembimbing Lapangan (Mentor)
// - Program lain (AM & KKN): SUM(nilai x bobot) / SUM(bobot)
func (c *defaultScoreCalculator) HitungProgram(jenisProgram string, items []NilaiInstrumen) (float64, string) {
	if len(items) == 0 {
		return 0, "E"
	}

	if jenisProgram == "magang" {
		var sumPembimbingNilai, sumPembimbingBobot float64
		var sumLapanganNilai, sumLapanganBobot float64
		hasPembimbing := false
		hasLapangan := false

		for _, item := range items {
			if item.RolePenilai == RoleDosenPembimbing {
				sumPembimbingNilai += item.Nilai * item.Bobot
				sumPembimbingBobot += item.Bobot
				hasPembimbing = true
			} else if item.RolePenilai == RolePembimbingLapangan {
				sumLapanganNilai += item.Nilai * item.Bobot
				sumLapanganBobot += item.Bobot
				hasLapangan = true
			}
		}

		// Jika kedua penilai (pembimbing dan lapangan) telah mengisi, hitung dengan proporsi 60% : 40%
		if hasPembimbing && hasLapangan && sumPembimbingBobot > 0 && sumLapanganBobot > 0 {
			nilaiPemb := sumPembimbingNilai / sumPembimbingBobot
			nilaiLap := sumLapanganNilai / sumLapanganBobot
			raw := (0.60 * nilaiPemb) + (0.40 * nilaiLap)
			nilaiAkhir := math.Round(raw*100) / 100
			return nilaiAkhir, konversiHuruf(nilaiAkhir)
		}
	}

	// Default fallback jika belum magang lengkap atau program AM / KKN
	var sumBobotNilai float64
	var sumBobot float64

	for _, item := range items {
		sumBobotNilai += item.Nilai * item.Bobot
		sumBobot += item.Bobot
	}

	if sumBobot <= 0 {
		return 0, "E"
	}

	raw := sumBobotNilai / sumBobot
	nilaiAkhir := math.Round(raw*100) / 100

	gradeHuruf := konversiHuruf(nilaiAkhir)
	return nilaiAkhir, gradeHuruf
}

func konversiHuruf(nilai float64) string {
	switch {
	case nilai >= 85:
		return "A"
	case nilai >= 70:
		return "B"
	case nilai >= 55:
		return "C"
	case nilai >= 40:
		return "D"
	default:
		return "E"
	}
}
