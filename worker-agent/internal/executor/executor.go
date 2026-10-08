package executor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/distributedcompute/cloud/worker-agent/internal/sandbox"
)

const (
	// MaxFileSize is the maximum size of a single decompressed file (25MB)
	MaxFileSize = 25 * 1024 * 1024
	// MaxTotalSize is the maximum total decompressed size (100MB)
	MaxTotalSize = 100 * 1024 * 1024
	// MaxCompressionRatio is the maximum allowed compression ratio
	MaxCompressionRatio = 100
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

	// Extract job payload to scratch dir with safety checks
	if err := extractZipSafe(job.Payload, scratchDir); err != nil {
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

// extractZipSafe extracts a zip file with path traversal and zip bomb protection.
func extractZipSafe(data []byte, dest string) error {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	var totalUncompressed int64

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Check for zip bomb
		totalUncompressed += int64(file.UncompressedSize64)
		if totalUncompressed > MaxTotalSize {
			return fmt.Errorf("zip bomb: total uncompressed size %d exceeds limit %d", totalUncompressed, MaxTotalSize)
		}

		// Check compression ratio
		if file.CompressedSize64 > 0 {
			ratio := float64(file.UncompressedSize64) / float64(file.CompressedSize64)
			if ratio > MaxCompressionRatio {
				return fmt.Errorf("zip bomb: compression ratio %.1fx exceeds limit %dx", ratio, MaxCompressionRatio)
			}
		}

		// Check for path traversal
		path := filepath.Join(dest, file.Name)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("path traversal detected: %s", file.Name)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		// Limit reader to prevent zip bombs
		limitedReader := io.LimitReader(rc, MaxFileSize)
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, limitedReader)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func collectOutput(scratchDir string) []byte {
	// Create a zip of all files in scratch dir
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	filepath.Walk(scratchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(scratchDir, path)
		f, _ := w.Create(relPath)
		data, _ := os.ReadFile(path)
		f.Write(data)
		return nil
	})

	w.Close()
	return buf.Bytes()
}
