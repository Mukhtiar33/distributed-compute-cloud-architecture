package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/batch"
	"github.com/distributedcompute/cloud/control-plane/internal/config"
	"github.com/distributedcompute/cloud/control-plane/internal/db"
	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
	"github.com/distributedcompute/cloud/control-plane/internal/registry"
	"github.com/distributedcompute/cloud/control-plane/internal/scheduler"
	"github.com/distributedcompute/cloud/control-plane/internal/server"
)

func main() {
	cfg := config.Load()

	// Initialize database
	dbCfg := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "distributed_compute_cloud"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	database, err := db.New(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize token manager with persistent key
	tokenManager, err := auth.NewTokenManager("certs/token-key.pem")
	if err != nil {
		log.Fatalf("Failed to initialize token manager: %v", err)
	}

	// Initialize CA
	ca, err := auth.NewCA("certs/ca.crt", "certs/ca.key")
	if err != nil {
		log.Fatalf("Failed to initialize CA: %v", err)
	}

	// Initialize database-backed stores
	workerStore := auth.NewWorkerStoreDB(database)
	jobStore := jobs.NewStoreDB(database)
	_ = batch.NewManagerDB(database) // Used in Phase R4

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

	// Worker mTLS server on port 9443
	workerAddr := ":9443"

	if err := srv.Start(addr, workerAddr, "certs/server.crt", "certs/server.key"); err != nil {
		log.Fatalf("Control Plane failed: %v", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
