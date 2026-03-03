package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/foodrelay/foodrelay/internal/common"
	"github.com/foodrelay/foodrelay/internal/node"
	"github.com/foodrelay/foodrelay/internal/registry"
)

func main() {
	common.LoadDotEnv(".env")

	cfg := node.LoadConfig()

	// ── ed25519 keypair ───────────────────────────────────────────────────────
	var pubKeyHex string
	var privKey ed25519.PrivateKey

	if cfg.PrivKeySeed == "" {
		pub, seed, err := common.GenerateKeyPair()
		if err != nil {
			log.Fatalf("generate keypair: %v", err)
		}
		_, privKey, _ = common.KeyPairFromSeedHex(seed)
		pubKeyHex = pub
		log.Printf("⚠  First run — add to .env: NODE_PRIVATE_KEY_HEX=%s", seed)
		log.Printf("   Public key: %s", pub)
	} else {
		pub, priv, err := common.KeyPairFromSeedHex(cfg.PrivKeySeed)
		if err != nil {
			log.Fatalf("load keypair: %v", err)
		}
		pubKeyHex, privKey = pub, priv
	}

	// ── Node database & migrations ────────────────────────────────────────────
	store, db, err := node.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	migrationsDir := common.Env("MIGRATIONS_DIR", "migrations")
	if err := common.RunMigrations(db, os.DirFS(migrationsDir), "node_"); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("node migrations ok")

	// ── Stop channel (shared by all background workers) ───────────────────────
	stop := make(chan struct{})

	// ── Combined mode: embedded registry ─────────────────────────────────────
	if cfg.RunRegistry {
		regCfg := registry.LoadConfig()

		regStore, regDB, err := registry.NewSQLiteStore(regCfg.DBPath)
		if err != nil {
			log.Fatalf("open registry store: %v", err)
		}
		defer regDB.Close()

		if err := common.RunMigrations(regDB, os.DirFS(migrationsDir), "registry_"); err != nil {
			log.Fatalf("registry migrations: %v", err)
		}
		log.Println("registry migrations ok")

		// Prepend local registry URL so the node announces to itself first.
		localURL := fmt.Sprintf("http://localhost%s", regCfg.Addr)
		cfg.RegistryURLs = append([]string{localURL}, cfg.RegistryURLs...)

		// Start federation syncer for the embedded registry.
		syncer := &registry.FederationSyncer{
			Store: regStore,
			Peers: regCfg.Peers,
			Self:  localURL,
		}
		go syncer.Run(stop)

		// Start embedded registry HTTP server.
		regHandler := &registry.Handler{Store: regStore}
		regRouter := chi.NewRouter()
		regRouter.Use(middleware.Logger)
		regRouter.Use(middleware.Recoverer)
		regRouter.Post("/v1/nodes/announce", regHandler.Announce)
		regRouter.Post("/v1/nodes/heartbeat", regHandler.Heartbeat)
		regRouter.Get("/v1/nodes", regHandler.ListNodes)
		regRouter.Get("/v1/federation/nodes", regHandler.FederationNodes)
		regRouter.Get("/map", regHandler.MapUI)
		regRouter.Get("/health", registry.HealthCheck)

		regSrv := &http.Server{Addr: regCfg.Addr, Handler: regRouter}
		go func() {
			log.Printf("embedded registry listening on %s", regCfg.Addr)
			if err := regSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("embedded registry: %v", err)
			}
		}()
	}

	// ── Node services ─────────────────────────────────────────────────────────
	sender := &node.Sender{
		AccountSID: cfg.TwilioSID,
		AuthToken:  cfg.TwilioToken,
		FromPhone:  cfg.TwilioFrom,
	}
	admin := &node.Admin{Phones: cfg.AdminPhones, Sender: sender}
	engine := &node.Engine{Store: store, Sender: sender}
	scheduler := &node.Scheduler{
		Store:         store,
		Engine:        engine,
		Admin:         admin,
		EscalateAfter: time.Duration(cfg.EscalateAfter) * time.Minute,
	}
	hbClient := &node.HeartbeatClient{
		Cfg:       cfg,
		PubKeyHex: pubKeyHex,
		PrivKey:   privKey,
	}
	twilioHandler := &node.TwilioHandler{Store: store, Engine: engine, Sender: sender}

	// ── Node router ───────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/twilio/sms", twilioHandler.ServeHTTP)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"node_id": cfg.NodeID,
		})
	})

	// ── Background workers ────────────────────────────────────────────────────
	go scheduler.Run(stop)
	go hbClient.Run(stop)

	// ── Node HTTP server ──────────────────────────────────────────────────────
	srv := &http.Server{Addr: cfg.Addr, Handler: r}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("node %s listening on %s", cfg.NodeID, cfg.Addr)
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
