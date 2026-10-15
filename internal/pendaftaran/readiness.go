// Package pendaftaran menyediakan logika pemeriksaan kesiapan pendaftaran (ReadinessChecker).
package pendaftaran

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ReadinessInput menampung seluruh informasi kelengkapan penempatan dan dokumen peserta.
type ReadinessInput struct {
	JenisProgram         string
	PeriodeStatus        string
	IDTempatProgram      *int
	IDPembimbingLapangan *int
	IDDosenPembimbing    *int
	IDKelompokKKN        *int
	DokumenTypes         []string
}

// ReadinessResult adalah hasil pengecekan kesiapan.
type ReadinessResult struct {
	Siap       bool     `json:"siap"`
	Kekurangan []string `json:"kekurangan,omitempty"`
}

// CheckReadiness adalah fungsi murni yang mengevaluasi kesiapan pendaftaran menjadi 'berjalan'
// sesuai aturan pada PRD bagian 4.3.
func CheckReadiness(input ReadinessInput) ReadinessResult {
	var kekurangan []string

	// Periode program harus aktif
	if input.PeriodeStatus != "aktif" {
		kekurangan = append(kekurangan, "periode program belum berstatus aktif")
	}

	hasDoc := func(docType string) bool {
		for _, d := range input.DokumenTypes {
			if d == docType {
				return true
			}
		}
		return false
	}

	switch input.JenisProgram {
	case "magang":
		if input.IDTempatProgram == nil {
			kekurangan = append(kekurangan, "tempat magang belum dipilih")
		}
		if input.IDPembimbingLapangan == nil {
			kekurangan = append(kekurangan, "mentor lapangan belum dipilih")
		}
		if input.IDDosenPembimbing == nil {
			kekurangan = append(kekurangan, "dosen pembimbing belum ditetapkan")
		}
		if !hasDoc("mou") {
			kekurangan = append(kekurangan, "dokumen mou belum diunggah")
		}
		if !hasDoc("proposal") {
			kekurangan = append(kekurangan, "dokumen proposal belum diunggah")
		}

	case "asistensi_mengajar":
		if input.IDTempatProgram == nil {
			kekurangan = append(kekurangan, "sekolah penempatan belum dipetakan")
		}
		if input.IDPembimbingLapangan == nil {
			kekurangan = append(kekurangan, "guru pamong belum ditetapkan")
		}
		if input.IDDosenPembimbing == nil {
			kekurangan = append(kekurangan, "dosen pembimbing belum ditetapkan")
		}
		if !hasDoc("proker") {
			kekurangan = append(kekurangan, "dokumen proker belum diunggah")
		}

	case "kkn":
		if input.IDKelompokKKN == nil {
			kekurangan = append(kekurangan, "kelompok KKN belum dipetakan")
		}
		if input.IDTempatProgram == nil {
			kekurangan = append(kekurangan, "desa penempatan belum dipetakan")
		}
		if input.IDDosenPembimbing == nil {
			kekurangan = append(kekurangan, "dosen pembimbing kelompok belum ditetapkan")
		}
		if !hasDoc("proker") {
			kekurangan = append(kekurangan, "dokumen proker belum diunggah")
		}

	default:
		kekurangan = append(kekurangan, fmt.Sprintf("jenis program '%s' tidak dikenali", input.JenisProgram))
	}

	if len(kekurangan) > 0 {
		return ReadinessResult{
			Siap:       false,
			Kekurangan: kekurangan,
		}
	}

	return ReadinessResult{
		Siap:       true,
		Kekurangan: nil,
	}
}

// TryTransitionToBerjalan memeriksa kesiapan peserta dan meng-update status menjadi 'berjalan'
// di dalam transaksi secara atomik menggunakan SELECT ... FOR UPDATE.
func TryTransitionToBerjalan(ctx context.Context, tx pgx.Tx, idPeserta int) (transitioned bool, err error) {
	const selectPesertaQ = `
		SELECT jenis_program, status, id_periode,
		       id_tempat_program, id_pembimbing_lapangan,
		       id_dosen_pembimbing, id_kelompok_kkn
		FROM peserta_program
		WHERE id = $1
		FOR UPDATE
	`
	var (
		jenisProgram         string
		status               string
		idPeriode            int
		idTempatProgram      *int
		idPembimbingLapangan *int
		idDosenPembimbing    *int
		idKelompokKKN        *int
	)

	err = tx.QueryRow(ctx, selectPesertaQ, idPeserta).Scan(
		&jenisProgram, &status, &idPeriode,
		&idTempatProgram, &idPembimbingLapangan,
		&idDosenPembimbing, &idKelompokKKN,
	)
	if err != nil {
		return false, fmt.Errorf("lock peserta %d: %w", idPeserta, err)
	}

	// Hanya peserta yang masih 'diajukan' yang dapat bertransisi menjadi 'berjalan'
	if status != StatusDiajukan {
		return false, nil
	}

	// Periksa status periode
	var periodeStatus string
	err = tx.QueryRow(ctx, "SELECT status FROM periode WHERE id = $1", idPeriode).Scan(&periodeStatus)
	if err != nil {
		return false, fmt.Errorf("cek status periode %d: %w", idPeriode, err)
	}

	// Ambil daftar dokumen yang sudah diunggah untuk peserta ini (dan kelompok KKN jika ada)
	const selectDocsQ = `
		SELECT DISTINCT jenis_dokumen
		FROM dokumen
		WHERE id_peserta_program = $1
		   OR ($2::int IS NOT NULL AND id_kelompok_kkn = $2)
	`
	rows, err := tx.Query(ctx, selectDocsQ, idPeserta, idKelompokKKN)
	if err != nil {
		return false, fmt.Errorf("ambil dokumen peserta %d: %w", idPeserta, err)
	}
	defer rows.Close()

	var docTypes []string
	for rows.Next() {
		var dt string
		if err := rows.Scan(&dt); err != nil {
			return false, fmt.Errorf("scan jenis dokumen: %w", err)
		}
		docTypes = append(docTypes, dt)
	}

	input := ReadinessInput{
		JenisProgram:         jenisProgram,
		PeriodeStatus:        periodeStatus,
		IDTempatProgram:      idTempatProgram,
		IDPembimbingLapangan: idPembimbingLapangan,
		IDDosenPembimbing:    idDosenPembimbing,
		IDKelompokKKN:        idKelompokKKN,
		DokumenTypes:         docTypes,
	}

	result := CheckReadiness(input)
	if !result.Siap {
		return false, nil
	}

	// Transisi menjadi 'berjalan'
	const updateQ = `
		UPDATE peserta_program
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err = tx.Exec(ctx, updateQ, StatusBerjalan, idPeserta)
	if err != nil {
		return false, fmt.Errorf("update status berjalan peserta %d: %w", idPeserta, err)
	}

	return true, nil
}
