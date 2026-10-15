package dokumen

import (
	"archive/zip"
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"splk-backend/internal/platform/httpx"
)

func makeDummyZip() []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("word/document.xml")
	_, _ = f.Write([]byte("<w:document></w:document>"))
	_ = w.Close()
	return buf.Bytes()
}

func TestValidateFileContent(t *testing.T) {
	maxSize := int64(5 * 1024 * 1024) // 5 MB

	pdfHeader := []byte("%PDF-1.4 header sample content")
	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpgHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF")
	docxHeader := makeDummyZip()
	exeHeader := []byte("MZ\x90\x00\x03\x00\x00\x00")
	txtHeader := []byte("Hello plain text file")

	tests := []struct {
		name        string
		filename    string
		size        int64
		header      []byte
		wantErrCode string
		wantMime    string
	}{
		{
			name:        "sukses: file PDF valid",
			filename:    "laporan.pdf",
			size:        1024,
			header:      pdfHeader,
			wantErrCode: "",
			wantMime:    "application/pdf",
		},
		{
			name:        "sukses: file PNG valid",
			filename:    "foto.png",
			size:        2048,
			header:      pngHeader,
			wantErrCode: "",
			wantMime:    "image/png",
		},
		{
			name:        "sukses: file JPG valid",
			filename:    "foto.jpg",
			size:        3072,
			header:      jpgHeader,
			wantErrCode: "",
			wantMime:    "image/jpeg",
		},
		{
			name:        "sukses: file DOCX valid",
			filename:    "dokumen.docx",
			size:        4096,
			header:      docxHeader,
			wantErrCode: "",
			wantMime:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:        "gagal: file kosong (size 0)",
			filename:    "kosong.pdf",
			size:        0,
			header:      []byte{},
			wantErrCode: "FILE_KOSONG",
		},
		{
			name:        "gagal: file melebihi batas ukuran maksimal (6 MB)",
			filename:    "besar.pdf",
			size:        6 * 1024 * 1024,
			header:      pdfHeader,
			wantErrCode: "FILE_TERLALU_BESAR",
		},
		{
			name:        "gagal: ekstensi terlarang (.exe)",
			filename:    "program.exe",
			size:        1024,
			header:      exeHeader,
			wantErrCode: "EKSTENSI_TIDAK_DIIZINKAN",
		},
		{
			name:        "gagal: ekstensi terlarang (.sh)",
			filename:    "script.sh",
			size:        512,
			header:      txtHeader,
			wantErrCode: "EKSTENSI_TIDAK_DIIZINKAN",
		},
		{
			name:        "gagal: .exe di-rename menjadi .pdf (isi file executable)",
			filename:    "malware.pdf",
			size:        1024,
			header:      exeHeader,
			wantErrCode: "KONTEN_TIDAK_SESUAI",
		},
		{
			name:        "gagal: .txt di-rename menjadi .pdf",
			filename:    "text.pdf",
			size:        512,
			header:      txtHeader,
			wantErrCode: "KONTEN_TIDAK_SESUAI",
		},
		{
			name:        "gagal: teks biasa di-rename menjadi .docx",
			filename:    "palsu.docx",
			size:        512,
			header:      txtHeader,
			wantErrCode: "KONTEN_TIDAK_SESUAI",
		},
		{
			name:        "gagal: PNG di-rename menjadi .jpg",
			filename:    "mismatch.jpg",
			size:        1024,
			header:      pngHeader,
			wantErrCode: "KONTEN_TIDAK_SESUAI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime, err := ValidateFileContent(tt.filename, tt.size, maxSize, tt.header)
			if tt.wantErrCode != "" {
				if err == nil {
					t.Fatalf("diharapkan error %s, tetapi nil", tt.wantErrCode)
				}
				appErr, ok := err.(*httpx.AppError)
				if !ok {
					t.Fatalf("error bukan *httpx.AppError: %v", err)
				}
				if appErr.Code != tt.wantErrCode {
					t.Errorf("kode error = %s, diharapkan %s", appErr.Code, tt.wantErrCode)
				}
				if appErr.HTTPStatus != http.StatusUnprocessableEntity {
					t.Errorf("HTTP status = %d, diharapkan 422", appErr.HTTPStatus)
				}
			} else {
				if err != nil {
					t.Fatalf("tidak diharapkan error: %v", err)
				}
				if mime != tt.wantMime {
					t.Errorf("mime = %s, diharapkan %s", mime, tt.wantMime)
				}
			}
		})
	}
}

