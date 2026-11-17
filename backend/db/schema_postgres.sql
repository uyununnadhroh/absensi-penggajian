-- Sistem Absensi & Penggajian (PostgreSQL)
-- Supabase Compatible

-- Drop tables if exists
DROP TABLE IF EXISTS payroll_periods CASCADE;
DROP TABLE IF EXISTS holidays CASCADE;
DROP TABLE IF EXISTS settings CASCADE;
DROP TABLE IF EXISTS leave_requests CASCADE;
DROP TABLE IF EXISTS attendance CASCADE;
DROP TABLE IF EXISTS employees CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Akun login
CREATE TABLE users (
  id            BIGSERIAL PRIMARY KEY,
  email         VARCHAR(150) NOT NULL UNIQUE,
  password_hash VARCHAR(100) NOT NULL,
  role          VARCHAR(20) NOT NULL DEFAULT 'karyawan' CHECK (role IN ('admin', 'karyawan')),
  nama          VARCHAR(100) NOT NULL,
  aktif         BOOLEAN NOT NULL DEFAULT TRUE,
  dibuat_pada   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Data karyawan + skema gaji
CREATE TABLE employees (
  id                   BIGSERIAL PRIMARY KEY,
  user_id              BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  nik                  VARCHAR(30) NOT NULL UNIQUE,
  nama                 VARCHAR(100) NOT NULL,
  jabatan              VARCHAR(60) NOT NULL DEFAULT '',
  telepon              VARCHAR(30) NOT NULL DEFAULT '',
  tanggal_masuk        DATE NULL,
  tipe_gaji            VARCHAR(20) NOT NULL DEFAULT 'per_jam' CHECK (tipe_gaji IN ('bulanan', 'harian', 'per_jam')),
  gaji_pokok           NUMERIC(14,2) NOT NULL DEFAULT 0,
  tarif_per_jam        NUMERIC(14,2) NOT NULL DEFAULT 0,
  tunjangan_tetap      NUMERIC(14,2) NOT NULL DEFAULT 0,
  uang_makan           NUMERIC(14,2) NOT NULL DEFAULT 0,
  tarif_lembur_per_jam NUMERIC(14,2) NOT NULL DEFAULT 0,
  jam_masuk            TIME NULL,
  jam_pulang           TIME NULL,
  aktif                BOOLEAN NOT NULL DEFAULT TRUE,
  dibuat_pada          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Absensi
CREATE TABLE attendance (
  id            BIGSERIAL PRIMARY KEY,
  employee_id   BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  tanggal       DATE NOT NULL,
  status        VARCHAR(20) NOT NULL CHECK (status IN ('hadir', 'izin', 'sakit', 'cuti', 'alpa')),
  jam_masuk     TIMESTAMP NULL,
  jam_pulang    TIMESTAMP NULL,
  jadwal_masuk  TIME NULL,
  jadwal_pulang TIME NULL,
  ip_masuk      VARCHAR(45) NULL,
  ip_pulang     VARCHAR(45) NULL,
  catatan       VARCHAR(255) NOT NULL DEFAULT '',
  sumber        VARCHAR(20) NOT NULL DEFAULT 'karyawan' CHECK (sumber IN ('karyawan', 'admin', 'pengajuan')),
  diubah_oleh   BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
  diubah_pada   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (employee_id, tanggal)
);

CREATE INDEX idx_att_tanggal ON attendance(tanggal);
CREATE INDEX idx_attendance_employee ON attendance(employee_id);

-- Pengajuan izin/sakit/cuti
CREATE TABLE leave_requests (
  id              BIGSERIAL PRIMARY KEY,
  employee_id     BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  jenis           VARCHAR(20) NOT NULL CHECK (jenis IN ('izin', 'sakit', 'cuti')),
  tanggal_mulai   DATE NOT NULL,
  tanggal_selesai DATE NOT NULL,
  alasan          VARCHAR(255) NOT NULL DEFAULT '',
  status          VARCHAR(20) NOT NULL DEFAULT 'menunggu' CHECK (status IN ('menunggu', 'disetujui', 'ditolak')),
  catatan_admin   VARCHAR(255) NOT NULL DEFAULT '',
  diputus_oleh    BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
  diputus_pada    TIMESTAMP NULL,
  dibuat_pada     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_leave_status ON leave_requests(status);
CREATE INDEX idx_leave_employee ON leave_requests(employee_id);

-- Hari libur
CREATE TABLE holidays (
  tanggal    DATE PRIMARY KEY,
  keterangan VARCHAR(120) NOT NULL
);

-- Settings
CREATE TABLE settings (
  kunci VARCHAR(50) PRIMARY KEY,
  nilai VARCHAR(50) NOT NULL
);

-- Periode gaji
CREATE TABLE payroll_periods (
  bulan        CHAR(7) PRIMARY KEY,
  data         JSONB NOT NULL,
  ditutup_oleh BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
  ditutup_pada TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings
INSERT INTO settings (kunci, nilai) VALUES
  ('toleransi_telat_menit', '10'),
  ('potongan_telat_per_menit', '1000'),
  ('hari_kerja_sebulan', '26'),
  ('lembur_minimal_menit', '30'),
  ('hari_libur_mingguan', '0'),
  ('awal_absen_menit', '120'),
  ('tarif_per_jam_default', '15000');

-- Create additional indexes for performance
CREATE INDEX idx_employees_aktif ON employees(aktif);
CREATE INDEX idx_employees_tipe_gaji ON employees(tipe_gaji);
