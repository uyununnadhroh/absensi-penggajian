-- Migrasi ke sistem gaji per jam
-- Jalankan di database absensi

-- Tambah tipe gaji 'per_jam'
ALTER TABLE employees 
  MODIFY COLUMN tipe_gaji ENUM('bulanan','harian','per_jam') NOT NULL DEFAULT 'per_jam';

-- Tambah kolom tarif per jam
ALTER TABLE employees 
  ADD COLUMN tarif_per_jam DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER gaji_pokok;

-- Update settings untuk sistem per jam
INSERT INTO settings (kunci, nilai) VALUES
  ('tarif_per_jam_default', '15000')
ON DUPLICATE KEY UPDATE nilai=VALUES(nilai);

-- Hapus field yang tidak relevan untuk sistem per jam (opsional, bisa dipertahankan untuk backward compatibility)
-- Field gaji_pokok, tunjangan_tetap, uang_makan tetap dipertahankan untuk karyawan dengan tipe_gaji bulanan/harian

