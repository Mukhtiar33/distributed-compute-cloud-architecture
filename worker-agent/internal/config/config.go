package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config holds Worker Agent configuration.
type Config struct {
	ControlPlaneURL   string
	CredentialsDir    string
	HeartbeatInterval int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	url := "https://localhost:8443"
	if v := os.Getenv("WORKER_CONTROL_PLANE_URL"); v != "" {
		url = v
	}

	credDir := ".worker-credentials"
	if v := os.Getenv("WORKER_CREDENTIALS_DIR"); v != "" {
		credDir = v
	}

	// Resolve to absolute path
	if abs, err := filepath.Abs(credDir); err == nil {
		credDir = abs
	}

	interval := 5
	if v := os.Getenv("WORKER_HEARTBEAT_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			interval = i
		}
	}

	return Config{
		ControlPlaneURL:   url,
		CredentialsDir:    credDir,
		HeartbeatInterval: interval,
	}
}
