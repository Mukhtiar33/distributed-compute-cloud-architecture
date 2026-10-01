package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/distributedcompute/cloud/control-plane/internal/config"
)

func main() {
	cfg := config.Load()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy","component":"control-plane"}`)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Control Plane starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Control Plane failed: %v", err)
		os.Exit(1)
	}
}
