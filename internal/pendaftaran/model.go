// Package pendaftaran mengelola siklus pendaftaran program SPLK, penempatan,
// evaluasi kesiapan, dan otorisasi akses per relasi.
package pendaftaran

import "time"

// Konstanta status peserta_program sesuai skema database v2.
const (
	StatusDiajukan = "diajukan"
	StatusDitolak  = "ditolak"
	StatusBerjalan = "berjalan"
	StatusSelesai  = "selesai"
)

// PesertaProgram merepresentasikan entitas inti peserta_program di database.
type PesertaProgram struct {
	ID                   int        `json:"id"`
	IDMahasiswa          int        `json:"id_mahasiswa"`
	IDPeriode            int        `json:"id_periode"`
	JenisProgram         string     `json:"jenis_program"`
	IDTempatProgram      *int       `json:"id_tempat_program,omitempty"`
	IDDosenPembimbing    *int       `json:"id_dosen_pembimbing,omitempty"`
	IDDosenPenguji       *int       `json:"id_dosen_penguji,omitempty"`
	IDPembimbingLapangan *int       `json:"id_pembimbing_lapangan,omitempty"`
	IDKelompokKKN        *int       `json:"id_kelompok_kkn,omitempty"`
	IsKetua              bool       `json:"is_ketua"`
	Status               string     `json:"status"`
	AlasanTolak          *string    `json:"alasan_tolak,omitempty"`
	TanggalDaftar        time.Time  `json:"tanggal_daftar"`
	TanggalMulaiAktif    *time.Time `json:"tanggal_mulai_aktif,omitempty"`
	TanggalSelesaiAktif  *time.Time `json:"tanggal_selesai_aktif,omitempty"`
	NilaiAkhir           *float64   `json:"nilai_akhir,omitempty"`
	GradeHuruf           *string    `json:"grade_huruf,omitempty"`
	SKSdiakui            *int       `json:"sks_diakui,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// SudahDinilai mengembalikan true bila peserta berstatus selesai dan nilai akhir sudah terhitung.
// Merujuk RULES.md bagian 7 nomor 6: status dinilai tidak ada di enum, melainkan diturunkan.
func SudahDinilai(p *PesertaProgram) bool {
	if p == nil {
		return false
	}
	return p.Status == StatusSelesai && p.NilaiAkhir != nil
}

// TempatProgramCheck menyimpan atribut tempat program untuk validasi penempatan.
type TempatProgramCheck struct {
	ID           int
	JenisProgram string
	IsAktif      bool
}

// PembimbingLapanganCheck menyimpan atribut pembimbing lapangan untuk validasi penempatan.
type PembimbingLapanganCheck struct {
	ID              int
	Jenis           string
	IDTempatProgram int
}

// DosenCheck menyimpan atribut dosen untuk validasi penetapan pembimbing/penguji.
type DosenCheck struct {
	ID   int
	Nama string
}
