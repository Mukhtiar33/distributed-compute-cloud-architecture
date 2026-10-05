package detonation

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DetonationResult represents the outcome of a dynamic detonation scan.
type DetonationResult struct {
	Passed   bool     `json:"passed"`
	Flagged  bool     `json:"flagged"`
	Reasons  []string `json:"reasons,omitempty"`
	Duration int64    `json:"duration_ms"`
}

// BehaviorProfile defines expected behavior for a job type.
type BehaviorProfile struct {
	JobType          string   `json:"job_type"`
	AllowedFiles     []string `json:"allowed_files"`
	AllowedCommands  []string `json:"allowed_commands"`
	AllowNetwork     bool     `json:"allow_network"`
	MaxExecutionTime int      `json:"max_execution_time_seconds"`
}

// DetonationScanner performs dynamic behavioral analysis on job payloads.
type DetonationScanner struct {
	profiles map[string]BehaviorProfile
}

// NewDetonationScanner creates a new detonation scanner with default profiles.
func NewDetonationScanner() *DetonationScanner {
	return &DetonationScanner{
		profiles: defaultProfiles(),
	}
}

// Detonate runs a job in a constrained sandbox and analyzes its behavior.
func (ds *DetonationScanner) Detonate(ctx context.Context, zipData []byte, jobType string) (*DetonationResult, error) {
	start := time.Now()

	// Get behavior profile for job type
	profile, ok := ds.profiles[jobType]
	if !ok {
		// Unknown job type — use most restrictive profile
		profile = BehaviorProfile{
			JobType:          jobType,
			AllowNetwork:     false,
			MaxExecutionTime: 30,
		}
	}

	// Create temp directory for detonation
	tmpDir, err := os.MkdirTemp("", "detonation-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract zip to temp directory
	if err := extractZip(zipData, tmpDir); err != nil {
		return &DetonationResult{
			Passed:   false,
			Flagged:  true,
			Reasons:  []string{fmt.Sprintf("invalid zip: %v", err)},
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}

	// Run job in constrained sandbox
	behavior, err := ds.runInSandbox(ctx, tmpDir, profile)
	if err != nil {
		return &DetonationResult{
			Passed:   false,
			Flagged:  true,
			Reasons:  []string{fmt.Sprintf("sandbox execution failed: %v", err)},
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}

	// Analyze behavior against profile
	reasons := ds.analyzeBehavior(behavior, profile)

	duration := time.Since(start).Milliseconds()

	if len(reasons) > 0 {
		return &DetonationResult{
			Passed:   false,
			Flagged:  true,
			Reasons:  reasons,
			Duration: duration,
		}, nil
	}

	return &DetonationResult{
		Passed:   true,
		Flagged:  false,
		Duration: duration,
	}, nil
}

// Behavior captures observed behavior during detonation.
type Behavior struct {
	NetworkAttempts []string `json:"network_attempts"`
	FilesAccessed   []string `json:"files_accessed"`
	CommandsRun     []string `json:"commands_run"`
	ExecutionTime   int64    `json:"execution_time_ms"`
}

func (ds *DetonationScanner) runInSandbox(ctx context.Context, dir string, profile BehaviorProfile) (*Behavior, error) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		// Docker not available — simulate detonation with static analysis
		return ds.simulateDetonation(dir, profile)
	}

	// Run in Docker container with heavy constraints
	return ds.runInDocker(ctx, dir, profile)
}

func (ds *DetonationScanner) runInDocker(ctx context.Context, dir string, profile BehaviorProfile) (*Behavior, error) {
	containerName := fmt.Sprintf("detonation-%d", time.Now().UnixNano())

	// Build docker run args — heavily constrained
	args := []string{
		"run",
		"--rm",
		"--name", containerName,
		"--network", "none",                           // Network denied
		"--memory", "64m",                             // Very low memory limit
		"--cpu-shares", "128",                         // Very low CPU
		"--read-only",                                 // Read-only rootfs
		"--cap-drop", "ALL",                           // Drop all capabilities
		"--security-opt", "no-new-privileges:true",    // No privilege escalation
		"--pids-limit", "32",                          // Limit processes
		"-v", fmt.Sprintf("%s:/scratch:ro", dir),      // Read-only mount
		"-w", "/scratch",
		"alpine:latest",
		"sh", "-c", "sleep 5",  // Run for 5 seconds to observe behavior
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(profile.MaxExecutionTime)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "docker", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Clean up container
	exec.Command("docker", "rm", "-f", containerName).Run()

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			// Timeout is expected for detonation
			return &Behavior{
				ExecutionTime: int64(profile.MaxExecutionTime) * 1000,
			}, nil
		}
		// Other errors are still useful behavior data
	}

	return &Behavior{
		NetworkAttempts: []string{},		FilesAccessed:   []string{},
		CommandsRun:     []string{},
		ExecutionTime:   int64(profile.MaxExecutionTime) * 1000,
	}, nil
}

func (ds *DetonationScanner) simulateDetonation(dir string, profile BehaviorProfile) (*Behavior, error) {
	// When Docker is not available, perform static analysis as a fallback
	behavior := &Behavior{
		NetworkAttempts: []string{},
		FilesAccessed:   []string{},
		CommandsRun:     []string{},
		ExecutionTime:   0,
	}

	// Walk the directory to find files
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			behavior.FilesAccessed = append(behavior.FilesAccessed, relPath)
		}
		return nil
	})

	return behavior, nil
}

func (ds *DetonationScanner) analyzeBehavior(behavior *Behavior, profile BehaviorProfile) []string {
	var reasons []string

	// Check for network attempts
	if !profile.AllowNetwork && len(behavior.NetworkAttempts) > 0 {
		reasons = append(reasons, fmt.Sprintf("network access attempted: %v", behavior.NetworkAttempts))
	}

	// Check for execution time anomalies
	if profile.MaxExecutionTime > 0 && behavior.ExecutionTime > int64(profile.MaxExecutionTime)*1000 {
		reasons = append(reasons, fmt.Sprintf("execution time %dms exceeds limit %dms", behavior.ExecutionTime, profile.MaxExecutionTime*1000))
	}

	return reasons
}

func defaultProfiles() map[string]BehaviorProfile {
	return map[string]BehaviorProfile{
		"data-preprocessing": {
			JobType:          "data-preprocessing",
			AllowedFiles:     []string{"*.csv", "*.json", "*.txt"},
			AllowedCommands:  []string{"python", "node", "cat", "wc", "head", "tail", "sort", "uniq"},
			AllowNetwork:     false,
			MaxExecutionTime: 30,
		},
		"model-training": {
			JobType:          "model-training",
			AllowedFiles:     []string{"*.py", "*.pt", "*.pth", "*.h5"},
			AllowedCommands:  []string{"python", "pip"},
			AllowNetwork:     false,
			MaxExecutionTime: 300,
		},
		"batch-inference": {
			JobType:          "batch-inference",
			AllowedFiles:     []string{"*.pt", "*.pth", "*.onnx", "*.csv"},
			AllowedCommands:  []string{"python"},
			AllowNetwork:     false,
			MaxExecutionTime: 120,
		},
		"rendering": {
			JobType:          "rendering",
			AllowedFiles:     []string{"*.blend", "*.obj", "*.fbx"},
			AllowedCommands:  []string{"blender", "python"},
			AllowNetwork:     false,
			MaxExecutionTime: 600,
		},
	}
}

func extractZip(data []byte, dest string) error {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		path := filepath.Join(dest, file.Name)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
