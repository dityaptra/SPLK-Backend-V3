package kkn

import "time"

// CreateKelompokRequest mendefinisikan payload untuk pembuatan kelompok KKN baru oleh admin.
type CreateKelompokRequest struct {
	IDPeriode       int    `json:"id_periode" binding:"required"`
	IDTempatProgram int    `json:"id_tempat_program" binding:"required"`
	NamaKelompok    string `json:"nama_kelompok" binding:"required"`
}

// AddAnggotaRequest mendefinisikan payload untuk penambahan anggota kelompok KKN.
type AddAnggotaRequest struct {
	IDPesertaProgram []int `json:"id_peserta_program" binding:"required"`
	IDKetua          *int  `json:"id_ketua,omitempty"`
}

// SetDosenPembimbingKelompokRequest mendefinisikan payload penetapan dosen pembimbing kelompok KKN.
type SetDosenPembimbingKelompokRequest struct {
	IDDosen int `json:"id_dosen" binding:"required"`
}

// SetPembimbingLapanganKelompokRequest mendefinisikan payload penetapan pembimbing lapangan (kepala desa) kelompok KKN.
type SetPembimbingLapanganKelompokRequest struct {
	IDPembimbingLapangan int `json:"id_pembimbing_lapangan" binding:"required"`
}

// AnggotaKelompokResponse mendefinisikan representasi JSON untuk anggota kelompok KKN.
type AnggotaKelompokResponse struct {
	IDPesertaProgram int       `json:"id_peserta_program"`
	IDMahasiswa      int       `json:"id_mahasiswa"`
	NIM              string    `json:"nim"`
	Nama             string    `json:"nama"`
	Prodi            string    `json:"prodi"`
	IsKetua          bool      `json:"is_ketua"`
	Status           string    `json:"status"`
	TanggalDaftar    time.Time `json:"tanggal_daftar"`
}

// KelompokSummaryResponse mendefinisikan representasi ringkas kelompok KKN pada daftar kelompok.
type KelompokSummaryResponse struct {
	ID                     int       `json:"id"`
	IDPeriode              int       `json:"id_periode"`
	NamaPeriode            string    `json:"nama_periode"`
	IDTempatProgram        int       `json:"id_tempat_program"`
	NamaTempat             string    `json:"nama_tempat"`
	AlamatTempat           string    `json:"alamat_tempat"`
	IDDosenPembimbing      *int      `json:"id_dosen_pembimbing,omitempty"`
	NamaDosenPembimbing    *string   `json:"nama_dosen_pembimbing,omitempty"`
	IDPembimbingLapangan   *int      `json:"id_pembimbing_lapangan,omitempty"`
	NamaPembimbingLapangan *string   `json:"nama_pembimbing_lapangan,omitempty"`
	NamaKelompok           string    `json:"nama_kelompok"`
	JumlahAnggota          int       `json:"jumlah_anggota"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// KelompokDetailResponse mendefinisikan detail lengkap kelompok KKN beserta anggota dan ketua.
type KelompokDetailResponse struct {
	ID                     int                       `json:"id"`
	IDPeriode              int                       `json:"id_periode"`
	NamaPeriode            string                    `json:"nama_periode"`
	IDTempatProgram        int                       `json:"id_tempat_program"`
	NamaTempat             string                    `json:"nama_tempat"`
	AlamatTempat           string                    `json:"alamat_tempat"`
	IDDosenPembimbing      *int                      `json:"id_dosen_pembimbing,omitempty"`
	NamaDosenPembimbing    *string                   `json:"nama_dosen_pembimbing,omitempty"`
	IDPembimbingLapangan   *int                      `json:"id_pembimbing_lapangan,omitempty"`
	NamaPembimbingLapangan *string                   `json:"nama_pembimbing_lapangan,omitempty"`
	NamaKelompok           string                    `json:"nama_kelompok"`
	Ketua                  *AnggotaKelompokResponse  `json:"ketua,omitempty"`
	Anggota                []AnggotaKelompokResponse `json:"anggota"`
	CreatedAt              time.Time                 `json:"created_at"`
	UpdatedAt              time.Time                 `json:"updated_at"`
}
