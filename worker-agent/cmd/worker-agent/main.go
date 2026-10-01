package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/distributedcompute/cloud/worker-agent/internal/agent"
	"github.com/distributedcompute/cloud/worker-agent/internal/config"
)

func main() {
	cfg := config.Load()

	log.Printf("Worker Agent starting (control-plane: %s)", cfg.ControlPlaneURL)

	a := agent.New(cfg)
	if err := a.Start(); err != nil {
		log.Fatalf("Worker Agent failed to start: %v", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down Worker Agent...")
	a.Stop()
}
