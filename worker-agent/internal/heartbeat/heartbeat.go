package heartbeat

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// HeartbeatClient sends periodic heartbeats to the Control Plane.
type HeartbeatClient struct {
	controlPlaneURL string
	workerID       string
	sessionToken   string
	httpClient     *http.Client
	interval       time.Duration
	mu             sync.RWMutex
	running        bool
	stopCh         chan struct{}
}

// NewHeartbeatClient creates a new heartbeat client.
func NewHeartbeatClient(controlPlaneURL, workerID, sessionToken string, caCertPEM []byte, clientCert tls.Certificate) (*HeartbeatClient, error) {
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &HeartbeatClient{
		controlPlaneURL: controlPlaneURL,
		workerID:       workerID,
		sessionToken:   sessionToken,
		httpClient:     &http.Client{Transport: transport, Timeout: 10 * time.Second},
		interval:       5 * time.Second,
		stopCh:         make(chan struct{}),
	}, nil
}

// Start begins sending periodic heartbeats.
func (h *HeartbeatClient) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	go h.loop()
}

// Stop halts the heartbeat loop.
func (h *HeartbeatClient) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running {
		h.running = false
		close(h.stopCh)
	}
}

// UpdateSessionToken updates the session token used for heartbeats.
func (h *HeartbeatClient) UpdateSessionToken(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessionToken = token
}

func (h *HeartbeatClient) loop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			if err := h.sendHeartbeat(); err != nil {
				fmt.Printf("Heartbeat failed: %v\n", err)
			}
		}
	}
}

func (h *HeartbeatClient) sendHeartbeat() error {
	h.mu.RLock()
	token := h.sessionToken
	h.mu.RUnlock()

	reqBody, _ := json.Marshal(map[string]string{
		"worker_id":      h.workerID,
		"session_token":  token,
		"status":         "alive",
	})

	resp, err := h.httpClient.Post(
		h.controlPlaneURL+"/heartbeat",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unauthorized: %s", string(body))
	}

	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("forbidden (revoked): %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