func TestValidateDokumenJenisForProgram(t *testing.T) {
	tests := []struct {
		name         string
		jenisProgram string
		jenisDokumen string
		isKelompok   bool
		wantErrCode  string
	}{
		// Program Magang (Peserta)
		{"sukses: mou untuk magang", "magang", JenisDokumenMou, false, ""},
		{"sukses: proposal untuk magang", "magang", JenisDokumenProposal, false, ""},
		{"sukses: laporan_akhir untuk magang", "magang", JenisDokumenLaporanAkhir, false, ""},
		{"sukses: lainnya untuk magang", "magang", JenisDokumenLainnya, false, ""},
		{"gagal: proker untuk peserta magang", "magang", JenisDokumenProker, false, "JENIS_DOKUMEN_TIDAK_SESUAI"},

		// Program Asistensi Mengajar (Peserta)
		{"sukses: proker untuk AM", "asistensi_mengajar", JenisDokumenProker, false, ""},
		{"gagal: mou untuk AM", "asistensi_mengajar", JenisDokumenMou, false, "JENIS_DOKUMEN_TIDAK_SESUAI"},
		{"gagal: proposal untuk AM", "asistensi_mengajar", JenisDokumenProposal, false, "JENIS_DOKUMEN_TIDAK_SESUAI"},

		// Program KKN (Peserta Individual)
		{"gagal: mou untuk peserta KKN", "kkn", JenisDokumenMou, false, "JENIS_DOKUMEN_TIDAK_SESUAI"},
		{"gagal: proposal untuk peserta KKN", "kkn", JenisDokumenProposal, false, "JENIS_DOKUMEN_TIDAK_SESUAI"},

		// Kelompok KKN
		{"sukses: proker untuk kelompok KKN", "kkn", JenisDokumenProker, true, ""},
		{"sukses: laporan_akhir untuk kelompok KKN", "kkn", JenisDokumenLaporanAkhir, true, ""},
		{"gagal: mou untuk kelompok KKN", "kkn", JenisDokumenMou, true, "JENIS_DOKUMEN_TIDAK_SESUAI"},
		{"gagal: proposal untuk kelompok KKN", "kkn", JenisDokumenProposal, true, "JENIS_DOKUMEN_TIDAK_SESUAI"},

		// Jenis tidak dikenal
		{"gagal: jenis dokumen aneh", "magang", "ijazah_palsu", false, "JENIS_DOKUMEN_TIDAK_VALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDokumenJenisForProgram(tt.jenisProgram, tt.jenisDokumen, tt.isKelompok)
			if tt.wantErrCode != "" {
				if err == nil {
					t.Fatalf("diharapkan error %s, tetapi nil", tt.wantErrCode)
				}
				appErr, ok := err.(*httpx.AppError)
				if !ok {
					t.Fatalf("error bukan *httpx.AppError: %v", err)
				}
				if appErr.Code != tt.wantErrCode {
					t.Errorf("kode error = %s, diharapkan %s", appErr.Code, tt.wantErrCode)
				}
				if appErr.HTTPStatus != http.StatusUnprocessableEntity {
					t.Errorf("status HTTP = %d, diharapkan 422", appErr.HTTPStatus)
				}
			} else if err != nil {
				t.Fatalf("tidak diharapkan error: %v", err)
			}
		})
	}
}

func TestStoragePath_DoesNotContainOriginalName(t *testing.T) {
	originalName := "Curriculum_Vitae_Sangat_Rahasia_12345.pdf"

	// 1. Path peserta
	idPeserta := 42
	pathPeserta := GeneratePesertaStoragePath(idPeserta, ".pdf")

	if strings.Contains(pathPeserta, "Curriculum") || strings.Contains(pathPeserta, "Rahasia") {
		t.Errorf("path peserta tidak boleh memuat nama asli file, didapat: %s", pathPeserta)
	}
	if !strings.HasPrefix(pathPeserta, "pendaftaran/42/") {
		t.Errorf("path peserta harus berformat 'pendaftaran/42/<uuid>.pdf', didapat: %s", pathPeserta)
	}
	if !strings.HasSuffix(pathPeserta, ".pdf") {
		t.Errorf("path peserta harus berakhiran .pdf, didapat: %s", pathPeserta)
	}

	// UUID validasi
	parts := strings.Split(pathPeserta, "/")
	if len(parts) != 3 {
		t.Fatalf("komponen path tidak sesuai: %v", parts)
	}
	uuidWithExt := parts[2]
	uuidStr := strings.TrimSuffix(uuidWithExt, ".pdf")
	if _, err := uuid.Parse(uuidStr); err != nil {
		t.Errorf("komponen nama file bukan UUID valid: %v", err)
	}

	// 2. Path kelompok
	idKelompok := 99
	pathKelompok := GenerateKelompokStoragePath(idKelompok, ".docx")
	if strings.Contains(pathKelompok, originalName) {
		t.Errorf("path kelompok tidak boleh memuat nama asli file")
	}
	if !strings.HasPrefix(pathKelompok, "kkn/99/") {
		t.Errorf("path kelompok harus berformat 'kkn/99/<uuid>.docx', didapat: %s", pathKelompok)
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"../../secret.pdf", "secret.pdf"},
		{"..\\..\\secret.docx", "secret.docx"},
		{"folder/sub/laporan.pdf", "laporan.pdf"},
		{"laporan final\x00v2.pdf", "laporan finalv2.pdf"},
		{"", "file"},
		{".", "file"},
		{"..", "file"},
	}

	for _, c := range cases {
		got := SanitizeFilename(c.input)
		if got != c.expected {
			t.Errorf("SanitizeFilename(%q) = %q, diharapkan %q", c.input, got, c.expected)
		}
	}
}
