package jurnal

import "time"

// CreateJurnalRequest mendefinisikan payload untuk pembuatan jurnal harian baru.
type CreateJurnalRequest struct {
	Tanggal string  `json:"tanggal" binding:"required"` // Format: YYYY-MM-DD
	Judul   string  `json:"judul" binding:"required"`
	Isi     string  `json:"isi" binding:"required"`
	Kendala *string `json:"kendala,omitempty"`
	Solusi  *string `json:"solusi,omitempty"`
	Status  string  `json:"status" binding:"required"` // draft | disubmit
}

// UpdateJurnalRequest mendefinisikan payload untuk pembaruan jurnal harian berstatus draft.
type UpdateJurnalRequest struct {
	Tanggal *string `json:"tanggal,omitempty"` // Format: YYYY-MM-DD (opsional jika tanggal diubah)
	Judul   string  `json:"judul" binding:"required"`
	Isi     string  `json:"isi" binding:"required"`
	Kendala *string `json:"kendala,omitempty"`
	Solusi  *string `json:"solusi,omitempty"`
	Status  string  `json:"status" binding:"required"` // draft | disubmit
}

// JurnalResponse mendefinisikan representasi JSON untuk jurnal harian.
type JurnalResponse struct {
	ID               int       `json:"id"`
	IDPesertaProgram int       `json:"id_peserta_program"`
	Tanggal          string    `json:"tanggal"` // Format: YYYY-MM-DD
	Judul            string    `json:"judul"`
	Isi              string    `json:"isi"`
	Kendala          *string   `json:"kendala,omitempty"`
	Solusi           *string   `json:"solusi,omitempty"`
	Status           string    `json:"status"`
	CatatanReviewer  *string   `json:"catatan_reviewer,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// JurnalFilter menyimpan parameter penyaringan daftar jurnal harian.
type JurnalFilter struct {
	TanggalMulai   string `json:"tanggal_mulai"`
	TanggalSelesai string `json:"tanggal_selesai"`
	Status         string `json:"status"`
}

// ToJurnalResponse mengonversi entitas JurnalHarian menjadi JurnalResponse.
func ToJurnalResponse(j JurnalHarian) JurnalResponse {
	return JurnalResponse{
		ID:               j.ID,
		IDPesertaProgram: j.IDPesertaProgram,
		Tanggal:          j.Tanggal.Format("2006-01-02"),
		Judul:            j.Judul,
		Isi:              j.Isi,
		Kendala:          j.Kendala,
		Solusi:           j.Solusi,
		Status:           j.Status,
		CatatanReviewer:  j.CatatanReviewer,
		CreatedAt:        j.CreatedAt,
		UpdatedAt:        j.UpdatedAt,
	}
}

// ToJurnalResponses mengonversi slice entitas JurnalHarian menjadi slice JurnalResponse.
func ToJurnalResponses(list []JurnalHarian) []JurnalResponse {
	res := make([]JurnalResponse, len(list))
	for i, j := range list {
		res[i] = ToJurnalResponse(j)
	}
	return res
}
