package config

import (
	"os"
)

// Config holds Worker Agent configuration.
type Config struct {
	ControlPlaneURL string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	url := "https://localhost:8443"
	if v := os.Getenv("WORKER_CONTROL_PLANE_URL"); v != "" {
		url = v
	}
	return Config{ControlPlaneURL: url}
}
