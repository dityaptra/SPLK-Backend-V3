package periode

// PeriodeFilter menyimpan opsi filter untuk GET /periode.
type PeriodeFilter struct {
	JenisProgram string
	Status       string
}

// CreatePeriodeRequest adalah DTO request untuk POST /admin/periode.
type CreatePeriodeRequest struct {
	Nama               string  `json:"nama" binding:"required,max=100"`
	JenisProgram       string  `json:"jenis_program" binding:"required,oneof=magang asistensi_mengajar kkn"`
	TahunSemester      string  `json:"tahun_semester" binding:"required,max=20"`
	TanggalMulai       string  `json:"tanggal_mulai" binding:"required"`   // format: YYYY-MM-DD
	TanggalSelesai     string  `json:"tanggal_selesai" binding:"required"` // format: YYYY-MM-DD
	TanggalBukaDaftar  *string `json:"tanggal_buka_daftar,omitempty"`      // opsional
	TanggalTutupDaftar *string `json:"tanggal_tutup_daftar,omitempty"`     // opsional
}

// UpdatePeriodeRequest adalah DTO request untuk PUT /admin/periode/:id.
// Hanya bisa diubah bila status masih 'draft'.
type UpdatePeriodeRequest struct {
	Nama               string  `json:"nama" binding:"required,max=100"`
	TahunSemester      string  `json:"tahun_semester" binding:"required,max=20"`
	TanggalMulai       string  `json:"tanggal_mulai" binding:"required"`
	TanggalSelesai     string  `json:"tanggal_selesai" binding:"required"`
	TanggalBukaDaftar  *string `json:"tanggal_buka_daftar,omitempty"`
	TanggalTutupDaftar *string `json:"tanggal_tutup_daftar,omitempty"`
}

// PeriodeDetail adalah DTO response detail periode.
type PeriodeDetail struct {
	ID                 int     `json:"id"`
	Nama               string  `json:"nama"`
	JenisProgram       string  `json:"jenis_program"`
	TahunSemester      string  `json:"tahun_semester"`
	TanggalMulai       string  `json:"tanggal_mulai"`
	TanggalSelesai     string  `json:"tanggal_selesai"`
	TanggalBukaDaftar  *string `json:"tanggal_buka_daftar,omitempty"`
	TanggalTutupDaftar *string `json:"tanggal_tutup_daftar,omitempty"`
	Status             string  `json:"status"`
	DibuatOleh         *int    `json:"dibuat_oleh,omitempty"`
	CreatedAt          string  `json:"created_at"`
}
