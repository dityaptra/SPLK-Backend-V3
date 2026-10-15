package dokumen

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"splk-backend/internal/platform/httpx"
)

var allowedExtensions = map[string]bool{
	".pdf":  true,
	".docx": true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

// SanitizeFilename membersihkan nama file dari path traversal dan karakter kontrol.
func SanitizeFilename(raw string) string {
	base := filepath.Base(raw)
	// Ganti pemisah path
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, "/", "_")

	// Hilangkan karakter non-printable / kontrol
	var clean strings.Builder
	for _, r := range base {
		if r >= 32 && r != 127 {
			clean.WriteRune(r)
		}
	}

	res := strings.TrimSpace(clean.String())
	if res == "" || res == "." || res == ".." {
		return "file"
	}
	return res
}

// ValidateFileContent memverifikasi ekstensi, ukuran, dan konsistensi tipe konten file.
// Mengembalikan mime type yang valid atau error 422 bila tidak konsisten.
func ValidateFileContent(filename string, size int64, maxSizeBytes int64, headerBytes []byte) (string, error) {
	if size <= 0 {
		return "", httpx.Unprocessable("FILE_KOSONG", "file yang diunggah tidak boleh kosong")
	}

	if size > maxSizeBytes {
		maxMB := maxSizeBytes / (1024 * 1024)
		return "", httpx.Unprocessable("FILE_TERLALU_BESAR", fmt.Sprintf("ukuran file melebihi batas maksimal %d MB", maxMB))
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExtensions[ext] {
		return "", httpx.Unprocessable("EKSTENSI_TIDAK_DIIZINKAN", fmt.Sprintf("ekstensi '%s' tidak diizinkan. Ekstensi yang didukung: pdf, docx, png, jpg, jpeg", ext))
	}

	// Sniff tipe konten dari cuplikan awal file
	detected := http.DetectContentType(headerBytes)

	// Validasi konsistensi ekstensi dengan tipe konten
	switch ext {
	case ".pdf":
		if !strings.HasPrefix(detected, "application/pdf") {
			return "", httpx.Unprocessable("KONTEN_TIDAK_SESUAI", "isi file tidak sesuai dengan ekstensi .pdf")
		}
		return "application/pdf", nil

	case ".png":
		if !strings.HasPrefix(detected, "image/png") {
			return "", httpx.Unprocessable("KONTEN_TIDAK_SESUAI", "isi file tidak sesuai dengan ekstensi .png")
		}
		return "image/png", nil

	case ".jpg", ".jpeg":
		if !strings.HasPrefix(detected, "image/jpeg") {
			return "", httpx.Unprocessable("KONTEN_TIDAK_SESUAI", "isi file tidak sesuai dengan ekstensi .jpg/.jpeg")
		}
		return "image/jpeg", nil

	case ".docx":
		// Format DOCX adalah zip archive (magic bytes PK\x03\x04)
		zipMagic := []byte{0x50, 0x4B, 0x03, 0x04}
		isZipMagic := len(headerBytes) >= 4 && bytes.Equal(headerBytes[:4], zipMagic)
		isZipMime := strings.HasPrefix(detected, "application/zip") ||
			strings.HasPrefix(detected, "application/x-zip-compressed") ||
			strings.HasPrefix(detected, "application/octet-stream") ||
			strings.HasPrefix(detected, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")

		if !isZipMagic || !isZipMime {
			return "", httpx.Unprocessable("KONTEN_TIDAK_SESUAI", "isi file tidak sesuai dengan format .docx")
		}
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil

	default:
		return "", httpx.Unprocessable("EKSTENSI_TIDAK_DIIZINKAN", "ekstensi tidak didukung")
	}
}

// ValidateDokumenJenisForProgram memvalidasi aturan kesesuaian jenis dokumen terhadap jenis program.
// mou dan proposal hanya untuk magang. proker untuk AM (peserta) atau KKN (kelompok).
func ValidateDokumenJenisForProgram(jenisProgram string, jenisDokumen string, isKelompok bool) error {
	if !IsValidJenisDokumen(jenisDokumen) {
		return httpx.Unprocessable("JENIS_DOKUMEN_TIDAK_VALID", fmt.Sprintf("jenis dokumen '%s' tidak valid", jenisDokumen))
	}

	if isKelompok {
		// Dokumen kelompok KKN
		if jenisDokumen == JenisDokumenMou || jenisDokumen == JenisDokumenProposal {
			return httpx.Unprocessable("JENIS_DOKUMEN_TIDAK_SESUAI", "dokumen mou dan proposal hanya berlaku untuk program magang peserta individual")
		}
		return nil
	}

	// Dokumen peserta individual
	switch jenisDokumen {
	case JenisDokumenMou, JenisDokumenProposal:
		if jenisProgram != "magang" {
			return httpx.Unprocessable("JENIS_DOKUMEN_TIDAK_SESUAI", "dokumen mou dan proposal hanya berlaku untuk program magang")
		}
	case JenisDokumenProker:
		if jenisProgram == "magang" {
			return httpx.Unprocessable("JENIS_DOKUMEN_TIDAK_SESUAI", "dokumen proker untuk magang tidak didukung (gunakan proposal)")
		}
	case JenisDokumenLaporanAkhir, JenisDokumenLainnya:
		// Bebas untuk semua jenis program
	}

	return nil
}

// GeneratePesertaStoragePath menghasilkan path penyimpanan relatif tanpa memuat nama asli.
// Format: pendaftaran/<id_peserta>/<uuid>.<ext>
func GeneratePesertaStoragePath(idPeserta int, ext string) string {
	cleanExt := strings.ToLower(ext)
	if !strings.HasPrefix(cleanExt, ".") {
		cleanExt = "." + cleanExt
	}
	return fmt.Sprintf("pendaftaran/%d/%s%s", idPeserta, uuid.New().String(), cleanExt)
}

// GenerateKelompokStoragePath menghasilkan path penyimpanan relatif tanpa memuat nama asli.
// Format: kkn/<id_kelompok>/<uuid>.<ext>
func GenerateKelompokStoragePath(idKelompok int, ext string) string {
	cleanExt := strings.ToLower(ext)
	if !strings.HasPrefix(cleanExt, ".") {
		cleanExt = "." + cleanExt
	}
	return fmt.Sprintf("kkn/%d/%s%s", idKelompok, uuid.New().String(), cleanExt)
}
