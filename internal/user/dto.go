package user

// --- Request dan Response Mahasiswa ---

type CreateMahasiswaRequest struct {
	Username         string  `json:"username" binding:"required,max=50"`
	Email            string  `json:"email" binding:"required,email,max=100"`
	Password         string  `json:"password" binding:"required,min=8"`
	NIM              string  `json:"nim" binding:"required,max=20"`
	Nama             string  `json:"nama" binding:"required,max=100"`
	ProdiID          int     `json:"id_prodi" binding:"required"`
	Angkatan         int     `json:"angkatan" binding:"required"`
	SemesterSekarang *int    `json:"semester_sekarang,omitempty"`
	NoHP             *string `json:"no_hp,omitempty" binding:"omitempty,max=20"`
}

type UpdateMahasiswaRequest struct {
	NIM              string  `json:"nim" binding:"required,max=20"`
	Nama             string  `json:"nama" binding:"required,max=100"`
	ProdiID          int     `json:"id_prodi" binding:"required"`
	Angkatan         int     `json:"angkatan" binding:"required"`
	SemesterSekarang *int    `json:"semester_sekarang,omitempty"`
	NoHP             *string `json:"no_hp,omitempty" binding:"omitempty,max=20"`
}

type MahasiswaDetail struct {
	ID               int     `json:"id"`
	UserID           int     `json:"id_user"`
	Username         string  `json:"username"`
	Email            string  `json:"email"`
	IsActive         bool    `json:"is_active"`
	NIM              string  `json:"nim"`
	Nama             string  `json:"nama"`
	ProdiID          int     `json:"id_prodi"`
	NamaProdi        string  `json:"nama_prodi,omitempty"`
	Angkatan         int     `json:"angkatan"`
	SemesterSekarang *int    `json:"semester_sekarang,omitempty"`
	NoHP             *string `json:"no_hp,omitempty"`
}

// --- Request dan Response Dosen ---

type CreateDosenRequest struct {
	Username      string  `json:"username" binding:"required,max=50"`
	Email         string  `json:"email" binding:"required,email,max=100"`
	Password      string  `json:"password" binding:"required,min=8"`
	NIP           string  `json:"nip" binding:"required,max=30"`
	Nama          string  `json:"nama" binding:"required,max=100"`
	ProdiID       int     `json:"id_prodi" binding:"required"`
	GelarDepan    *string `json:"gelar_depan,omitempty" binding:"omitempty,max=50"`
	GelarBelakang *string `json:"gelar_belakang,omitempty" binding:"omitempty,max=50"`
}

type UpdateDosenRequest struct {
	NIP           string  `json:"nip" binding:"required,max=30"`
	Nama          string  `json:"nama" binding:"required,max=100"`
	ProdiID       int     `json:"id_prodi" binding:"required"`
	GelarDepan    *string `json:"gelar_depan,omitempty" binding:"omitempty,max=50"`
	GelarBelakang *string `json:"gelar_belakang,omitempty" binding:"omitempty,max=50"`
}

type DosenDetail struct {
	ID            int     `json:"id"`
	UserID        int     `json:"id_user"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	IsActive      bool    `json:"is_active"`
	NIP           string  `json:"nip"`
	Nama          string  `json:"nama"`
	ProdiID       int     `json:"id_prodi"`
	NamaProdi     string  `json:"nama_prodi,omitempty"`
	GelarDepan    *string `json:"gelar_depan,omitempty"`
	GelarBelakang *string `json:"gelar_belakang,omitempty"`
}

// --- Request dan Response Pegawai ---

type CreatePegawaiRequest struct {
	Username      string  `json:"username" binding:"required,max=50"`
	Email         string  `json:"email" binding:"required,email,max=100"`
	Password      string  `json:"password" binding:"required,min=8"`
	NIP           string  `json:"nip" binding:"required,max=30"`
	Nama          string  `json:"nama" binding:"required,max=100"`
	GelarDepan    *string `json:"gelar_depan,omitempty" binding:"omitempty,max=50"`
	GelarBelakang *string `json:"gelar_belakang,omitempty" binding:"omitempty,max=50"`
	Jabatan       string  `json:"jabatan" binding:"required,max=100"`
}

type UpdatePegawaiRequest struct {
	NIP           string  `json:"nip" binding:"required,max=30"`
	Nama          string  `json:"nama" binding:"required,max=100"`
	GelarDepan    *string `json:"gelar_depan,omitempty" binding:"omitempty,max=50"`
	GelarBelakang *string `json:"gelar_belakang,omitempty" binding:"omitempty,max=50"`
	Jabatan       string  `json:"jabatan" binding:"required,max=100"`
}

type PegawaiDetail struct {
	ID            int     `json:"id"`
	UserID        int     `json:"id_user"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	IsActive      bool    `json:"is_active"`
	NIP           string  `json:"nip"`
	Nama          string  `json:"nama"`
	GelarDepan    *string `json:"gelar_depan,omitempty"`
	GelarBelakang *string `json:"gelar_belakang,omitempty"`
	Jabatan       string  `json:"jabatan"`
}

