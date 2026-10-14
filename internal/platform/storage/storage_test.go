package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/platform/storage"
)

func TestLocalStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "splk_storage_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store, err := storage.NewLocalStorage(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Simpan dan Buka File", func(t *testing.T) {
		relPath := "dokumen/pendaftaran/123/mou.pdf"
		content := []byte("konten file mou pdf untuk testing")

		size, err := store.Save(ctx, relPath, bytes.NewReader(content))
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), size)

		// Buka kembali file
		rc, err := store.Open(ctx, relPath)
		require.NoError(t, err)
		defer rc.Close()

		readContent, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, content, readContent)
	})

	t.Run("Hapus File", func(t *testing.T) {
		relPath := "dokumen/hapus/test.txt"
		content := []byte("akan dihapus")

		_, err := store.Save(ctx, relPath, bytes.NewReader(content))
		require.NoError(t, err)

		// Hapus file
		err = store.Delete(ctx, relPath)
		require.NoError(t, err)

		// Buka setelah dihapus harus gagal
		_, err = store.Open(ctx, relPath)
		assert.Error(t, err)

		// Hapus file yang tidak ada harus idempotent (tanpa error)
		err = store.Delete(ctx, relPath)
		assert.NoError(t, err)
	})

	t.Run("Tolak Path Traversal", func(t *testing.T) {
		traversalPaths := []string{
			"../secret.txt",
			"../../etc/passwd",
			"..\\secret.txt",
			"dokumen/../../escape.txt",
			"/absolute/path/file.txt",
			"dokumen/../../../root.txt",
			"",
		}

		for _, path := range traversalPaths {
			t.Run("Path: "+path, func(t *testing.T) {
				_, err := store.Save(ctx, path, bytes.NewReader([]byte("hacker")))
				assert.Error(t, err, "Save harus menolak path traversal")

				_, err = store.Open(ctx, path)
				assert.Error(t, err, "Open harus menolak path traversal")

				err = store.Delete(ctx, path)
				assert.Error(t, err, "Delete harus menolak path traversal")
			})
		}
	})

	t.Run("Verifikasi Subfolder Otomatis Terbentuk", func(t *testing.T) {
		relPath := "level1/level2/level3/deep.txt"
		_, err := store.Save(ctx, relPath, bytes.NewReader([]byte("deep content")))
		require.NoError(t, err)

		fullPath := filepath.Join(tempDir, "level1", "level2", "level3", "deep.txt")
		_, err = os.Stat(fullPath)
		assert.NoError(t, err, "file dan subdirektori harus benar-benar ada di filesystem")
	})
}
