// Package kkn mendefinisikan model domain, DTO, repositori, dan logika bisnis untuk cabang KKN.
package kkn

import "time"

// KelompokKKN merepresentasikan entitas kelompok KKN pada tabel kelompok_kkn.
type KelompokKKN struct {
	ID                   int       `json:"id"`
	IDPeriode            int       `json:"id_periode"`
	IDTempatProgram      int       `json:"id_tempat_program"`
	IDDosenPembimbing    *int      `json:"id_dosen_pembimbing,omitempty"`
	IDPembimbingLapangan *int      `json:"id_pembimbing_lapangan,omitempty"`
	NamaKelompok         string    `json:"nama_kelompok"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// AnggotaKelompok merepresentasikan data anggota peserta program dalam suatu kelompok KKN.
type AnggotaKelompok struct {
	IDPesertaProgram     int       `json:"id_peserta_program"`
	IDMahasiswa          int       `json:"id_mahasiswa"`
	NIM                  string    `json:"nim"`
	NamaMahasiswa        string    `json:"nama_mahasiswa"`
	NamaProdi            string    `json:"nama_prodi"`
	IsKetua              bool      `json:"is_ketua"`
	Status               string    `json:"status"`
	TanggalDaftar        time.Time `json:"tanggal_daftar"`
	IDDosenPembimbing    *int      `json:"id_dosen_pembimbing,omitempty"`
	IDPembimbingLapangan *int      `json:"id_pembimbing_lapangan,omitempty"`
	IDDosenPenguji       *int      `json:"id_dosen_penguji,omitempty"`
}

// PesertaKKNInfo merepresentasikan cuplikan data pendaftaran peserta untuk validasi KKN.
type PesertaKKNInfo struct {
	ID                   int    `json:"id"`
	IDMahasiswa          int    `json:"id_mahasiswa"`
	IDPeriode            int    `json:"id_periode"`
	JenisProgram         string `json:"jenis_program"`
	Status               string `json:"status"`
	IDKelompokKKN        *int   `json:"id_kelompok_kkn,omitempty"`
	IDDosenPembimbing    *int   `json:"id_dosen_pembimbing,omitempty"`
	IDDosenPenguji       *int   `json:"id_dosen_penguji,omitempty"`
	IDPembimbingLapangan *int   `json:"id_pembimbing_lapangan,omitempty"`
	IsKetua              bool   `json:"is_ketua"`
}
