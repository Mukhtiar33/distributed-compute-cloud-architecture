package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SandboxConfig configures a sandboxed execution environment.
type SandboxConfig struct {
	Image       string            // Docker image to use
	Command     []string          // Command to execute
	Env         []string          // Environment variables
	WorkingDir  string            // Working directory inside container
	NetworkMode string            // Network mode (default: "none")
	MemoryBytes int64             // Memory limit in bytes
	CPUShares   int64             // CPU shares
	Timeout     time.Duration     // Execution timeout
	PIDsLimit   int64             // Maximum number of processes
}

// SandboxResult contains the output of a sandboxed execution.
type SandboxResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Success  bool   `json:"success"`
}

// SandboxManager manages Docker-based sandboxed execution via Docker CLI.
type SandboxManager struct {
	available bool
}

// NewSandboxManager creates a new sandbox manager.
func NewSandboxManager() (*SandboxManager, error) {
	_, err := exec.LookPath("docker")
	if err != nil {
		return &SandboxManager{available: false}, nil
	}

	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return &SandboxManager{available: false}, nil
	}

	return &SandboxManager{available: true}, nil
}

// IsAvailable checks if Docker is available on this system.
func (sm *SandboxManager) IsAvailable() bool {
	return sm.available
}

// Close releases any resources held by the manager.
func (sm *SandboxManager) Close() error {
	return nil
}

// Execute runs a job inside a sandboxed Docker container.
func (sm *SandboxManager) Execute(ctx context.Context, config SandboxConfig, scratchDir string) (*SandboxResult, error) {
	if !sm.available {
		return nil, fmt.Errorf("Docker not available")
	}

	if config.NetworkMode == "" {
		config.NetworkMode = "none"
	}

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create scratch dir: %w", err)
	}

	containerName := fmt.Sprintf("dcc-sandbox-%d", time.Now().UnixNano())

	args := []string{
		"run",
		"--rm",
		"--name", containerName,
		"--label", "dcc.managed=true",
		"--label", fmt.Sprintf("dcc.created=%d", time.Now().Unix()),
		"--network", config.NetworkMode,
		"--memory", fmt.Sprintf("%dm", config.MemoryBytes/(1024*1024)),
		"--cpu-shares", fmt.Sprintf("%d", config.CPUShares),
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges:true",
		"--pids-limit", fmt.Sprintf("%d", config.PIDsLimit),
		"-v", fmt.Sprintf("%s:/scratch", scratchDir),
		"-w", config.WorkingDir,
	}

	for _, env := range config.Env {
		args = append(args, "-e", env)
	}

	args = append(args, config.Image)
	args = append(args, config.Command...)

	execCtx := ctx
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, "docker", args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Clean up container (in case it wasn't removed due to timeout)
	exec.Command("docker", "rm", "-f", containerName).Run()

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("container execution timed out")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &SandboxResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitErr.ExitCode(),
				Success:  false,
			}, nil
		}
		return nil, fmt.Errorf("docker execution failed: %w", err)
	}

	return &SandboxResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Success:  true,
	}, nil
}

// EnsureImage pulls a Docker image if not already present.
func (sm *SandboxManager) EnsureImage(ctx context.Context, image string) error {
	if !sm.available {
		return fmt.Errorf("Docker not available")
	}

	cmd := exec.CommandContext(ctx, "docker", "images", "-q", image)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check image: %w", err)
	}

	if len(output) > 0 {
		return nil
	}

	cmd = exec.CommandContext(ctx, "docker", "pull", image)
	return cmd.Run()
}

// Cleanup removes only DCC-managed containers (labeled with dcc.managed=true).
func (sm *SandboxManager) Cleanup(ctx context.Context) error {
	if !sm.available {
		return nil
	}

	// Only remove containers created by this platform
	cmd := exec.CommandContext(ctx, "docker", "container", "prune", "-f", "--filter", "label=dcc.managed=true")
	return cmd.Run()
}

// GetScratchDir returns a unique scratch directory path for a job.
func GetScratchDir(baseDir, jobID string) string {
	return filepath.Join(baseDir, jobID)
}
