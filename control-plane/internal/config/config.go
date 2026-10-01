package config

import (
	"os"
	"strconv"
)

// Config holds Control Plane configuration.
type Config struct {
	Port int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	port := 8080
	if v := os.Getenv("CONTROL_PLANE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	return Config{Port: port}
}
