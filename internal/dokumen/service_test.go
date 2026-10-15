package dokumen

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pendaftaran"
	"splk-backend/internal/platform/httpx"
)

type mockAccessResolver struct {
	resolveFn func(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error)
}

func (m *mockAccessResolver) Resolve(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
	if m.resolveFn != nil {
		return m.resolveFn(ctx, actor, idPeserta)
	}
	return pendaftaran.RelMahasiswa, nil
}

type mockStorage struct {
	savedFiles map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{savedFiles: make(map[string][]byte)}
}

func (m *mockStorage) Save(ctx context.Context, relPath string, r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	m.savedFiles[relPath] = data
	return int64(len(data)), nil
}

func (m *mockStorage) Open(ctx context.Context, relPath string) (io.ReadCloser, error) {
	data, ok := m.savedFiles[relPath]
	if !ok {
		return nil, httpx.NotFound("FILE_TIDAK_DITEMUKAN", "file tidak ditemukan")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorage) Delete(ctx context.Context, relPath string) error {
	delete(m.savedFiles, relPath)
	return nil
}

type mockDokumenRepo struct {
	infoFn           func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error)
	createFn         func(ctx context.Context, db database.DBTX, doc *Dokumen) error
	getByIDFn        func(ctx context.Context, db database.DBTX, id int) (*Dokumen, error)
	deleteFn         func(ctx context.Context, db database.DBTX, id int) error
	listPesertaFn    func(ctx context.Context, db database.DBTX, idPeserta int) ([]Dokumen, error)
	listKelompokFn   func(ctx context.Context, db database.DBTX, idKelompok int) ([]Dokumen, error)
	checkKelompokAcc func(ctx context.Context, db database.DBTX, idKelompok int, userID int, roles []string, isWrite bool) (bool, error)
}

func (m *mockDokumenRepo) Create(ctx context.Context, db database.DBTX, doc *Dokumen) error {
	if m.createFn != nil {
		return m.createFn(ctx, db, doc)
	}
	doc.ID = 1
	return nil
}

func (m *mockDokumenRepo) GetByID(ctx context.Context, db database.DBTX, id int) (*Dokumen, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, db, id)
	}
	return nil, nil
}

func (m *mockDokumenRepo) ListByPesertaID(ctx context.Context, db database.DBTX, idPeserta int) ([]Dokumen, error) {
	if m.listPesertaFn != nil {
		return m.listPesertaFn(ctx, db, idPeserta)
	}
	return nil, nil
}

func (m *mockDokumenRepo) ListByKelompokID(ctx context.Context, db database.DBTX, idKelompok int) ([]Dokumen, error) {
	if m.listKelompokFn != nil {
		return m.listKelompokFn(ctx, db, idKelompok)
	}
	return nil, nil
}

func (m *mockDokumenRepo) Delete(ctx context.Context, db database.DBTX, id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, db, id)
	}
	return nil
}

func (m *mockDokumenRepo) GetPendaftaranInfo(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
	if m.infoFn != nil {
		return m.infoFn(ctx, db, idPeserta)
	}
	return &PendaftaranInfo{ID: idPeserta, JenisProgram: "magang", Status: "diajukan", IDMahasiswa: 1}, nil
}

func (m *mockDokumenRepo) CheckKelompokAccess(ctx context.Context, db database.DBTX, idKelompok int, userID int, roles []string, isWrite bool) (bool, error) {
	if m.checkKelompokAcc != nil {
		return m.checkKelompokAcc(ctx, db, idKelompok, userID, roles, isWrite)
	}
	return true, nil
}

