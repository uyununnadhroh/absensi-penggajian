
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"absensi/internal/api"
	"absensi/internal/config"
	"absensi/internal/store"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[CONFIG] gagal membaca konfigurasi: %v", err)
	}

	// Pemeriksaan konfigurasi tanpa menampilkan nilai rahasia.
	if cfg.DSN == "" || len(cfg.JWTSecret) < 32 {
		log.Printf(
			"[CONFIG DEBUG] DSN terisi: %t, panjang JWT_SECRET: %d",
			cfg.DSN != "",
			len(cfg.JWTSecret),
		)
		log.Fatal("[CONFIG] konfigurasi belum valid")
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("[CONFIG] zona waktu %q: %v", cfg.Timezone, err)
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		log.Fatalf("[DB] gagal membuka koneksi PostgreSQL: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("[DB] tidak bisa terhubung ke PostgreSQL Supabase: %v", err)
	}

	log.Println("[DB] koneksi PostgreSQL berhasil")

	srv := &api.Server{
		Store:   &store.Store{DB: db, Loc: loc},
		Secret:  []byte(cfg.JWTSecret),
		Loc:     loc,
		Origins: cfg.CORSOrigins,
	}

	hs := &http.Server{
		Addr:              cfg.Port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[HTTP] server berjalan di %s", cfg.Port)

		if err := hs.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[HTTP] server gagal: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := hs.Shutdown(ctx); err != nil {
		log.Printf("[HTTP] shutdown: %v", err)
	}

	log.Println("[HTTP] server berhenti")
}
