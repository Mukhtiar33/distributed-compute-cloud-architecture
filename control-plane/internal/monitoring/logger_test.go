package monitoring

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestLogger_Log(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, err := NewLogger(tmpFile, false)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	logger.Log(LogLevelInfo, "test-component", "job-1", "worker-1", StageIntake, "test message", map[string]interface{}{
		"key": "value",
	})

	// Read log file
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.Level != LogLevelInfo {
		t.Errorf("expected INFO, got %s", entry.Level)
	}
	if entry.Component != "test-component" {
		t.Errorf("expected test-component, got %s", entry.Component)
	}
	if entry.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", entry.JobID)
	}
	if entry.WorkerID != "worker-1" {
		t.Errorf("expected worker-1, got %s", entry.WorkerID)
	}
	if entry.Stage != StageIntake {
		t.Errorf("expected intake, got %s", entry.Stage)
	}
	if entry.Message != "test message" {
		t.Errorf("expected 'test message', got %s", entry.Message)
	}
}

func TestLogger_LogJobStage(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	logger.LogJobStage("job-1", StageScanPassed, "scan passed", nil)

	data, _ := os.ReadFile(tmpFile)
	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.Stage != StageScanPassed {
		t.Errorf("expected scan_passed, got %s", entry.Stage)
	}
}

func TestLogger_LogWorkerHealth(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	logger.LogWorkerHealth("worker-1", "active", map[string]interface{}{
		"cpu_usage": 50,
	})

	data, _ := os.ReadFile(tmpFile)
	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.WorkerID != "worker-1" {
		t.Errorf("expected worker-1, got %s", entry.WorkerID)
	}
}

func TestLogger_LogCacheHit(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	logger.LogCacheHit("job-1", "python-3.11", "abc123")

	data, _ := os.ReadFile(tmpFile)
	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.Stage != StageExecuting {
		t.Errorf("expected executing, got %s", entry.Stage)
	}
	if entry.Fields["environment_id"] != "python-3.11" {
		t.Errorf("expected python-3.11, got %v", entry.Fields["environment_id"])
	}
}

func TestLogger_LogCacheMiss(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	logger.LogCacheMiss("job-1", "node-20")

	data, _ := os.ReadFile(tmpFile)
	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.Stage != StageExecuting {
		t.Errorf("expected executing, got %s", entry.Stage)
	}
}

func TestLogger_QueryByJobID(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	logger.LogJobStage("job-1", StageIntake, "intake", nil)
	logger.LogJobStage("job-1", StageScanPassed, "scan passed", nil)
	logger.LogJobStage("job-2", StageIntake, "intake", nil)

	entries := logger.QueryByJobID("job-1")
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for job-1, got %d", len(entries))
	}

	entries = logger.QueryByJobID("job-2")
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for job-2, got %d", len(entries))
	}

	entries = logger.QueryByJobID("nonexistent")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for nonexistent job, got %d", len(entries))
	}
}

func TestLogger_Timestamp(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	before := time.Now()
	logger.Log(LogLevelInfo, "test", "", "", "", "test", nil)
	after := time.Now()

	data, _ := os.ReadFile(tmpFile)
	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Error("timestamp should be between before and after")
	}
}

func TestLogger_AllStages(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	stages := []JobStage{
		StageIntake,
		StageManifest,
		StageScan,
		StageScanPassed,
		StageDetonation,
		StageDetonationPass,
		StageScheduling,
		StageExecuting,
		StageValidation,
		StageValidationPass,
		StageDelivered,
	}

	for _, stage := range stages {
		logger.LogJobStage("job-1", stage, string(stage), nil)
	}

	entries := logger.QueryByJobID("job-1")
	if len(entries) != len(stages) {
		t.Errorf("expected %d entries, got %d", len(stages), len(entries))
	}
}

func TestLogger_AllLevels(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}

	for _, level := range levels {
		logger.Log(level, "test", "", "", "", "test", nil)
	}

	data, _ := os.ReadFile(tmpFile)
	lines := splitLines(data)
	if len(lines) != len(levels) {
		t.Errorf("expected %d lines, got %d", len(levels), len(lines))
	}
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Log(LogLevelInfo, "test", "job-1", "", StageIntake, "concurrent", nil)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	entries := logger.QueryByJobID("job-1")
	if len(entries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(entries))
	}
}

func TestLogger_EmptyFields(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)
	defer logger.Close()

	logger.Log(LogLevelInfo, "test", "", "", "", "test", nil)

	data, _ := os.ReadFile(tmpFile)
	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.Fields != nil {
		t.Error("expected nil fields")
	}
}

func TestLogger_Close(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	logger, _ := NewLogger(tmpFile, false)

	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