// --- Request dan Response Admin ---

type CreateAdminRequest struct {
	Username string  `json:"username" binding:"required,max=50"`
	Email    string  `json:"email" binding:"required,email,max=100"`
	Password string  `json:"password" binding:"required,min=8"`
	NIP      *string `json:"nip,omitempty" binding:"omitempty,max=30"`
	Nama     string  `json:"nama" binding:"required,max=100"`
}

type UpdateAdminRequest struct {
	NIP  *string `json:"nip,omitempty" binding:"omitempty,max=30"`
	Nama string  `json:"nama" binding:"required,max=100"`
}

type AdminDetail struct {
	ID       int     `json:"id"`
	UserID   int     `json:"id_user"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	IsActive bool    `json:"is_active"`
	NIP      *string `json:"nip,omitempty"`
	Nama     string  `json:"nama"`
}

// --- Request dan Response Pembimbing Lapangan ---

type CreatePembimbingLapanganRequest struct {
	Username        string  `json:"username" binding:"required,max=50"`
	Email           string  `json:"email" binding:"required,email,max=100"`
	Password        string  `json:"password" binding:"required,min=8"`
	Jenis           string  `json:"jenis" binding:"required,oneof=mentor guru_pamong kepala_desa"`
	TempatProgramID int     `json:"id_tempat_program" binding:"required"`
	Nama            string  `json:"nama" binding:"required,max=100"`
	Jabatan         *string `json:"jabatan,omitempty" binding:"omitempty,max=100"`
	Kontak          *string `json:"kontak,omitempty" binding:"omitempty,max=50"`
}

type UpdatePembimbingLapanganRequest struct {
	Jenis           string  `json:"jenis" binding:"required,oneof=mentor guru_pamong kepala_desa"`
	TempatProgramID int     `json:"id_tempat_program" binding:"required"`
	Nama            string  `json:"nama" binding:"required,max=100"`
	Jabatan         *string `json:"jabatan,omitempty" binding:"omitempty,max=100"`
	Kontak          *string `json:"kontak,omitempty" binding:"omitempty,max=50"`
}

type PembimbingLapanganDetail struct {
	ID              int     `json:"id"`
	UserID          *int    `json:"id_user,omitempty"`
	Username        string  `json:"username,omitempty"`
	Email           string  `json:"email,omitempty"`
	IsActive        bool    `json:"is_active"`
	Jenis           string  `json:"jenis"`
	TempatProgramID int     `json:"id_tempat_program"`
	NamaTempat      string  `json:"nama_tempat,omitempty"`
	Nama            string  `json:"nama"`
	Jabatan         *string `json:"jabatan,omitempty"`
	Kontak          *string `json:"kontak,omitempty"`
}

// --- Request dan Response Manajemen User Umum ---

type UserFilter struct {
	Role  string
	Query string
}

type UserListItem struct {
	ID        int      `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	IsActive  bool     `json:"is_active"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}

type ResetPasswordRequest struct {
	PasswordBaru string `json:"password_baru" binding:"required,min=8"`
}
