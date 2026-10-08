package batch

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/db"
)

// ManagerDB is a PostgreSQL-backed batch manager.
type ManagerDB struct {
	db *db.DB
}

// NewManagerDB creates a new database-backed batch manager.
func NewManagerDB(database *db.DB) *ManagerDB {
	return &ManagerDB{db: database}
}

// GroupInputs groups inputs into batches using round-robin distribution.
func (m *ManagerDB) GroupInputs(jobID string, inputs []string, numWorkers int) ([]*Batch, error) {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	batches := make([]*Batch, 0, numWorkers)
	now := time.Now()

	// Create batches in a single transaction
	tx, err := m.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

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

		inputsJSON, _ := json.Marshal(batch.Inputs)
		_, err := tx.Exec(
			`INSERT INTO batches (id, job_id, status, inputs) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (id) DO NOTHING`,
			batchID, jobID, BatchStatusPending, inputsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert batch: %w", err)
		}
	}

	// Distribute inputs round-robin
	for i, input := range inputs {
		batchIndex := i % numWorkers
		batches[batchIndex].Inputs = append(batches[batchIndex].Inputs, input)
	}

	// Update batch inputs
	for _, batch := range batches {
		inputsJSON, _ := json.Marshal(batch.Inputs)
		_, err := tx.Exec(
			`UPDATE batches SET inputs = $1 WHERE id = $2`,
			inputsJSON, batch.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update batch inputs: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return batches, nil
}

// GetBatch retrieves a batch by ID.
func (m *ManagerDB) GetBatch(id string) (*Batch, bool) {
	row := m.db.QueryRow(
		`SELECT id, job_id, worker_id, status, inputs, result, created_at, started_at, completed_at, reassigned_to, reassign_count
		 FROM batches WHERE id = $1`,
		id,
	)

	var batch Batch
	var inputsJSON []byte
	var result []byte
	err := row.Scan(
		&batch.ID, &batch.JobID, &batch.WorkerID, &batch.Status, &inputsJSON, &result,
		&batch.CreatedAt, &batch.StartedAt, &batch.CompletedAt, &batch.ReassignedTo, &batch.ReassignCount,
	)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	if len(inputsJSON) > 0 {
		json.Unmarshal(inputsJSON, &batch.Inputs)
	}
	batch.Result = result

	return &batch, true
}

// UpdateStatus updates a batch's status.
func (m *ManagerDB) UpdateStatus(id string, status BatchStatus) error {
	var err error
	now := time.Now()

	switch status {
	case BatchStatusRunning:
		_, err = m.db.Exec(
			`UPDATE batches SET status = $1, started_at = $2 WHERE id = $3`,
			status, now, id,
		)
	case BatchStatusSucceeded, BatchStatusFailed:
		_, err = m.db.Exec(
			`UPDATE batches SET status = $1, completed_at = $2 WHERE id = $3`,
			status, now, id,
		)
	default:
		_, err = m.db.Exec(
			`UPDATE batches SET status = $1 WHERE id = $2`,
			status, id,
		)
	}
	return err
}

// AssignWorker assigns a worker to a batch.
func (m *ManagerDB) AssignWorker(batchID, workerID string) error {
	_, err := m.db.Exec(
		`UPDATE batches SET worker_id = $1, status = $2, started_at = $3 WHERE id = $4`,
		workerID, BatchStatusRunning, time.Now(), batchID,
	)
	return err
}

// Reassign marks a batch as failed and reassignable.
func (m *ManagerDB) Reassign(batchID, newWorkerID string) error {
	_, err := m.db.Exec(
		`UPDATE batches SET status = $1, reassigned_to = $2, reassign_count = reassign_count + 1 WHERE id = $3`,
		BatchStatusReassigned, newWorkerID, batchID,
	)
	return err
}

// Complete marks a batch as succeeded with its result.
func (m *ManagerDB) Complete(batchID string, result []byte) (bool, error) {
	// Idempotency: only update if not already succeeded
	resultTag, err := m.db.Exec(
		`UPDATE batches SET status = $1, result = $2, completed_at = $3
		 WHERE id = $4 AND status != $1`,
		BatchStatusSucceeded, result, time.Now(), batchID,
	)
	if err != nil {
		return false, err
	}
	rows, _ := resultTag.RowsAffected()
	return rows > 0, nil
}

// Fail marks a batch as failed.
func (m *ManagerDB) Fail(batchID string) error {
	_, err := m.db.Exec(
		`UPDATE batches SET status = $1, completed_at = $2 WHERE id = $3`,
		BatchStatusFailed, time.Now(), batchID,
	)
	return err
}

// GetBatchesByJob returns all batches for a given job.
func (m *ManagerDB) GetBatchesByJob(jobID string) []*Batch {
	rows, err := m.db.Query(
		`SELECT id, job_id, worker_id, status, inputs, result, created_at, started_at, completed_at, reassigned_to, reassign_count
		 FROM batches WHERE job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var batches []*Batch
	for rows.Next() {
		var batch Batch
		var inputsJSON []byte
		var result []byte
		if err := rows.Scan(
			&batch.ID, &batch.JobID, &batch.WorkerID, &batch.Status, &inputsJSON, &result,
			&batch.CreatedAt, &batch.StartedAt, &batch.CompletedAt, &batch.ReassignedTo, &batch.ReassignCount,
		); err != nil {
			continue
		}
		if len(inputsJSON) > 0 {
			json.Unmarshal(inputsJSON, &batch.Inputs)
		}
		batch.Result = result
		batches = append(batches, &batch)
	}
	return batches
}

// IsJobComplete checks if all batches for a job have succeeded.
func (m *ManagerDB) IsJobComplete(jobID string) bool {
	row := m.db.QueryRow(
		`SELECT COUNT(*) FROM batches WHERE job_id = $1 AND status != $2`,
		jobID, BatchStatusSucceeded,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count == 0
}

// GetPendingBatches returns all pending batches.
func (m *ManagerDB) GetPendingBatches() []*Batch {
	rows, err := m.db.Query(
		`SELECT id, job_id, worker_id, status, inputs, result, created_at, started_at, completed_at, reassigned_to, reassign_count
		 FROM batches WHERE status IN ($1, $2) ORDER BY created_at ASC`,
		BatchStatusPending, BatchStatusReassigned,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var batches []*Batch
	for rows.Next() {
		var batch Batch
		var inputsJSON []byte
		var result []byte
		if err := rows.Scan(
			&batch.ID, &batch.JobID, &batch.WorkerID, &batch.Status, &inputsJSON, &result,
			&batch.CreatedAt, &batch.StartedAt, &batch.CompletedAt, &batch.ReassignedTo, &batch.ReassignCount,
		); err != nil {
			continue
		}
		if len(inputsJSON) > 0 {
			json.Unmarshal(inputsJSON, &batch.Inputs)
		}
		batch.Result = result
		batches = append(batches, &batch)
	}
	return batches
}

// GetStats returns batch statistics.
func (m *ManagerDB) GetStats() map[string]int {
	stats := map[string]int{
		"total": 0, "pending": 0, "running": 0, "succeeded": 0, "failed": 0, "reassigned": 0,
	}

	rows, err := m.db.Query(`SELECT status, COUNT(*) FROM batches GROUP BY status`)
	if err != nil {
		return stats
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats["total"] += count
		switch BatchStatus(status) {
		case BatchStatusPending:
			stats["pending"] += count
		case BatchStatusRunning:
			stats["running"] += count
		case BatchStatusSucceeded:
			stats["succeeded"] += count
		case BatchStatusFailed:
			stats["failed"] += count
		case BatchStatusReassigned:
			stats["reassigned"] += count
		}
	}
	return stats
}
