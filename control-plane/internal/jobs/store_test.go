package jobs

import (
	"testing"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/manifest"
)

func TestStore_AddAndGet(t *testing.T) {
	store := NewStore()
	job := &Job{
		ID:          "job-1",
		Manifest:    &manifest.Manifest{JobType: manifest.JobTypeSingle, Environment: "python-3.11", Entrypoint: "main.py", ExpectedOutput: "result.zip"},
		Status:      JobStatusIntakeValidated,
		SubmittedAt: time.Now(),
	}

	store.AddJob(job)

	retrieved, ok := store.GetJob("job-1")
	if !ok {
		t.Fatal("expected to find job-1")
	}
	if retrieved.ID != "job-1" {
		t.Errorf("expected job-1, got %s", retrieved.ID)
	}
}

func TestStore_GetNonexistent(t *testing.T) {
	store := NewStore()
	_, ok := store.GetJob("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent job")
	}
}

func TestStore_ListJobs(t *testing.T) {
	store := NewStore()
	store.AddJob(&Job{ID: "job-1", Status: JobStatusIntakeValidated, SubmittedAt: time.Now()})
	store.AddJob(&Job{ID: "job-2", Status: JobStatusRejected, SubmittedAt: time.Now()})

	jobs := store.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}
