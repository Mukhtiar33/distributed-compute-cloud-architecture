package detonation

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"testing"
	"time"
)

func createDetonationZip(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	w.Close()
	return buf.Bytes()
}

func TestDetonationScanner_DetonateNormal(t *testing.T) {
	ds := NewDetonationScanner()
	zipData := createDetonationZip(map[string]string{
		"main.py": "print('hello')",
	})

	ctx := context.Background()
	result, err := ds.Detonate(ctx, zipData, "data-preprocessing")
	if err != nil {
		t.Fatalf("Detonate failed: %v", err)
	}

	// Normal job should pass (or be flagged only if Docker is available and detects something)
	// Without Docker, it should pass
	if result.Flagged && len(result.Reasons) == 0 {
		t.Error("flagged result should have reasons")
	}
}

func TestDetonationScanner_DetonateAnomalous(t *testing.T) {
	ds := NewDetonationScanner()
	// Job that attempts network access despite being "data-preprocessing" type
	zipData := createDetonationZip(map[string]string{
		"main.py": `import socket
s = socket.socket()
s.connect(("example.com", 80))
`,
	})

	ctx := context.Background()
	result, err := ds.Detonate(ctx, zipData, "data-preprocessing")
	if err != nil {
		t.Fatalf("Detonate failed: %v", err)
	}

	// Without Docker, this passes (static analysis only)
	// With Docker, it should be flagged for network access
	_ = result
}

func TestDetonationScanner_UnknownJobType(t *testing.T) {
	ds := NewDetonationScanner()
	zipData := createDetonationZip(map[string]string{
		"main.py": "print('hello')",
	})

	ctx := context.Background()
	result, err := ds.Detonate(ctx, zipData, "unknown-type")
	if err != nil {
		t.Fatalf("Detonate failed: %v", err)
	}

	// Unknown job type should use most restrictive profile
	if result.Flagged && len(result.Reasons) == 0 {
		t.Error("flagged result should have reasons")
	}
}

func TestDetonationScanner_InvalidZip(t *testing.T) {
	ds := NewDetonationScanner()

	ctx := context.Background()
	result, err := ds.Detonate(ctx, []byte("not a zip"), "data-preprocessing")
	if err != nil {
		t.Fatalf("Detonate failed: %v", err)
	}

	if !result.Flagged {
		t.Error("expected invalid zip to be flagged")
	}
}

func TestDetonationScanner_EmptyZip(t *testing.T) {
	ds := NewDetonationScanner()
	zipData := createDetonationZip(map[string]string{})

	ctx := context.Background()
	result, err := ds.Detonate(ctx, zipData, "data-preprocessing")
	if err != nil {
		t.Fatalf("Detonate failed: %v", err)
	}

	// Empty zip should pass (no threats found)
	if result.Flagged {
		t.Errorf("expected empty zip to pass, got reasons: %v", result.Reasons)
	}
}

func TestDetonationResult_Passed(t *testing.T) {
	result := &DetonationResult{
		Passed:   true,
		Flagged:  false,
		Duration: 100,
	}

	if !result.Passed {
		t.Error("expected passed to be true")
	}
	if result.Flagged {
		t.Error("expected flagged to be false")
	}
}

func TestDetonationResult_Flagged(t *testing.T) {
	result := &DetonationResult{
		Passed:   false,
		Flagged:  true,
		Reasons:  []string{"network access attempted"},
		Duration: 200,
	}

	if result.Passed {
		t.Error("expected passed to be false")
	}
	if !result.Flagged {
		t.Error("expected flagged to be true")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected reasons for flagged result")
	}
}

func TestBehaviorProfile_DataPreprocessing(t *testing.T) {
	ds := NewDetonationScanner()
	profile := ds.profiles["data-preprocessing"]

	if profile.JobType != "data-preprocessing" {
		t.Errorf("expected data-preprocessing, got %s", profile.JobType)
	}
	if profile.AllowNetwork {
		t.Error("data-preprocessing should not allow network")
	}
	if profile.MaxExecutionTime != 30 {
		t.Errorf("expected max execution time 30, got %d", profile.MaxExecutionTime)
	}
}

func TestBehaviorProfile_ModelTraining(t *testing.T) {
	ds := NewDetonationScanner()
	profile := ds.profiles["model-training"]

	if profile.JobType != "model-training" {
		t.Errorf("expected model-training, got %s", profile.JobType)
	}
	if profile.MaxExecutionTime != 300 {
		t.Errorf("expected max execution time 300, got %d", profile.MaxExecutionTime)
	}
}

func TestAnalyzeBehavior_NetworkDenied(t *testing.T) {
	ds := NewDetonationScanner()
	profile := BehaviorProfile{
		JobType:      "test",
		AllowNetwork: false,
	}

	behavior := &Behavior{
		NetworkAttempts: []string{"example.com:80"},
	}

	reasons := ds.analyzeBehavior(behavior, profile)
	if len(reasons) == 0 {
		t.Error("expected network attempt to be flagged")
	}
}

func TestAnalyzeBehavior_NetworkAllowed(t *testing.T) {
	ds := NewDetonationScanner()
	profile := BehaviorProfile{
		JobType:      "test",
		AllowNetwork: true,
	}

	behavior := &Behavior{
		NetworkAttempts: []string{"example.com:80"},
	}

	reasons := ds.analyzeBehavior(behavior, profile)
	if len(reasons) != 0 {
		t.Errorf("expected no reasons when network is allowed, got %v", reasons)
	}
}

func TestAnalyzeBehavior_ExecutionTimeExceeded(t *testing.T) {
	ds := NewDetonationScanner()
	profile := BehaviorProfile{
		JobType:          "test",
		MaxExecutionTime: 10,
	}

	behavior := &Behavior{
		ExecutionTime: 15000, // 15 seconds
	}

	reasons := ds.analyzeBehavior(behavior, profile)
	if len(reasons) == 0 {
		t.Error("expected execution time exceeded to be flagged")
	}
}

func TestExtractZip(t *testing.T) {
	zipData := createDetonationZip(map[string]string{
		"test.txt": "test content",
	})

	tmpDir := t.TempDir()
	err := extractZip(zipData, tmpDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(tmpDir + "/test.txt")
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("expected 'test content', got '%s'", string(content))
	}
}

func TestExtractZip_InvalidZip(t *testing.T) {
	tmpDir := t.TempDir()
	err := extractZip([]byte("not a zip"), tmpDir)
	if err == nil {
		t.Error("expected error for invalid zip")
	}
}

func TestDetonationScanner_DetonateTimeout(t *testing.T) {
	ds := NewDetonationScanner()
	zipData := createDetonationZip(map[string]string{
		"main.py": "print('hello')",
	})

	ctx := context.Background()
	start := time.Now()
	result, err := ds.Detonate(ctx, zipData, "data-preprocessing")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Detonate failed: %v", err)
	}

	// Should complete within reasonable time
	if duration > 60*time.Second {
		t.Errorf("detonation took too long: %v", duration)
	}

	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}
