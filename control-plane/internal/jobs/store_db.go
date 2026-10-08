package jobs

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/db"
)

// StoreDB is a PostgreSQL-backed job store.
type StoreDB struct {
	db *db.DB
}

// NewStoreDB creates a new database-backed job store.
func NewStoreDB(database *db.DB) *StoreDB {
	return &StoreDB{db: database}
}

// AddJob adds a job to the store.
func (s *StoreDB) AddJob(job *Job) {
	manifestJSON, _ := json.Marshal(job.Manifest)
	validationJSON, _ := json.Marshal(job.ValidationError)

	s.db.Exec(
		`INSERT INTO jobs (id, manifest, status, submitted_at, validation_errors)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET status = $3, manifest = $2, validation_errors = $5`,
		job.ID, manifestJSON, job.Status, job.SubmittedAt, validationJSON,
	)
}

// GetJob retrieves a job by ID.
func (s *StoreDB) GetJob(id string) (*Job, bool) {
	row := s.db.QueryRow(
		`SELECT id, manifest, status, submitted_at, validation_errors FROM jobs WHERE id = $1`,
		id,
	)

	var job Job
	var manifestJSON []byte
	var validationJSON []byte
	err := row.Scan(&job.ID, &manifestJSON, &job.Status, &job.SubmittedAt, &validationJSON)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	if len(manifestJSON) > 0 {
		json.Unmarshal(manifestJSON, &job.Manifest)
	}
	if len(validationJSON) > 0 {
		json.Unmarshal(validationJSON, &job.ValidationError)
	}

	return &job, true
}

// UpdateStatus updates a job's status.
func (s *StoreDB) UpdateStatus(id string, status JobStatus) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	return err
}

// ListJobs returns all jobs in the store.
func (s *StoreDB) ListJobs() []*Job {
	rows, err := s.db.Query(
		`SELECT id, manifest, status, submitted_at, validation_errors FROM jobs ORDER BY submitted_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var manifestJSON []byte
		var validationJSON []byte
		if err := rows.Scan(&job.ID, &manifestJSON, &job.Status, &job.SubmittedAt, &validationJSON); err != nil {
			continue
		}
		if len(manifestJSON) > 0 {
			json.Unmarshal(manifestJSON, &job.Manifest)
		}
		if len(validationJSON) > 0 {
			json.Unmarshal(validationJSON, &job.ValidationError)
		}
		jobs = append(jobs, &job)
	}
	return jobs
}

// GetJobsByStatus returns jobs filtered by status.
func (s *StoreDB) GetJobsByStatus(status JobStatus) []*Job {
	rows, err := s.db.Query(
		`SELECT id, manifest, status, submitted_at, validation_errors FROM jobs WHERE status = $1 ORDER BY submitted_at DESC`,
		status,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var manifestJSON []byte
		var validationJSON []byte
		if err := rows.Scan(&job.ID, &manifestJSON, &job.Status, &job.SubmittedAt, &validationJSON); err != nil {
			continue
		}
		if len(manifestJSON) > 0 {
			json.Unmarshal(manifestJSON, &job.Manifest)
		}
		if len(validationJSON) > 0 {
			json.Unmarshal(validationJSON, &job.ValidationError)
		}
		jobs = append(jobs, &job)
	}
	return jobs
}
