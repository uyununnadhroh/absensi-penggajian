#!/bin/bash
# Setup database untuk sistem absensi

echo "Creating database and importing schema..."

# Coba tanpa password dulu
mysql -u root << EOF
CREATE DATABASE IF NOT EXISTS absensi;
USE absensi;
SOURCE /Users/macprom12020/Downloads/absensi-penggajian/backend/db/schema.sql;
EOF

if [ $? -eq 0 ]; then
    echo "✓ Database 'absensi' berhasil dibuat dan schema diimport"
else
    echo "✗ Gagal membuat database. Silakan cek koneksi MySQL Anda."
    exit 1
fi
