package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"absensi/internal/payroll"
	"absensi/internal/store"
)

var reMonth = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type payslip struct {
	Karyawan store.Employee  `json:"karyawan"`
	Rekap    payroll.Summary `json:"rekap"`
	Rincian  payroll.Pay     `json:"rincian"`
	Harian   []payroll.Day   `json:"harian"`
}

type payrollReport struct {
	Bulan        string             `json:"bulan"`
	Ditutup      bool               `json:"ditutup"`
	DitutupPada  *time.Time         `json:"ditutup_pada,omitempty"`
	DihitungPada time.Time          `json:"dihitung_pada"`
	Aturan       payroll.Rules      `json:"aturan"`
	LiburMinggu  []int              `json:"hari_libur_mingguan"`
	HariLibur    []store.Holiday    `json:"hari_libur"`
	Karyawan     []payslip          `json:"karyawan"`
	Total        map[string]float64 `json:"total"`
}

func (s *Server) parseMonth(v string) (time.Time, error) {
	if v == "" {
		v = s.now().Format("2006-01")
	}
	if !reMonth.MatchString(v) {
		return time.Time{}, bad("bulan harus berformat YYYY-MM")
	}
	return time.ParseInLocation("2006-01", v, s.Loc)
}

// computePayroll menghitung gaji semua karyawan (atau satu karyawan bila onlyEmp != 0).
func (s *Server) computePayroll(ctx context.Context, month time.Time, onlyEmp int64) (payrollReport, error) {
	from := month.Format(payroll.DayFmt)
	to := month.AddDate(0, 1, -1).Format(payroll.DayFmt)
	rep := payrollReport{Bulan: month.Format("2006-01"), DihitungPada: s.now(), Karyawan: []payslip{},
		Total: map[string]float64{"total_pendapatan": 0, "total_potongan": 0, "gaji_bersih": 0}}
	rl, err := s.rules(ctx)
	if err != nil {
		return rep, err
	}
	rep.Aturan = rl
	rep.LiburMinggu = []int{}
	for d := 0; d <= 6; d++ {
		if rl.LiburMingguan[d] {
			rep.LiburMinggu = append(rep.LiburMinggu, d)
		}
	}
	if rep.HariLibur, err = s.Store.ListHolidays(ctx, from, to); err != nil {
		return rep, err
	}
	hol := map[string]string{}
	for _, h := range rep.HariLibur {
		hol[h.Tanggal] = h.Keterangan
	}
	emps, err := s.Store.ListEmployees(ctx, false)
	if err != nil {
		return rep, err
	}
	recs, err := s.Store.AttendanceRange(ctx, onlyEmp, from, to)
	if err != nil {
		return rep, err
	}
	byEmp := map[int64][]payroll.Record{}
	for _, a := range recs {
		byEmp[a.EmployeeID] = append(byEmp[a.EmployeeID], toRecord(a))
	}
	for _, e := range emps {
		if onlyEmp != 0 && e.ID != onlyEmp {
			continue
		}
		// karyawan nonaktif hanya muncul bila punya absensi di bulan itu;
		// karyawan yang baru masuk setelah bulan ini tidak dimunculkan
		if (!e.Aktif && len(byEmp[e.ID]) == 0) || (e.TanggalMasuk != "" && e.TanggalMasuk > to) {
			continue
		}
		pe := payroll.Employee{ID: e.ID, TipeGaji: e.TipeGaji, GajiPokok: e.GajiPokok, TarifPerJam: e.TarifPerJam, 
			TunjanganTetap: e.TunjanganTetap, UangMakan: e.UangMakan, TarifLemburPerJam: e.TarifLemburPerJam}
		if e.TanggalMasuk != "" {
			t, _ := s.parseDay(e.TanggalMasuk)
			pe.TanggalMasuk = &t
		}
		res := payroll.Calculate(pe, rl, month, s.now(), byEmp[e.ID], hol, s.Loc)
		rep.Karyawan = append(rep.Karyawan, payslip{Karyawan: e, Rekap: res.Summary, Rincian: res.Pay, Harian: res.Days})
		rep.Total["total_pendapatan"] += res.Pay.Pendapatan
		rep.Total["total_potongan"] += res.Pay.Potongan
		rep.Total["gaji_bersih"] += res.Pay.GajiBersih
	}
	return rep, nil
}

// report: snapshot bila periode sudah ditutup, selain itu dihitung langsung.
func (s *Server) report(ctx context.Context, month time.Time, onlyEmp int64) (payrollReport, error) {
	p, closed, err := s.Store.Period(ctx, month.Format("2006-01"))
	if err != nil {
		return payrollReport{}, err
	}
	if !closed {
		return s.computePayroll(ctx, month, onlyEmp)
	}
	var rep payrollReport
	if err := json.Unmarshal(p.Data, &rep); err != nil {
		return rep, err
	}
	rep.Ditutup = true
	t := p.DitutupPada.In(s.Loc)
	rep.DitutupPada = &t
	if onlyEmp != 0 {
		keep := []payslip{}
		for _, k := range rep.Karyawan {
			if k.Karyawan.ID == onlyEmp {
				keep = append(keep, k)
			}
		}
		rep.Karyawan = keep
		rep.Total = nil
	}
	return rep, nil
}

