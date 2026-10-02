package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/config"
	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
	"github.com/distributedcompute/cloud/control-plane/internal/registry"
	"github.com/distributedcompute/cloud/control-plane/internal/scheduler"
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

	// Initialize job store
	jobStore := jobs.NewStore()

	// Initialize scheduler
	sched := scheduler.NewScheduler(workerStore, jobStore)
	sched.Start()
	defer sched.Stop()

	// Initialize environment registry
	envRegistry := registry.NewRegistry("environments")

	// Register default environments
	envRegistry.Register(&registry.Environment{
		ID:        "python-3.11",
		Name:      "Python 3.11",
		Version:   "1.0.0",
		Hash:      "abc123def456",
		CreatedAt: time.Now(),
		Metadata:  map[string]string{"description": "Python 3.11 base environment"},
	})
	envRegistry.Register(&registry.Environment{
		ID:        "node-20",
		Name:      "Node.js 20",
		Version:   "1.0.0",
		Hash:      "def456abc123",
		CreatedAt: time.Now(),
		Metadata:  map[string]string{"description": "Node.js 20 base environment"},
	})

	// Create and start server
	srv := server.NewServer(tokenManager, ca, workerStore, jobStore, envRegistry)

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
