-- Sistem Absensi & Penggajian
-- MySQL 8 / MariaDB 10.6+
-- Jalankan sekali di database kosong, mis.:  CREATE DATABASE absensi;  lalu impor file ini.
SET NAMES utf8mb4;

-- Akun login. role: admin (HRD) atau karyawan.
CREATE TABLE users (
  id            BIGINT AUTO_INCREMENT PRIMARY KEY,
  email         VARCHAR(150) NOT NULL UNIQUE,
  password_hash VARCHAR(100) NOT NULL,
  role          ENUM('admin','karyawan') NOT NULL DEFAULT 'karyawan',
  nama          VARCHAR(100) NOT NULL,
  aktif         BOOLEAN NOT NULL DEFAULT TRUE,
  dibuat_pada   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Data karyawan + skema gaji. Satu karyawan punya satu akun login (users.role = karyawan).
--  tipe_gaji = bulanan : gaji_pokok per bulan, dipotong per hari alpa
--  tipe_gaji = harian  : gaji_pokok per hari hadir
CREATE TABLE employees (
  id                   BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id              BIGINT NOT NULL UNIQUE,
  nik                  VARCHAR(30) NOT NULL UNIQUE,
  nama                 VARCHAR(100) NOT NULL,
  jabatan              VARCHAR(60) NOT NULL DEFAULT '',
  telepon              VARCHAR(30) NOT NULL DEFAULT '',
  tanggal_masuk        DATE NULL,
  tipe_gaji            ENUM('bulanan','harian') NOT NULL DEFAULT 'bulanan',
  gaji_pokok           DECIMAL(14,2) NOT NULL DEFAULT 0,
  tunjangan_tetap      DECIMAL(14,2) NOT NULL DEFAULT 0,  -- per bulan (jabatan, dll.)
  uang_makan           DECIMAL(14,2) NOT NULL DEFAULT 0,  -- per hari hadir
  tarif_lembur_per_jam DECIMAL(14,2) NOT NULL DEFAULT 0,
  jam_masuk            TIME NOT NULL DEFAULT '08:00:00',
  jam_pulang           TIME NOT NULL DEFAULT '16:00:00',
  aktif                BOOLEAN NOT NULL DEFAULT TRUE,
  dibuat_pada          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Satu baris per karyawan per tanggal.
-- status: hadir (absen sendiri / dikoreksi admin), izin, sakit, cuti (dari pengajuan yang disetujui
-- atau diisi admin), alpa (ditandai admin). Hari kerja tanpa baris dihitung alpa di rekap gaji.
CREATE TABLE attendance (
  id            BIGINT AUTO_INCREMENT PRIMARY KEY,
  employee_id   BIGINT NOT NULL,
  tanggal       DATE NOT NULL,
  status        ENUM('hadir','izin','sakit','cuti','alpa') NOT NULL,
  jam_masuk     DATETIME NULL,
  jam_pulang    DATETIME NULL,
  jadwal_masuk  TIME NOT NULL,          -- jadwal saat absen, supaya rekap lama tidak berubah bila jadwal diganti
  jadwal_pulang TIME NOT NULL,
  ip_masuk      VARCHAR(45) NULL,
  ip_pulang     VARCHAR(45) NULL,
  catatan       VARCHAR(255) NOT NULL DEFAULT '',
  sumber        ENUM('karyawan','admin','pengajuan') NOT NULL DEFAULT 'karyawan',
  diubah_oleh   BIGINT NULL,
  diubah_pada   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_att (employee_id, tanggal),
  INDEX idx_att_tanggal (tanggal),
  FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
  FOREIGN KEY (diubah_oleh) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;

-- Pengajuan izin / sakit / cuti oleh karyawan, disetujui atau ditolak admin.
CREATE TABLE leave_requests (
  id              BIGINT AUTO_INCREMENT PRIMARY KEY,
  employee_id     BIGINT NOT NULL,
  jenis           ENUM('izin','sakit','cuti') NOT NULL,
  tanggal_mulai   DATE NOT NULL,
  tanggal_selesai DATE NOT NULL,
  alasan          VARCHAR(255) NOT NULL DEFAULT '',
  status          ENUM('menunggu','disetujui','ditolak') NOT NULL DEFAULT 'menunggu',
  catatan_admin   VARCHAR(255) NOT NULL DEFAULT '',
  diputus_oleh    BIGINT NULL,
  diputus_pada    DATETIME NULL,
  dibuat_pada     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_leave_status (status),
  FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
  FOREIGN KEY (diputus_oleh) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;

-- Hari libur (nasional / perusahaan). Tidak dihitung hari kerja.
CREATE TABLE holidays (
  tanggal    DATE PRIMARY KEY,
  keterangan VARCHAR(120) NOT NULL
) ENGINE=InnoDB;

-- Parameter aturan absensi & gaji (kunci -> nilai).
CREATE TABLE settings (
  kunci VARCHAR(50) PRIMARY KEY,
  nilai VARCHAR(50) NOT NULL
) ENGINE=InnoDB;

-- Periode gaji yang sudah ditutup: hasil hitung disimpan (snapshot) dan absensi bulan itu dikunci.
CREATE TABLE payroll_periods (
  bulan       CHAR(7) PRIMARY KEY,   -- 2026-11
  data        JSON NOT NULL,
  ditutup_oleh BIGINT NULL,
  ditutup_pada DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (ditutup_oleh) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;

INSERT INTO settings (kunci, nilai) VALUES
  ('toleransi_telat_menit', '10'),     -- masuk sampai jadwal + 10 menit belum dihitung telat
  ('potongan_telat_per_menit', '1000'),-- rupiah per menit telat
  ('hari_kerja_sebulan', '26'),        -- pembagi gaji bulanan untuk potongan alpa
  ('lembur_minimal_menit', '30'),      -- lembur dihitung per kelipatan menit ini
  ('hari_libur_mingguan', '0'),        -- hari libur rutin, dipisah koma: 0=Minggu ... 6=Sabtu
  ('awal_absen_menit', '120');         -- absen masuk paling cepat 120 menit sebelum jadwal
