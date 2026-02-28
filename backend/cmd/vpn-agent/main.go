package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	cfg := loadConfig()

	// Ensure required directories exist.
	certDir := filepath.Join(cfg.StateDir, "certificates")
	for _, dir := range []string{cfg.OvpnConfigDir, cfg.StateDir, cfg.LogDir, certDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Initialize components.
	store := NewConfigStore(cfg.OvpnConfigDir)
	stateStore := NewStateStore(cfg.StateDir)
	certStore := NewCertStore(certDir)
	callback := NewCallbackClient(cfg.Sub2APIURL, cfg.Sub2APIKey)
	tunnelMgr := NewTunnelManager(cfg, store, stateStore, callback, certStore)
	healthChecker := NewHealthChecker(tunnelMgr, store, callback, cfg)
	server := NewServer(cfg, store, certStore, tunnelMgr, healthChecker)

	// Restore previous tunnel state.
	if err := tunnelMgr.RestoreFromState(); err != nil {
		log.Printf("Warning: failed to restore state: %v", err)
	}

	// Start health checker in background.
	healthCtx, healthCancel := context.WithCancel(context.Background())
	defer healthCancel()
	go healthChecker.Start(healthCtx)

	// Start HTTP server.
	addr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.Port)
	srv := &http.Server{Addr: addr, Handler: server.Router()}
	log.Printf("VPN Agent listening on %s", addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	healthCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	tunnelMgr.StopAll()
	log.Println("VPN Agent stopped")
}
