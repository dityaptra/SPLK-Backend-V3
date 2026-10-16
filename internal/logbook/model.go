// Package logbook mengelola data domain dan use-case logbook bimbingan dosen untuk peserta program SPLK.
package logbook

import "time"

// LogbookBimbingan merepresentasikan baris pada tabel logbook_bimbingan.
type LogbookBimbingan struct {
	ID                  int        `json:"id"`
	IDPesertaProgram    int        `json:"id_peserta_program"`
	IDDosen             int        `json:"id_dosen"`
	Tanggal             time.Time  `json:"tanggal"`
	TopikBimbingan      string     `json:"topik_bimbingan"`
	Catatan             *string    `json:"catatan,omitempty"`
	PertemuanBerikutnya *time.Time `json:"pertemuan_berikutnya,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// PendaftaranInfo merepresentasikan data pendaftaran dan periode untuk validasi BR-08.
type PendaftaranInfo struct {
	IDPesertaProgram int
	StatusPeserta    string
	IDPeriode        int
	TanggalMulai     time.Time
	TanggalSelesai   time.Time
}