func TestUploadPesertaDokumen_AccessAndStatusRules(t *testing.T) {
	st := newMockStorage()

	t.Run("gagal: bukan mahasiswa pemilik atau admin (403)", func(t *testing.T) {
		repo := &mockDokumenRepo{}
		access := &mockAccessResolver{
			resolveFn: func(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
				return pendaftaran.RelPembimbing, nil // Relasi pembimbing bukan pemilik
			},
		}

		svc := &service{
			repo:           repo,
			storage:        st,
			accessSvc:      access,
			maxUploadBytes: 5 * 1024 * 1024,
		}

		reader := bytes.NewReader([]byte("%PDF-1.4 sample"))
		_, err := svc.UploadPesertaDokumen(context.Background(), middleware.Actor{UserID: 10, Roles: []string{"dosen"}}, 1, "laporan_akhir", "doc.pdf", int64(reader.Len()), reader)
		if err == nil {
			t.Fatal("diharapkan error 403, tetapi nil")
		}
		appErr, ok := err.(*httpx.AppError)
		if !ok || appErr.HTTPStatus != 403 {
			t.Fatalf("diharapkan status 403, didapat: %v", err)
		}
	})

	t.Run("gagal: status pendaftaran ditolak (422)", func(t *testing.T) {
		repo := &mockDokumenRepo{
			infoFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
				return &PendaftaranInfo{ID: idPeserta, JenisProgram: "magang", Status: "ditolak", IDMahasiswa: 1}, nil
			},
		}
		access := &mockAccessResolver{
			resolveFn: func(ctx context.Context, actor middleware.Actor, idPeserta int) (pendaftaran.Relation, error) {
				return pendaftaran.RelMahasiswa, nil
			},
		}

		svc := &service{
			repo:           repo,
			storage:        st,
			accessSvc:      access,
			maxUploadBytes: 5 * 1024 * 1024,
		}

		reader := bytes.NewReader([]byte("%PDF-1.4 sample"))
		_, err := svc.UploadPesertaDokumen(context.Background(), middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}, 1, "mou", "mou.pdf", int64(reader.Len()), reader)
		if err == nil {
			t.Fatal("diharapkan error 422, tetapi nil")
		}
		appErr, ok := err.(*httpx.AppError)
		if !ok || appErr.HTTPStatus != 422 {
			t.Fatalf("diharapkan status 422, didapat: %v", err)
		}
	})

	t.Run("gagal: unggah mou pada program AM (422)", func(t *testing.T) {
		repo := &mockDokumenRepo{
			infoFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
				return &PendaftaranInfo{ID: idPeserta, JenisProgram: "asistensi_mengajar", Status: "diajukan", IDMahasiswa: 1}, nil
			},
		}
		access := &mockAccessResolver{}

		svc := &service{
			repo:           repo,
			storage:        st,
			accessSvc:      access,
			maxUploadBytes: 5 * 1024 * 1024,
		}

		reader := bytes.NewReader([]byte("%PDF-1.4 sample"))
		_, err := svc.UploadPesertaDokumen(context.Background(), middleware.Actor{UserID: 1, Roles: []string{"mahasiswa"}}, 1, "mou", "mou.pdf", int64(reader.Len()), reader)
		if err == nil {
			t.Fatal("diharapkan error 422, tetapi nil")
		}
		appErr, ok := err.(*httpx.AppError)
		if !ok || appErr.HTTPStatus != 422 || appErr.Code != "JENIS_DOKUMEN_TIDAK_SESUAI" {
			t.Fatalf("diharapkan 422 JENIS_DOKUMEN_TIDAK_SESUAI, didapat: %v", err)
		}
	})
}

func TestDeleteDokumen_Rules(t *testing.T) {
	st := newMockStorage()

	t.Run("gagal: bukan pengunggah asli (403)", func(t *testing.T) {
		pesertaID := 10
		repo := &mockDokumenRepo{
			getByIDFn: func(ctx context.Context, db database.DBTX, id int) (*Dokumen, error) {
				return &Dokumen{
					ID:               id,
					IDPesertaProgram: &pesertaID,
					DiunggahOleh:     5, // Pemilik asli user ID 5
					FileURL:          "pendaftaran/10/file.pdf",
				}, nil
			},
		}

		svc := &service{
			repo:    repo,
			storage: st,
		}

		// User 99 mencoba menghapus dokumen user 5
		err := svc.DeleteDokumen(context.Background(), middleware.Actor{UserID: 99, Roles: []string{"mahasiswa"}}, 1)
		if err == nil {
			t.Fatal("diharapkan error 403, tetapi nil")
		}
		appErr, ok := err.(*httpx.AppError)
		if !ok || appErr.HTTPStatus != 403 {
			t.Fatalf("diharapkan 403, didapat: %v", err)
		}
	})

	t.Run("gagal: status pendaftaran sudah berjalan (422)", func(t *testing.T) {
		pesertaID := 10
		repo := &mockDokumenRepo{
			getByIDFn: func(ctx context.Context, db database.DBTX, id int) (*Dokumen, error) {
				return &Dokumen{
					ID:               id,
					IDPesertaProgram: &pesertaID,
					DiunggahOleh:     5,
					FileURL:          "pendaftaran/10/file.pdf",
				}, nil
			},
			infoFn: func(ctx context.Context, db database.DBTX, idPeserta int) (*PendaftaranInfo, error) {
				return &PendaftaranInfo{ID: idPeserta, JenisProgram: "magang", Status: "berjalan", IDMahasiswa: 1}, nil
			},
		}

		svc := &service{
			repo:    repo,
			storage: st,
		}

		err := svc.DeleteDokumen(context.Background(), middleware.Actor{UserID: 5, Roles: []string{"mahasiswa"}}, 1)
		if err == nil {
			t.Fatal("diharapkan error 422, tetapi nil")
		}
		appErr, ok := err.(*httpx.AppError)
		if !ok || appErr.HTTPStatus != 422 {
			t.Fatalf("diharapkan 422 STATUS_TIDAK_VALID, didapat: %v", err)
		}
		if !strings.Contains(appErr.Message, "diajukan") {
			t.Errorf("pesan error harus menyebutkan hanya saat diajukan: %s", appErr.Message)
		}
	})
}
