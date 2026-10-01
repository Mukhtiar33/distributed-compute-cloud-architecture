package agent

import (
	"testing"

	"github.com/distributedcompute/cloud/worker-agent/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.Config{ControlPlaneURL: "https://localhost:8443"}
	a := New(cfg)
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if a.IsRunning() {
		t.Error("new agent should not be running")
	}
}

func TestStartStop(t *testing.T) {
	cfg := config.Config{ControlPlaneURL: "https://localhost:8443"}
	a := New(cfg)

	if err := a.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !a.IsRunning() {
		t.Error("agent should be running after Start")
	}

	a.Stop()
	if a.IsRunning() {
		t.Error("agent should not be running after Stop")
	}
}
