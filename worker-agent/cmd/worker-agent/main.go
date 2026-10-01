package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distributedcompute/cloud/worker-agent/internal/config"
	"github.com/distributedcompute/cloud/worker-agent/internal/enroll"
	"github.com/distributedcompute/cloud/worker-agent/internal/heartbeat"
	"github.com/distributedcompute/cloud/worker-agent/internal/session"
)

func main() {
	cfg := config.Load()

	// Fetch CA certificate from Control Plane
	caCertPEM, err := fetchCACert(cfg.ControlPlaneURL)
	if err != nil {
		log.Fatalf("Failed to fetch CA cert: %v", err)
	}

	// Determine worker ID (from env or generate)
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}

	credentialsDir := cfg.CredentialsDir

	// Try to load existing credentials
	enrollClient, err := enroll.NewEnrollClient(cfg.ControlPlaneURL, caCertPEM)
	if err != nil {
		log.Fatalf("Failed to create enroll client: %v", err)
	}

	var enrollResult *enroll.EnrollResult

	// Check if we have saved credentials
	savedCreds, err := enrollClient.LoadCredentials(credentialsDir)
	if err == nil && savedCreds.WorkerID != "" {
		log.Printf("Loaded existing credentials for worker %s", savedCreds.WorkerID)
		enrollResult = savedCreds
	} else {
		// First run: enroll
		log.Printf("No existing credentials found, enrolling as new worker: %s", workerID)
		enrollResult, err = enrollClient.Enroll(workerID)
		if err != nil {
			log.Fatalf("Enrollment failed: %v", err)
		}

		// Save credentials for future restarts
		if err := enrollClient.SaveCredentials(enrollResult, credentialsDir); err != nil {
			log.Printf("Warning: failed to save credentials: %v", err)
		}
		log.Printf("Enrollment successful, credentials saved")
	}

	// Get session token
	sessionMgr, err := session.NewSessionManager(
		cfg.ControlPlaneURL,
		enrollResult.WorkerID,
		enrollResult.EnrollmentCredential,
		caCertPEM,
	)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	sessionToken, err := sessionMgr.GetSessionToken()
	if err != nil {
		log.Fatalf("Failed to get session token: %v", err)
	}
	log.Printf("Session token acquired")

	// Start heartbeat
	hbClient, err := heartbeat.NewHeartbeatClient(
		cfg.ControlPlaneURL,
		enrollResult.WorkerID,
		sessionToken,
		caCertPEM,
	)
	if err != nil {
		log.Fatalf("Failed to create heartbeat client: %v", err)
	}

	hbClient.Start()
	log.Printf("Worker %s enrolled and heartbeating", enrollResult.WorkerID)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down Worker Agent...")
	hbClient.Stop()
}

func fetchCACert(controlPlaneURL string) ([]byte, error) {
	resp, err := http.Get(controlPlaneURL + "/ca")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CA cert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
