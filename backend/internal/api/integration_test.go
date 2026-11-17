package api

// Tes integrasi: butuh database KOSONG berisi db/schema.sql (isi tabel DIHAPUS).
//   TEST_DB_DSN="root:@tcp(localhost:3306)/absensi_test?parseTime=true&loc=Asia%2FJakarta" go test ./internal/api/

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"absensi/internal/store"

	_ "github.com/go-sql-driver/mysql"
)

type env struct {
	t   *testing.T
	h   http.Handler
	now *time.Time
}

func (e env) call(method, path, token string, body any) (int, map[string]any, []any) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	var obj map[string]any
	var arr []any
	raw := rec.Body.Bytes()
	if len(raw) > 0 && raw[0] == '[' {
		json.Unmarshal(raw, &arr)
	} else {
		json.Unmarshal(raw, &obj)
	}
	return rec.Code, obj, arr
}

func (e env) must(code int, method, path, token string, body any) map[string]any {
	e.t.Helper()
	got, obj, _ := e.call(method, path, token, body)
	if got != code {
		e.t.Fatalf("%s %s: status %d, mau %d, body %v", method, path, got, code, obj)
	}
	return obj
}

func setup(t *testing.T) env {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN tidak diisi")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	for _, tb := range []string{"payroll_periods", "leave_requests", "attendance", "holidays", "employees", "users"} {
		if _, err := db.Exec("DELETE FROM " + tb); err != nil {
			t.Fatal(err)
		}
	}
	db.Exec(`UPDATE settings SET nilai='10' WHERE kunci='toleransi_telat_menit'`)
	db.Exec(`UPDATE settings SET nilai='1000' WHERE kunci='potongan_telat_per_menit'`)
	db.Exec(`UPDATE settings SET nilai='0' WHERE kunci='hari_libur_mingguan'`)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Date(2026, 11, 2, 7, 0, 0, 0, loc) // Senin
	s := &Server{Store: &store.Store{DB: db, Loc: loc}, Secret: []byte(strings.Repeat("x", 32)), Loc: loc}
	e := env{t: t, h: s.Routes(), now: &now}
	s.Now = func() time.Time { return *e.now }
	hash, _ := HashPassword("rahasia123")
	if _, err := s.Store.InsertUser(t.Context(), nil, "admin@contoh.com", hash, "admin", "Admin"); err != nil {
		t.Fatal(err)
	}
	return e
}

func login(e env, email, pw string) string {
	obj := e.must(200, "POST", "/api/login", "", map[string]string{"email": email, "password": pw})
	return obj["token"].(string)
}

