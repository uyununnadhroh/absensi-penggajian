package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"absensi/internal/payroll"
	"absensi/internal/store"
)

func (s *Server) myLeaves(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	return s.Store.ListLeaves(r.Context(), e.ID, "", 100)
}

func (s *Server) createLeave(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	var req store.Leave
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	switch req.Jenis {
	case "izin", "sakit", "cuti":
	default:
		return nil, bad("jenis pengajuan harus izin, sakit, atau cuti")
	}
	from, err := s.parseDay(req.TanggalMulai)
	if err != nil {
		return nil, err
	}
	to, err := s.parseDay(req.TanggalSelesai)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, bad("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	if to.Sub(from).Hours()/24 > 30 {
		return nil, bad("satu pengajuan maksimal 31 hari")
	}
	if from.Before(s.now().AddDate(0, 0, -14)) {
		return nil, bad("pengajuan untuk tanggal lampau maksimal 14 hari ke belakang")
	}
	req.Alasan = strings.TrimSpace(req.Alasan)
	if req.Alasan == "" || len(req.Alasan) > 255 {
		return nil, bad("alasan wajib diisi (maks. 255 karakter)")
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if closed, err := s.Store.IsClosed(r.Context(), nil, d.Format(payroll.DayFmt)); err != nil {
			return nil, err
		} else if closed {
			return nil, conflict("periode gaji %s sudah ditutup", d.Format("2006-01"))
		}
	}
	req.EmployeeID = e.ID
	id, err := s.Store.InsertLeave(r.Context(), req)
	if err != nil {
		return nil, err
	}
	return map[string]int64{"id": id}, nil
}

func (s *Server) cancelLeave(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	n, err := s.Store.DeletePendingLeave(r.Context(), id, e.ID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, conflict("pengajuan tidak ditemukan atau sudah diputuskan admin")
	}
	return ok(), nil
}

func (s *Server) listLeaves(w http.ResponseWriter, r *http.Request) (any, error) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", "menunggu", "disetujui", "ditolak":
	default:
		return nil, bad("status tidak dikenal")
	}
	return s.Store.ListLeaves(r.Context(), 0, status, 300)
}

// decideLeave POST /api/leaves/{id}/setujui | tolak  body {"catatan": "..."}
// Disetujui: setiap hari kerja dalam rentang dicatat sebagai izin/sakit/cuti.
// Hari yang sudah tercatat hadir (karyawan absen masuk) tidak ditimpa.
func (s *Server) decideLeave(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	action := r.PathValue("action")
	if action != "setujui" && action != "tolak" {
		return nil, notFound("aksi tidak dikenal")
	}
	var req struct {
		Catatan string `json:"catatan"`
	}
	if r.ContentLength > 0 {
		if err := decode(r, &req); err != nil {
			return nil, err
		}
	}
	if len(req.Catatan) > 255 {
		return nil, bad("catatan maksimal 255 karakter")
	}
	ctx := r.Context()
	rl, err := s.rules(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := s.Store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	l, err := s.Store.LeaveForUpdate(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if l.Status != "menunggu" {
		return nil, conflict("pengajuan ini sudah %s", l.Status)
	}
	status := "ditolak"
	dicatat := 0
	if action == "setujui" {
		status = "disetujui"
		e, err := s.Store.EmployeeByID(ctx, l.EmployeeID)
		if err != nil {
			return nil, err
		}
		hol, err := s.Store.HolidayMap(ctx, l.TanggalMulai, l.TanggalSelesai)
		if err != nil {
			return nil, err
		}
		from, _ := s.parseDay(l.TanggalMulai)
		to, _ := s.parseDay(l.TanggalSelesai)
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			if !rl.IsWorkday(d, hol) {
				continue
			}
			day := d.Format(payroll.DayFmt)
			if closed, err := s.Store.IsClosed(ctx, tx, day); err != nil {
				return nil, err
			} else if closed {
				return nil, conflict("periode gaji %s sudah ditutup", day[:7])
			}
			cur, err := s.Store.AttendanceOn(ctx, tx, e.ID, day)
			a := store.Attendance{EmployeeID: e.ID, Tanggal: day, Status: l.Jenis, Sumber: "pengajuan",
				JadwalMasuk: e.JamMasuk + ":00", JadwalPulang: e.JamPulang + ":00",
				Catatan: truncate(fmt.Sprintf("Pengajuan #%d: %s", l.ID, l.Alasan), 255)}
			switch {
			case err == nil && cur.Status == "hadir" && cur.JamMasuk != nil:
				continue // sudah masuk kerja, tidak ditimpa
			case err == nil:
				a.JadwalMasuk, a.JadwalPulang = cur.JadwalMasuk, cur.JadwalPulang
			case !errors.Is(err, store.ErrNotFound):
				return nil, err
			}
			if err := s.Store.UpsertAttendance(ctx, tx, a, "", actor(r).UserID); err != nil {
				return nil, err
			}
			dicatat++
		}
	}
	if err := s.Store.DecideLeave(ctx, tx, l.ID, status, strings.TrimSpace(req.Catatan), actor(r).UserID, s.now()); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "hari_dicatat": dicatat})
	return nil, nil
}

func truncate(v string, n int) string {
	if rs := []rune(v); len(rs) > n {
		return string(rs[:n])
	}
	return v
}
