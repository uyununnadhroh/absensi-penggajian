package api

import (
	"errors"
	"net/http"
	"strings"

	"absensi/internal/store"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword dipakai juga oleh cmd/createadmin.
func HashPassword(pw string) (string, error) {
	if len(pw) < 8 {
		return "", bad("kata sandi minimal 8 karakter")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	failed := httpError{http.StatusUnauthorized, "email atau kata sandi salah"}
	u, err := s.Store.UserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if errors.Is(err, store.ErrNotFound) {
		return nil, failed
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		return nil, failed
	}
	if !u.Aktif {
		return nil, httpError{http.StatusForbidden, "akun tidak aktif, silakan hubungi admin"}
	}
	tok, err := s.issueToken(u)
	if err != nil {
		return nil, err
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "user": u})
	return nil, nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) (any, error) {
	u, err := s.Store.UserByID(r.Context(), actor(r).UserID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"user": u}
	if u.EmployeeID != 0 {
		e, err := s.Store.EmployeeByID(r.Context(), u.EmployeeID)
		if err != nil {
			return nil, err
		}
		out["employee"] = e
	}
	return out, nil
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Lama string `json:"password_lama"`
		Baru string `json:"password_baru"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	u, err := s.Store.UserByID(r.Context(), actor(r).UserID)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Lama)) != nil {
		return nil, bad("kata sandi lama salah")
	}
	hash, err := HashPassword(req.Baru)
	if err != nil {
		return nil, err
	}
	return ok(), s.Store.UpdatePassword(r.Context(), u.ID, hash)
}
