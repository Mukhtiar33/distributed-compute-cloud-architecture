package monitoring

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// JobStage represents a stage in the job lifecycle.
type JobStage string

const (
	StageIntake         JobStage = "intake"
	StageManifest       JobStage = "manifest_validated"
	StageScan           JobStage = "scan"
	StageScanPassed     JobStage = "scan_passed"
	StageScanRejected   JobStage = "scan_rejected"
	StageDetonation     JobStage = "detonation"
	StageDetonationPass JobStage = "detonation_passed"
	StageDetonationFlag JobStage = "detonation_flagged"
	StageScheduling     JobStage = "scheduling"
	StageExecuting      JobStage = "executing"
	StageValidation     JobStage = "validation"
	StageValidationPass JobStage = "validation_passed"
	StageValidationFail JobStage = "validation_failed"
	StageDelivered      JobStage = "delivered"
	StageFailed         JobStage = "failed"
)

// LogEntry represents a structured log entry (Document 15 §3).
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Component   string                 `json:"component"`
	JobID       string                 `json:"job_id,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
	Stage       JobStage               `json:"stage,omitempty"`
	Message     string                 `json:"message"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// Logger writes structured log entries.
type Logger struct {
	mu     sync.Mutex
	file   *os.File
	stdout bool
}

// NewLogger creates a new structured logger.
func NewLogger(filePath string, stdout bool) (*Logger, error) {
	var file *os.File
	var err error

	if filePath != "" {
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	return &Logger{
		file:   file,
		stdout: stdout,
	}, nil
}

// Close closes the log file.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Log writes a structured log entry.
func (l *Logger) Log(level LogLevel, component, jobID, workerID string, stage JobStage, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Component: component,
		JobID:     jobID,
		WorkerID:  workerID,
		Stage:     stage,
		Message:   message,
		Fields:    fields,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.stdout {
		fmt.Println(string(data))
	}

	if l.file != nil {
		l.file.Write(data)
		l.file.Write([]byte("\n"))
	}
}

// LogJobStage logs a job lifecycle stage transition.
func (l *Logger) LogJobStage(jobID string, stage JobStage, message string, fields map[string]interface{}) {
	l.Log(LogLevelInfo, "control-plane", jobID, "", stage, message, fields)
}

// LogWorkerHealth logs a worker health signal.
func (l *Logger) LogWorkerHealth(workerID, status string, fields map[string]interface{}) {
	l.Log(LogLevelInfo, "worker-agent", "", workerID, "", fmt.Sprintf("worker health: %s", status), fields)
}

// LogCacheHit logs an environment cache hit.
func (l *Logger) LogCacheHit(jobID, environmentID, hash string) {
	l.Log(LogLevelInfo, "control-plane", jobID, "", StageExecuting, "cache hit", map[string]interface{}{
		"environment_id": environmentID,
		"hash":           hash,
	})
}

// LogCacheMiss logs an environment cache miss.
func (l *Logger) LogCacheMiss(jobID, environmentID string) {
	l.Log(LogLevelInfo, "control-plane", jobID, "", StageExecuting, "cache miss", map[string]interface{}{
		"environment_id": environmentID,
	})
}

// QueryByJobID returns all log entries for a given job ID.
func (l *Logger) QueryByJobID(jobID string) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var entries []LogEntry

	if l.file == nil {
		return entries
	}

	// Read all lines from the file
	data, err := os.ReadFile(l.file.Name())
	if err != nil {
		return entries
	}

	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.JobID == jobID {
			entries = append(entries, entry)
		}
	}

	return entries
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
