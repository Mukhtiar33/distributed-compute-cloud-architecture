package api

import (
	"encoding/json"
	"net/http"
)

// RegisterRoutes sets up all Control Plane API routes.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", healthHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"component": "control-plane",
	})
}
