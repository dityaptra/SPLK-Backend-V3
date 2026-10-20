package penilaian

// CreateInstrumenRequest payload untuk membuat instrumen penilaian baru.
type CreateInstrumenRequest struct {
	JenisProgram string  `json:"jenis_program" binding:"required"`
	NamaAspek    string  `json:"nama_aspek" binding:"required"`
	Bobot        float64 `json:"bobot" binding:"required,gt=0,lte=100"`
	RolePenilai  string  `json:"role_penilai" binding:"required"`
}

// UpdateInstrumenRequest payload untuk memperbarui instrumen penilaian.
type UpdateInstrumenRequest struct {
	JenisProgram *string  `json:"jenis_program"`
	NamaAspek    *string  `json:"nama_aspek"`
	Bobot        *float64 `json:"bobot"`
	RolePenilai  *string  `json:"role_penilai"`
}

// InstrumenResponse representasi JSON instrumen penilaian.
type InstrumenResponse struct {
	ID           int     `json:"id"`
	JenisProgram string  `json:"jenis_program"`
	NamaAspek    string  `json:"nama_aspek"`
	Bobot        float64 `json:"bobot"`
	RolePenilai  string  `json:"role_penilai"`
}

// RingkasanBobotRole memuat ringkasan bobot per role penilai.
type RingkasanBobotRole struct {
	RolePenilai     string  `json:"role_penilai"`
	TotalBobot      float64 `json:"total_bobot"`
	JumlahInstrumen int     `json:"jumlah_instrumen"`
}

// RingkasanBobotProgram memuat ringkasan bobot per jenis program.
type RingkasanBobotProgram struct {
	JenisProgram          string               `json:"jenis_program"`
	TotalBobotKeseluruhan float64              `json:"total_bobot_keseluruhan"`
	PerRole               []RingkasanBobotRole `json:"per_role"`
}

// RingkasanBobotResponse respon untuk endpoint ringkasan bobot.
type RingkasanBobotResponse struct {
	Programs []RingkasanBobotProgram `json:"programs"`
}

// SubmitPenilaianItem item nilai per instrumen.
type SubmitPenilaianItem struct {
	IDInstrumen int      `json:"id_instrumen" binding:"required"`
	Nilai       *float64 `json:"nilai" binding:"required"`
	Catatan     *string  `json:"catatan"`
}

// SubmitPenilaianRequest payload untuk mengirim penilaian.
type SubmitPenilaianRequest struct {
	Items []SubmitPenilaianItem `json:"items" binding:"required,min=1"`
}

// SubmitPenilaianResponse hasil pengiriman penilaian.
type SubmitPenilaianResponse struct {
	PendaftaranID        int      `json:"pendaftaran_id"`
	SemuaInstrumenTerisi bool     `json:"semua_instrumen_terisi"`
	NilaiAkhir           *float64 `json:"nilai_akhir,omitempty"`
	GradeHuruf           *string  `json:"grade_huruf,omitempty"`
	Pesan                string   `json:"pesan"`
}

// FormPenilaianItemResponse item instrumen pada form penilaian penilai.
type FormPenilaianItemResponse struct {
	IDInstrumen  int      `json:"id_instrumen"`
	NamaAspek    string   `json:"nama_aspek"`
	Bobot        float64  `json:"bobot"`
	RolePenilai  string   `json:"role_penilai"`
	Nilai        *float64 `json:"nilai"`
	Catatan      *string  `json:"catatan"`
	SudahDinilai bool     `json:"sudah_dinilai"`
}

// FormPenilaianResponse respon form penilaian untuk penilai sah.
type FormPenilaianResponse struct {
	PendaftaranID int                         `json:"pendaftaran_id"`
	JenisProgram  string                      `json:"jenis_program"`
	RolePenilai   string                      `json:"role_penilai"`
	Terkunci      bool                        `json:"terkunci"`
	NilaiAkhir    *float64                    `json:"nilai_akhir,omitempty"`
	GradeHuruf    *string                     `json:"grade_huruf,omitempty"`
	Items         []FormPenilaianItemResponse `json:"items"`
}

// NilaiItemDetailResponse rincian nilai individual per instrumen.
type NilaiItemDetailResponse struct {
	IDInstrumen  int     `json:"id_instrumen"`
	NamaAspek    string  `json:"nama_aspek"`
	Bobot        float64 `json:"bobot"`
	RolePenilai  string  `json:"role_penilai"`
	Nilai        float64 `json:"nilai"`
	Catatan      *string `json:"catatan"`
	NamaPenilai  string  `json:"nama_penilai"`
	TanggalNilai string  `json:"tanggal_nilai"`
}

// RincianNilaiResponse respon rincian nilai akhir dan komponennya.
type RincianNilaiResponse struct {
	PendaftaranID int                       `json:"pendaftaran_id"`
	JenisProgram  string                    `json:"jenis_program"`
	StatusPeserta string                    `json:"status_peserta"`
	NilaiAkhir    *float64                  `json:"nilai_akhir"`
	GradeHuruf    *string                   `json:"grade_huruf"`
	Items         []NilaiItemDetailResponse `json:"items"`
}

// HitungNilaiManualResponse respon penghitungan manual nilai akhir oleh admin.
type HitungNilaiManualResponse struct {
	PendaftaranID int     `json:"pendaftaran_id"`
	NilaiAkhir    float64 `json:"nilai_akhir"`
	GradeHuruf    string  `json:"grade_huruf"`
	Pesan         string  `json:"pesan"`
}

// SubmitVerifikasiPengujiRequest payload input penentuan kelulusan oleh dosen penguji (Gambar 4).
type SubmitVerifikasiPengujiRequest struct {
	StatusKelulusan string `json:"status_kelulusan" binding:"required"` // "Lulus" atau "Tidak Lulus"
	Catatan         string `json:"catatan"`
}

// VerifikasiPengujiResponse respon tampilan halaman verifikasi kelulusan dosen penguji.
type VerifikasiPengujiResponse struct {
	PendaftaranID     int      `json:"pendaftaran_id"`
	NIM               string   `json:"nim"`
	NamaMahasiswa     string   `json:"nama_mahasiswa"`
	JenisProgram      string   `json:"jenis_program"`
	IDDosenPenguji    *int     `json:"id_dosen_penguji,omitempty"`
	NamaDosenPenguji  *string  `json:"nama_dosen_penguji,omitempty"`
	StatusKelulusan   string   `json:"status_kelulusan"` // "Lulus", "Tidak Lulus", atau "" (belum diverifikasi)
	Catatan           string   `json:"catatan"`
	TanggalVerifikasi *string  `json:"tanggal_verifikasi,omitempty"`
	NilaiAkhir        *float64 `json:"nilai_akhir,omitempty"`
	GradeHuruf        *string  `json:"grade_huruf,omitempty"`
	CanVerify         bool     `json:"can_verify"`
}

