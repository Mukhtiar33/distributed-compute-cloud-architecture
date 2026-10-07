package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/distributedcompute/cloud/worker-agent/internal/agent"
	"github.com/distributedcompute/cloud/worker-agent/internal/config"
	"github.com/distributedcompute/cloud/worker-agent/internal/enroll"
	"github.com/distributedcompute/cloud/worker-agent/internal/heartbeat"
	"github.com/distributedcompute/cloud/worker-agent/internal/sandbox"
	"github.com/distributedcompute/cloud/worker-agent/internal/session"
)

func main() {
	cfg := config.Load()

	// Allow command-line override: worker-agent <control-plane-url>
	if len(os.Args) > 1 {
		cfg.ControlPlaneURL = os.Args[1]
	}

	// Daemonize: run as background service
	agent.Daemonize()

	log.Printf("Worker Agent starting (background service)...")
	log.Printf("Control Plane: %s", cfg.ControlPlaneURL)

	// Fetch CA certificate from Control Plane (public endpoint)
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
	enrollClient, err := enroll.NewEnrollClient(getPublicURL(cfg.ControlPlaneURL), caCertPEM)
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
		log.Printf("Enrolling as new worker: %s", workerID)
		enrollResult, err = enrollClient.Enroll(workerID)
		if err != nil {
			log.Fatalf("Enrollment failed: %v", err)
		}

		// Save credentials for future restarts
		if err := enrollClient.SaveCredentials(enrollResult, credentialsDir); err != nil {
			log.Printf("Warning: failed to save credentials: %v", err)
		}
		log.Printf("Enrollment successful")
	}

	// Load client certificate for mTLS
	clientCert, err := tls.X509KeyPair(enrollResult.ClientCertificate, enrollResult.ClientKey)
	if err != nil {
		log.Fatalf("Failed to load client certificate: %v", err)
	}

	// Get session token using mTLS identity
	sessionMgr, err := session.NewSessionManager(
		cfg.ControlPlaneURL,
		enrollResult.WorkerID,
		caCertPEM,
		clientCert,
	)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	sessionToken, err := sessionMgr.GetSessionToken()
	if err != nil {
		log.Fatalf("Failed to get session token: %v", err)
	}
	log.Printf("Session token acquired via mTLS")

	// Initialize sandbox manager (optional — Docker may not be available)
	sandboxMgr, err := sandbox.NewSandboxManager()
	if err != nil {
		log.Printf("Warning: Sandbox not available: %v", err)
	} else {
		log.Printf("Sandbox available: %v", sandboxMgr.IsAvailable())
		defer sandboxMgr.Close()
	}

	// Start heartbeat
	hbClient, err := heartbeat.NewHeartbeatClient(
		cfg.ControlPlaneURL,
		enrollResult.WorkerID,
		sessionToken,
		caCertPEM,
		clientCert,
	)
	if err != nil {
		log.Fatalf("Failed to create heartbeat client: %v", err)
	}

	hbClient.Start()
	log.Printf("Worker %s connected and heartbeating (background service)", enrollResult.WorkerID)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down Worker Agent...")
	hbClient.Stop()
}

func fetchCACert(controlPlaneURL string) ([]byte, error) {
	// The CA cert is served on a separate public HTTP port (8080)
	// so workers can fetch it without mTLS
	publicURL := controlPlaneURL
	publicURL = strings.Replace(publicURL, "https://", "http://", 1)
	publicURL = strings.Replace(publicURL, ":8443", ":8080", 1)

	resp, err := http.Get(publicURL + "/ca")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CA cert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func getPublicURL(controlPlaneURL string) string {
	publicURL := controlPlaneURL
	publicURL = strings.Replace(publicURL, "https://", "http://", 1)
	publicURL = strings.Replace(publicURL, ":8443", ":8080", 1)
	return publicURL
}
