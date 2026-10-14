package tempat

// CreateTempatRequest adalah DTO request untuk membuat tempat program baru.
type CreateTempatRequest struct {
	JenisProgram string  `json:"jenis_program" binding:"required,oneof=magang asistensi_mengajar kkn"`
	Nama         string  `json:"nama" binding:"required,max=150"`
	Alamat       string  `json:"alamat" binding:"required"`
	Kota         *string `json:"kota,omitempty" binding:"omitempty,max=100"`
	Kontak       *string `json:"kontak,omitempty" binding:"omitempty,max=50"`
}

// UpdateTempatRequest adalah DTO request untuk memperbarui tempat program.
type UpdateTempatRequest struct {
	JenisProgram string  `json:"jenis_program" binding:"required,oneof=magang asistensi_mengajar kkn"`
	Nama         string  `json:"nama" binding:"required,max=150"`
	Alamat       string  `json:"alamat" binding:"required"`
	Kota         *string `json:"kota,omitempty" binding:"omitempty,max=100"`
	Kontak       *string `json:"kontak,omitempty" binding:"omitempty,max=50"`
	IsAktif      *bool   `json:"is_aktif,omitempty"`
}

// TempatFilter menyimpan opsi filter untuk GET /tempat-program.
type TempatFilter struct {
	JenisProgram string
	IsAktif      *bool
}
