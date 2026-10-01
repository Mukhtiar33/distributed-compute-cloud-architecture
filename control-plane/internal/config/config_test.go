package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultPort(t *testing.T) {
	os.Unsetenv("CONTROL_PLANE_PORT")
	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("CONTROL_PLANE_PORT", "9090")
	defer os.Unsetenv("CONTROL_PLANE_PORT")
	cfg := Load()
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Setenv("CONTROL_PLANE_PORT", "not-a-number")
	defer os.Unsetenv("CONTROL_PLANE_PORT")
	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected fallback to default port 8080, got %d", cfg.Port)
	}
}
