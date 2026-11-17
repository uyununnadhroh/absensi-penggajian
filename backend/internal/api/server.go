// Package api berisi route, middleware, dan handler HTTP.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"absensi/internal/payroll"
	"absensi/internal/store"

	"github.com/golang-jwt/jwt/v5"
)

type Server struct {
	Store   *store.Store
	Secret  []byte
	Loc     *time.Location
	Origins []string
	Now     func() time.Time // opsional, untuk pengujian
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now().In(s.Loc)
	}
	return time.Now().In(s.Loc)
}

func (s *Server) today() string { return s.now().Format(payroll.DayFmt) }

// ------------------------------------------------------------------ errors & response

type httpError struct {
	code int
	msg  string
}

func (e httpError) Error() string { return e.msg }

func bad(format string, a ...any) error {
	return httpError{http.StatusBadRequest, fmt.Sprintf(format, a...)}
}
func notFound(format string, a ...any) error {
	return httpError{http.StatusNotFound, fmt.Sprintf(format, a...)}
}
func conflict(format string, a ...any) error {
	return httpError{http.StatusConflict, fmt.Sprintf(format, a...)}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	var he httpError
	code, msg := http.StatusInternalServerError, "terjadi kesalahan pada server"
	switch {
	case errors.As(err, &he):
		code, msg = he.code, he.msg
	case errors.Is(err, store.ErrNotFound):
		code, msg = http.StatusNotFound, "data tidak ditemukan"
	}
	if code >= 500 {
		log.Printf("[ERROR] %s %s: %v", r.Method, r.URL.Path, err)
	}
	writeJSON(w, code, map[string]string{"error": msg})
}

// handlerFunc versi yang mengembalikan (data, error).
type hf func(w http.ResponseWriter, r *http.Request) (any, error)

func (s *Server) wrap(fn hf) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fn(w, r)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		if data == nil { // handler sudah menulis respons sendiri (mis. CSV)
			return
		}
		code := http.StatusOK
		if r.Method == http.MethodPost {
			code = http.StatusCreated
		}
		writeJSON(w, code, data)
	}
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return bad("body JSON tidak valid")
	}
	return nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, bad("id tidak valid")
	}
	return id, nil
}

func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func ok() map[string]string { return map[string]string{"status": "ok"} }

// ------------------------------------------------------------------ auth

type ctxKey struct{}

type Claims struct {
	Role       string `json:"role"`
	EmployeeID int64  `json:"emp,omitempty"`
	jwt.RegisteredClaims
}

type Actor struct {
	UserID     int64
	Role       string
	EmployeeID int64
}

func actor(r *http.Request) Actor {
	a, _ := r.Context().Value(ctxKey{}).(Actor)
	return a
}

func (s *Server) issueToken(u store.User) (string, error) {
	c := Claims{
		Role:       u.Role,
		EmployeeID: u.EmployeeID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(u.ID, 10),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.Secret)
}

// auth memeriksa token; roles kosong = semua role boleh.
func (s *Server) auth(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var c Claims
		tok, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) { return s.Secret, nil },
			jwt.WithValidMethods([]string{"HS256"}))
		if raw == "" || err != nil || !tok.Valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sesi berakhir, silakan masuk lagi"})
			return
		}
		uid, _ := strconv.ParseInt(c.Subject, 10, 64)
		// akun yang dinonaktifkan langsung tidak bisa dipakai walau token masih berlaku
		u, err := s.Store.UserByID(r.Context(), uid)
		if err != nil || !u.Aktif {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "akun tidak aktif, silakan hubungi admin"})
			return
		}
		if len(roles) > 0 {
			allowed := false
			for _, role := range roles {
				allowed = allowed || role == u.Role
			}
			if !allowed {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "anda tidak punya akses ke menu ini"})
				return
			}
		}
		a := Actor{UserID: u.ID, Role: u.Role, EmployeeID: u.EmployeeID}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, a)))
	}
}

// ------------------------------------------------------------------ routes

func (s *Server) Routes() http.Handler {
	m := http.NewServeMux()
	admin := func(fn hf) http.HandlerFunc { return s.auth(s.wrap(fn), "admin") }
	emp := func(fn hf) http.HandlerFunc { return s.auth(s.wrap(fn), "karyawan") }
	all := func(fn hf) http.HandlerFunc { return s.auth(s.wrap(fn)) }

	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, ok()) })
	m.HandleFunc("POST /api/login", s.wrap(s.login))
	m.HandleFunc("GET /api/me", all(s.me))
	m.HandleFunc("PUT /api/me/password", all(s.changePassword))

	// Karyawan: absen sendiri, riwayat, pengajuan izin, slip gaji
	m.HandleFunc("GET /api/my/today", emp(s.myToday))
	m.HandleFunc("POST /api/my/check-in", emp(s.checkIn))
	m.HandleFunc("POST /api/my/check-out", emp(s.checkOut))
	m.HandleFunc("GET /api/my/payroll", emp(s.myPayroll))
	m.HandleFunc("GET /api/my/leaves", emp(s.myLeaves))
	m.HandleFunc("POST /api/my/leaves", emp(s.createLeave))
	m.HandleFunc("DELETE /api/my/leaves/{id}", emp(s.cancelLeave))

	// Admin
	m.HandleFunc("GET /api/dashboard", admin(s.dashboard))
	m.HandleFunc("GET /api/employees", admin(s.listEmployees))
	m.HandleFunc("POST /api/employees", admin(s.createEmployee))
	m.HandleFunc("PUT /api/employees/{id}", admin(s.updateEmployee))
	m.HandleFunc("GET /api/attendance", admin(s.attendanceByDate))
	m.HandleFunc("PUT /api/attendance", admin(s.saveAttendance))
	m.HandleFunc("GET /api/leaves", admin(s.listLeaves))
	m.HandleFunc("POST /api/leaves/{id}/{action}", admin(s.decideLeave))
	m.HandleFunc("GET /api/holidays", admin(s.listHolidays))
	m.HandleFunc("POST /api/holidays", admin(s.saveHoliday))
	m.HandleFunc("DELETE /api/holidays/{tanggal}", admin(s.deleteHoliday))
	m.HandleFunc("GET /api/settings", admin(s.getSettings))
	m.HandleFunc("PUT /api/settings", admin(s.putSettings))
	m.HandleFunc("GET /api/payroll", admin(s.payrollMonth))
	m.HandleFunc("GET /api/payroll/export", admin(s.exportPayroll))
	m.HandleFunc("POST /api/payroll/{month}/close", admin(s.closePayroll))
	m.HandleFunc("DELETE /api/payroll/{month}/close", admin(s.reopenPayroll))

	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "endpoint tidak ditemukan"})
	})
	return s.cors(m)
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.Origins {
		allowed[strings.TrimRight(strings.TrimSpace(o), "/")] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && (allowed[o] || allowed["*"]) {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
