#!/bin/bash
# Setup database untuk sistem absensi (dengan prompt password)

echo "=== Setup Database Absensi ==="
echo ""
echo "Masukkan password MySQL root (tekan Enter jika tidak ada password):"
read -s MYSQL_PASSWORD

if [ -z "$MYSQL_PASSWORD" ]; then
    # Tanpa password
    mysql -u root << EOF
CREATE DATABASE IF NOT EXISTS absensi;
USE absensi;
SOURCE /Users/macprom12020/Downloads/absensi-penggajian/backend/db/schema.sql;
SELECT 'Database berhasil dibuat!' AS Status;
EOF
else
    # Dengan password
    mysql -u root -p"$MYSQL_PASSWORD" << EOF
CREATE DATABASE IF NOT EXISTS absensi;
USE absensi;
SOURCE /Users/macprom12020/Downloads/absensi-penggajian/backend/db/schema.sql;
SELECT 'Database berhasil dibuat!' AS Status;
EOF
fi

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Database 'absensi' berhasil dibuat dan schema diimport"
else
    echo ""
    echo "✗ Gagal membuat database. Pastikan:"
    echo "  1. MySQL sudah berjalan"
    echo "  2. Password yang dimasukkan benar"
    exit 1
fi
