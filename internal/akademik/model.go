// Package akademik mengelola master data struktur hierarki akademik: fakultas, jurusan, dan program studi.
package akademik

// Fakultas mewakili baris dalam tabel fakultas.
type Fakultas struct {
	ID   int     `json:"id"`
	Nama string  `json:"nama"`
	Kode *string `json:"kode,omitempty"`
}

// Jurusan mewakili baris dalam tabel jurusan.
type Jurusan struct {
	ID          int     `json:"id"`
	FakultasID  int     `json:"id_fakultas"`
	Nama        string  `json:"nama"`
	KodeJurusan *string `json:"kode_jurusan,omitempty"`
}

// ProgramStudi mewakili baris dalam tabel program_studi.
type ProgramStudi struct {
	ID        int     `json:"id"`
	JurusanID int     `json:"id_jurusan"`
	Nama      string  `json:"nama"`
	Kode      *string `json:"kode,omitempty"`
}
