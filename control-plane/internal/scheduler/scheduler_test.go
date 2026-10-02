package scheduler

import (
	"testing"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
	"github.com/distributedcompute/cloud/control-plane/internal/manifest"
)

func TestScheduler_AssignJob(t *testing.T) {
	workerStore := auth.NewWorkerStore()
	jobStore := jobs.NewStore()

	// Add an active worker
	workerStore.AddWorker(&auth.Worker{
		ID:         "worker-1",
		Status:     "active",
		EnrolledAt: time.Now(),
	})

	scheduler := NewScheduler(workerStore, jobStore)

	workerID, err := scheduler.AssignJob("job-1")
	if err != nil {
		t.Fatalf("AssignJob failed: %v", err)
	}
	if workerID != "worker-1" {
		t.Errorf("expected worker-1, got %s", workerID)
	}
}

func TestScheduler_NoAvailableWorkers(t *testing.T) {
	workerStore := auth.NewWorkerStore()
	jobStore := jobs.NewStore()

	// Add only inactive workers
	workerStore.AddWorker(&auth.Worker{
		ID:         "worker-1",
		Status:     "enrolled",
		EnrolledAt: time.Now(),
	})

	scheduler := NewScheduler(workerStore, jobStore)

	_, err := scheduler.AssignJob("job-1")
	if err == nil {
		t.Error("expected error when no workers available")
	}
}

func TestScheduler_Stats(t *testing.T) {
	workerStore := auth.NewWorkerStore()
	jobStore := jobs.NewStore()

	workerStore.AddWorker(&auth.Worker{
		ID:         "worker-1",
		Status:     "active",
		EnrolledAt: time.Now(),
	})

	jobStore.AddJob(&jobs.Job{
		ID:     "job-1",
		Status: jobs.JobStatusScanPassed,
		Manifest: &manifest.Manifest{
			JobType: manifest.JobTypeSingle,
		},
		SubmittedAt: time.Now(),
	})

	scheduler := NewScheduler(workerStore, jobStore)
	stats := scheduler.Stats()

	if stats["total_workers"].(int) != 1 {
		t.Errorf("expected 1 worker, got %d", stats["total_workers"])
	}
	if stats["active_workers"].(int) != 1 {
		t.Errorf("expected 1 active worker, got %d", stats["active_workers"])
	}
	if stats["total_jobs"].(int) != 1 {
		t.Errorf("expected 1 job, got %d", stats["total_jobs"])
	}
	if stats["pending_jobs"].(int) != 1 {
		t.Errorf("expected 1 pending job, got %d", stats["pending_jobs"])
	}
}

func TestScheduler_HealthCheck(t *testing.T) {
	workerStore := auth.NewWorkerStore()
	jobStore := jobs.NewStore()

	scheduler := NewScheduler(workerStore, jobStore)

	if err := scheduler.HealthCheck(); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}
