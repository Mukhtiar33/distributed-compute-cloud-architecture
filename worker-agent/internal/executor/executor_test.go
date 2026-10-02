package executor

import (
	"context"
	"testing"

	"github.com/distributedcompute/cloud/worker-agent/internal/sandbox"
)

func TestExecutor_Execute(t *testing.T) {
	mgr, err := sandbox.NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer mgr.Close()

	if !mgr.IsAvailable() {
		t.Skip("Docker daemon not available")
	}

	executor := NewExecutor(mgr, t.TempDir())

	job := Job{
		ID: "test-job-1",
		Manifest: map[string]interface{}{
			"entrypoint": "echo hello",
		},
		Payload: []byte("fake zip"),
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, job)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got: %s", result.Stderr)
	}
}

func TestExecutor_NoEntrypoint(t *testing.T) {
	mgr, err := sandbox.NewSandboxManager()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer mgr.Close()

	executor := NewExecutor(mgr, t.TempDir())

	job := Job{
		ID:       "test-job-2",
		Manifest: map[string]interface{}{},
		Payload:  []byte("fake zip"),
	}

	ctx := context.Background()
	_, err = executor.Execute(ctx, job)
	if err == nil {
		t.Error("expected error for missing entrypoint")
	}
}

func TestDetermineCommand(t *testing.T) {
	tests := []struct {
		name     string
		manifest map[string]interface{}
		want     int
	}{
		{
			name:     "valid entrypoint",
			manifest: map[string]interface{}{"entrypoint": "echo hello"},
			want:     3,
		},
		{
			name:     "missing entrypoint",
			manifest: map[string]interface{}{},
			want:     0,
		},
		{
			name:     "empty entrypoint",
			manifest: map[string]interface{}{"entrypoint": ""},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineCommand(tt.manifest)
			if len(got) != tt.want {
				t.Errorf("determineCommand() = %v, want %d elements", got, tt.want)
			}
		})
	}
}
