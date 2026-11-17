// Package payroll berisi aturan murni perhitungan absensi & gaji (tanpa database),
// supaya mudah dites dan dipakai ulang oleh rekap bulanan maupun slip gaji.
package payroll

import (
	"math"
	"time"
)

const DayFmt = "2006-01-02"

// Rules parameter dari tabel settings.
type Rules struct {
	ToleransiTelatMenit   int          `json:"toleransi_telat_menit"`
	PotonganTelatPerMenit float64      `json:"potongan_telat_per_menit"`
	HariKerjaSebulan      int          `json:"hari_kerja_sebulan"`
	LemburMinimalMenit    int          `json:"lembur_minimal_menit"`
	LiburMingguan         map[int]bool `json:"-"` // time.Weekday -> libur
	AwalAbsenMenit        int          `json:"awal_absen_menit"`
}

// Employee data karyawan yang dibutuhkan perhitungan.
type Employee struct {
	ID                int64
	TipeGaji          string // bulanan | harian | per_jam
	GajiPokok         float64
	TarifPerJam       float64 // untuk tipe_gaji = per_jam
	TunjanganTetap    float64
	UangMakan         float64
	TarifLemburPerJam float64
	TanggalMasuk      *time.Time
}

// Record satu baris absensi.
type Record struct {
	Tanggal      string // YYYY-MM-DD
	Status       string // hadir izin sakit cuti alpa
	JamMasuk     *time.Time
	JamPulang    *time.Time
	JadwalMasuk  string // HH:MM:SS
	JadwalPulang string
	Catatan      string
}

// Day hasil olahan satu hari untuk rekap/slip.
type Day struct {
	Tanggal          string  `json:"tanggal"`
	Status           string  `json:"status"` // hadir izin sakit cuti alpa libur belum
	Keterangan       string  `json:"keterangan,omitempty"`
	JamMasuk         string  `json:"jam_masuk,omitempty"`
	JamPulang        string  `json:"jam_pulang,omitempty"`
	TelatMenit       int     `json:"telat_menit"`
	PulangCepatMenit int     `json:"pulang_cepat_menit"`
	LemburMenit      int     `json:"lembur_menit"`
	TanpaAbsenPulang bool    `json:"tanpa_absen_pulang,omitempty"`
	JamKerja         float64 `json:"jam_kerja,omitempty"` // total jam kerja hari ini
}

// Summary rekap jumlah hari & menit.
type Summary struct {
	HariKerja        int     `json:"hari_kerja"`
	Hadir            int     `json:"hadir"`
	Izin             int     `json:"izin"`
	Sakit            int     `json:"sakit"`
	Cuti             int     `json:"cuti"`
	Alpa             int     `json:"alpa"`
	Telat            int     `json:"telat"` // jumlah hari telat
	TelatMenit       int     `json:"telat_menit"`
	PulangCepatMenit int     `json:"pulang_cepat_menit"`
	LemburMenit      int     `json:"lembur_menit"`
	TanpaAbsenPulang int     `json:"tanpa_absen_pulang"`
	TotalJamKerja    float64 `json:"total_jam_kerja"` // untuk sistem per jam
}

// Pay rincian gaji dalam rupiah (dibulatkan).
type Pay struct {
	GajiPokok      float64 `json:"gaji_pokok"`
	TunjanganTetap float64 `json:"tunjangan_tetap"`
	UangMakan      float64 `json:"uang_makan"`
	Lembur         float64 `json:"lembur"`
	PotonganAlpa   float64 `json:"potongan_alpa"`
	PotonganTelat  float64 `json:"potongan_telat"`
	Pendapatan     float64 `json:"total_pendapatan"`
	Potongan       float64 `json:"total_potongan"`
	GajiBersih     float64 `json:"gaji_bersih"`
}

type Result struct {
	Summary Summary `json:"rekap"`
	Pay     Pay     `json:"rincian"`
	Days    []Day   `json:"harian"`
}

// IsWorkday hari kerja = bukan libur mingguan dan bukan hari libur.
func (r Rules) IsWorkday(d time.Time, holidays map[string]string) bool {
	if r.LiburMingguan[int(d.Weekday())] {
		return false
	}
	_, libur := holidays[d.Format(DayFmt)]
	return !libur
}

