package api

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"absensi/internal/payroll"
	"absensi/internal/store"
)

type rule struct{ min, max float64 }

// Parameter yang boleh diubah lewat API beserta batas wajarnya.
var numericSettings = map[string]rule{
	"toleransi_telat_menit":    {0, 240},
	"potongan_telat_per_menit": {0, 1_000_000},
	"hari_kerja_sebulan":       {1, 31},
	"lembur_minimal_menit":     {1, 240},
	"awal_absen_menit":         {0, 720},
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	m, err := s.Store.Settings(r.Context())
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k := range numericSettings {
		out[k] = store.Atof(m, k, 0)
	}
	libur := []int{}
	rl, err := s.rules(r.Context())
	if err != nil {
		return nil, err
	}
	for d := 0; d <= 6; d++ {
		if rl.LiburMingguan[d] {
			libur = append(libur, d)
		}
	}
	out["hari_libur_mingguan"] = libur
	return out, nil
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	var req map[string]any
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	save := map[string]string{}
	for k, v := range req {
		if k == "hari_libur_mingguan" {
			list, ok := v.([]any)
			if !ok || len(list) > 6 {
				return nil, bad("hari_libur_mingguan harus daftar angka 0-6 (minimal 1 hari kerja)")
			}
			parts := []string{}
			seen := map[int]bool{}
			for _, x := range list {
				f, ok := x.(float64)
				if !ok || f != math.Trunc(f) || f < 0 || f > 6 || seen[int(f)] {
					return nil, bad("hari_libur_mingguan harus daftar angka 0-6 tanpa duplikat")
				}
				seen[int(f)] = true
				parts = append(parts, strconv.Itoa(int(f)))
			}
			save[k] = strings.Join(parts, ",")
			continue
		}
		rl, ok := numericSettings[k]
		if !ok {
			return nil, bad("parameter %q tidak dikenal", k)
		}
		f, ok := v.(float64)
		if !ok || math.IsNaN(f) || f < rl.min || f > rl.max || f != math.Trunc(f) {
			return nil, bad("%s harus bilangan bulat antara %g dan %g", k, rl.min, rl.max)
		}
		save[k] = strconv.FormatFloat(f, 'f', -1, 64)
	}
	if len(save) == 0 {
		return nil, bad("tidak ada parameter yang diubah")
	}
	if err := s.Store.SaveSettings(r.Context(), save); err != nil {
		return nil, err
	}
	return s.getSettings(w, r)
}

// ------------------------------------------------------------------ hari libur

func (s *Server) listHolidays(w http.ResponseWriter, r *http.Request) (any, error) {
	year, err := strconv.Atoi(r.URL.Query().Get("tahun"))
	if err != nil || year < 2000 || year > 2100 {
		year = s.now().Year()
	}
	return s.Store.ListHolidays(r.Context(), strconv.Itoa(year)+"-01-01", strconv.Itoa(year)+"-12-31")
}

func (s *Server) saveHoliday(w http.ResponseWriter, r *http.Request) (any, error) {
	var h store.Holiday
	if err := decode(r, &h); err != nil {
		return nil, err
	}
	if _, err := time.ParseInLocation(payroll.DayFmt, h.Tanggal, s.Loc); err != nil {
		return nil, bad("tanggal harus berformat YYYY-MM-DD")
	}
	h.Keterangan = strings.TrimSpace(h.Keterangan)
	if h.Keterangan == "" || len(h.Keterangan) > 120 {
		return nil, bad("keterangan wajib diisi (maks. 120 karakter)")
	}
	if closed, err := s.Store.IsClosed(r.Context(), nil, h.Tanggal); err != nil {
		return nil, err
	} else if closed {
		return nil, conflict("periode gaji %s sudah ditutup", h.Tanggal[:7])
	}
	return ok(), s.Store.UpsertHoliday(r.Context(), h)
}

func (s *Server) deleteHoliday(w http.ResponseWriter, r *http.Request) (any, error) {
	day := r.PathValue("tanggal")
	if _, err := time.ParseInLocation(payroll.DayFmt, day, s.Loc); err != nil {
		return nil, bad("tanggal tidak valid")
	}
	if closed, err := s.Store.IsClosed(r.Context(), nil, day); err != nil {
		return nil, err
	} else if closed {
		return nil, conflict("periode gaji %s sudah ditutup", day[:7])
	}
	return ok(), s.Store.DeleteHoliday(r.Context(), day)
}
