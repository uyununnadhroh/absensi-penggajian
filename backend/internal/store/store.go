// Package store berisi semua query SQL.
package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("data tidak ditemukan")

type Store struct {
	DB  *sql.DB
	Loc *time.Location
}

// q bisa *sql.DB atau *sql.Tx
type q interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type postgresQ struct {
	base q
}

func (p postgresQ) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.base.ExecContext(ctx, postgresPlaceholders(query), args...)
}

func (p postgresQ) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return p.base.QueryContext(ctx, postgresPlaceholders(query), args...)
}

func (p postgresQ) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return p.base.QueryRowContext(ctx, postgresPlaceholders(query), args...)
}

// postgresPlaceholders mengubah placeholder ? menjadi $1, $2, dan seterusnya.
func postgresPlaceholders(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 1
	for _, r := range query {
		if r == '?' {
			b.WriteString("$")
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Store) Begin(ctx context.Context) (*sql.Tx, error) {
	return s.DB.BeginTx(ctx, nil)
}

func (s *Store) conn(tx *sql.Tx) q {
	if tx != nil {
		return postgresQ{base: tx}
	}
	return postgresQ{base: s.DB}
}

// ------------------------------------------------------------------ users

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	Nama         string `json:"nama"`
	Aktif        bool   `json:"aktif"`
	EmployeeID   int64  `json:"employee_id,omitempty"`
}

const userCols = `u.id, u.email, u.password_hash, u.role, u.nama, u.aktif, COALESCE(e.id,0)`

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Nama, &u.Aktif, &u.EmployeeID)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(s.conn(nil).QueryRowContext(ctx, `SELECT `+userCols+` FROM users u LEFT JOIN employees e ON e.user_id = u.id WHERE u.email = ?`, email))
}

func (s *Store) UserByID(ctx context.Context, id int64) (User, error) {
	return scanUser(s.conn(nil).QueryRowContext(ctx, `SELECT `+userCols+` FROM users u LEFT JOIN employees e ON e.user_id = u.id WHERE u.id = ?`, id))
}

func (s *Store) InsertUser(ctx context.Context, tx *sql.Tx, email, hash, role, nama string) (int64, error) {
	var id int64
	err := s.conn(tx).QueryRowContext(ctx, `INSERT INTO users (email, password_hash, role, nama) VALUES (?,?,?,?) RETURNING id`, email, hash, role, nama).Scan(&id)
	return id, err
}

func (s *Store) UpdatePassword(ctx context.Context, userID int64, hash string) error {
	_, err := s.conn(nil).ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID)
	return err
}

// ------------------------------------------------------------------ employees

type Employee struct {
	ID                int64   `json:"id"`
	UserID            int64   `json:"user_id"`
	Email             string  `json:"email"`
	NIK               string  `json:"nik"`
	Nama              string  `json:"nama"`
	Jabatan           string  `json:"jabatan"`
	Telepon           string  `json:"telepon"`
	TanggalMasuk      string  `json:"tanggal_masuk"` // YYYY-MM-DD atau ""
	TipeGaji          string  `json:"tipe_gaji"`
	GajiPokok         float64 `json:"gaji_pokok"`
	TarifPerJam       float64 `json:"tarif_per_jam"` // untuk tipe_gaji = per_jam
	TunjanganTetap    float64 `json:"tunjangan_tetap"`
	UangMakan         float64 `json:"uang_makan"`
	TarifLemburPerJam float64 `json:"tarif_lembur_per_jam"`
	JamMasuk          string  `json:"jam_masuk"` // HH:MM
	JamPulang         string  `json:"jam_pulang"`
	Aktif             bool    `json:"aktif"`
}

const empCols = `e.id, e.user_id, u.email, e.nik, e.nama, e.jabatan, e.telepon,
	COALESCE(TO_CHAR(e.tanggal_masuk, 'YYYY-MM-DD'), ''), e.tipe_gaji, e.gaji_pokok, e.tarif_per_jam, e.tunjangan_tetap,
	e.uang_makan, e.tarif_lembur_per_jam, COALESCE(TO_CHAR(e.jam_masuk, 'HH24:MI'), ''), COALESCE(TO_CHAR(e.jam_pulang, 'HH24:MI'), ''),
	e.aktif AND u.aktif`

