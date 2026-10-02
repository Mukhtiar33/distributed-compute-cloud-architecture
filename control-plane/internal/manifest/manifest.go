package manifest

import (
	"encoding/json"
	"fmt"
)

// JobType represents the type of job being submitted.
type JobType string

const (
	JobTypeSingle  JobType = "single"
	JobTypeParallel JobType = "parallel"
)

// Manifest represents the v1 job manifest schema (Document 10 §3).
type Manifest struct {
	JobType       JobType       `json:"job_type"`
	Environment   string        `json:"environment"`
	Entrypoint     string        `json:"entrypoint"`
	Inputs        []string      `json:"inputs,omitempty"`
	ExpectedOutput string        `json:"expected_output"`
}

// ValidationError represents a specific manifest validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks the manifest against the v1 schema and returns a list of validation errors.
func (m *Manifest) Validate() []ValidationError {
	var errors []ValidationError

	// job_type: required, must be "single" or "parallel"
	if m.JobType == "" {
		errors = append(errors, ValidationError{Field: "job_type", Message: "required field missing"})
	} else if m.JobType != JobTypeSingle && m.JobType != JobTypeParallel {
		errors = append(errors, ValidationError{Field: "job_type", Message: fmt.Sprintf("must be 'single' or 'parallel', got '%s'", m.JobType)})
	}

	// environment: required, non-empty string
	if m.Environment == "" {
		errors = append(errors, ValidationError{Field: "environment", Message: "required field missing"})
	}

	// entrypoint: required, non-empty string
	if m.Entrypoint == "" {
		errors = append(errors, ValidationError{Field: "entrypoint", Message: "required field missing"})
	}

	// inputs: required for parallel jobs, must be non-empty list
	if m.JobType == JobTypeParallel {
		if len(m.Inputs) == 0 {
			errors = append(errors, ValidationError{Field: "inputs", Message: "required for parallel jobs: must be a non-empty list"})
		}
	}

	// expected_output: required, non-empty string
	if m.ExpectedOutput == "" {
		errors = append(errors, ValidationError{Field: "expected_output", Message: "required field missing"})
	}

	return errors
}

// ParseManifest parses a JSON manifest and validates it.
func ParseManifest(data []byte) (*Manifest, []ValidationError, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON: %w", err)
	}

	errors := m.Validate()
	return &m, errors, nil
}
