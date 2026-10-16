// Package pendaftaran menyediakan DTO request dan response untuk modul pendaftaran.
package pendaftaran

// CreatePendaftaranRequest adalah DTO request untuk POST /pendaftaran.
type CreatePendaftaranRequest struct {
	IDPeriode int `json:"id_periode" binding:"required"`
}

// PenempatanMagangRequest adalah DTO request untuk PUT /pendaftaran/:id/penempatan-magang.
type PenempatanMagangRequest struct {
	IDTempatProgram      int `json:"id_tempat_program" binding:"required"`
	IDPembimbingLapangan int `json:"id_pembimbing_lapangan" binding:"required"`
}

// SetDosenPembimbingRequest adalah DTO request untuk PUT /pendaftaran/:id/dosen-pembimbing.
type SetDosenPembimbingRequest struct {
	IDDosen int `json:"id_dosen" binding:"required"`
}

// AdminPenempatanRequest adalah DTO request untuk PUT /admin/pendaftaran/:id/penempatan.
type AdminPenempatanRequest struct {
	IDTempatProgram      int `json:"id_tempat_program" binding:"required"`
	IDPembimbingLapangan int `json:"id_pembimbing_lapangan" binding:"required"`
}

// SetDosenPengujiRequest adalah DTO request untuk PUT /admin/pendaftaran/:id/dosen-penguji.
type SetDosenPengujiRequest struct {
	IDDosen int `json:"id_dosen" binding:"required"`
}

// PendaftaranFilter adalah opsi penyaringan untuk GET /pendaftaran.
type PendaftaranFilter struct {
	IDPeriode    *int
	JenisProgram string
	Status       string
	Q            string // pencarian nama atau nim mahasiswa
}

// MahasiswaSummary berisi rangkuman profil mahasiswa untuk pendaftaran.
type MahasiswaSummary struct {
	ID       int    `json:"id"`
	NIM      string `json:"nim"`
	Nama     string `json:"nama"`
	Angkatan int    `json:"angkatan"`
	Prodi    string `json:"prodi"`
}

// PeriodeSummary berisi rangkuman informasi periode untuk pendaftaran.
type PeriodeSummary struct {
	ID            int    `json:"id"`
	Nama          string `json:"nama"`
	JenisProgram  string `json:"jenis_program"`
	TahunSemester string `json:"tahun_semester"`
	Status        string `json:"status"`
}

// TempatSummary berisi rangkuman tempat program untuk pendaftaran.
type TempatSummary struct {
	ID           int    `json:"id"`
	Nama         string `json:"nama"`
	JenisProgram string `json:"jenis_program"`
	Alamat       string `json:"alamat"`
}

// DosenSummary berisi rangkuman profil dosen untuk pendaftaran.
type DosenSummary struct {
	ID   int    `json:"id"`
	NIP  string `json:"nip"`
	Nama string `json:"nama"`
}

// PembimbingLapanganSummary berisi rangkuman profil pembimbing lapangan untuk pendaftaran.
type PembimbingLapanganSummary struct {
	ID    int    `json:"id"`
	Nama  string `json:"nama"`
	Jenis string `json:"jenis"`
}

// KelompokKKNSummary berisi rangkuman kelompok KKN untuk pendaftaran.
type KelompokKKNSummary struct {
	ID           int    `json:"id"`
	NamaKelompok string `json:"nama_kelompok"`
}

// PendaftaranDetail adalah response detail lengkap dari GET /pendaftaran/:id.
type PendaftaranDetail struct {
	ID                  int                        `json:"id"`
	Mahasiswa           MahasiswaSummary           `json:"mahasiswa"`
	Periode             PeriodeSummary             `json:"periode"`
	JenisProgram        string                     `json:"jenis_program"`
	Status              string                     `json:"status"`
	AlasanTolak         *string                    `json:"alasan_tolak,omitempty"`
	TanggalDaftar       string                     `json:"tanggal_daftar"`
	TanggalMulaiAktif   *string                    `json:"tanggal_mulai_aktif,omitempty"`
	TanggalSelesaiAktif *string                    `json:"tanggal_selesai_aktif,omitempty"`
	NilaiAkhir          *float64                   `json:"nilai_akhir,omitempty"`
	GradeHuruf          *string                    `json:"grade_huruf,omitempty"`
	SKSdiakui           *int                       `json:"sks_diakui,omitempty"`
	TempatProgram       *TempatSummary             `json:"tempat_program,omitempty"`
	DosenPembimbing     *DosenSummary              `json:"dosen_pembimbing,omitempty"`
	DosenPenguji        *DosenSummary              `json:"dosen_penguji,omitempty"`
	PembimbingLapangan  *PembimbingLapanganSummary `json:"pembimbing_lapangan,omitempty"`
	KelompokKKN         *KelompokKKNSummary        `json:"kelompok_kkn,omitempty"`
	Kesiapan            ReadinessResult            `json:"kesiapan"`
	CreatedAt           string                     `json:"created_at"`
	UpdatedAt           string                     `json:"updated_at"`
}

// PendaftaranListItem adalah item dalam response daftar GET /pendaftaran.
type PendaftaranListItem struct {
	ID            int      `json:"id"`
	IDMahasiswa   int      `json:"id_mahasiswa"`
	NIMMahasiswa  string   `json:"nim_mahasiswa"`
	NamaMahasiswa string   `json:"nama_mahasiswa"`
	IDPeriode     int      `json:"id_periode"`
	NamaPeriode   string   `json:"nama_periode"`
	JenisProgram  string   `json:"jenis_program"`
	Status        string   `json:"status"`
	TanggalDaftar string   `json:"tanggal_daftar"`
	NilaiAkhir    *float64 `json:"nilai_akhir,omitempty"`
	GradeHuruf    *string  `json:"grade_huruf,omitempty"`
}
