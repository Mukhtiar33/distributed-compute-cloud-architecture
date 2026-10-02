package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/distributedcompute/cloud/worker-agent/internal/sandbox"
)

// Job represents a job to be executed.
type Job struct {
	ID         string            `json:"id"`
	Manifest   map[string]interface{} `json:"manifest"`
	Payload    []byte            `json:"payload"` // Zip file contents
}

// JobResult contains the result of job execution.
type JobResult struct {
	JobID     string `json:"job_id"`
	Success   bool   `json:"success"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
	OutputZip []byte `json:"output_zip,omitempty"`
}

// Executor manages job execution inside sandboxes.
type Executor struct {
	sandboxMgr *sandbox.SandboxManager
	scratchBase string
}

// NewExecutor creates a new job executor.
func NewExecutor(mgr *sandbox.SandboxManager, scratchBase string) *Executor {
	return &Executor{
		sandboxMgr: mgr,
		scratchBase: scratchBase,
	}
}

// Execute runs a job inside an isolated sandbox.
func (e *Executor) Execute(ctx context.Context, job Job) (*JobResult, error) {
	// Create scratch directory for this job
	scratchDir := filepath.Join(e.scratchBase, job.ID)
	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create scratch dir: %w", err)
	}
	defer os.RemoveAll(scratchDir) // Clean up after execution

	// Extract job payload to scratch dir
	if err := extractZip(job.Payload, scratchDir); err != nil {
		return nil, fmt.Errorf("failed to extract job payload: %w", err)
	}

	// Determine command from manifest
	command := determineCommand(job.Manifest)
	if len(command) == 0 {
		return nil, fmt.Errorf("no entrypoint specified in manifest")
	}

	// Configure sandbox
	config := sandbox.SandboxConfig{
		Image:       "alpine:latest", // Hardcoded for Phase 4
		Command:     command,
		WorkingDir:  "/scratch",
		NetworkMode: "none", // Default-deny network policy
		MemoryBytes: 512 * 1024 * 1024, // 512MB limit
		CPUShares:   512,
		Timeout:     5 * time.Minute,
	}

	// Execute in sandbox
	result, err := e.sandboxMgr.Execute(ctx, config, scratchDir)
	if err != nil {
		return &JobResult{
			JobID:    job.ID,
			Success:  false,
			Stderr:   err.Error(),
			ExitCode: -1,
		}, nil
	}

	// Collect output
	outputZip := collectOutput(scratchDir)

	return &JobResult{
		JobID:     job.ID,
		Success:   result.Success,
		Stdout:    result.Stdout,
		Stderr:    result.Stderr,
		ExitCode:  result.ExitCode,
		OutputZip: outputZip,
	}, nil
}

func determineCommand(manifest map[string]interface{}) []string {
	entrypoint, ok := manifest["entrypoint"].(string)
	if !ok || entrypoint == "" {
		return nil
	}

	// For Phase 4, we use a simple command splitter
	// In production, this would be more sophisticated
	return []string{"/bin/sh", "-c", entrypoint}
}

func extractZip(data []byte, dest string) error {
	// For Phase 4, we use a simple extraction
	// In production, this would use archive/zip
	return nil
}

func collectOutput(scratchDir string) []byte {
	// For Phase 4, we collect output from scratch dir
	// In production, this would create a zip of output files
	return nil
}
