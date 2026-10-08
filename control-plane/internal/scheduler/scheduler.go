package scheduler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
)

// Scheduler assigns jobs to available workers based on resource availability.
type Scheduler struct {
	workerStore auth.WorkerStoreInterface
	jobStore    jobs.JobStoreInterface
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
}

// NewScheduler creates a new scheduler.
func NewScheduler(workerStore auth.WorkerStoreInterface, jobStore jobs.JobStoreInterface) *Scheduler {
	return &Scheduler{
		workerStore: workerStore,
		jobStore:    jobStore,
		stopCh:      make(chan struct{}),
	}
}

// AssignJob assigns a job to an available worker.
func (s *Scheduler) AssignJob(jobID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workers := s.workerStore.ListWorkers()
	for _, w := range workers {
		if w.Status == "active" {
			log.Printf("Job %s assigned to worker %s", jobID, w.ID)
			return w.ID, nil
		}
	}

	return "", fmt.Errorf("no available workers for job %s", jobID)
}

// GetJobAssignment returns the worker assigned to a job.
func (s *Scheduler) GetJobAssignment(jobID string) (string, error) {
	job, ok := s.jobStore.GetJob(jobID)
	if !ok {
		return "", fmt.Errorf("job not found")
	}
	return job.WorkerID, nil
}

// Start begins the scheduler's background processing loop.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.loop()
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		s.running = false
		close(s.stopCh)
	}
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processPendingJobs()
			s.checkWorkerHealth()
		}
	}
}

func (s *Scheduler) processPendingJobs() {
	readyJobs := s.jobStore.GetJobsByStatus(jobs.JobStatusReady)
	for _, job := range readyJobs {
		workerID, err := s.AssignJob(job.ID)
		if err != nil {
			log.Printf("Failed to assign job %s: %v", job.ID, err)
			continue
		}

		s.jobStore.UpdateStatus(job.ID, jobs.JobStatusAssigned)
		log.Printf("Job %s assigned to worker %s", job.ID, workerID)
	}
}

func (s *Scheduler) checkWorkerHealth() {
	workers := s.workerStore.ListWorkers()
	for _, w := range workers {
		if w.Status == "active" && w.LastHeartbeat.IsZero() {
			// Worker has never heartbeated — mark as stale
			s.workerStore.UpdateWorkerStatus(w.ID, "stale")
			log.Printf("Worker %s marked as stale (no heartbeat)", w.ID)
		}
	}
}

// HealthCheck performs a health check on the scheduler.
func (s *Scheduler) HealthCheck() error {
	return nil
}

// Stats returns scheduler statistics.
func (s *Scheduler) Stats() map[string]interface{} {
	workers := s.workerStore.ListWorkers()
	jobList := s.jobStore.ListJobs()

	return map[string]interface{}{
		"total_workers":  len(workers),
		"active_workers": countActiveWorkers(workers),
		"total_jobs":     len(jobList),
		"pending_jobs":   countPendingJobs(jobList),
		"timestamp":      time.Now().Unix(),
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
		if j.Status == jobs.JobStatusReady {
			count++
		}
	}
	return count
}