func TestAlurLengkap(t *testing.T) {
	e := setup(t)
	loc := e.now.Location()
	at := func(day, hhmm string) {
		tm, _ := time.ParseInLocation("2006-01-02 15:04", day+" "+hhmm, loc)
		*e.now = tm
	}

	// login salah
	if c, _, _ := e.call("POST", "/api/login", "", map[string]string{"email": "admin@contoh.com", "password": "salah"}); c != 401 {
		t.Fatalf("login salah harus 401, dapat %d", c)
	}
	admin := login(e, "admin@contoh.com", "rahasia123")

	// admin menambah karyawan
	emp := map[string]any{"nik": "K001", "nama": "Budi", "email": "budi@contoh.com", "password": "budi12345",
		"jabatan": "Staf", "tipe_gaji": "bulanan", "gaji_pokok": 2600000, "tunjangan_tetap": 200000,
		"uang_makan": 20000, "tarif_lembur_per_jam": 15000, "jam_masuk": "08:00", "jam_pulang": "16:00"}
	id := int64(e.must(201, "POST", "/api/employees", admin, emp)["id"].(float64))
	if c, obj, _ := e.call("POST", "/api/employees", admin, emp); c != 409 {
		t.Fatalf("email ganda harus 409: %d %v", c, obj)
	}
	budi := login(e, "budi@contoh.com", "budi12345")

	// karyawan tidak boleh membuka menu admin
	if c, _, _ := e.call("GET", "/api/employees", budi, nil); c != 403 {
		t.Fatalf("karyawan ke menu admin harus 403, dapat %d", c)
	}

	// Senin 2 Nov: absen terlalu pagi (dibuka 06.00) lalu tepat waktu, pulang lembur
	at("2026-11-02", "05:30")
	e.must(400, "POST", "/api/my/check-in", budi, nil)
	at("2026-11-02", "07:58")
	today := e.must(201, "POST", "/api/my/check-in", budi, nil)
	if today["bisa_pulang"] != true || today["bisa_masuk"] != false {
		t.Fatalf("setelah absen masuk: %v", today)
	}
	e.must(409, "POST", "/api/my/check-in", budi, nil)
	at("2026-11-02", "18:10")
	e.must(201, "POST", "/api/my/check-out", budi, nil)
	e.must(409, "POST", "/api/my/check-out", budi, nil)

	// Selasa 3 Nov: telat 25 menit
	at("2026-11-03", "08:25")
	e.must(201, "POST", "/api/my/check-in", budi, nil)
	at("2026-11-03", "16:00")
	e.must(201, "POST", "/api/my/check-out", budi, nil)

	// pengajuan sakit 4-5 Nov, disetujui admin
	lv := e.must(201, "POST", "/api/my/leaves", budi, map[string]string{"jenis": "sakit", "tanggal_mulai": "2026-11-04", "tanggal_selesai": "2026-11-05", "alasan": "Demam"})
	res := e.must(200, "POST", fmt.Sprintf("/api/leaves/%d/setujui", int64(lv["id"].(float64))), admin, nil)
	if res["hari_dicatat"].(float64) != 2 {
		t.Fatalf("hari sakit dicatat: %v", res)
	}
	e.must(409, "POST", fmt.Sprintf("/api/leaves/%d/tolak", int64(lv["id"].(float64))), admin, nil)

	at("2026-11-10", "09:00")
	// tanggal yang akan datang ditolak
	e.must(400, "PUT", "/api/attendance", admin, map[string]any{"employee_id": id, "tanggal": "2026-11-11", "status": "izin"})
	// admin input manual: 6 Nov hadir 08.00-16.00, hari libur 7 Nov
	e.must(200, "PUT", "/api/attendance", admin, map[string]any{"employee_id": id, "tanggal": "2026-11-06", "status": "hadir", "jam_masuk": "08:00", "jam_pulang": "16:00"})
	e.must(201, "POST", "/api/holidays", admin, map[string]string{"tanggal": "2026-11-07", "keterangan": "Libur perusahaan"})

	// dashboard tgl 10 Nov jam 09.00: Budi belum absen
	dash := e.must(200, "GET", "/api/dashboard", admin, nil)
	if dash["jumlah"].(map[string]any)["belum"].(float64) != 1 {
		t.Fatalf("dashboard: %v", dash["jumlah"])
	}

	// rekap gaji November (hari ini 10 Nov): alpa = 9 Nov (8 Minggu, 7 libur)
	rep := e.must(200, "GET", "/api/payroll?bulan=2026-11", admin, nil)
	k := rep["karyawan"].([]any)[0].(map[string]any)
	rk, rc := k["rekap"].(map[string]any), k["rincian"].(map[string]any)
	if rk["hadir"].(float64) != 3 || rk["sakit"].(float64) != 2 || rk["alpa"].(float64) != 1 ||
		rk["telat_menit"].(float64) != 25 || rk["lembur_menit"].(float64) != 120 {
		t.Fatalf("rekap: %v", rk)
	}
	// pokok 2.600.000 + tunj 200.000 + makan 60.000 + lembur 30.000 - alpa 100.000 - telat 25.000
	if rc["gaji_bersih"].(float64) != 2_765_000 {
		t.Fatalf("rincian: %v", rc)
	}

	// slip milik karyawan sama dengan rekap admin
	slip := e.must(200, "GET", "/api/my/payroll?bulan=2026-11", budi, nil)["slip"].(map[string]any)
	if slip["rincian"].(map[string]any)["gaji_bersih"].(float64) != 2_765_000 {
		t.Fatalf("slip karyawan: %v", slip["rincian"])
	}

	// tutup periode: belum boleh sebelum bulan berakhir
	e.must(400, "POST", "/api/payroll/2026-11/close", admin, nil)
	at("2026-12-01", "09:00")
	closed := e.must(201, "POST", "/api/payroll/2026-11/close", admin, nil)
	if closed["ditutup"] != true {
		t.Fatalf("periode harus ditutup: %v", closed["ditutup"])
	}
	// absensi bulan yang ditutup terkunci
	e.must(409, "PUT", "/api/attendance", admin, map[string]any{"employee_id": id, "tanggal": "2026-11-09", "status": "izin"})
	// snapshot tidak berubah walau gaji karyawan diubah
	emp["gaji_pokok"] = 5000000
	e.must(200, "PUT", fmt.Sprintf("/api/employees/%d", id), admin, emp)
	rep = e.must(200, "GET", "/api/payroll?bulan=2026-11", admin, nil)
	if rep["total"].(map[string]any)["gaji_bersih"].(float64) <= 0 ||
		rep["karyawan"].([]any)[0].(map[string]any)["rincian"].(map[string]any)["gaji_pokok"].(float64) != 2_600_000 {
		t.Fatalf("snapshot berubah: %v", rep["total"])
	}
	// buka kembali -> hitung ulang dengan gaji baru, absensi bisa diubah lagi
	e.must(200, "DELETE", "/api/payroll/2026-11/close", admin, nil)
	e.must(200, "PUT", "/api/attendance", admin, map[string]any{"employee_id": id, "tanggal": "2026-11-09", "status": "izin", "catatan": "susulan"})

	// karyawan dinonaktifkan tidak bisa login lagi
	emp["aktif"] = false
	e.must(200, "PUT", fmt.Sprintf("/api/employees/%d", id), admin, emp)
	if c, _, _ := e.call("GET", "/api/my/today", budi, nil); c != 401 {
		t.Fatalf("token karyawan nonaktif harus ditolak, dapat %d", c)
	}

	// ekspor CSV
	req := httptest.NewRequest("GET", "/api/payroll/export?bulan=2026-11", nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Budi") {
		t.Fatalf("CSV: %d %s", rec.Code, rec.Body.String())
	}
}

