package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"absensi/internal/payroll"
	"absensi/internal/store"
)

// rules membaca parameter absensi & gaji dari tabel settings.
func (s *Server) rules(ctx context.Context) (payroll.Rules, error) {
	m, err := s.Store.Settings(ctx)
	if err != nil {
		return payroll.Rules{}, err
	}
	r := payroll.Rules{
		ToleransiTelatMenit:   store.Atoi(m, "toleransi_telat_menit", 10),
		PotonganTelatPerMenit: store.Atof(m, "potongan_telat_per_menit", 0),
		HariKerjaSebulan:      store.Atoi(m, "hari_kerja_sebulan", 26),
		LemburMinimalMenit:    store.Atoi(m, "lembur_minimal_menit", 30),
		AwalAbsenMenit:        store.Atoi(m, "awal_absen_menit", 120),
		LiburMingguan:         map[int]bool{},
	}
	for _, p := range strings.Split(m["hari_libur_mingguan"], ",") {
		if d, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && d >= 0 && d <= 6 {
			r.LiburMingguan[d] = true
		}
	}
	return r, nil
}

func (s *Server) parseDay(v string) (time.Time, error) {
	t, err := time.ParseInLocation(payroll.DayFmt, v, s.Loc)
	if err != nil {
		return t, bad("tanggal harus berformat YYYY-MM-DD")
	}
	return t, nil
}

func toRecord(a store.Attendance) payroll.Record {
	return payroll.Record{Tanggal: a.Tanggal, Status: a.Status, JamMasuk: a.JamMasuk, JamPulang: a.JamPulang,
		JadwalMasuk: a.JadwalMasuk, JadwalPulang: a.JadwalPulang, Catatan: a.Catatan}
}

func overnight(masuk, pulang string) bool { return pulang <= masuk }

func hm(t time.Time) string { return t.Format("15:04") }

// ------------------------------------------------------------------ karyawan: absen sendiri

type todayView struct {
	Tanggal      string            `json:"tanggal"`
	ServerTime   time.Time         `json:"server_time"`
	JadwalMasuk  string            `json:"jadwal_masuk"`
	JadwalPulang string            `json:"jadwal_pulang"`
	HariKerja    bool              `json:"hari_kerja"`
	Libur        string            `json:"libur,omitempty"`
	Absensi      *store.Attendance `json:"absensi"`
	Hasil        *payroll.Day      `json:"hasil,omitempty"`
	Terbuka      *store.Attendance `json:"absen_terbuka,omitempty"` // shift malam kemarin yang belum absen pulang
	BisaMasuk    bool              `json:"bisa_masuk"`
	BisaPulang   bool              `json:"bisa_pulang"`
	MasukMulai   string            `json:"absen_masuk_mulai"`
	Pesan        string            `json:"pesan"`
}

func (s *Server) buildToday(ctx context.Context, emp store.Employee) (todayView, error) {
	now := s.now()
	day := now.Format(payroll.DayFmt)
	v := todayView{Tanggal: day, ServerTime: now, JadwalMasuk: emp.JamMasuk, JadwalPulang: emp.JamPulang}
	r, err := s.rules(ctx)
	if err != nil {
		return v, err
	}
	hol, err := s.Store.HolidayMap(ctx, day, day)
	if err != nil {
		return v, err
	}
	dayT, _ := s.parseDay(day)
	v.HariKerja = r.IsWorkday(dayT, hol)
	v.Libur = hol[day]
	start := payroll.At(dayT, emp.JamMasuk, s.Loc).Add(-time.Duration(r.AwalAbsenMenit) * time.Minute)
	v.MasukMulai = hm(start)

	if open, err := s.openShift(ctx, nil, emp.ID, now); err == nil && open.Tanggal != day {
		v.Terbuka = &open
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		return v, err
	}

	a, err := s.Store.AttendanceOn(ctx, nil, emp.ID, day)
	switch {
	case errors.Is(err, store.ErrNotFound):
		v.BisaMasuk = v.Terbuka == nil && !now.Before(start)
		switch {
		case v.Terbuka != nil:
			v.Pesan = "Selesaikan dulu absen pulang shift sebelumnya."
		case now.Before(start):
			v.Pesan = "Absen masuk dibuka mulai pukul " + v.MasukMulai + "."
		case !v.HariKerja:
			v.Pesan = "Hari ini hari libur. Absen hanya bila memang dijadwalkan masuk."
		default:
			v.Pesan = "Anda belum absen masuk hari ini."
		}
	case err != nil:
		return v, err
	default:
		v.Absensi = &a
		d := r.Evaluate(toRecord(a), s.Loc)
		v.Hasil = &d
		switch {
		case a.Status != "hadir":
			v.Pesan = "Hari ini tercatat " + a.Status + "."
		case a.JamPulang == nil:
			v.Pesan = "Sudah absen masuk pukul " + hm(*a.JamMasuk) + ". Jangan lupa absen pulang."
		default:
			v.Pesan = "Absensi hari ini lengkap. Terima kasih!"
		}
	}
	v.BisaPulang = v.Terbuka != nil || (v.Absensi != nil && v.Absensi.Status == "hadir" && v.Absensi.JamMasuk != nil && v.Absensi.JamPulang == nil)
	return v, nil
}

