package tempat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"splk-backend/internal/tempat"
)

func TestPembimbingMatchValidation(t *testing.T) {
	t.Run("Validasi Kesesuaian Jenis Pembimbing Lapangan dengan Jenis Tempat", func(t *testing.T) {
		tests := []struct {
			name            string
			jenisPembimbing string
			jenisProgram    string
			expectSuccess   bool
		}{
			{
				name:            "Mentor cocok dengan Magang",
				jenisPembimbing: tempat.PembimbingMentor,
				jenisProgram:    tempat.ProgramMagang,
				expectSuccess:   true,
			},
			{
				name:            "Guru Pamong cocok dengan Asistensi Mengajar",
				jenisPembimbing: tempat.PembimbingGuruPamong,
				jenisProgram:    tempat.ProgramAsistensiMengajar,
				expectSuccess:   true,
			},
			{
				name:            "Kepala Desa cocok dengan KKN",
				jenisPembimbing: tempat.PembimbingKepalaDesa,
				jenisProgram:    tempat.ProgramKKN,
				expectSuccess:   true,
			},
			{
				name:            "Mentor TIDAK cocok dengan KKN",
				jenisPembimbing: tempat.PembimbingMentor,
				jenisProgram:    tempat.ProgramKKN,
				expectSuccess:   false,
			},
			{
				name:            "Guru Pamong TIDAK cocok dengan Magang",
				jenisPembimbing: tempat.PembimbingGuruPamong,
				jenisProgram:    tempat.ProgramMagang,
				expectSuccess:   false,
			},
			{
				name:            "Kepala Desa TIDAK cocok dengan Asistensi Mengajar",
				jenisPembimbing: tempat.PembimbingKepalaDesa,
				jenisProgram:    tempat.ProgramAsistensiMengajar,
				expectSuccess:   false,
			},
			{
				name:            "Jenis program tidak valid",
				jenisPembimbing: tempat.PembimbingMentor,
				jenisProgram:    "program_fiktif",
				expectSuccess:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tempat.ValidatePembimbingMatch(tt.jenisPembimbing, tt.jenisProgram)
				if tt.expectSuccess {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			})
		}
	})

	t.Run("Validasi IsValidProgram dan IsValidPembimbingJenis", func(t *testing.T) {
		assert.True(t, tempat.IsValidProgram("magang"))
		assert.True(t, tempat.IsValidProgram("asistensi_mengajar"))
		assert.True(t, tempat.IsValidProgram("kkn"))
		assert.False(t, tempat.IsValidProgram("lainnya"))

		assert.True(t, tempat.IsValidPembimbingJenis("mentor"))
		assert.True(t, tempat.IsValidPembimbingJenis("guru_pamong"))
		assert.True(t, tempat.IsValidPembimbingJenis("kepala_desa"))
		assert.False(t, tempat.IsValidPembimbingJenis("dosen"))
	})
}
