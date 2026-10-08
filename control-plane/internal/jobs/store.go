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
	JobStatusIntakeValidated JobStatus = "intake_validated"
	JobStatusScanning         JobStatus = "scanning"
	JobStatusScanPassed       JobStatus = "scan_passed"
	JobStatusScanRejected     JobStatus = "scan_rejected"
	JobStatusRejected         JobStatus = "rejected"
)

// Job represents a submitted job in the Control Plane.
type Job struct {
	ID            string              `json:"id"`
	Manifest      *manifest.Manifest  `json:"manifest"`
	Status        JobStatus           `json:"status"`
	SubmittedAt   time.Time           `json:"submitted_at"`
	ValidationError []string          `json:"validation_errors,omitempty"`
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

// UpdateStatus updates a job's status.
func (s *Store) UpdateStatus(id string, status JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.Status = status
		return nil
	}
	return fmt.Errorf("job %s not found", id)
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

// GetJob retrieves a job by ID.
func (s *Store) GetJob(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
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
