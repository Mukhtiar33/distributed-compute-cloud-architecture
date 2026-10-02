package sandbox

import (
	"context"
	"testing"
	"time"
)

func TestSandboxConfig_Defaults(t *testing.T) {
	config := SandboxConfig{
		Image:   "alpine:latest",
		Command: []string{"echo", "hello"},
	}

	if config.NetworkMode != "" {
		t.Error("network mode should be empty by default (set during Execute)")
	}
}

func TestSandboxResult_Success(t *testing.T) {
	result := &SandboxResult{
		Stdout:   "hello\n",
		ExitCode: 0,
		Success:  true,
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestSandboxResult_Failure(t *testing.T) {
	result := &SandboxResult{
		Stdout:   "",
		Stderr:   "error\n",
		ExitCode: 1,
		Success:  false,
	}

	if result.Success {
		t.Error("expected success to be false")
	}
}

func TestGetScratchDir(t *testing.T) {
	dir := GetScratchDir("/tmp/scratch", "job-123")
	expected := "/tmp/scratch/job-123"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestSandboxManager_NotAvailable(t *testing.T) {
	sm, err := NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sm.Close()

	available := sm.IsAvailable()
	_ = available
}

func TestSandboxManager_ExecuteTimeout(t *testing.T) {
	sm, err := NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sm.Close()

	if !sm.IsAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	config := SandboxConfig{
		Image:   "alpine:latest",
		Command: []string{"sleep", "10"},
		Timeout: 1 * time.Second,
	}

	_, err = sm.Execute(ctx, config, t.TempDir())
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestSandboxManager_ExecuteEcho(t *testing.T) {
	sm, err := NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sm.Close()

	if !sm.IsAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()

	if err := sm.EnsureImage(ctx, "alpine:latest"); err != nil {
		t.Skipf("Failed to pull image: %v", err)
	}

	config := SandboxConfig{
		Image:   "alpine:latest",
		Command: []string{"echo", "hello sandbox"},
		Timeout: 30 * time.Second,
	}

	result, err := sm.Execute(ctx, config, t.TempDir())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
}

func TestSandboxManager_NetworkDeny(t *testing.T) {
	sm, err := NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sm.Close()

	if !sm.IsAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()

	if err := sm.EnsureImage(ctx, "alpine:latest"); err != nil {
		t.Skipf("Failed to pull image: %v", err)
	}

	config := SandboxConfig{
		Image:       "alpine:latest",
		Command:     []string{"wget", "-qO-", "http://example.com"},
		NetworkMode: "none",
		Timeout:     10 * time.Second,
	}

	result, err := sm.Execute(ctx, config, t.TempDir())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("expected network request to fail with default-deny policy")
	}
}

func TestSandboxManager_FilesystemIsolation(t *testing.T) {
	sm, err := NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sm.Close()

	if !sm.IsAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()

	if err := sm.EnsureImage(ctx, "alpine:latest"); err != nil {
		t.Skipf("Failed to pull image: %v", err)
	}

	// Try to write to /etc/ which should fail with read-only rootfs
	config := SandboxConfig{
		Image:   "alpine:latest",
		Command: []string{"sh", "-c", "echo test > /etc/testfile"},
		Timeout: 10 * time.Second,
	}

	result, err := sm.Execute(ctx, config, t.TempDir())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// The write should fail because rootfs is read-only
	if result.Success {
		t.Error("expected filesystem isolation to prevent writing to /etc/")
	}
}
