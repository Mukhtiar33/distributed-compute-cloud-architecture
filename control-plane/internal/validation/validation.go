package validation

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
)

// ValidationStage represents a stage in the output validation pipeline.
type ValidationStage int

const (
	StageWorkerLocal ValidationStage = iota + 1
	StageControlPlane
	StagePreDelivery
)

// ValidationResult represents the outcome of a validation stage.
type ValidationResult struct {
	Stage   ValidationStage `json:"stage"`
	Passed  bool            `json:"passed"`
	Message string          `json:"message"`
}

// ExpectedOutput describes the expected output shape from the manifest.
type ExpectedOutput struct {
	Files       []string `json:"files"`        // Expected output file names
	MinSize     int64    `json:"min_size"`     // Minimum total output size in bytes
	MaxSize     int64    `json:"max_size"`     // Maximum total output size in bytes
	ContentType string   `json:"content_type"` // Expected content type (e.g., "application/zip")
}

// ValidateWorkerLocal performs worker-local sanity check before upload.
// Catches ordinary job/worker bugs early and cheaply.
func ValidateWorkerLocal(outputData []byte, expected ExpectedOutput) ValidationResult {
	if len(outputData) == 0 {
		return ValidationResult{
			Stage:   StageWorkerLocal,
			Passed:  false,
			Message: "output is empty",
		}
	}

	if expected.MinSize > 0 && int64(len(outputData)) < expected.MinSize {
		return ValidationResult{
			Stage:   StageWorkerLocal,
			Passed:  false,
			Message: fmt.Sprintf("output size %d below minimum %d", len(outputData), expected.MinSize),
		}
	}

	if expected.MaxSize > 0 && int64(len(outputData)) > expected.MaxSize {
		return ValidationResult{
			Stage:   StageWorkerLocal,
			Passed:  false,
			Message: fmt.Sprintf("output size %d exceeds maximum %d", len(outputData), expected.MaxSize),
		}
	}

	// Always validate that output is a valid zip
	zipReader, err := zip.NewReader(bytes.NewReader(outputData), int64(len(outputData)))
	if err != nil {
		return ValidationResult{
			Stage:   StageWorkerLocal,
			Passed:  false,
			Message: fmt.Sprintf("output is not a valid zip: %v", err),
		}
	}

	if len(expected.Files) > 0 {
		foundFiles := make(map[string]bool)
		for _, f := range zipReader.File {
			foundFiles[f.Name] = true
		}

		for _, expectedFile := range expected.Files {
			if !foundFiles[expectedFile] {
				return ValidationResult{
					Stage:   StageWorkerLocal,
					Passed:  false,
					Message: fmt.Sprintf("expected output file '%s' not found", expectedFile),
				}
			}
		}
	}

	return ValidationResult{
		Stage:   StageWorkerLocal,
		Passed:  true,
		Message: "worker-local sanity check passed",
	}
}

// ValidateControlPlane performs authoritative validation at the Control Plane.
// Catches a compromised or malicious worker attempting to use its result payload as an attack vector.
func ValidateControlPlane(outputData []byte, expected ExpectedOutput) ValidationResult {
	stage1 := ValidateWorkerLocal(outputData, expected)
	if !stage1.Passed {
		return ValidationResult{
			Stage:   StageControlPlane,
			Passed:  false,
			Message: fmt.Sprintf("stage 1 check failed at Control Plane: %s", stage1.Message),
		}
	}

	zipReader, err := zip.NewReader(bytes.NewReader(outputData), int64(len(outputData)))
	if err != nil {
		return ValidationResult{
			Stage:   StageControlPlane,
			Passed:  false,
			Message: fmt.Sprintf("output is not a valid zip: %v", err),
		}
	}

	var totalUncompressed int64
	for _, f := range zipReader.File {
		totalUncompressed += int64(f.UncompressedSize64)
	}

	if totalUncompressed > int64(len(outputData))*100 {
		return ValidationResult{
			Stage:   StageControlPlane,
			Passed:  false,
			Message: fmt.Sprintf("potential zip bomb: uncompressed size %d is %dx compressed size", totalUncompressed, totalUncompressed/int64(len(outputData))),
		}
	}

	return ValidationResult{
		Stage:   StageControlPlane,
		Passed:  true,
		Message: "Control Plane authoritative validation passed",
	}
}

// ValidatePreDelivery performs integrity check at result assembly.
// For single-worker jobs, this is trivial. For parallel jobs (Phase 8), this checks all batches are present.
func ValidatePreDelivery(outputData []byte, expected ExpectedOutput) ValidationResult {
	zipReader, err := zip.NewReader(bytes.NewReader(outputData), int64(len(outputData)))
	if err != nil {
		return ValidationResult{
			Stage:   StagePreDelivery,
			Passed:  false,
			Message: fmt.Sprintf("output is not a valid zip: %v", err),
		}
	}

	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return ValidationResult{
				Stage:   StagePreDelivery,
				Passed:  false,
				Message: fmt.Sprintf("failed to read file '%s': %v", f.Name, err),
			}
		}

		_, err = io.Copy(io.Discard, rc)
		rc.Close()
		if err != nil {
			return ValidationResult{
				Stage:   StagePreDelivery,
				Passed:  false,
				Message: fmt.Sprintf("failed to read file '%s': %v", f.Name, err),
			}
		}
	}

	return ValidationResult{
		Stage:   StagePreDelivery,
		Passed:  true,
		Message: "pre-delivery integrity check passed",
	}
}

// ValidateAll runs all three validation stages.
func ValidateAll(outputData []byte, expected ExpectedOutput) []ValidationResult {
	return []ValidationResult{
		ValidateWorkerLocal(outputData, expected),
		ValidateControlPlane(outputData, expected),
		ValidatePreDelivery(outputData, expected),
	}
}

func detectContentType(data []byte) string {
	if len(data) >= 4 {
		if data[0] == 0x50 && data[1] == 0x4B {
			return "application/zip"
		}
		if data[0] == 0x1F && data[1] == 0x8B {
			return "application/gzip"
		}
	}
	return "application/octet-stream"
}
