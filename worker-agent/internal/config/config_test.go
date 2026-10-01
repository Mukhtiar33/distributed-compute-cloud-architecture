package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultURL(t *testing.T) {
	os.Unsetenv("WORKER_CONTROL_PLANE_URL")
	cfg := Load()
	if cfg.ControlPlaneURL != "https://localhost:8443" {
		t.Errorf("expected default URL https://localhost:8443, got %s", cfg.ControlPlaneURL)
	}
}

func TestLoad_CustomURL(t *testing.T) {
	os.Setenv("WORKER_CONTROL_PLANE_URL", "https://cp.example.com:443")
	defer os.Unsetenv("WORKER_CONTROL_PLANE_URL")
	cfg := Load()
	if cfg.ControlPlaneURL != "https://cp.example.com:443" {
		t.Errorf("expected custom URL, got %s", cfg.ControlPlaneURL)
	}
}
