
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
		log.Fatalf("[CONFIG] %v", err)
	}

	if cfg.DSN == "" || len(cfg.JWTSecret) < 32 {
		log.Fatal("[CONFIG] dsn wajib diisi dan jwt_secret minimal 32 karakter (lihat config.yaml)")
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("[CONFIG] zona waktu %q: %v", cfg.Timezone, err)
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		log.Fatalf("[DB] %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("[DB] tidak bisa terhubung ke PostgreSQL Supabase: %v", err)
	}

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
		log.Printf("[HTTP] berjalan di %s", cfg.Port)
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[HTTP] %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := hs.Shutdown(ctx); err != nil {
		log.Printf("[HTTP] shutdown: %v", err)
	}
}
