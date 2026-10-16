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
}

// defaultScoreCalculator mengimplementasikan ScoreCalculator standar SPLK.
type defaultScoreCalculator struct{}

// NewScoreCalculator membuat instance default ScoreCalculator.
func NewScoreCalculator() ScoreCalculator {
	return &defaultScoreCalculator{}
}

// Hitung menghitung nilai akhir dengan formula:
// nilai_akhir = ROUND( SUM(nilai_i * bobot_i) / SUM(bobot_i), 2 )
//
// ASUMSI: A4 - rumus nilai akhir sementara: SUM(nilai x bobot) / SUM(bobot) atas seluruh instrumen program tersebut.
// TODO(mentor): konfirmasi apakah aturan bobot total harus tepat 100 atau boleh variatif, dan apakah ada bobot lintas instrumen program.
//
// ASUMSI: A5 - konversi nilai huruf sementara: A >= 85, B >= 70, C >= 55, D >= 40, selainnya E.
// TODO(mentor): konfirmasi rentang dan skala nilai huruf resmi universitas (misal: A, A-, B+, dst).
//
// ASUMSI: A6 - sks_diakui tidak diisi (dibiarkan NULL) sampai aturan konversi SKS ditentukan.
func (c *defaultScoreCalculator) Hitung(items []NilaiInstrumen) (float64, string) {
	if len(items) == 0 {
		return 0, "E"
	}

	var sumBobotNilai float64
	var sumBobot float64

	for _, item := range items {
		sumBobotNilai += item.Nilai * item.Bobot
		sumBobot += item.Bobot
	}

	if sumBobot <= 0 {
		return 0, "E"
	}

	// Pembulatan 2 desimal: ROUND(..., 2)
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
