package config

import (
	"os"
	"strconv"
)

// Config holds Control Plane configuration.
type Config struct {
	Port        int
	BindAddress string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	port := 8443
	if v := os.Getenv("CONTROL_PLANE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	bindAddr := ":" + strconv.Itoa(port)
	if v := os.Getenv("CONTROL_PLANE_BIND"); v != "" {
		bindAddr = v
	}

	return Config{
		Port:        port,
		BindAddress: bindAddr,
	}
}
