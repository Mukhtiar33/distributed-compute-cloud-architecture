package jobs

import (
	"sync"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/manifest"
)

// JobStatus represents the current state of a job.
type JobStatus string

const (
	JobStatusIntakeValidated JobStatus = "intake_validated"
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
