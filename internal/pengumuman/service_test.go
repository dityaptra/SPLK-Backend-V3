package pengumuman_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"splk-backend/internal/database"
	"splk-backend/internal/middleware"
	"splk-backend/internal/pengumuman"
	"splk-backend/internal/platform/httpx"
)

type mockPengumumanRepo struct {
	items  map[int]*pengumuman.Pengumuman
	nextID int
}

func newMockPengumumanRepo() *mockPengumumanRepo {
	return &mockPengumumanRepo{
		items:  make(map[int]*pengumuman.Pengumuman),
		nextID: 1,
	}
}

func (m *mockPengumumanRepo) Create(ctx context.Context, db database.DBTX, p *pengumuman.Pengumuman) error {
	p.ID = m.nextID
	m.nextID++
	if p.TanggalTerbit.IsZero() {
		p.TanggalTerbit = time.Now()
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}

func (m *mockPengumumanRepo) Update(ctx context.Context, db database.DBTX, p *pengumuman.Pengumuman) error {
	existing, ok := m.items[p.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	existing.Judul = p.Judul
	existing.Isi = p.Isi
	existing.TargetRole = p.TargetRole
	return nil
}

func (m *mockPengumumanRepo) Delete(ctx context.Context, db database.DBTX, id int) error {
	if _, ok := m.items[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.items, id)
	return nil
}

func (m *mockPengumumanRepo) FindByID(ctx context.Context, db database.DBTX, id int) (*pengumuman.PengumumanDetail, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	username := "admin_tester"
	return &pengumuman.PengumumanDetail{
		ID:            p.ID,
		Judul:         p.Judul,
		Isi:           p.Isi,
		DibuatOleh:    p.DibuatOleh,
		NamaPembuat:   &username,
		TanggalTerbit: p.TanggalTerbit,
		TargetRole:    p.TargetRole,
	}, nil
}

func (m *mockPengumumanRepo) ListAdmin(ctx context.Context, db database.DBTX, filter pengumuman.PengumumanFilter, p httpx.Pagination) ([]pengumuman.PengumumanDetail, int64, error) {
	var list []pengumuman.PengumumanDetail
	for _, item := range m.items {
		if filter.TargetRole != "" && (item.TargetRole == nil || *item.TargetRole != filter.TargetRole) {
			continue
		}
		if filter.Search != "" && !strings.Contains(item.Judul, filter.Search) && !strings.Contains(item.Isi, filter.Search) {
			continue
		}
		username := "admin_tester"
		list = append(list, pengumuman.PengumumanDetail{
			ID:            item.ID,
			Judul:         item.Judul,
			Isi:           item.Isi,
			DibuatOleh:    item.DibuatOleh,
			NamaPembuat:   &username,
			TanggalTerbit: item.TanggalTerbit,
			TargetRole:    item.TargetRole,
		})
	}
	return list, int64(len(list)), nil
}

func (m *mockPengumumanRepo) ListForRoles(ctx context.Context, db database.DBTX, roles []string, filter pengumuman.PengumumanFilter, p httpx.Pagination) ([]pengumuman.PengumumanDetail, int64, error) {
	var list []pengumuman.PengumumanDetail
	for _, item := range m.items {
		// Target role filter
		if item.TargetRole != nil && *item.TargetRole != "" {
			matched := false
			for _, r := range roles {
				if r == *item.TargetRole {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if filter.Search != "" && !strings.Contains(item.Judul, filter.Search) && !strings.Contains(item.Isi, filter.Search) {
			continue
		}

		username := "admin_tester"
		list = append(list, pengumuman.PengumumanDetail{
			ID:            item.ID,
			Judul:         item.Judul,
			Isi:           item.Isi,
			DibuatOleh:    item.DibuatOleh,
			NamaPembuat:   &username,
			TanggalTerbit: item.TanggalTerbit,
			TargetRole:    item.TargetRole,
		})
	}
	return list, int64(len(list)), nil
}

type mockActivityLogger struct{}

func (m *mockActivityLogger) Log(ctx context.Context, userID int, action string, status bool, ip, userAgent string) error {
	return nil
}

func TestPengumumanService(t *testing.T) {
	ctx := context.Background()
	repo := newMockPengumumanRepo()
	logger := &mockActivityLogger{}
	svc := pengumuman.NewService(repo, nil, logger)

	adminActor := middleware.Actor{UserID: 1, Roles: []string{"admin"}}
	mhsActor := middleware.Actor{UserID: 2, Roles: []string{"mahasiswa"}}
	dosenActor := middleware.Actor{UserID: 3, Roles: []string{"dosen"}}

	t.Run("CreatePengumuman - Akses Ditolak untuk Non-Admin", func(t *testing.T) {
		req := pengumuman.CreatePengumumanRequest{
			Judul: "Pengumuman 1",
			Isi:   "Isi pengumuman",
		}
		_, err := svc.CreatePengumuman(ctx, mhsActor, req, "127.0.0.1", "agent")
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, "HAK_AKSES_DITOLAK", appErr.Code)
	})

	t.Run("CreatePengumuman - Sukses oleh Admin untuk Semua Role (target_role = nil)", func(t *testing.T) {
		req := pengumuman.CreatePengumumanRequest{
			Judul: "Pengumuman Publik",
			Isi:   "Semua pengguna dapat melihat pengumuman ini.",
		}
		resp, err := svc.CreatePengumuman(ctx, adminActor, req, "127.0.0.1", "agent")
		require.NoError(t, err)
		assert.Equal(t, "Pengumuman Publik", resp.Judul)
		assert.Nil(t, resp.TargetRole)
	})

	t.Run("CreatePengumuman - Sukses oleh Admin untuk Role Mahasiswa", func(t *testing.T) {
		roleMhs := "mahasiswa"
		req := pengumuman.CreatePengumumanRequest{
			Judul:      "Pengumuman Khusus Mahasiswa",
			Isi:        "Batas akhir pengumpulan proposal.",
			TargetRole: &roleMhs,
		}
		resp, err := svc.CreatePengumuman(ctx, adminActor, req, "127.0.0.1", "agent")
		require.NoError(t, err)
		assert.Equal(t, "Pengumuman Khusus Mahasiswa", resp.Judul)
		assert.NotNil(t, resp.TargetRole)
		assert.Equal(t, "mahasiswa", *resp.TargetRole)
	})

	t.Run("GetPengumumanByID - Pengujian Otorisasi Berdasarkan Target Role", func(t *testing.T) {
		roleDosen := "dosen"
		req := pengumuman.CreatePengumumanRequest{
			Judul:      "Pengumuman Khusus Dosen",
			Isi:        "Jadwal sidang penguji.",
			TargetRole: &roleDosen,
		}
		respDosen, err := svc.CreatePengumuman(ctx, adminActor, req, "127.0.0.1", "agent")
		require.NoError(t, err)

		// Dosen dapat mengakses
		dosenView, err := svc.GetPengumumanByID(ctx, dosenActor, respDosen.ID)
		require.NoError(t, err)
		assert.Equal(t, respDosen.ID, dosenView.ID)

		// Admin dapat mengakses
		adminView, err := svc.GetPengumumanByID(ctx, adminActor, respDosen.ID)
		require.NoError(t, err)
		assert.Equal(t, respDosen.ID, adminView.ID)

		// Mahasiswa ditolak (404 Not Found)
		_, err = svc.GetPengumumanByID(ctx, mhsActor, respDosen.ID)
		require.Error(t, err)
		var appErr *httpx.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, "PENGUMUMAN_TIDAK_DITEMUKAN", appErr.Code)
	})

	t.Run("ListPengumumanUser - Filter Otomatis Sesuai Role", func(t *testing.T) {
		p := httpx.Pagination{Page: 1, Limit: 20}
		filter := pengumuman.PengumumanFilter{}

		// Mahasiswa harus melihat: Pengumuman Publik & Pengumuman Khusus Mahasiswa (total: 2)
		// Tidak boleh melihat Pengumuman Khusus Dosen
		items, meta, err := svc.ListPengumumanUser(ctx, mhsActor, filter, p)
		require.NoError(t, err)
		assert.Equal(t, int64(2), meta.Total)
		for _, item := range items {
			if item.TargetRole != nil {
				assert.Equal(t, "mahasiswa", *item.TargetRole)
			}
		}

		// Dosen harus melihat: Pengumuman Publik & Pengumuman Khusus Dosen (total: 2)
		itemsDosen, metaDosen, err := svc.ListPengumumanUser(ctx, dosenActor, filter, p)
		require.NoError(t, err)
		assert.Equal(t, int64(2), metaDosen.Total)
		for _, item := range itemsDosen {
			if item.TargetRole != nil {
				assert.Equal(t, "dosen", *item.TargetRole)
			}
		}

		// Admin melihat seluruh 3 pengumuman
		itemsAdmin, metaAdmin, err := svc.ListPengumumanAdmin(ctx, adminActor, filter, p)
		require.NoError(t, err)
		assert.Equal(t, int64(3), metaAdmin.Total)
		assert.Len(t, itemsAdmin, 3)
	})

	t.Run("Update & Delete Pengumuman", func(t *testing.T) {
		roleMhs := "mahasiswa"
		reqCreate := pengumuman.CreatePengumumanRequest{
			Judul:      "Akan Diubah",
			Isi:        "Isi lama",
			TargetRole: &roleMhs,
		}
		created, err := svc.CreatePengumuman(ctx, adminActor, reqCreate, "127.0.0.1", "agent")
		require.NoError(t, err)

		// Update
		reqUpdate := pengumuman.UpdatePengumumanRequest{
			Judul:      "Sudah Diperbarui",
			Isi:        "Isi baru yang telah diperbarui.",
			TargetRole: nil, // diubah jadi publik
		}
		updated, err := svc.UpdatePengumuman(ctx, adminActor, created.ID, reqUpdate, "127.0.0.1", "agent")
		require.NoError(t, err)
		assert.Equal(t, "Sudah Diperbarui", updated.Judul)
		assert.Nil(t, updated.TargetRole)

		// Delete
		err = svc.DeletePengumuman(ctx, adminActor, created.ID, "127.0.0.1", "agent")
		require.NoError(t, err)

		// Verifikasi sudah terhapus
		_, err = svc.GetPengumumanByID(ctx, adminActor, created.ID)
		require.Error(t, err)
	})
}