func (s *Server) payrollMonth(w http.ResponseWriter, r *http.Request) (any, error) {
	month, err := s.parseMonth(r.URL.Query().Get("bulan"))
	if err != nil {
		return nil, err
	}
	return s.report(r.Context(), month, 0)
}

// myPayroll rekap absensi + slip gaji karyawan yang login.
func (s *Server) myPayroll(w http.ResponseWriter, r *http.Request) (any, error) {
	e, err := s.myEmployee(r)
	if err != nil {
		return nil, err
	}
	month, err := s.parseMonth(r.URL.Query().Get("bulan"))
	if err != nil {
		return nil, err
	}
	rep, err := s.report(r.Context(), month, e.ID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"bulan": rep.Bulan, "ditutup": rep.Ditutup, "ditutup_pada": rep.DitutupPada,
		"aturan": rep.Aturan, "hari_libur": rep.HariLibur, "slip": nil}
	if len(rep.Karyawan) > 0 {
		out["slip"] = rep.Karyawan[0]
	}
	return out, nil
}

func (s *Server) closePayroll(w http.ResponseWriter, r *http.Request) (any, error) {
	month, err := s.parseMonth(r.PathValue("month"))
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	if month.AddDate(0, 1, 0).After(s.now()) {
		return nil, bad("periode %s belum berakhir; tutup periode setelah bulan selesai", month.Format("2006-01"))
	}
	if _, closed, err := s.Store.Period(ctx, month.Format("2006-01")); err != nil {
		return nil, err
	} else if closed {
		return nil, conflict("periode %s sudah ditutup", month.Format("2006-01"))
	}
	rep, err := s.computePayroll(ctx, month, 0)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(rep)
	if err != nil {
		return nil, err
	}
	if err := s.Store.ClosePeriod(ctx, rep.Bulan, data, actor(r).UserID); err != nil {
		return nil, duplicateClosed(err, rep.Bulan)
	}
	return s.report(ctx, month, 0)
}

func duplicateClosed(err error, bulan string) error {
	if e := duplicate(err); e != err {
		return conflict("periode %s sudah ditutup", bulan)
	}
	return err
}

func (s *Server) reopenPayroll(w http.ResponseWriter, r *http.Request) (any, error) {
	month, err := s.parseMonth(r.PathValue("month"))
	if err != nil {
		return nil, err
	}
	n, err := s.Store.ReopenPeriod(r.Context(), month.Format("2006-01"))
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, conflict("periode %s belum ditutup", month.Format("2006-01"))
	}
	return ok(), nil
}

// exportPayroll GET /api/payroll/export?bulan=YYYY-MM -> CSV (dibuka di Excel).
func (s *Server) exportPayroll(w http.ResponseWriter, r *http.Request) (any, error) {
	month, err := s.parseMonth(r.URL.Query().Get("bulan"))
	if err != nil {
		return nil, err
	}
	rep, err := s.report(r.Context(), month, 0)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="gaji-%s.csv"`, rep.Bulan))
	w.Write([]byte("\xEF\xBB\xBF")) // BOM supaya Excel membaca UTF-8
	cw := csv.NewWriter(w)
	cw.Comma = ';' // pemisah yang dikenali Excel berbahasa Indonesia
	cw.Write([]string{"NIK", "Nama", "Jabatan", "Tipe Gaji", "Hari Kerja", "Hadir", "Total Jam Kerja", "Izin", "Sakit", "Cuti", "Alpa",
		"Telat (menit)", "Pulang Cepat (menit)", "Lembur (menit)", "Gaji Pokok", "Tunjangan Tetap", "Uang Makan", "Lembur",
		"Potongan Alpa", "Potongan Telat", "Total Pendapatan", "Total Potongan", "Gaji Bersih"})
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	i := strconv.Itoa
	for _, k := range rep.Karyawan {
		e, a, p := k.Karyawan, k.Rekap, k.Rincian
		cw.Write([]string{e.NIK, e.Nama, e.Jabatan, e.TipeGaji, i(a.HariKerja), i(a.Hadir), f(a.TotalJamKerja), i(a.Izin), i(a.Sakit), i(a.Cuti), i(a.Alpa),
			i(a.TelatMenit), i(a.PulangCepatMenit), i(a.LemburMenit), f(p.GajiPokok), f(p.TunjanganTetap), f(p.UangMakan), f(p.Lembur),
			f(p.PotonganAlpa), f(p.PotonganTelat), f(p.Pendapatan), f(p.Potongan), f(p.GajiBersih)})
	}
	cw.Flush()
	return nil, cw.Error()
}
