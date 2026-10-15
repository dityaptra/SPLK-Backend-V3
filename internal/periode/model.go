// Package periode menyediakan model domain untuk manajemen periode program.
package periode

import "time"

// Konstanta status periode (satu arah: draft -> aktif -> selesai).
const (
	StatusDraft   = "draft"
	StatusAktif   = "aktif"
	StatusSelesai = "selesai"
)

// ValidStatusTransisi mendefinisikan transisi status yang diizinkan.
var ValidStatusTransisi = map[string]string{
	StatusDraft: StatusAktif,
	StatusAktif: StatusSelesai,
}

// Periode mewakili satu periode program MBKM.
type Periode struct {
	ID                 int        `json:"id"`
	Nama               string     `json:"nama"`
	JenisProgram       string     `json:"jenis_program"`
	TahunSemester      string     `json:"tahun_semester"`
	TanggalMulai       time.Time  `json:"tanggal_mulai"`
	TanggalSelesai     time.Time  `json:"tanggal_selesai"`
	TanggalBukaDaftar  *time.Time `json:"tanggal_buka_daftar,omitempty"`
	TanggalTutupDaftar *time.Time `json:"tanggal_tutup_daftar,omitempty"`
	Status             string     `json:"status"`
	DibuatOleh         *int       `json:"dibuat_oleh,omitempty"` // hanya untuk profil pegawai
	CreatedAt          time.Time  `json:"created_at"`
}

// IsTransisiValid memeriksa apakah transisi dari status saat ini ke statusBaru diizinkan.
func IsTransisiValid(statusSekarang, statusBaru string) bool {
	next, ok := ValidStatusTransisi[statusSekarang]
	return ok && next == statusBaru
}

// CanUpdate mengembalikan true bila periode masih boleh diubah (hanya saat status draft).
func CanUpdate(status string) bool {
	return status == StatusDraft
}
