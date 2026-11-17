# Sistem Absensi & Penggajian

Website absensi karyawan untuk keperluan penggajian.
- **Backend:** Go + MySQL (bisa pakai XAMPP)
- **Frontend:** Next.js + Tailwind

## Fitur

**Karyawan (login sendiri, bisa dari HP)**
- Absen Masuk / Absen Pulang memakai jam server, jadi tidak bisa diakali dengan mengubah jam HP. IP perangkat ikut dicatat.
- Riwayat absensi per bulan: hadir, izin, sakit, cuti, alpa, telat, pulang cepat, dan lembur.
- Pengajuan izin / sakit / cuti, bisa dibatalkan selama belum diputuskan admin.
- Slip gaji bulanan yang bisa dicetak atau disimpan sebagai PDF.
- Ganti kata sandi.

**Admin / HRD**
- Dashboard hari ini: siapa yang hadir, telat, belum absen, izin, atau alpa, plus jumlah pengajuan yang menunggu.
- Kelola karyawan: akun login, jadwal kerja (termasuk shift malam), dan skema gaji **bulanan** atau **harian**.
- Data absensi per tanggal, termasuk koreksi atau input manual.
- Persetujuan pengajuan. Hari yang disetujui otomatis tercatat di absensi.
- Penggajian bulanan:
  - rekap semua karyawan dan slip per karyawan
  - unduh Excel (CSV)
  - **Tutup periode**: angka jadi final dan absensi bulan itu terkunci. Periode bisa dibuka kembali.
- Pengaturan:
  - toleransi telat, potongan per menit, hari kerja sebulan, kelipatan lembur
  - hari libur mingguan
  - daftar hari libur nasional / perusahaan

## Cara hitung gaji

| Komponen | Rumus |
|---|---|
| Gaji pokok (bulanan) | gaji pokok penuh; prorata bila mulai bekerja di tengah bulan |
| Gaji pokok (harian) | upah harian × hari hadir |
| Tunjangan tetap | nominal per bulan |
| Uang makan / transport | nominal × hari hadir |
| Lembur | menit setelah jadwal pulang (dibulatkan ke bawah per 30 menit) ÷ 60 × tarif per jam |
| Potongan alpa (bulanan) | gaji pokok ÷ hari kerja sebulan (default 26) × hari alpa |
| Potongan telat | (menit telat + menit pulang cepat) × potongan per menit |

- **Alpa** = hari kerja yang sudah lewat tanpa absensi dan tanpa izin, sakit, atau cuti. Hari libur mingguan dan hari libur di daftar tidak dihitung.
- **Telat** = masuk lewat dari jadwal + toleransi (default 10 menit). Kalau lewat toleransi, telat dihitung penuh dari jam jadwal.
- Izin, sakit, dan cuti tetap dibayar untuk gaji bulanan, tetapi tidak mendapat uang makan.
- Gaji bersih tidak pernah minus.

Semua angka di atas bisa diubah di menu **Pengaturan**.

## Cara menjalankan (Windows + XAMPP)

### 1. Database
1. Start MySQL di XAMPP.
2. Buka phpMyAdmin, lalu buat database `absensi`.
3. Impor file `backend/db/schema.sql`.

### 2. Backend
1. Pasang Go 1.22 atau lebih baru.
2. Di folder `backend`, salin `config.example.yaml` menjadi `config.yaml`.
3. Ganti `jwt_secret` dengan teks acak minimal 32 karakter.
4. Jalankan perintah berikut dari folder `backend`:

```powershell
go mod tidy
go run ./cmd/createadmin -email admin@perusahaan.com -nama "Admin HRD" -password "minimal8karakter"
go run ./cmd/server
```

Backend berjalan di `http://localhost:8080`. Untuk mengecek, buka `http://localhost:8080/api/health`.

### 3. Frontend
1. Pasang Node.js 18 atau lebih baru.
2. Di folder `frontend`, salin `.env.example` menjadi `.env.local`. Isinya alamat backend.
3. Jalankan:

```powershell
npm install
npm run dev
```

4. Buka `http://localhost:3000` dan login sebagai admin.
5. Tambahkan karyawan di menu **Karyawan**. Email dan kata sandi awal yang diisi di sana dipakai karyawan untuk login dan absen.

> Supaya karyawan bisa absen dari HP, server harus bisa diakses dari jaringan mereka, misalnya lewat Wi-Fi kantor dengan IP komputer server.
> Kalau begitu:
> - ganti `NEXT_PUBLIC_API_URL` dengan IP server, dan
> - tambahkan alamat frontend ke `cors_origins` di `config.yaml`.

## Pengujian

```powershell
go test ./internal/payroll/
```

Tes integrasi butuh database **kosong** `absensi_test` yang sudah berisi `schema.sql`. Isi tabelnya akan dihapus saat tes berjalan.

```powershell
$env:TEST_DB_DSN="root:@tcp(localhost:3306)/absensi_test?parseTime=true&loc=Asia%2FJakarta"; go test ./...
```

## Struktur

```
backend/
  cmd/server        server API
  cmd/createadmin   membuat akun admin pertama
  db/schema.sql     tabel + pengaturan awal
  internal/payroll  aturan hitung telat, lembur, alpa, dan gaji (murni, ada unit test)
  internal/store    semua query SQL
  internal/api      route, login (JWT), dan handler
frontend/
  pages/            absen, riwayat, pengajuan, slip, akun, admin/*
  components/       layout, slip gaji, komponen UI
  lib/              api client, sesi, format rupiah/tanggal
```
