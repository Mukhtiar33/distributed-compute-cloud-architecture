package manifest

import (
	"encoding/json"
	"testing"
)

func TestParseManifest_ValidSingle(t *testing.T) {
	data := []byte(`{
		"job_type": "single",
		"environment": "python-3.11",
		"entrypoint": "main.py",
		"expected_output": "result.zip"
	}`)

	m, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) > 0 {
		t.Fatalf("expected no validation errors, got %d: %v", len(errors), errors)
	}
	if m.JobType != JobTypeSingle {
		t.Errorf("expected job_type 'single', got '%s'", m.JobType)
	}
	if m.Environment != "python-3.11" {
		t.Errorf("expected environment 'python-3.11', got '%s'", m.Environment)
	}
}

func TestParseManifest_ValidParallel(t *testing.T) {
	data := []byte(`{
		"job_type": "parallel",
		"environment": "python-3.11",
		"entrypoint": "process.py",
		"inputs": ["input1.csv", "input2.csv"],
		"expected_output": "results.zip"
	}`)

	m, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) > 0 {
		t.Fatalf("expected no validation errors, got %d: %v", len(errors), errors)
	}
	if m.JobType != JobTypeParallel {
		t.Errorf("expected job_type 'parallel', got '%s'", m.JobType)
	}
	if len(m.Inputs) != 2 {
		t.Errorf("expected 2 inputs, got %d", len(m.Inputs))
	}
}

func TestParseManifest_MissingJobType(t *testing.T) {
	data := []byte(`{
		"environment": "python-3.11",
		"entrypoint": "main.py",
		"expected_output": "result.zip"
	}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("expected validation errors for missing job_type")
	}
	if errors[0].Field != "job_type" {
		t.Errorf("expected error for job_type, got %s", errors[0].Field)
	}
}

func TestParseManifest_MissingEnvironment(t *testing.T) {
	data := []byte(`{
		"job_type": "single",
		"entrypoint": "main.py",
		"expected_output": "result.zip"
	}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("expected validation errors for missing environment")
	}
	if errors[0].Field != "environment" {
		t.Errorf("expected error for environment, got %s", errors[0].Field)
	}
}

func TestParseManifest_MissingEntrypoint(t *testing.T) {
	data := []byte(`{
		"job_type": "single",
		"environment": "python-3.11",
		"expected_output": "result.zip"
	}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("expected validation errors for missing entrypoint")
	}
	if errors[0].Field != "entrypoint" {
		t.Errorf("expected error for entrypoint, got %s", errors[0].Field)
	}
}

func TestParseManifest_MissingExpectedOutput(t *testing.T) {
	data := []byte(`{
		"job_type": "single",
		"environment": "python-3.11",
		"entrypoint": "main.py"
	}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("expected validation errors for missing expected_output")
	}
	if errors[0].Field != "expected_output" {
		t.Errorf("expected error for expected_output, got %s", errors[0].Field)
	}
}

func TestParseManifest_ParallelMissingInputs(t *testing.T) {
	data := []byte(`{
		"job_type": "parallel",
		"environment": "python-3.11",
		"entrypoint": "process.py",
		"expected_output": "results.zip"
	}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("expected validation errors for parallel job missing inputs")
	}
	if errors[0].Field != "inputs" {
		t.Errorf("expected error for inputs, got %s", errors[0].Field)
	}
}

func TestParseManifest_InvalidJobType(t *testing.T) {
	data := []byte(`{
		"job_type": "distributed",
		"environment": "python-3.11",
		"entrypoint": "main.py",
		"expected_output": "result.zip"
	}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("expected validation errors for invalid job_type")
	}
	if errors[0].Field != "job_type" {
		t.Errorf("expected error for job_type, got %s", errors[0].Field)
	}
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json}`)

	_, _, err := ParseManifest(data)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseManifest_MultipleErrors(t *testing.T) {
	data := []byte(`{}`)

	_, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) < 3 {
		t.Errorf("expected at least 3 validation errors for empty manifest, got %d", len(errors))
	}
}

func TestManifest_JSONRoundTrip(t *testing.T) {
	original := &Manifest{
		JobType:        JobTypeSingle,
		Environment:    "python-3.11",
		Entrypoint:      "main.py",
		ExpectedOutput: "result.zip",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, errors, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if len(errors) > 0 {
		t.Fatalf("expected no validation errors, got %v", errors)
	}
	if parsed.JobType != original.JobType {
		t.Errorf("job_type mismatch: %s vs %s", parsed.JobType, original.JobType)
	}
	if parsed.Environment != original.Environment {
		t.Errorf("environment mismatch: %s vs %s", parsed.Environment, original.Environment)
	}
}
