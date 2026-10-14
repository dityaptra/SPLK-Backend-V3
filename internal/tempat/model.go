// Package tempat mengelola master data tempat pelaksanaan program (perusahaan, sekolah, desa)
// dan pembimbing lapangan (mentor, guru pamong, kepala desa).
package tempat

import "fmt"

// Konstanta jenis program pembelajaran luar kampus
const (
	ProgramMagang            = "magang"
	ProgramAsistensiMengajar = "asistensi_mengajar"
	ProgramKKN               = "kkn"
)

// Konstanta jenis pembimbing lapangan
const (
	PembimbingMentor     = "mentor"
	PembimbingGuruPamong = "guru_pamong"
	PembimbingKepalaDesa = "kepala_desa"
)

// IsValidProgram memeriksa apakah jenis_program valid.
func IsValidProgram(program string) bool {
	switch program {
	case ProgramMagang, ProgramAsistensiMengajar, ProgramKKN:
		return true
	default:
		return false
	}
}

// IsValidPembimbingJenis memeriksa apakah jenis pembimbing lapangan valid.
func IsValidPembimbingJenis(jenis string) bool {
	switch jenis {
	case PembimbingMentor, PembimbingGuruPamong, PembimbingKepalaDesa:
		return true
	default:
		return false
	}
}

// ValidatePembimbingMatch memeriksa kesesuaian antara jenis pembimbing lapangan
// dengan jenis program dari tempat pelaksanaan:
// - mentor -> magang
// - guru_pamong -> asistensi_mengajar
// - kepala_desa -> kkn
func ValidatePembimbingMatch(jenisPembimbing, jenisProgram string) error {
	switch jenisProgram {
	case ProgramMagang:
		if jenisPembimbing != PembimbingMentor {
			return fmt.Errorf("tempat program magang harus memiliki pembimbing lapangan bertipe '%s', bukan '%s'", PembimbingMentor, jenisPembimbing)
		}
	case ProgramAsistensiMengajar:
		if jenisPembimbing != PembimbingGuruPamong {
			return fmt.Errorf("tempat program asistensi mengajar harus memiliki pembimbing lapangan bertipe '%s', bukan '%s'", PembimbingGuruPamong, jenisPembimbing)
		}
	case ProgramKKN:
		if jenisPembimbing != PembimbingKepalaDesa {
			return fmt.Errorf("tempat program KKN harus memiliki pembimbing lapangan bertipe '%s', bukan '%s'", PembimbingKepalaDesa, jenisPembimbing)
		}
	default:
		return fmt.Errorf("jenis program tempat '%s' tidak valid", jenisProgram)
	}
	return nil
}

// TempatProgram mewakili baris dalam tabel tempat_program.
type TempatProgram struct {
	ID           int     `json:"id"`
	JenisProgram string  `json:"jenis_program"`
	Nama         string  `json:"nama"`
	Alamat       string  `json:"alamat"`
	Kota         *string `json:"kota,omitempty"`
	Kontak       *string `json:"kontak,omitempty"`
	IsAktif      bool    `json:"is_aktif"`
}

// PembimbingLapangan mewakili baris dalam tabel pembimbing_lapangan.
type PembimbingLapangan struct {
	ID              int     `json:"id"`
	Jenis           string  `json:"jenis"`
	TempatProgramID int     `json:"id_tempat_program"`
	UserID          *int    `json:"id_user,omitempty"`
	Nama            string  `json:"nama"`
	Jabatan         *string `json:"jabatan,omitempty"`
	Kontak          *string `json:"kontak,omitempty"`
}
