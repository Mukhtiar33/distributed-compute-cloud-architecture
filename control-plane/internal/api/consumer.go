package api

import (
	"encoding/json"
	"net/http"

	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
)

// ConsumerHandler handles consumer-facing job lifecycle APIs.
type ConsumerHandler struct {
	jobStore jobs.JobStoreInterface
}

// NewConsumerHandler creates a new consumer handler.
func NewConsumerHandler(store jobs.JobStoreInterface) *ConsumerHandler {
	return &ConsumerHandler{jobStore: store}
}

// GetJobStatus returns the status of a job.
func (h *ConsumerHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
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

	// Return only consumer-visible fields
	response := map[string]interface{}{
		"job_id":      job.ID,
		"status":      job.Status,
		"submitted_at": job.SubmittedAt,
	}
	if job.CompletedAt != nil {
		response["completed_at"] = job.CompletedAt
	}
	if len(job.ValidationError) > 0 {
		response["validation_errors"] = job.ValidationError
	}

	json.NewEncoder(w).Encode(response)
}

// DownloadResult downloads a completed job's result.
func (h *ConsumerHandler) DownloadResult(w http.ResponseWriter, r *http.Request) {
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

	if job.Status != jobs.JobStatusCompleted {
		http.Error(w, "Job not completed", http.StatusBadRequest)
		return
	}

	if len(job.ResultData) == 0 {
		http.Error(w, "No result available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"result.zip\"")
	w.Write(job.ResultData)
}

// CancelJob requests job cancellation.
func (h *ConsumerHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	job, ok := h.jobStore.GetJob(req.JobID)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Can only cancel jobs that haven't completed yet
	if job.Status == jobs.JobStatusCompleted || job.Status == jobs.JobStatusFailed {
		http.Error(w, "Job already finished", http.StatusBadRequest)
		return
	}

	h.jobStore.UpdateStatus(req.JobID, jobs.JobStatusFailed)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": req.JobID,
		"status": "cancelled",
	})
}

// ListConsumerJobs returns all jobs with consumer-visible fields.
func (h *ConsumerHandler) ListConsumerJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allJobs := h.jobStore.ListJobs()
	response := make([]map[string]interface{}, 0, len(allJobs))
	for _, job := range allJobs {
		j := map[string]interface{}{
			"job_id":      job.ID,
			"status":      job.Status,
			"submitted_at": job.SubmittedAt,
		}
		if job.CompletedAt != nil {
			j["completed_at"] = job.CompletedAt
		}
		response = append(response, j)
	}

	json.NewEncoder(w).Encode(response)
}

// GetJobMetadata returns execution metadata for a completed job.
func (h *ConsumerHandler) GetJobMetadata(w http.ResponseWriter, r *http.Request) {
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

	if job.Status != jobs.JobStatusCompleted {
		http.Error(w, "Job not completed", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":      job.ID,
		"status":      job.Status,
		"submitted_at": job.SubmittedAt,
		"completed_at": job.CompletedAt,
		"worker_id":   job.WorkerID,
	})
}
