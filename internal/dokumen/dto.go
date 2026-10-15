package dokumen

import "time"

// DokumenResponse adalah representasi aman dokumen yang disajikan ke klien.
// Kolom file_url / path internal server TIDAK PERNAH dikembalikan ke klien.
type DokumenResponse struct {
	ID               int        `json:"id"`
	IDPesertaProgram *int       `json:"id_peserta_program,omitempty"`
	IDKelompokKKN    *int       `json:"id_kelompok_kkn,omitempty"`
	JenisDokumen     string     `json:"jenis_dokumen"`
	NamaFile         string     `json:"nama_file"`
	UkuranFile       int64      `json:"ukuran_file"`
	MimeType         string     `json:"mime_type"`
	DiunggahOleh     int        `json:"diunggah_oleh"`
	UploadedAt       time.Time  `json:"uploaded_at"`
	IsVerified       bool       `json:"is_verified"`
	VerifiedBy       *int       `json:"verified_by,omitempty"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
}

// ToDokumenResponse mengubah entitas model Dokumen menjadi DokumenResponse aman.
func ToDokumenResponse(d Dokumen) DokumenResponse {
	return DokumenResponse{
		ID:               d.ID,
		IDPesertaProgram: d.IDPesertaProgram,
		IDKelompokKKN:    d.IDKelompokKKN,
		JenisDokumen:     d.JenisDokumen,
		NamaFile:         d.NamaFile,
		UkuranFile:       d.UkuranFile,
		MimeType:         d.MimeType,
		DiunggahOleh:     d.DiunggahOleh,
		UploadedAt:       d.UploadedAt,
		IsVerified:       d.IsVerified,
		VerifiedBy:       d.VerifiedBy,
		VerifiedAt:       d.VerifiedAt,
	}
}

// ToDokumenResponses mengubah kumpulan entitas Dokumen menjadi kumpulan DokumenResponse.
func ToDokumenResponses(list []Dokumen) []DokumenResponse {
	if list == nil {
		return []DokumenResponse{}
	}
	res := make([]DokumenResponse, len(list))
	for i, d := range list {
		res[i] = ToDokumenResponse(d)
	}
	return res
}
