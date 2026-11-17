package payroll

import (
	"testing"
	"time"
)

var jkt, _ = time.LoadLocation("Asia/Jakarta")

func rules() Rules {
	return Rules{
		ToleransiTelatMenit: 10, PotonganTelatPerMenit: 1000, HariKerjaSebulan: 26,
		LemburMinimalMenit: 30, LiburMingguan: map[int]bool{0: true},
	}
}

func tm(s string) *time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, jkt)
	if err != nil {
		panic(err)
	}
	return &t
}

func hadir(day, in, out string) Record {
	r := Record{Tanggal: day, Status: "hadir", JadwalMasuk: "08:00:00", JadwalPulang: "16:00:00"}
	if in != "" {
		r.JamMasuk = tm(day + " " + in)
	}
	if out != "" {
		r.JamPulang = tm(day + " " + out)
	}
	return r
}

func TestEvaluate(t *testing.T) {
	r := rules()
	cases := []struct {
		name                 string
		rec                  Record
		telat, cepat, lembur int
		tanpaPulang          bool
	}{
		{"tepat waktu", hadir("2026-11-02", "07:55", "16:05"), 0, 0, 0, false},
		{"masih toleransi", hadir("2026-11-02", "08:10", "16:00"), 0, 0, 0, false},
		{"telat lewat toleransi dihitung penuh", hadir("2026-11-02", "08:11", "16:00"), 11, 0, 0, false},
		{"pulang cepat", hadir("2026-11-02", "08:00", "15:20"), 0, 40, 0, false},
		{"lembur dibulatkan ke bawah per 30 menit", hadir("2026-11-02", "08:00", "17:25"), 0, 0, 60, false},
		{"lupa absen pulang", hadir("2026-11-02", "08:00", ""), 0, 0, 0, true},
	}
	for _, c := range cases {
		d := r.Evaluate(c.rec, jkt)
		if d.TelatMenit != c.telat || d.PulangCepatMenit != c.cepat || d.LemburMenit != c.lembur || d.TanpaAbsenPulang != c.tanpaPulang {
			t.Errorf("%s: dapat telat=%d cepat=%d lembur=%d tanpaPulang=%v", c.name, d.TelatMenit, d.PulangCepatMenit, d.LemburMenit, d.TanpaAbsenPulang)
		}
	}
}

func TestEvaluateShiftMalam(t *testing.T) {
	r := rules()
	rec := Record{Tanggal: "2026-11-02", Status: "hadir", JadwalMasuk: "22:00:00", JadwalPulang: "06:00:00",
		JamMasuk: tm("2026-11-02 22:00"), JamPulang: tm("2026-11-03 07:00")}
	d := r.Evaluate(rec, jkt)
	if d.LemburMenit != 60 || d.PulangCepatMenit != 0 {
		t.Fatalf("shift malam: %+v", d)
	}
}

// November 2026: 30 hari, 5 hari Minggu -> 25 hari kerja (tanpa libur nasional).
func TestCalculateBulanan(t *testing.T) {
	emp := Employee{TipeGaji: "bulanan", GajiPokok: 2_600_000, TunjanganTetap: 200_000, UangMakan: 20_000, TarifLemburPerJam: 15_000}
	recs := []Record{
		hadir("2026-11-02", "08:00", "18:00"), // lembur 120 menit
		hadir("2026-11-03", "08:20", "16:00"), // telat 20 menit
		{Tanggal: "2026-11-04", Status: "sakit"},
		{Tanggal: "2026-11-05", Status: "izin"},
	}
	holidays := map[string]string{"2026-11-06": "Libur perusahaan"}
	today := time.Date(2026, 11, 10, 9, 0, 0, 0, jkt)
	res := Calculate(emp, rules(), time.Date(2026, 11, 1, 0, 0, 0, 0, jkt), today, recs, holidays, jkt)

	s := res.Summary
	// hari kerja: 25 - 1 libur = 24. Sebelum tgl 10: 2,3,4,5,7,9 (6 libur, 8 Minggu) -> alpa = 7 dan 9.
	if s.HariKerja != 24 || s.Hadir != 2 || s.Sakit != 1 || s.Izin != 1 || s.Alpa != 2 {
		t.Fatalf("rekap salah: %+v", s)
	}
	if s.TelatMenit != 20 || s.LemburMenit != 120 || s.Telat != 1 {
		t.Fatalf("menit salah: %+v", s)
	}
	p := res.Pay
	want := Pay{GajiPokok: 2_600_000, TunjanganTetap: 200_000, UangMakan: 40_000, Lembur: 30_000,
		PotonganAlpa: 200_000, PotonganTelat: 20_000}
	want.Pendapatan = 2_870_000
	want.Potongan = 220_000
	want.GajiBersih = 2_650_000
	if p != want {
		t.Fatalf("gaji salah:\n dapat %+v\n mau   %+v", p, want)
	}
	if len(res.Days) != 30 || res.Days[9].Status != "belum" || res.Days[0].Status != "libur" {
		t.Fatalf("harian salah: %v %v", res.Days[0], res.Days[9])
	}
}

func TestCalculateHarianDanMasukTengahBulan(t *testing.T) {
	start := time.Date(2026, 11, 16, 0, 0, 0, 0, jkt)
	recs := []Record{hadir("2026-11-16", "08:00", "16:00"), hadir("2026-11-17", "08:00", "16:00")}
	today := time.Date(2026, 12, 5, 0, 0, 0, 0, jkt)

	harian := Employee{TipeGaji: "harian", GajiPokok: 100_000, TanggalMasuk: &start}
	res := Calculate(harian, rules(), start, today, recs, nil, jkt)
	// sejak 16 Nov: 16-21, 23-28, 30 = 13 hari kerja, hadir 2, alpa 11
	if res.Summary.HariKerja != 13 || res.Summary.Alpa != 11 || res.Pay.GajiPokok != 200_000 || res.Pay.PotonganAlpa != 0 {
		t.Fatalf("harian: %+v %+v", res.Summary, res.Pay)
	}

	bulanan := Employee{TipeGaji: "bulanan", GajiPokok: 2_500_000, TanggalMasuk: &start}
	res = Calculate(bulanan, rules(), start, today, recs, nil, jkt)
	// prorata 13/25 hari kerja = 1.300.000, potongan alpa 11 x 2.500.000/26 = 1.057.692
	if res.Pay.GajiPokok != 1_300_000 || res.Pay.PotonganAlpa != 1_057_692 || res.Pay.GajiBersih != 242_308 {
		t.Fatalf("bulanan prorata: %+v", res.Pay)
	}
}

func TestGajiTidakMinus(t *testing.T) {
	emp := Employee{TipeGaji: "bulanan", GajiPokok: 260_000}
	today := time.Date(2026, 12, 1, 0, 0, 0, 0, jkt)
	res := Calculate(emp, rules(), time.Date(2026, 11, 1, 0, 0, 0, 0, jkt), today, nil, nil, jkt)
	if res.Summary.Alpa != 25 || res.Pay.GajiBersih != 0 || res.Pay.Potongan != res.Pay.Pendapatan {
		t.Fatalf("%+v %+v", res.Summary, res.Pay)
	}
}
