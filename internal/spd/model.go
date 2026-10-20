// Package spd mengelola data domain dan use-case untuk Surat Perjalanan Dinas (SPD).
package spd

import "time"

// SuratPerjalananDinas merepresentasikan baris pada tabel surat_perjalanan_dinas.
type SuratPerjalananDinas struct {
	ID               int       `json:"id"`
	IDDosen          int       `json:"id_dosen"`
	IDPesertaProgram int       `json:"id_peserta_program"`
	Tujuan           string    `json:"tujuan"`
	TanggalBerangkat time.Time `json:"tanggal_berangkat"`
	TanggalKembali   time.Time `json:"tanggal_kembali"`
	FileURL          *string   `json:"file_url,omitempty"`
	DibuatOleh       int       `json:"dibuat_oleh"` // pegawai.id
	CreatedAt        time.Time `json:"created_at"`
}

// SPDDetail merepresentasikan detail lengkap surat perjalanan dinas dengan relasi dosen, mahasiswa, dan pegawai.
type SPDDetail struct {
	ID               int       `json:"id"`
	IDDosen          int       `json:"id_dosen"`
	NamaDosen        string    `json:"nama_dosen"`
	NIPDosen         string    `json:"nip_dosen"`
	IDPesertaProgram int       `json:"id_peserta_program"`
	NamaMahasiswa    string    `json:"nama_mahasiswa"`
	NIMMahasiswa     string    `json:"nim_mahasiswa"`
	JenisProgram     string    `json:"jenis_program"`
	Tujuan           string    `json:"tujuan"`
	TanggalBerangkat time.Time `json:"tanggal_berangkat"`
	TanggalKembali   time.Time `json:"tanggal_kembali"`
	FileURL          *string   `json:"file_url,omitempty"`
	DibuatOleh       int       `json:"dibuat_oleh"`
	NamaPegawai      string    `json:"nama_pegawai"`
	CreatedAt        time.Time `json:"created_at"`
}