func TestShiftMalam(t *testing.T) {
	e := setup(t)
	loc := e.now.Location()
	admin := login(e, "admin@contoh.com", "rahasia123")
	e.must(201, "POST", "/api/employees", admin, map[string]any{"nik": "K002", "nama": "Sari", "email": "sari@contoh.com",
		"password": "sari12345", "tipe_gaji": "harian", "gaji_pokok": 120000, "tarif_lembur_per_jam": 20000,
		"jam_masuk": "22:00", "jam_pulang": "06:00"})
	sari := login(e, "sari@contoh.com", "sari12345")
	*e.now = time.Date(2026, 11, 2, 21, 55, 0, 0, loc)
	e.must(201, "POST", "/api/my/check-in", sari, nil)
	*e.now = time.Date(2026, 11, 3, 7, 5, 0, 0, loc) // keesokan pagi
	today := e.must(200, "GET", "/api/my/today", sari, nil)
	if today["bisa_pulang"] != true || today["bisa_masuk"] != false {
		t.Fatalf("shift malam harus bisa absen pulang: %v", today)
	}
	e.must(201, "POST", "/api/my/check-out", sari, nil)
	slip := e.must(200, "GET", "/api/my/payroll?bulan=2026-11", sari, nil)["slip"].(map[string]any)
	if slip["rekap"].(map[string]any)["lembur_menit"].(float64) != 60 || slip["rincian"].(map[string]any)["gaji_pokok"].(float64) != 120000 {
		t.Fatalf("slip shift malam: %v %v", slip["rekap"], slip["rincian"])
	}
}
