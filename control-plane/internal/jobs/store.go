package jobs

import (
	"fmt"
	"sync"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/manifest"
)

// JobStatus represents the current state of a job.
type JobStatus string

const (
	JobStatusSubmitted       JobStatus = "submitted"
	JobStatusValidating      JobStatus = "validating"
	JobStatusScanning        JobStatus = "scanning"
	JobStatusDetonating      JobStatus = "detonating"
	JobStatusReady           JobStatus = "ready"
	JobStatusAssigned        JobStatus = "assigned"
	JobStatusRunning         JobStatus = "running"
	JobStatusValidatingOutput JobStatus = "validating_output"
	JobStatusCompleted       JobStatus = "completed"
	JobStatusFailed          JobStatus = "failed"
	JobStatusRejected        JobStatus = "rejected"
)

// Job represents a submitted job in the Control Plane.
type Job struct {
	ID              string              `json:"id"`
	Manifest        *manifest.Manifest  `json:"manifest"`
	Status          JobStatus           `json:"status"`
	SubmittedAt     time.Time           `json:"submitted_at"`
	CompletedAt     *time.Time          `json:"completed_at,omitempty"`
	ValidationError []string          `json:"validation_errors,omitempty"`
	WorkerID        string              `json:"worker_id,omitempty"`
	ResultData      []byte              `json:"result_data,omitempty"`
}

// Store is an in-memory job store.
type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewStore creates a new job store.
func NewStore() *Store {
	return &Store{
		jobs: make(map[string]*Job),
	}
}

// AddJob adds a job to the store.
func (s *Store) AddJob(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// GetJob retrieves a job by ID.
func (s *Store) GetJob(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

// UpdateStatus updates a job's status.
func (s *Store) UpdateStatus(id string, status JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.Status = status
		if status == JobStatusCompleted || status == JobStatusFailed {
			now := time.Now()
			j.CompletedAt = &now
		}
		return nil
	}
	return fmt.Errorf("job %s not found", id)
}

// ListJobs returns all jobs in the store.
func (s *Store) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, job)
	}
	return result
}

// GetJobsByStatus returns jobs filtered by status.
func (s *Store) GetJobsByStatus(status JobStatus) []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Job
	for _, j := range s.jobs {
		if j.Status == status {
			result = append(result, j)
		}
	}
	return result
}
