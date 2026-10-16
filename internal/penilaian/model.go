package penilaian

import "time"

// Konstanta role penilai instrumen penilaian
const (
	RoleDosenPembimbing    = "dosen_pembimbing"
	RoleDosenPenguji       = "dosen_penguji"
	RolePembimbingLapangan = "pembimbing_lapangan"
)

// Konstanta jenis program
const (
	JenisMagang = "magang"
	JenisAM     = "asistensi_mengajar"
	JenisKKN    = "kkn"
)

// InstrumenPenilaian merepresentasikan template instrumen penilaian.
type InstrumenPenilaian struct {
	ID           int     `json:"id"`
	JenisProgram string  `json:"jenis_program"`
	NamaAspek    string  `json:"nama_aspek"`
	Bobot        float64 `json:"bobot"`
	RolePenilai  string  `json:"role_penilai"`
}

// Penilaian merepresentasikan nilai aktual yang diberikan penilai untuk satu aspek instrumen.
type Penilaian struct {
	ID               int       `json:"id"`
	IDPesertaProgram int       `json:"id_peserta_program"`
	IDInstrumen      int       `json:"id_instrumen"`
	IDPenilai        int       `json:"id_penilai"`
	Nilai            float64   `json:"nilai"`
	Catatan          *string   `json:"catatan,omitempty"`
	TanggalNilai     time.Time `json:"tanggal_nilai"`
}

// PesertaPenilaianInfo memuat informasi pendaftaran yang dibutuhkan untuk proses penilaian.
type PesertaPenilaianInfo struct {
	ID                   int
	IDMahasiswa          int
	IDPeriode            int
	JenisProgram         string
	Status               string
	IDDosenPembimbing    *int
	IDDosenPenguji       *int
	IDPembimbingLapangan *int
	NilaiAkhir           *float64
	GradeHuruf           *string
	SksDiakui            *int
}

// NilaiDetail memuat nilai beserta metadata instrumen dan penilai.
type NilaiDetail struct {
	IDPenilaian  int
	IDInstrumen  int
	NamaAspek    string
	Bobot        float64
	RolePenilai  string
	IDPenilai    int
	NamaPenilai  string
	Nilai        float64
	Catatan      *string
	TanggalNilai time.Time
}