// openShift absen masuk yang belum absen pulang: hari ini, atau kemarin bila shift
// malam (jadwal pulang melewati tengah malam) dan belum lewat 12 jam dari jadwal pulang.
func (s *Server) openShift(ctx context.Context, tx *sql.Tx, empID int64, now time.Time) (store.Attendance, error) {
	yesterday := now.AddDate(0, 0, -1).Format(payroll.DayFmt)
	var a store.Attendance
	var err error
	if tx == nil {
		var t, e = s.Store.Begin(ctx)
		if e != nil {
			return a, e
		}
		defer t.Rollback()
		a, err = s.Store.OpenAttendance(ctx, t, empID, yesterday)
	} else {
		a, err = s.Store.OpenAttendance(ctx, tx, empID, yesterday)
	}
	if err != nil {
		return a, err
	}
	if a.Tanggal == now.Format(payroll.DayFmt) {
		return a, nil
	}
	if !overnight(a.JadwalMasuk, a.JadwalPulang) {
		return a, store.ErrNotFound
	}
	d, _ := s.parseDay(a.Tanggal)
	end := payroll.At(d.AddDate(0, 0, 1), a.JadwalPulang, s.Loc)
	if now.After(end.Add(12 * time.Hour)) {
		return a, store.ErrNotFound
	}
	return a, nil
}

func (s *Server) myEmployee(r *http.Request) (store.Employee, error) {
	e, err := s.Store.EmployeeByID(r.Context(), actor(r).EmployeeID)
	if errors.Is(err, store.ErrNotFound) {
		return e, httpError{http.StatusForbidden, "akun ini belum terhubung ke data karyawan"}
	}
	if err == nil && !e.Aktif {
		return e, httpError{http.StatusForbidden, "data karyawan tidak aktif"}
	}
	return e, err
}

func (s *Server) myToday(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	return s.buildToday(r.Context(), e)
}

