package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/distributedcompute/cloud/control-plane/internal/api"
	"github.com/distributedcompute/cloud/control-plane/internal/auth"
	"github.com/distributedcompute/cloud/control-plane/internal/jobs"
	"github.com/distributedcompute/cloud/control-plane/internal/registry"
)

// Server is the Control Plane HTTP server with separate public and worker listeners.
type Server struct {
	tokenManager *auth.TokenManager
	ca           *auth.CA
	workerStore  auth.WorkerStoreInterface
	jobStore     jobs.JobStoreInterface
	jobHandler   *api.JobHandler
	registry     *registry.Registry
	mux          *http.ServeMux
	httpServer   *http.Server
	workerServer *http.Server
}

// NewServer creates a new Control Plane server.
func NewServer(tm *auth.TokenManager, ca *auth.CA, workerStore auth.WorkerStoreInterface, jobStore jobs.JobStoreInterface, reg *registry.Registry) *Server {
	s := &Server{
		tokenManager: tm,
		ca:           ca,
		workerStore:  workerStore,
		jobStore:     jobStore,
		jobHandler:   api.NewJobHandler(jobStore),
		registry:     reg,
		mux:          http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Public endpoints (no mTLS required) — consumer-facing
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/ca", s.handleGetCA)
	s.mux.HandleFunc("/enroll", s.handleEnroll)
	s.mux.HandleFunc("/jobs", s.jobHandler.SubmitJob)
	s.mux.HandleFunc("/jobs/list", s.jobHandler.ListJobs)
	s.mux.HandleFunc("/jobs/get", s.jobHandler.GetJob)

	// Worker endpoints (mTLS required) — worker-facing
	s.mux.HandleFunc("/session", s.handleSession)
	s.mux.HandleFunc("/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("/revoke", s.handleRevoke)
	s.mux.HandleFunc("/workers", s.handleListWorkers)
	s.mux.HandleFunc("/environments", s.handleListEnvironments)
	s.mux.HandleFunc("/environments/get", s.handleGetEnvironment)
	s.mux.HandleFunc("/environments/pull", s.handlePullEnvironment)
}

// Start starts both the public HTTPS server and the worker mTLS server.
func (s *Server) Start(addr, workerAddr, certPath, keyPath string) error {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load server cert: %w", err)
	}

	// Load CA cert for client verification
	caPEM, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return fmt.Errorf("failed to read CA cert: %w", err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caPEM)

	// Public server — standard TLS, no client cert required
	publicTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	s.httpServer = &http.Server{
		Addr:      addr,
		Handler:   s.mux,
		TLSConfig: publicTLSConfig,
	}

	// Worker server — mTLS required
	workerTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}

	s.workerServer = &http.Server{
		Addr:      workerAddr,
		Handler:   s.mux,
		TLSConfig: workerTLSConfig,
	}

	// Start worker server in background
	go func() {
		log.Printf("Worker mTLS server starting on %s", workerAddr)
		if err := s.workerServer.ListenAndServeTLS("", ""); err != nil {
			log.Printf("Worker server error: %v", err)
		}
	}()

	log.Printf("Public HTTPS server starting on %s", addr)
	return s.httpServer.ListenAndServeTLS("", "")
}

// Stop gracefully shuts down both servers.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	if s.workerServer != nil {
		s.workerServer.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkerID string `json:"worker_id"`
		CSR      string `json:"csr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Input validation
	if req.WorkerID == "" || req.CSR == "" {
		http.Error(w, "worker_id and csr are required", http.StatusBadRequest)
		return
	}
	if len(req.WorkerID) > 128 {
		http.Error(w, "worker_id too long", http.StatusBadRequest)
		return
	}
	if len(req.CSR) > 100000 {
		http.Error(w, "CSR too large", http.StatusBadRequest)
		return
	}

	// Sign the CSR to create a client certificate
	clientCertPEM, err := s.ca.SignCSR([]byte(req.CSR), req.WorkerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("CSR signing failed: %v", err), http.StatusBadRequest)
		return
	}

	// Issue enrollment credential
	enrollmentCred, err := s.tokenManager.IssueToken(req.WorkerID, "enrollment", auth.EnrollmentTokenLifetime)
	if err != nil {
		http.Error(w, "Failed to issue enrollment credential", http.StatusInternalServerError)
		return
	}

	// Register worker
	s.workerStore.AddWorker(&auth.Worker{
		ID:         req.WorkerID,
		Status:     "enrolled",
		EnrolledAt: time.Now(),
	})

	log.Printf("Worker %s enrolled successfully", req.WorkerID)

	json.NewEncoder(w).Encode(map[string]string{
		"enrollment_credential": enrollmentCred,
		"client_certificate":    string(clientCertPEM),
	})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify mTLS peer certificate
	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	workerID := r.TLS.PeerCertificates[0].Subject.CommonName

	// Check worker exists and is not revoked
	worker, ok := s.workerStore.GetWorker(workerID)
	if !ok {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}
	if worker.Status == "revoked" {
		http.Error(w, "Worker has been revoked", http.StatusForbidden)
		return
	}

	// Issue session token based on mTLS identity
	sessionToken, err := s.tokenManager.IssueToken(workerID, "session", auth.SessionTokenLifetime)
	if err != nil {
		http.Error(w, "Failed to issue session token", http.StatusInternalServerError)
		return
	}

	log.Printf("Worker %s obtained session token via mTLS", workerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_token": sessionToken,
		"expires_at":    time.Now().Add(auth.SessionTokenLifetime).Unix(),
	})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify mTLS peer certificate
	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}

	workerID := r.TLS.PeerCertificates[0].Subject.CommonName

	// Check worker is not revoked
	worker, ok := s.workerStore.GetWorker(workerID)
	if !ok {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}
	if worker.Status == "revoked" {
		http.Error(w, "Worker has been revoked", http.StatusForbidden)
		return
	}

	// Update heartbeat
	s.workerStore.UpdateWorkerStatus(workerID, "active")

	log.Printf("Heartbeat received from worker %s", workerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": true,
		"message":  "Heartbeat accepted",
	})
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkerID string `json:"worker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !s.workerStore.RevokeWorker(req.WorkerID) {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}

	log.Printf("Worker %s revoked", req.WorkerID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"revoked": true,
	})
}

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	workers := s.workerStore.ListWorkers()
	json.NewEncoder(w).Encode(workers)
}

func (s *Server) handleGetCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Write(s.ca.GetCACertPEM())
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	envs := s.registry.List()
	json.NewEncoder(w).Encode(envs)
}

func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	env, ok := s.registry.Get(id)
	if !ok {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(env)
}

func (s *Server) handlePullEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EnvironmentID string `json:"environment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	env, ok := s.registry.Get(req.EnvironmentID)
	if !ok {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"environment_id": env.ID,
		"hash":           env.Hash,
		"content_path":   env.ContentPath,
	})
}
