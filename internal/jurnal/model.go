// Package jurnal mengelola data domain dan logika use-case untuk jurnal harian peserta program SPLK.
package jurnal

import "time"

const (
	StatusDraft     = "draft"
	StatusDisubmit  = "disubmit"
	StatusDisetujui = "disetujui"
	StatusDitolak   = "ditolak"
)

// JurnalHarian merepresentasikan baris pada tabel jurnal_harian.
type JurnalHarian struct {
	ID               int        `json:"id"`
	IDPesertaProgram int        `json:"id_peserta_program"`
	Tanggal          time.Time  `json:"tanggal"`
	Judul            string     `json:"judul"`
	Isi              string     `json:"isi"`
	Kendala          *string    `json:"kendala,omitempty"`
	Solusi           *string    `json:"solusi,omitempty"`
	Status           string     `json:"status"`
	CatatanReviewer  *string    `json:"catatan_reviewer,omitempty"`
	ReviewedBy       *int       `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// PendaftaranInfo merepresentasikan data pendaftaran dan periode untuk validasi BR-08.
type PendaftaranInfo struct {
	IDPesertaProgram int
	StatusPeserta    string
	IDPeriode        int
	TanggalMulai     time.Time
	TanggalSelesai   time.Time
}
