package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
	"github.com/distributedcompute/cloud/control-plane/internal/manifest"
	"github.com/distributedcompute/cloud/control-plane/internal/scanner"
)

// JobHandler handles job submission and retrieval.
type JobHandler struct {
	jobStore jobs.JobStoreInterface
	scanner  *scanner.Scanner
}

// NewJobHandler creates a new job handler.
func NewJobHandler(store jobs.JobStoreInterface) *JobHandler {
	return &JobHandler{
		jobStore: store,
		scanner:  scanner.NewScanner(),
	}
}

// SubmitJob handles job submission via HTTP.
func (h *JobHandler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (zip file + manifest)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract manifest JSON
	manifestJSON := r.FormValue("manifest")
	if manifestJSON == "" {
		http.Error(w, "manifest field is required", http.StatusBadRequest)
		return
	}

	// Extract zip file
	file, _, err := r.FormFile("zip")
	if err != nil {
		http.Error(w, "zip file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	zipData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read zip file", http.StatusBadRequest)
		return
	}

	// Validate zip file is not empty
	if len(zipData) == 0 {
		http.Error(w, "zip file is empty", http.StatusBadRequest)
		return
	}

	// Parse and validate manifest
	manifest, validationErrors, err := manifest.ParseManifest([]byte(manifestJSON))
	if err != nil {
		// JSON parse error
		job := &jobs.Job{
			ID:              generateJobID(),
			Status:          jobs.JobStatusRejected,
			SubmittedAt:     time.Now(),
			ValidationError: []string{fmt.Sprintf("JSON parse error: %v", err)},
		}
		h.jobStore.AddJob(job)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":            job.ID,
			"status":            job.Status,
			"validation_errors": job.ValidationError,
		})
		return
	}

	// Check validation errors
	if len(validationErrors) > 0 {
		errorMessages := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			errorMessages[i] = ve.Error()
		}

		job := &jobs.Job{
			ID:              generateJobID(),
			Manifest:        manifest,
			Status:          jobs.JobStatusRejected,
			SubmittedAt:     time.Now(),
			ValidationError: errorMessages,
		}
		h.jobStore.AddJob(job)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":            job.ID,
			"status":            job.Status,
			"validation_errors": errorMessages,
		})
		return
	}

	// Job passed manifest validation — now run static scanning
	job := &jobs.Job{
		ID:            generateJobID(),
		Manifest:      manifest,
		Status:        jobs.JobStatusScanning,
		SubmittedAt:   time.Now(),
	}
	h.jobStore.AddJob(job)

	// Run static scanner
	scanResult, err := h.scanner.ScanZip(zipData)
	if err != nil {
		job.Status = jobs.JobStatusScanRejected
		job.ValidationError = []string{fmt.Sprintf("scan error: %v", err)}
		h.jobStore.AddJob(job)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":            job.ID,
			"status":            job.Status,
			"validation_errors": job.ValidationError,
		})
		return
	}

	// Check entrypoint exists
	entrypointResult, err := h.scanner.ScanManifest(zipData, manifest.Entrypoint)
	if err != nil {
		job.Status = jobs.JobStatusScanRejected
		job.ValidationError = []string{fmt.Sprintf("entrypoint check error: %v", err)}
		h.jobStore.AddJob(job)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":            job.ID,
			"status":            job.Status,
			"validation_errors": job.ValidationError,
		})
		return
	}

	// Combine scan reasons
	allReasons := append(scanResult.Reasons, entrypointResult.Reasons...)

	if len(allReasons) > 0 {
		job.Status = jobs.JobStatusScanRejected
		job.ValidationError = allReasons
		h.jobStore.AddJob(job)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":            job.ID,
			"status":            job.Status,
			"validation_errors": allReasons,
		})
		return
	}

	// Job passed all scans
	job.Status = jobs.JobStatusScanPassed
	h.jobStore.AddJob(job)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "Job accepted and passed static scanning",
	})
}

// GetJob retrieves a job by ID.
func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	job, ok := h.jobStore.GetJob(jobID)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(job)
}

// ListJobs returns all jobs.
func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobs := h.jobStore.ListJobs()
	json.NewEncoder(w).Encode(jobs)
}

func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}