// At menggabungkan tanggal dan jam "HH:MM[:SS]" di zona loc.
func At(day time.Time, hhmm string, loc *time.Location) time.Time {
	var h, m, s int
	parseClock(hhmm, &h, &m, &s)
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, s, 0, loc)
}

func parseClock(v string, h, m, s *int) {
	n := 0
	parts := [3]int{}
	cur := 0
	for i := 0; i < len(v) && n < 3; i++ {
		c := v[i]
		if c >= '0' && c <= '9' {
			cur = cur*10 + int(c-'0')
			continue
		}
		if c == ':' {
			parts[n] = cur
			n++
			cur = 0
		}
	}
	if n < 3 {
		parts[n] = cur
	}
	*h, *m, *s = parts[0], parts[1], parts[2]
}

// Evaluate menghitung jam kerja aktual atau telat/lembur jika ada jadwal.
func (r Rules) Evaluate(rec Record, loc *time.Location) Day {
	d := Day{Tanggal: rec.Tanggal, Status: rec.Status, Keterangan: rec.Catatan}
	if rec.JamMasuk != nil {
		d.JamMasuk = rec.JamMasuk.In(loc).Format("15:04")
	}
	if rec.JamPulang != nil {
		d.JamPulang = rec.JamPulang.In(loc).Format("15:04")
	}
	if rec.Status != "hadir" {
		return d
	}
	
	// Hitung total jam kerja dari masuk sampai pulang (untuk semua sistem)
	if rec.JamMasuk != nil && rec.JamPulang != nil {
		duration := rec.JamPulang.Sub(*rec.JamMasuk)
		d.JamKerja = math.Round(duration.Hours()*100) / 100 // bulatkan 2 desimal
	}
	
	if rec.JamPulang == nil {
		d.TanpaAbsenPulang = true
		return d
	}
	
	// Hitung telat, pulang cepat, dan lembur HANYA jika ada jadwal (untuk sistem bulanan/harian)
	if rec.JadwalMasuk == "" || rec.JadwalPulang == "" {
		return d // Tidak ada jadwal = tidak ada konsep telat/lembur
	}
	
	day, err := time.ParseInLocation(DayFmt, rec.Tanggal, loc)
	if err != nil {
		return d
	}
	start := At(day, rec.JadwalMasuk, loc)
	end := At(day, rec.JadwalPulang, loc)
	if !end.After(start) { // shift malam: pulang keesokan harinya
		end = end.Add(24 * time.Hour)
	}
	if rec.JamMasuk != nil {
		late := int(rec.JamMasuk.Sub(start).Minutes())
		if late > r.ToleransiTelatMenit {
			d.TelatMenit = late
		}
	}
	if diff := int(end.Sub(*rec.JamPulang).Minutes()); diff > 0 {
		d.PulangCepatMenit = diff
	}
	if over := int(rec.JamPulang.Sub(end).Minutes()); over > 0 {
		step := r.LemburMinimalMenit
		if step <= 0 {
			step = 1
		}
		d.LemburMenit = over / step * step
	}
	return d
}

