package auth

import (
	"testing"
	"time"
)

func TestWorkerStore_AddAndGet(t *testing.T) {
	store := NewWorkerStore()
	w := &Worker{
		ID:         "worker-1",
		Status:     "enrolled",
		EnrolledAt: time.Now(),
	}

	store.AddWorker(w)

	retrieved, ok := store.GetWorker("worker-1")
	if !ok {
		t.Fatal("expected to find worker-1")
	}
	if retrieved.ID != "worker-1" {
		t.Errorf("expected worker-1, got %s", retrieved.ID)
	}
}

func TestWorkerStore_UpdateStatus(t *testing.T) {
	store := NewWorkerStore()
	w := &Worker{
		ID:         "worker-1",
		Status:     "enrolled",
		EnrolledAt: time.Now(),
	}

	store.AddWorker(w)
	store.UpdateWorkerStatus("worker-1", "active")

	retrieved, _ := store.GetWorker("worker-1")
	if retrieved.Status != "active" {
		t.Errorf("expected active, got %s", retrieved.Status)
	}
}

func TestWorkerStore_Revoke(t *testing.T) {
	store := NewWorkerStore()
	w := &Worker{
		ID:         "worker-1",
		Status:     "active",
		EnrolledAt: time.Now(),
	}

	store.AddWorker(w)
	store.RevokeWorker("worker-1")

	retrieved, _ := store.GetWorker("worker-1")
	if retrieved.Status != "revoked" {
		t.Errorf("expected revoked, got %s", retrieved.Status)
	}
}

func TestWorkerStore_ListWorkers(t *testing.T) {
	store := NewWorkerStore()
	store.AddWorker(&Worker{ID: "worker-1", Status: "enrolled"})
	store.AddWorker(&Worker{ID: "worker-2", Status: "active"})

	workers := store.ListWorkers()
	if len(workers) != 2 {
		t.Errorf("expected 2 workers, got %d", len(workers))
	}
}

func TestWorkerStore_GetNonexistent(t *testing.T) {
	store := NewWorkerStore()
	_, ok := store.GetWorker("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent worker")
	}
}
