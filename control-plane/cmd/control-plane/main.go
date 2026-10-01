package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/config"
	"github.com/distributedcompute/cloud/control-plane/internal/server"
)

func main() {
	cfg := config.Load()

	// Initialize token manager
	tokenManager, err := auth.NewTokenManager()
	if err != nil {
		log.Fatalf("Failed to initialize token manager: %v", err)
	}

	// Initialize CA
	ca, err := auth.NewCA("certs/ca.crt", "certs/ca.key")
	if err != nil {
		log.Fatalf("Failed to initialize CA: %v", err)
	}

	// Initialize worker store
	workerStore := auth.NewWorkerStore()

	// Create and start server
	srv := server.NewServer(tokenManager, ca, workerStore)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down Control Plane...")
		srv.Stop()
	}()

	addr := cfg.BindAddress
	if addr == "" {
		addr = ":8443"
	}

	if err := srv.Start(addr, "certs/server.crt", "certs/server.key"); err != nil {
		log.Fatalf("Control Plane failed: %v", err)
		os.Exit(1)
	}
}