// Calculate rekap dan gaji satu karyawan untuk satu bulan.
//
//   - Hari kerja tanpa absensi sebelum hari ini dihitung alpa (hari ini belum, karena masih berjalan).
//   - Hari sebelum tanggal masuk karyawan tidak dihitung.
//   - Bulanan: gaji pokok penuh (prorata bila masuk di tengah bulan), dipotong
//     gaji_pokok / hari_kerja_sebulan per hari alpa. Izin/sakit/cuti tetap dibayar.
//   - Harian: gaji pokok x hari hadir.
//   - Uang makan x hari hadir, lembur = menit/60 x tarif per jam,
//     potongan telat = (menit telat + menit pulang cepat) x potongan per menit.
func Calculate(emp Employee, r Rules, month time.Time, today time.Time, recs []Record, holidays map[string]string, loc *time.Location) Result {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, loc)
	next := first.AddDate(0, 1, 0)
	todayDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)

	byDay := make(map[string]Record, len(recs))
	for _, rc := range recs {
		byDay[rc.Tanggal] = rc
	}

	var start time.Time
	if emp.TanggalMasuk != nil {
		start = time.Date(emp.TanggalMasuk.Year(), emp.TanggalMasuk.Month(), emp.TanggalMasuk.Day(), 0, 0, 0, 0, loc)
	}

	res := Result{Days: []Day{}}
	s := &res.Summary
	workdaysMonth, workdaysEmployed := 0, 0
	for d := first; d.Before(next); d = d.AddDate(0, 0, 1) {
		key := d.Format(DayFmt)
		work := r.IsWorkday(d, holidays)
		employed := start.IsZero() || !d.Before(start)
		if work {
			workdaysMonth++
			if employed {
				workdaysEmployed++
			}
		}
		rec, has := byDay[key]
		var day Day
		switch {
		case has:
			day = r.Evaluate(rec, loc)
		case !employed:
			day = Day{Tanggal: key, Status: "belum", Keterangan: "Belum bekerja"}
		case !work:
			day = Day{Tanggal: key, Status: "libur", Keterangan: holidays[key]}
		case d.Before(todayDay):
			day = Day{Tanggal: key, Status: "alpa", Keterangan: "Tidak absen"}
		default:
			day = Day{Tanggal: key, Status: "belum"}
		}
		switch day.Status {
		case "hadir":
			s.Hadir++
			s.TotalJamKerja += day.JamKerja // akumulasi total jam kerja
			if day.TelatMenit > 0 {
				s.Telat++
			}
			s.TelatMenit += day.TelatMenit
			s.PulangCepatMenit += day.PulangCepatMenit
			s.LemburMenit += day.LemburMenit
			if day.TanpaAbsenPulang {
				s.TanpaAbsenPulang++
			}
		case "izin":
			s.Izin++
		case "sakit":
			s.Sakit++
		case "cuti":
			s.Cuti++
		case "alpa":
			s.Alpa++
		}
		res.Days = append(res.Days, day)
	}
	s.HariKerja = workdaysEmployed

	p := &res.Pay
	
	// Sistem Gaji Per Jam
	if emp.TipeGaji == "per_jam" {
		p.GajiPokok = emp.TarifPerJam * s.TotalJamKerja
		// Tidak ada potongan alpa untuk sistem per jam, karena tidak masuk = tidak dapat gaji
	} else if emp.TipeGaji == "harian" {
		p.GajiPokok = emp.GajiPokok * float64(s.Hadir)
	} else {
		// Sistem bulanan (existing)
		p.GajiPokok = emp.GajiPokok
		if workdaysMonth > 0 && workdaysEmployed < workdaysMonth {
			p.GajiPokok = emp.GajiPokok * float64(workdaysEmployed) / float64(workdaysMonth)
		}
		div := r.HariKerjaSebulan
		if div <= 0 {
			div = 26
		}
		p.PotonganAlpa = emp.GajiPokok / float64(div) * float64(s.Alpa)
		// potongan alpa tidak melebihi gaji pokok; alpa sepanjang hari kerja = gaji pokok hangus
		if p.PotonganAlpa > p.GajiPokok || (s.Alpa > 0 && s.Alpa >= workdaysEmployed) {
			p.PotonganAlpa = p.GajiPokok
		}
	}
	p.TunjanganTetap = emp.TunjanganTetap
	p.UangMakan = emp.UangMakan * float64(s.Hadir)
	p.Lembur = float64(s.LemburMenit) / 60 * emp.TarifLemburPerJam
	p.PotonganTelat = float64(s.TelatMenit+s.PulangCepatMenit) * r.PotonganTelatPerMenit

	p.GajiPokok = round(p.GajiPokok)
	p.UangMakan = round(p.UangMakan)
	p.Lembur = round(p.Lembur)
	p.PotonganAlpa = round(p.PotonganAlpa)
	p.PotonganTelat = round(p.PotonganTelat)
	p.TunjanganTetap = round(p.TunjanganTetap)
	p.Pendapatan = p.GajiPokok + p.TunjanganTetap + p.UangMakan + p.Lembur
	p.Potongan = p.PotonganAlpa + p.PotonganTelat
	if p.Potongan > p.Pendapatan { // gaji bersih tidak pernah minus
		p.Potongan = p.Pendapatan
	}
	p.GajiBersih = p.Pendapatan - p.Potongan
	return res
}

func round(v float64) float64 { return math.Round(v) }
