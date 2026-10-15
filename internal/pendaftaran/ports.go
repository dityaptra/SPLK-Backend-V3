// Package pendaftaran menyediakan abstraksi validasi eksternal untuk KRS dan eligibilitas.
package pendaftaran

import (
	"context"
)

// MahasiswaInfo berisi informasi ringkas mahasiswa yang dibutuhkan untuk validasi eksternal.
type MahasiswaInfo struct {
	ID       int
	UserID   int
	NIM      string
	Nama     string
	IDProdi  int
	Angkatan int
}

// PeriodeInfo berisi informasi periode yang dibutuhkan untuk validasi eksternal.
type PeriodeInfo struct {
	ID            int
	Nama          string
	JenisProgram  string
	TahunSemester string
	Status        string
}

// KRSValidator memeriksa apakah mahasiswa telah memprogram KRS yang sesuai pada periode berjalan.
type KRSValidator interface {
	Validate(ctx context.Context, m MahasiswaInfo, p PeriodeInfo) (ok bool, alasan string, err error)
}

// EligibilityChecker memeriksa syarat eligibilitas program (misalnya SKS tempuh minimal, IPK, atau prasyarat prodi).
type EligibilityChecker interface {
	Check(ctx context.Context, m MahasiswaInfo, p PeriodeInfo) (ok bool, alasan string, err error)
}

// StubKRSValidator adalah implementasi tiruan default untuk KRSValidator.
// ASUMSI: A1 - validasi KRS di-stub (selalu mengembalikan true) sesuai arahan mentor.
// TODO(mentor): ganti implementasi ini dengan integrasi nyata ke sistem SIAKAD / KRS kampus.
type StubKRSValidator struct{}

// NewStubKRSValidator membuat instance baru dari StubKRSValidator.
func NewStubKRSValidator() KRSValidator {
	return &StubKRSValidator{}
}

// Validate selalu mengembalikan true pada implementasi stub.
func (s *StubKRSValidator) Validate(ctx context.Context, m MahasiswaInfo, p PeriodeInfo) (ok bool, alasan string, err error) {
	// ASUMSI: A1 - mahasiswa dianggap telah memprogram mata kuliah KRS yang valid
	return true, "", nil
}

// StubEligibilityChecker adalah implementasi tiruan default untuk EligibilityChecker.
// ASUMSI: A2 - eligibilitas program di-stub (selalu mengembalikan true).
// TODO(mentor): tentukan aturan baku eligibilitas program (misal SKS minimal >= 80 untuk AM/KKN).
type StubEligibilityChecker struct{}

// NewStubEligibilityChecker membuat instance baru dari StubEligibilityChecker.
func NewStubEligibilityChecker() EligibilityChecker {
	return &StubEligibilityChecker{}
}

// Check selalu mengembalikan true pada implementasi stub.
func (s *StubEligibilityChecker) Check(ctx context.Context, m MahasiswaInfo, p PeriodeInfo) (ok bool, alasan string, err error) {
	// ASUMSI: A2 - mahasiswa dianggap memenuhi syarat eligibilitas program
	return true, "", nil
}
