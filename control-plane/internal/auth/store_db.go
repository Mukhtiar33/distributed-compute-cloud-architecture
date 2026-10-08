package auth

import (
	"database/sql"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/db"
)

// WorkerStoreDB is a PostgreSQL-backed worker store.
type WorkerStoreDB struct {
	db *db.DB
}

// NewWorkerStoreDB creates a new database-backed worker store.
func NewWorkerStoreDB(database *db.DB) *WorkerStoreDB {
	return &WorkerStoreDB{db: database}
}

// AddWorker registers a new worker.
func (s *WorkerStoreDB) AddWorker(w *Worker) {
	s.db.Exec(
		`INSERT INTO workers (id, status, enrolled_at) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET status = $2, enrolled_at = $3`,
		w.ID, w.Status, w.EnrolledAt,
	)
}

// GetWorker retrieves a worker by ID.
func (s *WorkerStoreDB) GetWorker(id string) (*Worker, bool) {
	row := s.db.QueryRow(
		`SELECT id, status, enrolled_at, last_heartbeat FROM workers WHERE id = $1`,
		id,
	)

	var w Worker
	err := row.Scan(&w.ID, &w.Status, &w.EnrolledAt, &w.LastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	return &w, true
}

// UpdateWorkerStatus updates a worker's status.
func (s *WorkerStoreDB) UpdateWorkerStatus(id, status string) bool {
	var result sql.Result
	var err error

	if status == "active" {
		result, err = s.db.Exec(
			`UPDATE workers SET status = $1, last_heartbeat = $2, updated_at = $2 WHERE id = $3`,
			status, time.Now(), id,
		)
	} else {
		result, err = s.db.Exec(
			`UPDATE workers SET status = $1, updated_at = $2 WHERE id = $3`,
			status, time.Now(), id,
		)
	}

	if err != nil {
		return false
	}

	rows, _ := result.RowsAffected()
	return rows > 0
}

// RevokeWorker marks a worker as revoked.
func (s *WorkerStoreDB) RevokeWorker(id string) bool {
	result, err := s.db.Exec(
		`UPDATE workers SET status = 'revoked', updated_at = $1 WHERE id = $2`,
		time.Now(), id,
	)
	if err != nil {
		return false
	}
	rows, _ := result.RowsAffected()
	return rows > 0
}

// ListWorkers returns all registered workers.
func (s *WorkerStoreDB) ListWorkers() []*Worker {
	rows, err := s.db.Query(
		`SELECT id, status, enrolled_at, last_heartbeat FROM workers ORDER BY enrolled_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var workers []*Worker
	for rows.Next() {
		var w Worker
		if err := rows.Scan(&w.ID, &w.Status, &w.EnrolledAt, &w.LastHeartbeat); err != nil {
			continue
		}
		workers = append(workers, &w)
	}
	return workers
}

// IsWorkerRevoked checks if a worker has been revoked.
func (s *WorkerStoreDB) IsWorkerRevoked(id string) bool {
	row := s.db.QueryRow(`SELECT status FROM workers WHERE id = $1`, id)
	var status string
	if err := row.Scan(&status); err != nil {
		return false
	}
	return status == "revoked"
}
