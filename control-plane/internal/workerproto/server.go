package workerproto

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/auth"
)

// WorkerServer implements the worker-facing API over mTLS.
type WorkerServer struct {
	tokenManager *auth.TokenManager
	workerStore  auth.WorkerStoreInterface
	mux          *http.ServeMux
}

// NewWorkerServer creates a new worker protocol server.
func NewWorkerServer(tm *auth.TokenManager, ws auth.WorkerStoreInterface) *WorkerServer {
	s := &WorkerServer{
		tokenManager: tm,
		workerStore:  ws,
		mux:          http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *WorkerServer) setupRoutes() {
	s.mux.HandleFunc("/v1/session", s.handleSession)
	s.mux.HandleFunc("/v1/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("/v1/resources", s.handleResourceReport)
	s.mux.HandleFunc("/v1/job/assign", s.handleJobAssignment)
	s.mux.HandleFunc("/v1/job/status", s.handleJobStatus)
	s.mux.HandleFunc("/v1/job/result", s.handleResultSubmission)
	s.mux.HandleFunc("/v1/job/cancel", s.handleCancelJob)
	s.mux.HandleFunc("/v1/environment", s.handleEnvironment)
}

// Start starts the worker mTLS server.
func (s *WorkerServer) Start(addr, certPath, keyPath string) error {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load server cert: %w", err)
	}

	caPEM, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return fmt.Errorf("failed to read CA cert: %w", err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caPEM)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}

	server := &http.Server{
		Addr:      addr,
		Handler:   s.mux,
		TLSConfig: tlsConfig,
	}

	log.Printf("Worker mTLS server starting on %s", addr)
	return server.ListenAndServeTLS("", "")
}

func (s *WorkerServer) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	workerID := r.TLS.PeerCertificates[0].Subject.CommonName

	worker, ok := s.workerStore.GetWorker(workerID)
	if !ok {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}
	if worker.Status == "revoked" {
		http.Error(w, "Worker has been revoked", http.StatusForbidden)
		return
	}

	sessionToken, err := s.tokenManager.IssueToken(workerID, "session", auth.SessionTokenLifetime)
	if err != nil {
		http.Error(w, "Failed to issue session token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_token": sessionToken,
		"expires_at":    time.Now().Add(auth.SessionTokenLifetime).Unix(),
	})
}

func (s *WorkerServer) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	workerID := r.TLS.PeerCertificates[0].Subject.CommonName

	worker, ok := s.workerStore.GetWorker(workerID)
	if !ok {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}
	if worker.Status == "revoked" {
		http.Error(w, "Worker has been revoked", http.StatusForbidden)
		return
	}

	s.workerStore.UpdateWorkerStatus(workerID, "active")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
		"message":  "Heartbeat accepted",
	})
}

func (s *WorkerServer) handleResourceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	var req struct {
		WorkerID     string  `json:"worker_id"`
		CPUPercent   float64 `json:"cpu_percent"`
		MemoryMB     int64   `json:"memory_mb"`
		DiskMB       int64   `json:"disk_mb"`
		ActiveJobs   int32   `json:"active_jobs"`
		GPUAvailable bool    `json:"gpu_available"`
		GPUModel     string  `json:"gpu_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.workerStore.UpdateWorkerStatus(req.WorkerID, "active")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
	})
}

func (s *WorkerServer) handleJobAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	var req struct {
		JobID       string   `json:"job_id"`
		WorkerID    string   `json:"worker_id"`
		Environment string   `json:"environment"`
		Payload     []byte   `json:"payload"`
		Entrypoint  string   `json:"entrypoint"`
		Inputs      []string `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Job %s assigned to worker %s", req.JobID, req.WorkerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
		"message":  "Job accepted",
	})
}

func (s *WorkerServer) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	var req struct {
		JobID    string `json:"job_id"`
		WorkerID string `json:"worker_id"`
		Status   string `json:"status"`
		Message  string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Job %s status update from worker %s: %s", req.JobID, req.WorkerID, req.Status)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
	})
}

func (s *WorkerServer) handleResultSubmission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	var req struct {
		JobID      string `json:"job_id"`
		WorkerID   string `json:"worker_id"`
		ResultData []byte `json:"result_data"`
		ExitCode   int32  `json:"exit_code"`
		Stdout     string `json:"stdout"`
		Stderr     string `json:"stderr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Result submitted for job %s from worker %s", req.JobID, req.WorkerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
		"message":  "Result accepted",
	})
}

func (s *WorkerServer) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	var req struct {
		JobID    string `json:"job_id"`
		WorkerID string `json:"worker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Cancel request for job %s from worker %s", req.JobID, req.WorkerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
		"message":  "Cancel request accepted",
	})
}

func (s *WorkerServer) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	var req struct {
		EnvironmentID string `json:"environment_id"`
		WorkerID      string `json:"worker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"environment_id": req.EnvironmentID,
		"hash":           "placeholder",
		"content":        []byte{},
	})
}
