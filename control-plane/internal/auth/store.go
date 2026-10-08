package auth

import (
	"sync"
	"time"
)

// Worker represents an enrolled worker agent.
type Worker struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"` // "enrolled", "active", "revoked"
	EnrolledAt    time.Time `json:"enrolled_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// WorkerStore is an in-memory store of enrolled workers.
type WorkerStore struct {
	mu      sync.RWMutex
	workers map[string]*Worker
}

// NewWorkerStore creates a new worker store.
func NewWorkerStore() *WorkerStore {
	return &WorkerStore{
		workers: make(map[string]*Worker),
	}
}

// AddWorker registers a new worker.
func (s *WorkerStore) AddWorker(w *Worker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workers[w.ID] = w
}

// IsWorkerRevoked checks if a worker has been revoked.
func (s *WorkerStore) IsWorkerRevoked(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workers[id]
	if !ok {
		return false
	}
	return w.Status == "revoked"
}

// GetWorker retrieves a worker by ID.
func (s *WorkerStore) GetWorker(id string) (*Worker, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workers[id]
	return w, ok
}

// UpdateWorkerStatus updates a worker's status.
func (s *WorkerStore) UpdateWorkerStatus(id, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.workers[id]; ok {
		w.Status = status
		if status == "active" {
			w.LastHeartbeat = time.Now()
		}
		return true
	}
	return false
}

// RevokeWorker marks a worker as revoked.
func (s *WorkerStore) RevokeWorker(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.workers[id]; ok {
		w.Status = "revoked"
		return true
	}
	return false
}

// ListWorkers returns all registered workers.
func (s *WorkerStore) ListWorkers() []*Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Worker, 0, len(s.workers))
	for _, w := range s.workers {
		result = append(result, w)
	}
	return result
}
