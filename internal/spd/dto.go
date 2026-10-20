package spd

import "time"

// CreateSPDRequest adalah payload untuk membuat Surat Perjalanan Dinas oleh pegawai/admin.
type CreateSPDRequest struct {
	IDDosen          int    `json:"id_dosen" binding:"required"`
	IDPesertaProgram int    `json:"id_peserta_program" binding:"required"`
	Tujuan           string `json:"tujuan" binding:"required,max=150"`
	TanggalBerangkat string `json:"tanggal_berangkat" binding:"required"` // Format: YYYY-MM-DD
	TanggalKembali   string `json:"tanggal_kembali" binding:"required"`   // Format: YYYY-MM-DD
}

// UpdateSPDRequest adalah payload untuk memperbarui data Surat Perjalanan Dinas.
type UpdateSPDRequest struct {
	IDDosen          int    `json:"id_dosen" binding:"required"`
	IDPesertaProgram int    `json:"id_peserta_program" binding:"required"`
	Tujuan           string `json:"tujuan" binding:"required,max=150"`
	TanggalBerangkat string `json:"tanggal_berangkat" binding:"required"`
	TanggalKembali   string `json:"tanggal_kembali" binding:"required"`
}

// SPDResponse adalah format response JSON untuk detail surat perjalanan dinas.
type SPDResponse struct {
	ID               int       `json:"id"`
	IDDosen          int       `json:"id_dosen"`
	NamaDosen        string    `json:"nama_dosen"`
	NIPDosen         string    `json:"nip_dosen"`
	IDPesertaProgram int       `json:"id_peserta_program"`
	NamaMahasiswa    string    `json:"nama_mahasiswa"`
	NIMMahasiswa     string    `json:"nim_mahasiswa"`
	JenisProgram     string    `json:"jenis_program"`
	Tujuan           string    `json:"tujuan"`
	TanggalBerangkat string    `json:"tanggal_berangkat"` // YYYY-MM-DD
	TanggalKembali   string    `json:"tanggal_kembali"`   // YYYY-MM-DD
	FileURL          *string   `json:"file_url,omitempty"`
	DibuatOleh       int       `json:"dibuat_oleh"`
	NamaPegawai      string    `json:"nama_pegawai"`
	CreatedAt        time.Time `json:"created_at"`
}

// SPDFilter menyimpan parameter penyaringan daftar SPD.
type SPDFilter struct {
	IDDosen          int
	IDPesertaProgram int
	Search           string
}

// ToResponse mengonversi entitas SPDDetail ke SPDResponse.
func ToResponse(d SPDDetail) SPDResponse {
	return SPDResponse{
		ID:               d.ID,
		IDDosen:          d.IDDosen,
		NamaDosen:        d.NamaDosen,
		NIPDosen:         d.NIPDosen,
		IDPesertaProgram: d.IDPesertaProgram,
		NamaMahasiswa:    d.NamaMahasiswa,
		NIMMahasiswa:     d.NIMMahasiswa,
		JenisProgram:     d.JenisProgram,
		Tujuan:           d.Tujuan,
		TanggalBerangkat: d.TanggalBerangkat.Format("2006-01-02"),
		TanggalKembali:   d.TanggalKembali.Format("2006-01-02"),
		FileURL:          d.FileURL,
		DibuatOleh:       d.DibuatOleh,
		NamaPegawai:      d.NamaPegawai,
		CreatedAt:        d.CreatedAt,
	}
}

// ToResponses mengonversi slice SPDDetail ke slice SPDResponse.
func ToResponses(list []SPDDetail) []SPDResponse {
	res := make([]SPDResponse, len(list))
	for i, item := range list {
		res[i] = ToResponse(item)
	}
	return res
}
