package resources

import (
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("expected non-nil monitor")
	}
	if !m.IsAvailable() {
		t.Error("expected monitor to be available initially")
	}
}

func TestMonitor_CheckResources(t *testing.T) {
	m := NewMonitor()
	m.CheckResources()

	if !m.IsAvailable() {
		t.Error("expected monitor to be available after check")
	}
}

func TestMonitor_ShouldThrottle(t *testing.T) {
	m := NewMonitor()

	if m.ShouldThrottle() {
		t.Error("expected monitor not to throttle initially")
	}

	m.mu.Lock()
	m.available = false
	m.mu.Unlock()

	if !m.ShouldThrottle() {
		t.Error("expected monitor to throttle when unavailable")
	}
}

func TestMonitor_GetStats(t *testing.T) {
	m := NewMonitor()
	stats := m.GetStats()

	if stats["available"] != true {
		t.Error("expected available to be true")
	}
	if stats["goroutines"] == nil {
		t.Error("expected goroutines to be set")
	}
}

func TestMonitor_LastCheck(t *testing.T) {
	m := NewMonitor()
	before := time.Now()
	m.CheckResources()
	after := time.Now()

	m.mu.RLock()
	lastCheck := m.lastCheck
	m.mu.RUnlock()

	if lastCheck.Before(before) || lastCheck.After(after) {
		t.Error("lastCheck should be between before and after")
	}
}
