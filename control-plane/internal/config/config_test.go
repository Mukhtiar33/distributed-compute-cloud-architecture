package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultPort(t *testing.T) {
	os.Unsetenv("CONTROL_PLANE_PORT")
	os.Unsetenv("CONTROL_PLANE_BIND")

	cfg := Load()
	if cfg.Port != 8443 {
		t.Errorf("expected default port 8443, got %d", cfg.Port)
	}
	if cfg.BindAddress != ":8443" {
		t.Errorf("expected :8443, got %s", cfg.BindAddress)
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

func TestLoad_CustomBind(t *testing.T) {
	os.Setenv("CONTROL_PLANE_BIND", "0.0.0.0:8443")
	defer os.Unsetenv("CONTROL_PLANE_BIND")

	cfg := Load()
	if cfg.BindAddress != "0.0.0.0:8443" {
		t.Errorf("expected 0.0.0.0:8443, got %s", cfg.BindAddress)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Setenv("CONTROL_PLANE_PORT", "not-a-number")
	defer os.Unsetenv("CONTROL_PLANE_PORT")

	cfg := Load()
	if cfg.Port != 8443 {
		t.Errorf("expected fallback to default port 8443, got %d", cfg.Port)
	}
}
