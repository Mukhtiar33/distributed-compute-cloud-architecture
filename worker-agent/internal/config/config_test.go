package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("WORKER_CONTROL_PLANE_URL")
	os.Unsetenv("WORKER_CREDENTIALS_DIR")
	os.Unsetenv("WORKER_HEARTBEAT_INTERVAL")

	cfg := Load()
	if cfg.ControlPlaneURL != "https://localhost:8443" {
		t.Errorf("expected default URL, got %s", cfg.ControlPlaneURL)
	}
	if cfg.HeartbeatInterval != 5 {
		t.Errorf("expected default interval 5, got %d", cfg.HeartbeatInterval)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Setenv("WORKER_CONTROL_PLANE_URL", "https://cp.example.com:443")
	os.Setenv("WORKER_CREDENTIALS_DIR", "/tmp/creds")
	os.Setenv("WORKER_HEARTBEAT_INTERVAL", "10")
	defer os.Unsetenv("WORKER_CONTROL_PLANE_URL")
	defer os.Unsetenv("WORKER_CREDENTIALS_DIR")
	defer os.Unsetenv("WORKER_HEARTBEAT_INTERVAL")

	cfg := Load()
	if cfg.ControlPlaneURL != "https://cp.example.com:443" {
		t.Errorf("expected custom URL, got %s", cfg.ControlPlaneURL)
	}
	if cfg.CredentialsDir != "/tmp/creds" {
		t.Errorf("expected /tmp/creds, got %s", cfg.CredentialsDir)
	}
	if cfg.HeartbeatInterval != 10 {
		t.Errorf("expected interval 10, got %d", cfg.HeartbeatInterval)
	}
}
