package logbook

import "time"

// CreateLogbookRequest mendefinisikan payload untuk pembuatan logbook bimbingan baru oleh dosen pembimbing.
type CreateLogbookRequest struct {
	Tanggal             string  `json:"tanggal" binding:"required"` // Format: YYYY-MM-DD
	TopikBimbingan      string  `json:"topik_bimbingan" binding:"required"`
	Catatan             *string `json:"catatan,omitempty"`
	PertemuanBerikutnya *string `json:"pertemuan_berikutnya,omitempty"` // Format: YYYY-MM-DD
}

// UpdateLogbookRequest mendefinisikan payload untuk pembaruan logbook bimbingan oleh dosen pembuatnya.
type UpdateLogbookRequest struct {
	Tanggal             *string `json:"tanggal,omitempty"` // Format: YYYY-MM-DD
	TopikBimbingan      string  `json:"topik_bimbingan" binding:"required"`
	Catatan             *string `json:"catatan,omitempty"`
	PertemuanBerikutnya *string `json:"pertemuan_berikutnya,omitempty"` // Format: YYYY-MM-DD
}

// LogbookResponse mendefinisikan representasi JSON untuk logbook bimbingan.
type LogbookResponse struct {
	ID                  int       `json:"id"`
	IDPesertaProgram    int       `json:"id_peserta_program"`
	IDDosen             int       `json:"id_dosen"`
	NamaDosen           string    `json:"nama_dosen,omitempty"`
	Tanggal             string    `json:"tanggal"` // Format: YYYY-MM-DD
	TopikBimbingan      string    `json:"topik_bimbingan"`
	Catatan             *string   `json:"catatan,omitempty"`
	PertemuanBerikutnya *string   `json:"pertemuan_berikutnya,omitempty"` // Format: YYYY-MM-DD
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ToLogbookResponse mengonversi entitas LogbookBimbingan menjadi LogbookResponse.
func ToLogbookResponse(l LogbookBimbingan, namaDosen string) LogbookResponse {
	resp := LogbookResponse{
		ID:               l.ID,
		IDPesertaProgram: l.IDPesertaProgram,
		IDDosen:          l.IDDosen,
		NamaDosen:        namaDosen,
		Tanggal:          l.Tanggal.Format("2006-01-02"),
		TopikBimbingan:   l.TopikBimbingan,
		Catatan:          l.Catatan,
		CreatedAt:        l.CreatedAt,
		UpdatedAt:        l.UpdatedAt,
	}
	if l.PertemuanBerikutnya != nil {
		s := l.PertemuanBerikutnya.Format("2006-01-02")
		resp.PertemuanBerikutnya = &s
	}
	return resp
}
