package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/foodrelay/foodrelay/internal/common"
	"github.com/foodrelay/foodrelay/internal/registry"
)

func main() {
	common.LoadDotEnv(".env")

	cfg := registry.LoadConfig()

	// ── Database & migrations ─────────────────────────────────────────────────
	store, db, err := registry.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	migrationsDir := common.Env("MIGRATIONS_DIR", "migrations")
	if err := common.RunMigrations(db, os.DirFS(migrationsDir), "registry_"); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("migrations ok")

	// ── Handler & router ──────────────────────────────────────────────────────
	h := &registry.Handler{Store: store}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/v1/nodes/announce", h.Announce)
	r.Post("/v1/nodes/heartbeat", h.Heartbeat)
	r.Get("/v1/nodes", h.ListNodes)
	r.Get("/v1/federation/nodes", h.FederationNodes)
	r.Get("/map", h.MapUI)
	r.Get("/health", registry.HealthCheck)

	// ── Federation syncer ─────────────────────────────────────────────────────
	stop := make(chan struct{})

	selfURL := "http://localhost" + cfg.Addr
	syncer := &registry.FederationSyncer{
		Store: store,
		Peers: cfg.Peers,
		Self:  selfURL,
	}
	go syncer.Run(stop)

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{Addr: cfg.Addr, Handler: r}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("registry listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx) //nolint:errcheck
}
