// Package dokumen menangani penyimpanan dan pengelolaan dokumen peserta dan kelompok KKN.
package dokumen

import (
	"time"
)

// Jenis dokumen yang sah
const (
	JenisDokumenMou          = "mou"
	JenisDokumenProposal     = "proposal"
	JenisDokumenProker       = "proker"
	JenisDokumenLaporanAkhir = "laporan_akhir"
	JenisDokumenLainnya      = "lainnya"
)

// IsValidJenisDokumen memeriksa apakah jenis dokumen termasuk dalam daftar yang diizinkan.
func IsValidJenisDokumen(j string) bool {
	switch j {
	case JenisDokumenMou, JenisDokumenProposal, JenisDokumenProker, JenisDokumenLaporanAkhir, JenisDokumenLainnya:
		return true
	default:
		return false
	}
}

// Dokumen merepresentasikan baris pada tabel dokumen.
type Dokumen struct {
	ID               int        `json:"id"`
	IDPesertaProgram *int       `json:"id_peserta_program"`
	JenisDokumen     string     `json:"jenis_dokumen"`
	NamaFile         string     `json:"nama_file"`
	FileURL          string     `json:"file_url"` // Path relatif di penyimpanan lokal
	UkuranFile       int64      `json:"ukuran_file"`
	MimeType         string     `json:"mime_type"`
	DiunggahOleh     int        `json:"diunggah_oleh"`
	UploadedAt       time.Time  `json:"uploaded_at"`
	IDKelompokKKN    *int       `json:"id_kelompok_kkn"`
	IsVerified       bool       `json:"is_verified"`
	VerifiedBy       *int       `json:"verified_by"`
	VerifiedAt       *time.Time `json:"verified_at"`
}

// PendaftaranInfo menyimpan informasi penting pendaftaran untuk validasi dokumen.
type PendaftaranInfo struct {
	ID           int
	JenisProgram string
	Status       string
	IDMahasiswa  int
}