func scanEmployee(row interface{ Scan(...any) error }) (Employee, error) {
	var e Employee
	err := row.Scan(&e.ID, &e.UserID, &e.Email, &e.NIK, &e.Nama, &e.Jabatan, &e.Telepon, &e.TanggalMasuk,
		&e.TipeGaji, &e.GajiPokok, &e.TarifPerJam, &e.TunjanganTetap, &e.UangMakan, &e.TarifLemburPerJam, &e.JamMasuk, &e.JamPulang, &e.Aktif)
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

func (s *Store) ListEmployees(ctx context.Context, onlyActive bool) ([]Employee, error) {
	where := ""
	if onlyActive {
		where = "WHERE e.aktif = TRUE AND u.aktif = TRUE"
	}
	rows, err := s.conn(nil).QueryContext(ctx, `SELECT `+empCols+` FROM employees e JOIN users u ON u.id = e.user_id `+where+` ORDER BY e.nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Employee{}
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) EmployeeByID(ctx context.Context, id int64) (Employee, error) {
	return scanEmployee(s.conn(nil).QueryRowContext(ctx, `SELECT `+empCols+` FROM employees e JOIN users u ON u.id = e.user_id WHERE e.id = ?`, id))
}

func nullDate(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Store) InsertEmployee(ctx context.Context, tx *sql.Tx, e Employee) (int64, error) {
	var id int64
	err := s.conn(tx).QueryRowContext(ctx, `INSERT INTO employees (user_id, nik, nama, jabatan, telepon, tanggal_masuk, tipe_gaji,
		gaji_pokok, tarif_per_jam, tunjangan_tetap, uang_makan, tarif_lembur_per_jam, jam_masuk, jam_pulang, aktif)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
		e.UserID, e.NIK, e.Nama, e.Jabatan, e.Telepon, nullDate(e.TanggalMasuk), e.TipeGaji,
		e.GajiPokok, e.TarifPerJam, e.TunjanganTetap, e.UangMakan, e.TarifLemburPerJam, nullStr(e.JamMasuk), nullStr(e.JamPulang), e.Aktif).Scan(&id)
	return id, err
}

func (s *Store) UpdateEmployee(ctx context.Context, tx *sql.Tx, e Employee) error {
	_, err := s.conn(tx).ExecContext(ctx, `UPDATE employees SET nik=?, nama=?, jabatan=?, telepon=?, tanggal_masuk=?, tipe_gaji=?,
		gaji_pokok=?, tarif_per_jam=?, tunjangan_tetap=?, uang_makan=?, tarif_lembur_per_jam=?, jam_masuk=?, jam_pulang=?, aktif=? WHERE id=?`,
		e.NIK, e.Nama, e.Jabatan, e.Telepon, nullDate(e.TanggalMasuk), e.TipeGaji,
		e.GajiPokok, e.TarifPerJam, e.TunjanganTetap, e.UangMakan, e.TarifLemburPerJam, nullStr(e.JamMasuk), nullStr(e.JamPulang), e.Aktif, e.ID)
	if err != nil {
		return err
	}
	_, err = s.conn(tx).ExecContext(ctx, `UPDATE users SET email=?, nama=?, aktif=? WHERE id=?`, e.Email, e.Nama, e.Aktif, e.UserID)
	return err
}

func (s *Store) UpdateUserPasswordTx(ctx context.Context, tx *sql.Tx, userID int64, hash string) error {
	_, err := s.conn(tx).ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID)
	return err
}

// ------------------------------------------------------------------ attendance

type Attendance struct {
	ID           int64      `json:"id"`
	EmployeeID   int64      `json:"employee_id"`
	Tanggal      string     `json:"tanggal"`
	Status       string     `json:"status"`
	JamMasuk     *time.Time `json:"jam_masuk"`
	JamPulang    *time.Time `json:"jam_pulang"`
	JadwalMasuk  string     `json:"jadwal_masuk"`
	JadwalPulang string     `json:"jadwal_pulang"`
	Catatan      string     `json:"catatan"`
	Sumber       string     `json:"sumber"`
}

