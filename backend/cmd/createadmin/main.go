// Membuat akun admin (HRD) pertama.
//
//	go run ./cmd/createadmin -email admin@perusahaan.com -nama "Admin HRD" -password "minimal8karakter"
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"strings"

	"absensi/internal/api"
	"absensi/internal/config"
	"absensi/internal/store"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	email := flag.String("email", "", "email login admin")
	nama := flag.String("nama", "Admin", "nama admin")
	pw := flag.String("password", "", "kata sandi (min. 8 karakter)")
	flag.Parse()
	if *email == "" || *pw == "" {
		log.Fatal("wajib: -email dan -password")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := api.HashPassword(*pw)
	if err != nil {
		log.Fatal(err)
	}
	st := &store.Store{DB: db}
	id, err := st.InsertUser(context.Background(), nil, strings.ToLower(strings.TrimSpace(*email)), hash, "admin", *nama)
	if err != nil {
		log.Fatalf("gagal membuat admin: %v", err)
	}
	log.Printf("admin dibuat (id %d): %s", id, *email)
}
