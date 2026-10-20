package pengumuman

import "time"

// CreatePengumumanRequest mendefinisikan payload untuk pembuatan pengumuman baru oleh admin.
type CreatePengumumanRequest struct {
	Judul      string  `json:"judul" binding:"required,max=150"`
	Isi        string  `json:"isi" binding:"required"`
	TargetRole *string `json:"target_role,omitempty"` // Jika null/kosong, berarti untuk semua role
}

// UpdatePengumumanRequest mendefinisikan payload untuk memperbarui pengumuman oleh admin.
type UpdatePengumumanRequest struct {
	Judul      string  `json:"judul" binding:"required,max=150"`
	Isi        string  `json:"isi" binding:"required"`
	TargetRole *string `json:"target_role,omitempty"`
}

// PengumumanResponse mendefinisikan format JSON response untuk detail dan item pengumuman.
type PengumumanResponse struct {
	ID            int       `json:"id"`
	Judul         string    `json:"judul"`
	Isi           string    `json:"isi"`
	DibuatOleh    *int      `json:"dibuat_oleh,omitempty"`
	NamaPembuat   *string   `json:"nama_pembuat,omitempty"`
	TanggalTerbit time.Time `json:"tanggal_terbit"`
	TargetRole    *string   `json:"target_role,omitempty"`
}

// PengumumanFilter menyimpan kriteria filter untuk daftar pengumuman.
type PengumumanFilter struct {
	TargetRole string
	Search     string
}

// ToResponse mengonversi PengumumanDetail ke PengumumanResponse.
func ToResponse(p PengumumanDetail) PengumumanResponse {
	return PengumumanResponse{
		ID:            p.ID,
		Judul:         p.Judul,
		Isi:           p.Isi,
		DibuatOleh:    p.DibuatOleh,
		NamaPembuat:   p.NamaPembuat,
		TanggalTerbit: p.TanggalTerbit,
		TargetRole:    p.TargetRole,
	}
}

// ToResponses mengonversi slice PengumumanDetail ke slice PengumumanResponse.
func ToResponses(list []PengumumanDetail) []PengumumanResponse {
	res := make([]PengumumanResponse, len(list))
	for i, item := range list {
		res[i] = ToResponse(item)
	}
	return res
}
