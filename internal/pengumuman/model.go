// Package pengumuman mengelola data domain dan logika use-case untuk pengumuman sistem SPLK.
package pengumuman

import "time"

// Pengumuman merepresentasikan baris pada tabel pengumuman.
type Pengumuman struct {
	ID            int       `json:"id"`
	Judul         string    `json:"judul"`
	Isi           string    `json:"isi"`
	DibuatOleh    *int      `json:"dibuat_oleh,omitempty"`
	TanggalTerbit time.Time `json:"tanggal_terbit"`
	TargetRole    *string   `json:"target_role,omitempty"`
}

// PengumumanDetail berisi informasi pengumuman beserta nama username pembuatnya.
type PengumumanDetail struct {
	ID            int       `json:"id"`
	Judul         string    `json:"judul"`
	Isi           string    `json:"isi"`
	DibuatOleh    *int      `json:"dibuat_oleh,omitempty"`
	NamaPembuat   *string   `json:"nama_pembuat,omitempty"`
	TanggalTerbit time.Time `json:"tanggal_terbit"`
	TargetRole    *string   `json:"target_role,omitempty"`
}
