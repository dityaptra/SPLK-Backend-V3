// Package storage menyediakan antarmuka penyimpanan file (Storage) dan implementasi lokal (LocalStorage).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrPathTraversal terjadi ketika path relatif mencoba keluar dari direktori dasar.
	ErrPathTraversal = errors.New("path traversal tidak diizinkan")
	// ErrInvalidPath terjadi ketika path relatif tidak valid atau kosong.
	ErrInvalidPath = errors.New("path file tidak valid")
)

// Storage adalah interface penyimpanan file abstrak (lokal atau cloud storage).
type Storage interface {
	Save(ctx context.Context, relPath string, r io.Reader) (size int64, err error)
	Open(ctx context.Context, relPath string) (io.ReadCloser, error)
	Delete(ctx context.Context, relPath string) error
}

// LocalStorage menyimpan file ke sistem file lokal di bawah baseDir.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage membuat instance LocalStorage pada direktori dasar baseDir.
// Memastikan baseDir dibuat otomatis jika belum ada.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if baseDir == "" {
		return nil, errors.New("baseDir tidak boleh kosong")
	}

	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return nil, fmt.Errorf("resolusi baseDir: %w", err)
	}

	if err := os.MkdirAll(absBase, 0755); err != nil {
		return nil, fmt.Errorf("buat direktori baseDir: %w", err)
	}

	return &LocalStorage{baseDir: absBase}, nil
}

// resolvePath memastikan relPath bersih, aman dari path traversal, dan berada di dalam baseDir.
func (l *LocalStorage) resolvePath(relPath string) (string, error) {
	if relPath == "" || strings.Contains(relPath, "\x00") {
		return "", ErrInvalidPath
	}

	// Path harus relatif murni; tolak jika diawali slash atau backslash
	if strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, "\\") {
		return "", ErrPathTraversal
	}

	// Normalisasi path
	cleanRel := filepath.Clean(filepath.FromSlash(relPath))

	// Tolak jika path mutlak, memiliki volume name (misal C:), diawali .., atau titik
	if filepath.IsAbs(cleanRel) || filepath.VolumeName(cleanRel) != "" || strings.HasPrefix(cleanRel, "..") || cleanRel == "." {
		return "", ErrPathTraversal
	}

	fullPath := filepath.Join(l.baseDir, cleanRel)

	// Verifikasi apakah fullPath benar-benar berada di bawah baseDir
	rel, err := filepath.Rel(l.baseDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", ErrPathTraversal
	}

	return fullPath, nil
}

// Save menyimpan isi dari reader r ke path relatif relPath.
// Subfolder dibuat secara otomatis bila belum ada.
func (l *LocalStorage) Save(ctx context.Context, relPath string, r io.Reader) (int64, error) {
	fullPath, err := l.resolvePath(relPath)
	if err != nil {
		return 0, err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("buat direktori tujuan: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return 0, fmt.Errorf("buat file: %w", err)
	}
	defer f.Close()

	size, err := io.Copy(f, r)
	if err != nil {
		_ = os.Remove(fullPath)
		return 0, fmt.Errorf("salin konten file: %w", err)
	}

	return size, nil
}

// Open membuka file pada relPath untuk dibaca.
func (l *LocalStorage) Open(ctx context.Context, relPath string) (io.ReadCloser, error) {
	fullPath, err := l.resolvePath(relPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// Delete menghapus file pada relPath.
// Jika file tidak ada, tidak dianggap sebagai error (idempotent).
func (l *LocalStorage) Delete(ctx context.Context, relPath string) error {
	fullPath, err := l.resolvePath(relPath)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("hapus file: %w", err)
	}

	return nil
}