func (s *Server) checkIn(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	now := s.now()
	day := now.Format(payroll.DayFmt)
	
	// Validasi waktu absen hanya jika ada jadwal
	if e.JamMasuk != "" {
		rl, err := s.rules(ctx)
		if err != nil {
			return nil, err
		}
		dayT, _ := s.parseDay(day)
		start := payroll.At(dayT, e.JamMasuk, s.Loc).Add(-time.Duration(rl.AwalAbsenMenit) * time.Minute)
		if now.Before(start) {
			return nil, bad("absen masuk baru dibuka pukul %s", hm(start))
		}
	}

	tx, err := s.Store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if closed, err := s.Store.IsClosed(ctx, tx, day); err != nil {
		return nil, err
	} else if closed {
		return nil, conflict("periode gaji bulan ini sudah ditutup, hubungi admin")
	}
	if _, err := s.openShift(ctx, tx, e.ID, now); err == nil {
		return nil, conflict("anda belum absen pulang dari shift sebelumnya")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	cur, err := s.Store.AttendanceOn(ctx, tx, e.ID, day)
	if err == nil {
		if cur.Status == "hadir" {
			return nil, conflict("anda sudah absen masuk hari ini")
		}
		return nil, conflict("hari ini anda tercatat %s, hubungi admin bila ingin mengubah", cur.Status)
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	
	// Jadwal diisi dari data karyawan jika ada, atau kosong jika tidak ada
	jadwalMasuk := ""
	jadwalPulang := ""
	if e.JamMasuk != "" {
		jadwalMasuk = e.JamMasuk + ":00"
	}
	if e.JamPulang != "" {
		jadwalPulang = e.JamPulang + ":00"
	}
	
	a := store.Attendance{EmployeeID: e.ID, Tanggal: day, Status: "hadir", JamMasuk: &now,
		JadwalMasuk: jadwalMasuk, JadwalPulang: jadwalPulang, Sumber: "karyawan"}
	if err := s.Store.UpsertAttendance(ctx, tx, a, clientIP(r), actor(r).UserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.buildToday(ctx, e)
}

func (s *Server) checkOut(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	now := s.now()
	tx, err := s.Store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	a, err := s.openShift(ctx, tx, e.ID, now)
	if errors.Is(err, store.ErrNotFound) {
		return nil, conflict("tidak ada absen masuk yang bisa ditutup (belum absen masuk atau sudah absen pulang)")
	} else if err != nil {
		return nil, err
	}
	if closed, err := s.Store.IsClosed(ctx, tx, a.Tanggal); err != nil {
		return nil, err
	} else if closed {
		return nil, conflict("periode gaji sudah ditutup, hubungi admin")
	}
	if err := s.Store.SetCheckOut(ctx, tx, a.ID, now, clientIP(r)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.buildToday(ctx, e)
}

// ------------------------------------------------------------------ admin: absensi per tanggal

type attRow struct {
	Employee store.Employee    `json:"karyawan"`
	Absensi  *store.Attendance `json:"absensi"`
	Hasil    payroll.Day       `json:"hasil"`
}

type attDay struct {
	Tanggal   string   `json:"tanggal"`
	HariKerja bool     `json:"hari_kerja"`
	Libur     string   `json:"libur,omitempty"`
	Ditutup   bool     `json:"ditutup"`
	Rows      []attRow `json:"rows"`
}

func (s *Server) buildDay(ctx context.Context, day string) (attDay, error) {
	out := attDay{Tanggal: day, Rows: []attRow{}}
	dayT, err := s.parseDay(day)
	if err != nil {
		return out, err
	}
	r, err := s.rules(ctx)
	if err != nil {
		return out, err
	}
	hol, err := s.Store.HolidayMap(ctx, day, day)
	if err != nil {
		return out, err
	}
	out.HariKerja = r.IsWorkday(dayT, hol)
	out.Libur = hol[day]
	if out.Ditutup, err = s.Store.IsClosed(ctx, nil, day); err != nil {
		return out, err
	}
	emps, err := s.Store.ListEmployees(ctx, false)
	if err != nil {
		return out, err
	}
	recs, err := s.Store.AttendanceByDate(ctx, day)
	if err != nil {
		return out, err
	}
	byEmp := map[int64]store.Attendance{}
	for _, a := range recs {
		byEmp[a.EmployeeID] = a
	}
	today := s.today()
	for _, e := range emps {
		a, has := byEmp[e.ID]
		started := e.TanggalMasuk == "" || e.TanggalMasuk <= day
		if !has && (!e.Aktif || !started) {
			continue
		}
		row := attRow{Employee: e}
		switch {
		case has:
			row.Absensi = &a
			row.Hasil = r.Evaluate(toRecord(a), s.Loc)
		case !out.HariKerja:
			row.Hasil = payroll.Day{Tanggal: day, Status: "libur", Keterangan: out.Libur}
		case day < today:
			row.Hasil = payroll.Day{Tanggal: day, Status: "alpa", Keterangan: "Tidak absen"}
		default:
			row.Hasil = payroll.Day{Tanggal: day, Status: "belum", Keterangan: "Belum absen"}
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func (s *Server) attendanceByDate(w http.ResponseWriter, r *http.Request) (any, error) {
	day := r.URL.Query().Get("tanggal")
	if day == "" {
		day = s.today()
	}
	return s.buildDay(r.Context(), day)
}

// saveAttendance koreksi / input manual oleh admin. status kosong = hapus catatan hari itu.
func (s *Server) saveAttendance(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		EmployeeID int64  `json:"employee_id"`
		Tanggal    string `json:"tanggal"`
		Status     string `json:"status"`
		JamMasuk   string `json:"jam_masuk"`  // HH:MM
		JamPulang  string `json:"jam_pulang"` // HH:MM, boleh kosong
		Catatan    string `json:"catatan"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	ctx := r.Context()
	dayT, err := s.parseDay(req.Tanggal)
	if err != nil {
		return nil, err
	}
	if req.Tanggal > s.today() {
		return nil, bad("tidak bisa mengisi absensi untuk tanggal yang akan datang")
	}
	e, err := s.Store.EmployeeByID(ctx, req.EmployeeID)
	if err != nil {
		return nil, err
	}
	if closed, err := s.Store.IsClosed(ctx, nil, req.Tanggal); err != nil {
		return nil, err
	} else if closed {
		return nil, conflict("periode gaji %s sudah ditutup; buka kembali periode untuk mengubah absensi", req.Tanggal[:7])
	}
	if req.Status == "" {
		return ok(), s.Store.DeleteAttendance(ctx, e.ID, req.Tanggal)
	}
	switch req.Status {
	case "hadir", "izin", "sakit", "cuti", "alpa":
	default:
		return nil, bad("status tidak dikenal")
	}
	if len(req.Catatan) > 255 {
		return nil, bad("catatan maksimal 255 karakter")
	}
	a := store.Attendance{EmployeeID: e.ID, Tanggal: req.Tanggal, Status: req.Status, Catatan: strings.TrimSpace(req.Catatan),
		JadwalMasuk: e.JamMasuk + ":00", JadwalPulang: e.JamPulang + ":00", Sumber: "admin"}
	if cur, err := s.Store.AttendanceOn(ctx, nil, e.ID, req.Tanggal); err == nil {
		a.JadwalMasuk, a.JadwalPulang = cur.JadwalMasuk, cur.JadwalPulang // jadwal saat itu dipertahankan
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	if req.Status == "hadir" {
		if !reClock.MatchString(req.JamMasuk) {
			return nil, bad("jam masuk wajib diisi (HH:MM) untuk status hadir")
		}
		in := payroll.At(dayT, req.JamMasuk, s.Loc)
		a.JamMasuk = &in
		if req.JamPulang != "" {
			if !reClock.MatchString(req.JamPulang) {
				return nil, bad("jam pulang harus berformat HH:MM")
			}
			out := payroll.At(dayT, req.JamPulang, s.Loc)
			if !out.After(in) { // pulang lewat tengah malam
				out = out.Add(24 * time.Hour)
			}
			a.JamPulang = &out
		}
	}
	return ok(), s.Store.UpsertAttendance(ctx, nil, a, "", actor(r).UserID)
}

// ------------------------------------------------------------------ dashboard admin

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) (any, error) {
	ctx := r.Context()
	d, err := s.buildDay(ctx, s.today())
	if err != nil {
		return nil, err
	}
	count := map[string]int{"karyawan": len(d.Rows), "hadir": 0, "telat": 0, "belum": 0, "izin": 0, "sakit": 0, "cuti": 0, "alpa": 0, "libur": 0, "belum_pulang": 0}
	for _, row := range d.Rows {
		count[row.Hasil.Status]++
		if row.Hasil.TelatMenit > 0 {
			count["telat"]++
		}
		if row.Hasil.Status == "hadir" && row.Hasil.TanpaAbsenPulang {
			count["belum_pulang"]++
		}
	}
	pending, err := s.Store.CountPendingLeaves(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"hari_ini": d, "jumlah": count, "pengajuan_menunggu": pending, "server_time": s.now()}, nil
}
