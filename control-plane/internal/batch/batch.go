package batch

import (
	"fmt"
	"sync"
	"time"
)

// BatchStatus represents the current state of a batch.
type BatchStatus string

const (
	BatchStatusPending    BatchStatus = "pending"
	BatchStatusRunning    BatchStatus = "running"
	BatchStatusSucceeded  BatchStatus = "succeeded"
	BatchStatusFailed     BatchStatus = "failed"
	BatchStatusReassigned BatchStatus = "reassigned"
)

// Batch represents a single batch of work within a parallel job.
type Batch struct {
	ID            string      `json:"id"`
	JobID         string      `json:"job_id"`
	WorkerID      string      `json:"worker_id,omitempty"`
	Status        BatchStatus `json:"status"`
	Inputs        []string    `json:"inputs"`
	Result        []byte      `json:"result,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	StartedAt     *time.Time  `json:"started_at,omitempty"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
	ReassignedTo  string      `json:"reassigned_to,omitempty"`
	ReassignCount int         `json:"reassign_count"`
}

// Manager manages batch lifecycle, grouping, and reassignment.
type Manager struct {
	mu     sync.RWMutex
	batches map[string]*Batch // key: batch ID
}

// NewManager creates a new batch manager.
func NewManager() *Manager {
	return &Manager{
		batches: make(map[string]*Batch),
	}
}

// GroupInputs groups inputs into batches using round-robin distribution.
func (m *Manager) GroupInputs(jobID string, inputs []string, numWorkers int) []*Batch {
	m.mu.Lock()
	defer m.mu.Unlock()

	if numWorkers <= 0 {
		numWorkers = 1
	}

	batches := make([]*Batch, 0, numWorkers)
	now := time.Now()

	// Create one batch per worker
	for i := 0; i < numWorkers; i++ {
		batchID := fmt.Sprintf("%s-batch-%d", jobID, i)
		batch := &Batch{
			ID:        batchID,
			JobID:     jobID,
			Status:    BatchStatusPending,
			Inputs:    []string{},
			CreatedAt: now,
		}
		batches = append(batches, batch)
		m.batches[batchID] = batch
	}

	// Distribute inputs round-robin
	for i, input := range inputs {
		batchIndex := i % numWorkers
		batches[batchIndex].Inputs = append(batches[batchIndex].Inputs, input)
	}

	return batches
}

// GetBatch retrieves a batch by ID.
func (m *Manager) GetBatch(id string) (*Batch, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.batches[id]
	return b, ok
}

// UpdateStatus updates a batch's status.
func (m *Manager) UpdateStatus(id string, status BatchStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.batches[id]
	if !ok {
		return fmt.Errorf("batch %s not found", id)
	}

	b.Status = status

	now := time.Now()
	switch status {
	case BatchStatusRunning:
		b.StartedAt = &now
	case BatchStatusSucceeded, BatchStatusFailed:
		b.CompletedAt = &now
	}

	return nil
}

// AssignWorker assigns a worker to a batch.
func (m *Manager) AssignWorker(batchID, workerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.batches[batchID]
	if !ok {
		return fmt.Errorf("batch %s not found", batchID)
	}

	b.WorkerID = workerID
	b.Status = BatchStatusRunning

	now := time.Now()
	b.StartedAt = &now

	return nil
}

// Reassign marks a batch as failed and reassignable.
func (m *Manager) Reassign(batchID, newWorkerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.batches[batchID]
	if !ok {
		return fmt.Errorf("batch %s not found", batchID)
	}

	b.Status = BatchStatusReassigned
	b.ReassignedTo = newWorkerID
	b.ReassignCount++

	return nil
}

// Complete marks a batch as succeeded with its result.
// Returns false if the batch was already completed (idempotency check).
func (m *Manager) Complete(batchID string, result []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.batches[batchID]
	if !ok {
		return false, fmt.Errorf("batch %s not found", batchID)
	}

	// Idempotency: if already succeeded, discard duplicate result
	if b.Status == BatchStatusSucceeded {
		return false, nil
	}

	b.Status = BatchStatusSucceeded
	b.Result = result

	now := time.Now()
	b.CompletedAt = &now

	return true, nil
}

// Fail marks a batch as failed.
func (m *Manager) Fail(batchID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.batches[batchID]
	if !ok {
		return fmt.Errorf("batch %s not found", batchID)
	}

	b.Status = BatchStatusFailed

	now := time.Now()
	b.CompletedAt = &now

	return nil
}

// GetBatchesByJob returns all batches for a given job.
func (m *Manager) GetBatchesByJob(jobID string) []*Batch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Batch
	for _, b := range m.batches {
		if b.JobID == jobID {
			result = append(result, b)
		}
	}
	return result
}

// IsJobComplete checks if all batches for a job have succeeded.
func (m *Manager) IsJobComplete(jobID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	batches := make([]*Batch, 0)
	for _, b := range m.batches {
		if b.JobID == jobID {
			batches = append(batches, b)
		}
	}

	if len(batches) == 0 {
		return false
	}

	for _, b := range batches {
		if b.Status != BatchStatusSucceeded {
			return false
		}
	}

	return true
}

// GetPendingBatches returns all pending batches.
func (m *Manager) GetPendingBatches() []*Batch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Batch
	for _, b := range m.batches {
		if b.Status == BatchStatusPending || b.Status == BatchStatusReassigned {
			result = append(result, b)
		}
	}
	return result
}

// GetRunningBatches returns all running batches.
func (m *Manager) GetRunningBatches() []*Batch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Batch
	for _, b := range m.batches {
		if b.Status == BatchStatusRunning {
			result = append(result, b)
		}
	}
	return result
}

// GetFailedBatches returns all failed batches.
func (m *Manager) GetFailedBatches() []*Batch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Batch
	for _, b := range m.batches {
		if b.Status == BatchStatusFailed {
			result = append(result, b)
		}
	}
	return result
}

// GetStats returns batch statistics.
func (m *Manager) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     0,
		"pending":   0,
		"running":   0,
		"succeeded": 0,
		"failed":    0,
		"reassigned": 0,
	}

	for _, b := range m.batches {
		stats["total"]++
		switch b.Status {
		case BatchStatusPending:
			stats["pending"]++
		case BatchStatusRunning:
			stats["running"]++
		case BatchStatusSucceeded:
			stats["succeeded"]++
		case BatchStatusFailed:
			stats["failed"]++
		case BatchStatusReassigned:
			stats["reassigned"]++
		}
	}

	return stats
}
