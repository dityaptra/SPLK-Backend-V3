package akademik

// Request dan DTO Fakultas
type CreateFakultasRequest struct {
	Nama string  `json:"nama" binding:"required,max=100"`
	Kode *string `json:"kode,omitempty" binding:"omitempty,max=10"`
}

type UpdateFakultasRequest struct {
	Nama string  `json:"nama" binding:"required,max=100"`
	Kode *string `json:"kode,omitempty" binding:"omitempty,max=10"`
}

// Request dan DTO Jurusan
type CreateJurusanRequest struct {
	FakultasID  int     `json:"id_fakultas" binding:"required"`
	Nama        string  `json:"nama" binding:"required,max=100"`
	KodeJurusan *string `json:"kode_jurusan,omitempty" binding:"omitempty,max=10"`
}

type UpdateJurusanRequest struct {
	FakultasID  int     `json:"id_fakultas" binding:"required"`
	Nama        string  `json:"nama" binding:"required,max=100"`
	KodeJurusan *string `json:"kode_jurusan,omitempty" binding:"omitempty,max=10"`
}

// Request dan DTO Program Studi
type CreateProdiRequest struct {
	JurusanID int     `json:"id_jurusan" binding:"required"`
	Nama      string  `json:"nama" binding:"required,max=100"`
	Kode      *string `json:"kode,omitempty" binding:"omitempty,max=10"`
}

type UpdateProdiRequest struct {
	JurusanID int     `json:"id_jurusan" binding:"required"`
	Nama      string  `json:"nama" binding:"required,max=100"`
	Kode      *string `json:"kode,omitempty" binding:"omitempty,max=10"`
}
