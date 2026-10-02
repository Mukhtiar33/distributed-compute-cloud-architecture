package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
)

// Scheduler assigns jobs to available workers.
type Scheduler struct {
	workerStore *auth.WorkerStore
	jobStore    *jobs.Store
	mu          sync.RWMutex
}

// NewScheduler creates a new scheduler.
func NewScheduler(workerStore *auth.WorkerStore, jobStore *jobs.Store) *Scheduler {
	return &Scheduler{
		workerStore: workerStore,
		jobStore:    jobStore,
	}
}

// AssignJob assigns a job to an available worker.
// For Phase 4: simple "any available worker" — no ranking logic yet.
func (s *Scheduler) AssignJob(jobID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find an available worker
	workers := s.workerStore.ListWorkers()
	for _, w := range workers {
		if w.Status == "active" {
			// In a real implementation, we would send the job to the worker here
			// For Phase 4, we just track the assignment
			return w.ID, nil
		}
	}

	return "", fmt.Errorf("no available workers for job %s", jobID)
}

// GetJobAssignment returns the worker assigned to a job.
func (s *Scheduler) GetJobAssignment(jobID string) (string, error) {
	// For Phase 4, we don't track assignments persistently
	// This would be implemented in a later phase
	return "", fmt.Errorf("job assignment tracking not implemented yet")
}

// Start begins the scheduler's background processing loop.
func (s *Scheduler) Start() {
	// For Phase 4, the scheduler is minimal
	// In later phases, this would include:
	// - Monitoring job queue
	// - Assigning jobs to workers
	// - Handling worker failures
	// - Reassigning failed jobs
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	// Cleanup resources
}

// HealthCheck performs a health check on the scheduler.
func (s *Scheduler) HealthCheck() error {
	// For Phase 4, always healthy
	return nil
}

// Stats returns scheduler statistics.
func (s *Scheduler) Stats() map[string]interface{} {
	workers := s.workerStore.ListWorkers()
	jobList := s.jobStore.ListJobs()

	return map[string]interface{}{
		"total_workers":   len(workers),
		"active_workers":  countActiveWorkers(workers),
		"total_jobs":      len(jobList),
		"pending_jobs":    countPendingJobs(jobList),
		"timestamp":       time.Now().Unix(),
	}
}

func countActiveWorkers(workers []*auth.Worker) int {
	count := 0
	for _, w := range workers {
		if w.Status == "active" {
			count++
		}
	}
	return count
}

func countPendingJobs(jobList []*jobs.Job) int {
	count := 0
	for _, j := range jobList {
		if j.Status == jobs.JobStatusScanPassed {
			count++
		}
	}
	return count
}
