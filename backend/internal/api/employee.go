package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"absensi/internal/payroll"
	"absensi/internal/store"

	"github.com/go-sql-driver/mysql"
)

var (
	reClock = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	reEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type employeeReq struct {
	store.Employee
	Password string `json:"password"`
}

func (s *Server) validateEmployee(e *store.Employee) error {
	e.Nama = strings.TrimSpace(e.Nama)
	e.NIK = strings.TrimSpace(e.NIK)
	e.Email = strings.ToLower(strings.TrimSpace(e.Email))
	e.Jabatan = strings.TrimSpace(e.Jabatan)
	e.Telepon = strings.TrimSpace(e.Telepon)
	switch {
	case e.Nama == "" || len(e.Nama) > 100:
		return bad("nama wajib diisi (maks. 100 karakter)")
	case e.NIK == "" || len(e.NIK) > 30:
		return bad("NIK / nomor karyawan wajib diisi (maks. 30 karakter)")
	case !reEmail.MatchString(e.Email):
		return bad("email tidak valid")
	case e.TipeGaji != "bulanan" && e.TipeGaji != "harian" && e.TipeGaji != "per_jam":
		return bad("tipe gaji harus bulanan, harian, atau per_jam")
	case e.GajiPokok < 0 || e.TarifPerJam < 0 || e.TunjanganTetap < 0 || e.UangMakan < 0 || e.TarifLemburPerJam < 0:
		return bad("nominal gaji tidak boleh negatif")
	}
	// Validasi jadwal hanya jika diisi (opsional untuk per_jam)
	if e.JamMasuk != "" || e.JamPulang != "" {
		if !reClock.MatchString(e.JamMasuk) || !reClock.MatchString(e.JamPulang) {
			return bad("jam masuk/pulang harus berformat HH:MM")
		}
		if e.JamMasuk == e.JamPulang {
			return bad("jam masuk dan jam pulang tidak boleh sama")
		}
	}
	// Untuk sistem bulanan dan harian, jadwal wajib
	if (e.TipeGaji == "bulanan" || e.TipeGaji == "harian") && (e.JamMasuk == "" || e.JamPulang == "") {
		return bad("jadwal jam masuk/pulang wajib diisi untuk sistem %s", e.TipeGaji)
	}
	if e.TipeGaji == "per_jam" && e.TarifPerJam == 0 {
		return bad("tarif per jam wajib diisi untuk sistem per jam")
	}
	if e.TanggalMasuk != "" {
		if _, err := time.ParseInLocation(payroll.DayFmt, e.TanggalMasuk, s.Loc); err != nil {
			return bad("tanggal masuk harus berformat YYYY-MM-DD")
		}
	}
	return nil
}

func duplicate(err error) error {
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == 1062 {
		switch {
		case strings.Contains(me.Message, "email"):
			return conflict("email sudah dipakai akun lain")
		case strings.Contains(me.Message, "nik"):
			return conflict("NIK sudah dipakai karyawan lain")
		}
		return conflict("data sudah ada")
	}
	return err
}

func (s *Server) listEmployees(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.Store.ListEmployees(r.Context(), r.URL.Query().Get("aktif") == "1")
}

func (s *Server) createEmployee(w http.ResponseWriter, r *http.Request) (any, error) {
	var req employeeReq
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	e := req.Employee
	e.Aktif = true
	if err := s.validateEmployee(&e); err != nil {
		return nil, err
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	tx, err := s.Store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	e.UserID, err = s.Store.InsertUser(ctx, tx, e.Email, hash, "karyawan", e.Nama)
	if err != nil {
		return nil, duplicate(err)
	}
	id, err := s.Store.InsertEmployee(ctx, tx, e)
	if err != nil {
		return nil, duplicate(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]int64{"id": id}, nil
}

func (s *Server) updateEmployee(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	var req employeeReq
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	ctx := r.Context()
	cur, err := s.Store.EmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	e := req.Employee
	e.ID, e.UserID = cur.ID, cur.UserID
	if err := s.validateEmployee(&e); err != nil {
		return nil, err
	}
	tx, err := s.Store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.Store.UpdateEmployee(ctx, tx, e); err != nil {
		return nil, duplicate(err)
	}
	if req.Password != "" { // admin mereset kata sandi karyawan
		hash, err := HashPassword(req.Password)
		if err != nil {
			return nil, err
		}
		if err := s.Store.UpdateUserPasswordTx(ctx, tx, e.UserID, hash); err != nil {
			return nil, err
		}
	}
	return ok(), tx.Commit()
}