const attCols = `id, employee_id, TO_CHAR(tanggal, 'YYYY-MM-DD'), status, jam_masuk, jam_pulang,
	TO_CHAR(jadwal_masuk, 'HH24:MI:SS'), TO_CHAR(jadwal_pulang, 'HH24:MI:SS'), catatan, sumber`

func (s *Store) scanAtt(row interface{ Scan(...any) error }) (Attendance, error) {
	var a Attendance
	var in, out sql.NullTime
	err := row.Scan(&a.ID, &a.EmployeeID, &a.Tanggal, &a.Status, &in, &out, &a.JadwalMasuk, &a.JadwalPulang, &a.Catatan, &a.Sumber)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	if in.Valid {
		t := in.Time.In(s.Loc)
		a.JamMasuk = &t
	}
	if out.Valid {
		t := out.Time.In(s.Loc)
		a.JamPulang = &t
	}
	return a, err
}

func (s *Store) queryAtt(ctx context.Context, c q, query string, args ...any) ([]Attendance, error) {
	rows, err := c.QueryContext(ctx, `SELECT `+attCols+` FROM attendance `+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attendance{}
	for rows.Next() {
		a, err := s.scanAtt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AttendanceOn(ctx context.Context, tx *sql.Tx, empID int64, day string) (Attendance, error) {
	return s.scanAtt(s.conn(tx).QueryRowContext(ctx, `SELECT `+attCols+` FROM attendance WHERE employee_id = ? AND tanggal = ? FOR UPDATE`, empID, day))
}

// OpenAttendance absen masuk terakhir (sejak `since`) yang belum absen pulang.
func (s *Store) OpenAttendance(ctx context.Context, tx *sql.Tx, empID int64, since string) (Attendance, error) {
	return s.scanAtt(s.conn(tx).QueryRowContext(ctx, `SELECT `+attCols+` FROM attendance
		WHERE employee_id = ? AND tanggal >= ? AND status = 'hadir' AND jam_masuk IS NOT NULL AND jam_pulang IS NULL
		ORDER BY tanggal DESC LIMIT 1 FOR UPDATE`, empID, since))
}

func (s *Store) AttendanceByDate(ctx context.Context, day string) ([]Attendance, error) {
	return s.queryAtt(ctx, s.conn(nil), `WHERE tanggal = ?`, day)
}

func (s *Store) AttendanceRange(ctx context.Context, empID int64, from, to string) ([]Attendance, error) {
	if empID == 0 {
		return s.queryAtt(ctx, s.conn(nil), `WHERE tanggal BETWEEN ? AND ? ORDER BY tanggal`, from, to)
	}

	return s.queryAtt(ctx, s.conn(nil), `WHERE employee_id = ? AND tanggal BETWEEN ? AND ? ORDER BY tanggal`, empID, from, to)
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullStr(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Store) UpsertAttendance(ctx context.Context, tx *sql.Tx, a Attendance, ip string, actor int64) error {
	_, err := s.conn(tx).ExecContext(ctx, `INSERT INTO attendance
		(employee_id, tanggal, status, jam_masuk, jam_pulang, jadwal_masuk, jadwal_pulang, ip_masuk, catatan, sumber, diubah_oleh)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT (employee_id, tanggal) DO UPDATE SET
		  status=EXCLUDED.status, jam_masuk=EXCLUDED.jam_masuk, jam_pulang=EXCLUDED.jam_pulang,
		  jadwal_masuk=EXCLUDED.jadwal_masuk, jadwal_pulang=EXCLUDED.jadwal_pulang, catatan=EXCLUDED.catatan,
		  sumber=EXCLUDED.sumber, diubah_oleh=EXCLUDED.diubah_oleh`,
		a.EmployeeID,
		a.Tanggal,
		a.Status,
		nullTime(a.JamMasuk),
		nullTime(a.JamPulang),
		nullStr(a.JadwalMasuk),
		nullStr(a.JadwalPulang),
		nullStr(ip),
		a.Catatan,
		a.Sumber,
		nullID(actor),
	)
	return err
}

func (s *Store) SetCheckOut(ctx context.Context, tx *sql.Tx, id int64, at time.Time, ip string) error {
	_, err := s.conn(tx).ExecContext(ctx, `UPDATE attendance SET jam_pulang = ?, ip_pulang = ? WHERE id = ?`, at, nullStr(ip), id)
	return err
}

func (s *Store) DeleteAttendance(ctx context.Context, empID int64, day string) error {
	_, err := s.conn(nil).ExecContext(ctx, `DELETE FROM attendance WHERE employee_id = ? AND tanggal = ?`, empID, day)
	return err
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ------------------------------------------------------------------ leave requests

type Leave struct {
	ID             int64  `json:"id"`
	EmployeeID     int64  `json:"employee_id"`
	Nama           string `json:"nama"`
	Jabatan        string `json:"jabatan"`
	Jenis          string `json:"jenis"`
	TanggalMulai   string `json:"tanggal_mulai"`
	TanggalSelesai string `json:"tanggal_selesai"`
	Alasan         string `json:"alasan"`
	Status         string `json:"status"`
	CatatanAdmin   string `json:"catatan_admin"`
	DibuatPada     string `json:"dibuat_pada"`
	DiputusPada    string `json:"diputus_pada"`
}

const leaveCols = `l.id, l.employee_id, e.nama, e.jabatan, l.jenis, TO_CHAR(l.tanggal_mulai, 'YYYY-MM-DD'),
	TO_CHAR(l.tanggal_selesai, 'YYYY-MM-DD'), l.alasan, l.status, l.catatan_admin,
	TO_CHAR(l.dibuat_pada, 'YYYY-MM-DD HH24:MI'), COALESCE(TO_CHAR(l.diputus_pada, 'YYYY-MM-DD HH24:MI'), '')`

func scanLeave(row interface{ Scan(...any) error }) (Leave, error) {
	var l Leave
	err := row.Scan(&l.ID, &l.EmployeeID, &l.Nama, &l.Jabatan, &l.Jenis, &l.TanggalMulai, &l.TanggalSelesai,
		&l.Alasan, &l.Status, &l.CatatanAdmin, &l.DibuatPada, &l.DiputusPada)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// ListLeaves empID 0 = semua karyawan; status "" = semua status.
func (s *Store) ListLeaves(ctx context.Context, empID int64, status string, limit int) ([]Leave, error) {
	where, args := []string{"1=1"}, []any{}
	if empID != 0 {
		where, args = append(where, "l.employee_id = ?"), append(args, empID)
	}
	if status != "" {
		where, args = append(where, "l.status = ?"), append(args, status)
	}
	args = append(args, limit)
	rows, err := s.conn(nil).QueryContext(ctx, `SELECT `+leaveCols+` FROM leave_requests l JOIN employees e ON e.id = l.employee_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY (l.status = 'menunggu') DESC, l.id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Leave{}
	for rows.Next() {
		l, err := scanLeave(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) LeaveForUpdate(ctx context.Context, tx *sql.Tx, id int64) (Leave, error) {
	return scanLeave(s.conn(tx).QueryRowContext(ctx, `SELECT `+leaveCols+` FROM leave_requests l JOIN employees e ON e.id = l.employee_id
		WHERE l.id = ? FOR UPDATE`, id))
}

func (s *Store) InsertLeave(ctx context.Context, l Leave) (int64, error) {
	var id int64
	err := s.conn(nil).QueryRowContext(ctx, `INSERT INTO leave_requests (employee_id, jenis, tanggal_mulai, tanggal_selesai, alasan)
		VALUES (?,?,?,?,?) RETURNING id`, l.EmployeeID, l.Jenis, l.TanggalMulai, l.TanggalSelesai, l.Alasan).Scan(&id)
	return id, err
}

func (s *Store) DecideLeave(ctx context.Context, tx *sql.Tx, id int64, status, catatan string, actor int64, at time.Time) error {
	_, err := s.conn(tx).ExecContext(ctx, `UPDATE leave_requests SET status=?, catatan_admin=?, diputus_oleh=?, diputus_pada=? WHERE id=?`,
		status, catatan, nullID(actor), at, id)
	return err
}

// DeletePendingLeave hanya menghapus pengajuan milik karyawan itu yang masih menunggu.
func (s *Store) DeletePendingLeave(ctx context.Context, id, empID int64) (int64, error) {
	r, err := s.conn(nil).ExecContext(ctx, `DELETE FROM leave_requests WHERE id=? AND employee_id=? AND status='menunggu'`, id, empID)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func (s *Store) CountPendingLeaves(ctx context.Context) (int, error) {
	var n int
	err := s.conn(nil).QueryRowContext(ctx, `SELECT COUNT(*) FROM leave_requests WHERE status='menunggu'`).Scan(&n)
	return n, err
}

// ------------------------------------------------------------------ holidays

type Holiday struct {
	Tanggal    string `json:"tanggal"`
	Keterangan string `json:"keterangan"`
}

func (s *Store) ListHolidays(ctx context.Context, from, to string) ([]Holiday, error) {
	rows, err := s.conn(nil).QueryContext(ctx, `SELECT TO_CHAR(tanggal, 'YYYY-MM-DD'), keterangan FROM holidays
		WHERE tanggal BETWEEN ? AND ? ORDER BY tanggal`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Holiday{}
	for rows.Next() {
		var h Holiday
		if err := rows.Scan(&h.Tanggal, &h.Keterangan); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) HolidayMap(ctx context.Context, from, to string) (map[string]string, error) {
	list, err := s.ListHolidays(ctx, from, to)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(list))
	for _, h := range list {
		m[h.Tanggal] = h.Keterangan
	}
	return m, nil
}

func (s *Store) UpsertHoliday(ctx context.Context, h Holiday) error {
	_, err := s.conn(nil).ExecContext(ctx, `INSERT INTO holidays (tanggal, keterangan) VALUES (?,?)
		ON CONFLICT (tanggal) DO UPDATE SET keterangan = EXCLUDED.keterangan`, h.Tanggal, h.Keterangan)
	return err
}

func (s *Store) DeleteHoliday(ctx context.Context, day string) error {
	_, err := s.conn(nil).ExecContext(ctx, `DELETE FROM holidays WHERE tanggal = ?`, day)
	return err
}

// ------------------------------------------------------------------ settings

func (s *Store) Settings(ctx context.Context) (map[string]string, error) {
	rows, err := s.conn(nil).QueryContext(ctx, `SELECT kunci, nilai FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (s *Store) SaveSettings(ctx context.Context, in map[string]string) error {
	tx, err := s.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range in {
		if _, err := s.conn(tx).ExecContext(ctx, `INSERT INTO settings (kunci, nilai) VALUES (?,?)
			ON CONFLICT (kunci) DO UPDATE SET nilai = EXCLUDED.nilai`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ------------------------------------------------------------------ payroll periods

type Period struct {
	Bulan       string
	Data        []byte
	DitutupPada time.Time
}

func (s *Store) Period(ctx context.Context, bulan string) (Period, bool, error) {
	var p Period
	err := s.conn(nil).QueryRowContext(ctx, `SELECT bulan, data, ditutup_pada FROM payroll_periods WHERE bulan = ?`, bulan).
		Scan(&p.Bulan, &p.Data, &p.DitutupPada)
	if errors.Is(err, sql.ErrNoRows) {
		return p, false, nil
	}
	return p, err == nil, err
}

// IsClosed true bila bulan dari tanggal YYYY-MM-DD sudah ditutup (absensi terkunci).
func (s *Store) IsClosed(ctx context.Context, tx *sql.Tx, day string) (bool, error) {
	if len(day) < 7 {
		return false, nil
	}
	var n int
	err := s.conn(tx).QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_periods WHERE bulan = ?`, day[:7]).Scan(&n)
	return n > 0, err
}

func (s *Store) ClosePeriod(ctx context.Context, bulan string, data []byte, actor int64) error {
	_, err := s.conn(nil).ExecContext(ctx, `INSERT INTO payroll_periods (bulan, data, ditutup_oleh) VALUES (?,?,?)`, bulan, data, nullID(actor))
	return err
}

func (s *Store) ReopenPeriod(ctx context.Context, bulan string) (int64, error) {
	r, err := s.conn(nil).ExecContext(ctx, `DELETE FROM payroll_periods WHERE bulan = ?`, bulan)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

// Atoi helper untuk settings.
func Atoi(m map[string]string, k string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(m[k])); err == nil {
		return v
	}
	return def
}

func Atof(m map[string]string, k string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(m[k]), 64); err == nil {
		return v
	}
	return def
}
